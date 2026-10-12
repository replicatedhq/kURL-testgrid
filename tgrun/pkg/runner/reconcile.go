package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
	tghandlers "github.com/replicatedhq/kurl-testgrid/tgapi/pkg/handlers"
	"github.com/replicatedhq/kurl-testgrid/tgrun/pkg/runner/helpers"
	"github.com/replicatedhq/kurl-testgrid/tgrun/pkg/runner/types"
	runnerVmi "github.com/replicatedhq/kurl-testgrid/tgrun/pkg/runner/vmi"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"kubevirt.io/client-go/kubecli"
)

// apiHTTPClient bounds API calls so a hung tgapi does not stall the shared
// cleanup goroutine (VMI timeout cleanup, data cleanup, metrics).
var apiHTTPClient = &http.Client{Timeout: 30 * time.Second}

// ReconcileCancelledVMIs polls the testgrid api for recently cancelled
// instances and deletes the corresponding local VMIs and their data volumes
// (PVCs, PVs, cloud-init secrets), so cancelled runs stop consuming hosts
// within one reconciliation interval instead of running until the 8h timeout.
func ReconcileCancelledVMIs(runnerOptions types.RunnerOptions) error {
	cancelledIDs, err := getCancelledInstanceIDs(runnerOptions.APIEndpoint)
	if err != nil {
		return errors.Wrap(err, "failed to get cancelled instances")
	}
	if len(cancelledIDs) == 0 {
		return nil
	}

	virtClient, err := helpers.GetKubevirtClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get kubevirt clientset")
	}
	if err := deleteCancelledVMIs(virtClient, cancelledIDs); err != nil {
		return err
	}

	// VMI deletion is async and virt-launcher may still have the disk mounted,
	// so leave the data of instances whose VMIs are still terminating for the
	// next pass instead of pulling the PVC out from under qemu.
	stillTerminating, err := instanceIDsWithVMIs(virtClient, cancelledIDs)
	if err != nil {
		return err
	}

	clientset, err := helpers.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}
	if err := deleteCancelledData(clientset, cancelledIDs, stillTerminating); err != nil {
		return err
	}

	return nil
}

func getCancelledInstanceIDs(apiEndpoint string) (map[string]bool, error) {
	resp, err := apiHTTPClient.Get(fmt.Sprintf("%s/v1/instances/cancelled", apiEndpoint))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cancelled instances")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read body")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("cancelled instances endpoint returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	listCancelledResponse := tghandlers.ListCancelledResponse{}
	if err := json.Unmarshal(body, &listCancelledResponse); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal %q", string(body))
	}

	cancelledIDs := map[string]bool{}
	for _, instance := range listCancelledResponse.Instances {
		cancelledIDs[instance.ID] = true
	}

	return cancelledIDs, nil
}

// deleteCancelledVMIs deletes every VMI belonging to a cancelled instance.
func deleteCancelledVMIs(virtClient kubecli.KubevirtClient, cancelledIDs map[string]bool) error {
	vmiList, err := virtClient.VirtualMachineInstance(Namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to list vmis")
	}

	for _, vmi := range vmiList.Items {
		if _, ok := matchCancelledInstance(cancelledIDs, vmi.Name, vmi.Annotations[runnerVmi.TestIDAnnotation]); !ok {
			continue
		}

		err := virtClient.VirtualMachineInstance(Namespace).Delete(context.TODO(), vmi.Name, metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			fmt.Printf("Failed to delete cancelled vmi %s: %v\n", vmi.Name, err)
		} else {
			fmt.Printf("Deleted cancelled vmi %s\n", vmi.Name)
		}
	}

	return nil
}

// instanceIDsWithVMIs returns the subset of cancelledIDs that still have a VMI
// object on this host. A VMI with a deletion timestamp is still terminating
// and its virt-launcher pod may still have the disk mounted.
func instanceIDsWithVMIs(virtClient kubecli.KubevirtClient, cancelledIDs map[string]bool) (map[string]bool, error) {
	vmiList, err := virtClient.VirtualMachineInstance(Namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list vmis")
	}

	remaining := map[string]bool{}
	for _, vmi := range vmiList.Items {
		if id, ok := matchCancelledInstance(cancelledIDs, vmi.Name, vmi.Annotations[runnerVmi.TestIDAnnotation]); ok {
			remaining[id] = true
		}
	}

	return remaining, nil
}

// deleteCancelledData deletes the PVCs, PVs and cloud-init secrets of
// cancelled instances immediately instead of waiting for the age-based
// CleanUpData loop. Instances in stillTerminating are skipped until their
// VMIs are gone.
func deleteCancelledData(clientset *kubernetes.Clientset, cancelledIDs, stillTerminating map[string]bool) error {
	if err := deleteCancelledPVCsAndPVs(clientset, cancelledIDs, stillTerminating); err != nil {
		return err
	}
	if err := deleteCancelledSecrets(clientset, cancelledIDs, stillTerminating); err != nil {
		return err
	}
	return nil
}

func deleteCancelledPVCsAndPVs(clientset *kubernetes.Clientset, cancelledIDs, stillTerminating map[string]bool) error {
	pvcs, err := clientset.CoreV1().PersistentVolumeClaims(Namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to get pvc list")
	}

	for _, pvc := range pvcs.Items {
		id, ok := matchCancelledInstance(cancelledIDs, pvc.Name, "")
		if !ok || stillTerminating[id] {
			continue
		}

		// capture the bound PV before deleting the pvc; pv names are
		// pvc-<uuid> and cannot be derived from the instance id
		volumeName := pvc.Spec.VolumeName

		if err := deletePVC(clientset, pvc); err != nil {
			fmt.Printf("Failed to delete cancelled pvc %s: %v\n", pvc.Name, err)
			continue
		}

		if volumeName != "" {
			deleteCancelledPV(clientset, volumeName)
		}
	}

	return nil
}

func deleteCancelledPV(clientset *kubernetes.Clientset, volumeName string) {
	pv, err := clientset.CoreV1().PersistentVolumes().Get(context.TODO(), volumeName, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			fmt.Printf("Failed to get pv %s: %v\n", volumeName, err)
		}
		return
	}

	if err := deletePV(clientset, *pv); err != nil {
		fmt.Printf("Failed to delete cancelled pv %s: %v\n", pv.Name, err)
	}
}

func deleteCancelledSecrets(clientset *kubernetes.Clientset, cancelledIDs, stillTerminating map[string]bool) error {
	secrets, err := clientset.CoreV1().Secrets(Namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to get secrets list")
	}

	for _, sec := range secrets.Items {
		// testgrid secrets are named cloud-init-{instanceId}-{node}[-sendlogs]
		if !strings.HasPrefix(sec.Name, "cloud-init-") {
			continue
		}
		id, ok := matchCancelledInstance(cancelledIDs, strings.TrimPrefix(sec.Name, "cloud-init-"), "")
		if !ok || stillTerminating[id] {
			continue
		}

		if err := deleteSecret(clientset, sec.Name); err != nil {
			fmt.Printf("Failed to delete cancelled secret %s: %v\n", sec.Name, err)
		}
	}

	return nil
}

// matchCancelledInstance returns the cancelled instance ID owning the named
// resource, by exact testid annotation match or "{instanceId}-" name prefix
// (the latter covers resources without annotations, e.g. sendlogs VMs).
func matchCancelledInstance(cancelledIDs map[string]bool, name string, testIDAnnotation string) (string, bool) {
	if cancelledIDs[testIDAnnotation] {
		return testIDAnnotation, true
	}
	for id := range cancelledIDs {
		if strings.HasPrefix(name, id+"-") {
			return id, true
		}
	}
	return "", false
}
