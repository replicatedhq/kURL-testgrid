package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

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

	clientset, err := helpers.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}
	if err := deleteCancelledData(clientset, cancelledIDs); err != nil {
		return err
	}

	return nil
}

func getCancelledInstanceIDs(apiEndpoint string) (map[string]bool, error) {
	resp, err := http.DefaultClient.Get(fmt.Sprintf("%s/v1/instances/cancelled", apiEndpoint))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cancelled instances")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read body")
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
// Matching is by testid annotation, or by "{instanceId}-" name prefix for
// VMIs without annotations (e.g. sendlogs VMs).
func deleteCancelledVMIs(virtClient kubecli.KubevirtClient, cancelledIDs map[string]bool) error {
	vmiList, err := virtClient.VirtualMachineInstance(Namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to list vmis")
	}

	for _, vmi := range vmiList.Items {
		if !isCancelledResource(cancelledIDs, vmi.Name, vmi.Annotations[runnerVmi.TestIDAnnotation]) {
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

// deleteCancelledData deletes the PVCs, PVs and cloud-init secrets of
// cancelled instances immediately instead of waiting for the age-based
// CleanUpData loop.
func deleteCancelledData(clientset *kubernetes.Clientset, cancelledIDs map[string]bool) error {
	if err := deleteCancelledPVCsAndPVs(clientset, cancelledIDs); err != nil {
		return err
	}
	if err := deleteCancelledSecrets(clientset, cancelledIDs); err != nil {
		return err
	}
	return nil
}

func deleteCancelledPVCsAndPVs(clientset *kubernetes.Clientset, cancelledIDs map[string]bool) error {
	pvcs, err := clientset.CoreV1().PersistentVolumeClaims(Namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to get pvc list")
	}

	for _, pvc := range pvcs.Items {
		if !isCancelledResource(cancelledIDs, pvc.Name, "") {
			continue
		}

		// capture the bound PV before deleting the pvc; pv names are
		// pvc-<uuid> and cannot be derived from the instance id
		volumeName := pvc.Spec.VolumeName

		// strip finalizers first; an OpenEBS webhook bug can wedge them
		pvc.SetFinalizers(nil)
		if _, err := clientset.CoreV1().PersistentVolumeClaims(Namespace).Update(context.TODO(), &pvc, metav1.UpdateOptions{}); err != nil {
			fmt.Printf("Failed removing finalizers for cancelled pvc %s: %v\n", pvc.Name, err)
		}

		err := clientset.CoreV1().PersistentVolumeClaims(Namespace).Delete(context.TODO(), pvc.Name, metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			fmt.Printf("Failed to delete cancelled pvc %s: %v\n", pvc.Name, err)
			continue
		}
		fmt.Printf("Deleted cancelled pvc %s\n", pvc.Name)

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

	// only ever touch testgrid local-path volumes
	localPath := ""
	if pv.Spec.Local != nil && strings.HasPrefix(pv.Spec.Local.Path, "/var/openebs/local/pvc-") {
		localPath = pv.Spec.Local.Path
	}

	pv.SetFinalizers(nil)
	if _, err := clientset.CoreV1().PersistentVolumes().Update(context.TODO(), pv, metav1.UpdateOptions{}); err != nil {
		fmt.Printf("Failed removing finalizers for cancelled pv %s: %v\n", pv.Name, err)
	}

	if err := clientset.CoreV1().PersistentVolumes().Delete(context.TODO(), pv.Name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		fmt.Printf("Failed to delete cancelled pv %s: %v\n", pv.Name, err)
		return
	}
	fmt.Printf("Deleted cancelled pv %s\n", pv.Name)

	if localPath != "" {
		if err := os.RemoveAll(localPath); err != nil {
			fmt.Printf("Failed to delete %s: %v\n", localPath, err)
		}
	}
}

func deleteCancelledSecrets(clientset *kubernetes.Clientset, cancelledIDs map[string]bool) error {
	secrets, err := clientset.CoreV1().Secrets(Namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to get secrets list")
	}

	for _, sec := range secrets.Items {
		// testgrid secrets are named cloud-init-{instanceId}-{node}[-sendlogs]
		if !strings.HasPrefix(sec.Name, "cloud-init-") {
			continue
		}
		if !isCancelledResource(cancelledIDs, strings.TrimPrefix(sec.Name, "cloud-init-"), "") {
			continue
		}

		err := clientset.CoreV1().Secrets(Namespace).Delete(context.TODO(), sec.Name, metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			fmt.Printf("Failed to delete cancelled secret %s: %v\n", sec.Name, err)
		} else {
			fmt.Printf("Deleted cancelled secret %s\n", sec.Name)
		}
	}

	return nil
}

// isCancelledResource reports whether a resource belongs to a cancelled
// instance, by exact annotation match or "{instanceId}-" name prefix.
func isCancelledResource(cancelledIDs map[string]bool, name string, testIDAnnotation string) bool {
	if cancelledIDs[testIDAnnotation] {
		return true
	}
	for id := range cancelledIDs {
		if strings.HasPrefix(name, id+"-") {
			return true
		}
	}
	return false
}
