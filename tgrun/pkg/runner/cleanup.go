package runner

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kurl-testgrid/tgrun/pkg/runner/helpers"
	runnerVmi "github.com/replicatedhq/kurl-testgrid/tgrun/pkg/runner/vmi"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	kubevirtv1 "kubevirt.io/api/core/v1"
)

const (
	cleanupAfterMinutes = 90
	timeoutAfterMinutes = 480
)

// CleanUpVMIs deletes "Succeeded" VMIs
func CleanUpVMIs() error {
	virtClient, err := helpers.GetKubevirtClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	vmiList, err := virtClient.VirtualMachineInstance(Namespace).List(context.TODO(), &metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to list vmis")
	}

	for _, vmi := range vmiList.Items {
		// cleanup succeeded VMIs
		// leaving them for a few hours for debug cases
		if vmi.Status.Phase == kubevirtv1.Succeeded && time.Since(vmi.CreationTimestamp.Time).Minutes() > cleanupAfterMinutes {
			err := virtClient.VirtualMachineInstance(Namespace).Delete(context.TODO(), vmi.Name, &metav1.DeleteOptions{})
			if err != nil {
				fmt.Printf("Failed to delete successful vmi %s: %v\n", vmi.Name, err)
			} else {
				fmt.Printf("Delete successful vmi %s\n", vmi.Name)
			}

			if !strings.HasSuffix(vmi.Name, "sendlogs") {
				// make a VMI to send the logs and nothing else
				err = runnerVmi.SendLogs(vmi.Annotations[runnerVmi.ApiEndpointAnnotation], vmi.Name)
				if err != nil {
					fmt.Printf("Failed to send logs of successful vmi %s: %v\n", vmi.Name, err)
				} else {
					fmt.Printf("Sent logs for successful vmi %s\n", vmi.Name)
				}
			}
		} else if vmi.Status.Phase == kubevirtv1.Succeeded && strings.HasSuffix(vmi.Name, "-initialprimary") {
			// cleanup secondary nodes if the primary has been done for 10 minutes

			completedAt := time.Time{}
			for _, condition := range vmi.Status.PhaseTransitionTimestamps {
				if condition.Phase == kubevirtv1.Succeeded {
					completedAt = condition.PhaseTransitionTimestamp.Time
				}
			}
			if time.Since(completedAt).Minutes() > 10 {
				vmiPrefix := strings.TrimSuffix(vmi.Name, "-initialprimary")
				for _, vmi2 := range vmiList.Items {
					if strings.HasPrefix(vmi2.Name, vmiPrefix) && vmi.Status.Phase == kubevirtv1.Running {
						err := virtClient.VirtualMachineInstance(Namespace).Delete(context.TODO(), vmi2.Name, &metav1.DeleteOptions{})
						if err != nil {
							fmt.Printf("Failed to delete secondary vmi %s: %v\n", vmi.Name, err)
						} else {
							fmt.Printf("Delete secondary vmi %s\n", vmi.Name)
						}
					}
				}
			}
		}

		// cleanup VMIs that are not running and have not succeeded after 1.5 hours (cleanupAfterMinutes)
		// or VMIs that are running and have not succeeded after 5 hours (timeoutAfterMinutes)
		// the in-script timeout for install is 30m, upgrade is 180m
		if cleanupIsTimeout(vmi) {
			if apiEndpoint := vmi.Annotations[runnerVmi.ApiEndpointAnnotation]; apiEndpoint != "" {
				url := fmt.Sprintf("%s/v1/instance/%s/finish", apiEndpoint, vmi.Annotations[runnerVmi.TestIDAnnotation])
				data := `{"success": false, "failureReason": "timeout"}`
				resp, err := http.Post(url, "application/json", strings.NewReader(data))
				if err != nil {
					fmt.Printf("Failed to post timeout failure to testgrid api for vmi %s: %v\n", vmi.Name, err)
				} else {
					if resp.StatusCode != 200 {
						fmt.Printf("Failed to post timeout failure to testgrid api for vmi %s: got %d\n", vmi.Name, resp.StatusCode)
					}

					resp.Body.Close()
				}
			}

			err = virtClient.VirtualMachineInstance(Namespace).Delete(context.TODO(), vmi.Name, &metav1.DeleteOptions{})
			if err != nil {
				fmt.Printf("Failed to delete long-running vmi %s: %v\n", vmi.Name, err)
			} else {
				fmt.Printf("Delete long-running vmi %s\n", vmi.Name)
			}

			if !strings.HasSuffix(vmi.Name, "sendlogs") {
				// make a VMI to send the logs and nothing else
				err = runnerVmi.SendLogs(vmi.Annotations[runnerVmi.ApiEndpointAnnotation], vmi.Name)
				if err != nil {
					fmt.Printf("Failed to send logs of deleted long-running vmi %s: %v\n", vmi.Name, err)
				} else {
					fmt.Printf("Sent logs for long-running vmi %s\n", vmi.Name)
				}
			}
		}
	}

	return nil
}

// cleanup VMIs that are not running and have not succeeded after 1.5 hours (cleanupAfterMinutes)
// or VMIs that are running and have not succeeded after 8 hours (timeoutAfterMinutes)
func cleanupIsTimeout(vmi kubevirtv1.VirtualMachineInstance) bool {
	if vmi.Status.Phase != kubevirtv1.Running && vmi.Status.Phase != kubevirtv1.Succeeded && time.Since(vmi.CreationTimestamp.Time).Minutes() > cleanupAfterMinutes {
		return true
	}
	if vmi.Status.Phase == kubevirtv1.Running && time.Since(vmi.CreationTimestamp.Time).Minutes() > timeoutAfterMinutes {
		return true
	}
	return false
}

// CleanUpData cleans stale PV/PVC/Secrets and localpath to reclaim space
func CleanUpData() error {
	clientset, err := helpers.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	if err := cleanupPVCs(clientset); err != nil {
		return err
	}
	if err := cleanupPVs(clientset); err != nil {
		return err
	}
	if err := cleanupSecrets(clientset); err != nil {
		return err
	}
	return nil
}

func cleanupPVCs(clientset *kubernetes.Clientset) error {
	pvcs, err := clientset.CoreV1().PersistentVolumeClaims(Namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to get pvc list")
	}

	// NOTE: OpenEbs webhook should be absent. For LocalPV volumes it has a bug preventing api calls, the webhook is only needed for cStor.
	for _, pvc := range pvcs.Items {
		// clean pvc older timeoutAfterMinutes+10 minutes
		if time.Since(pvc.CreationTimestamp.Time).Minutes() > timeoutAfterMinutes+10 {
			pvc.ObjectMeta.SetFinalizers(nil)
			p, err := clientset.CoreV1().PersistentVolumeClaims(Namespace).Update(context.TODO(), &pvc, metav1.UpdateOptions{})
			if err != nil {
				fmt.Printf("Failed removing finalizers for pvc %s; EROOR: %s\n", p.Name, err)
			} else {
				fmt.Printf("Removed finalizers on %s\n", p.Name)
			}

			err = clientset.CoreV1().PersistentVolumeClaims(Namespace).Delete(context.TODO(), pvc.Name, metav1.DeleteOptions{})
			if err != nil {
				fmt.Printf("Failed to delete pvc %s: %v\n", pvc.Name, err)
			} else {
				fmt.Printf("Deleted pvc %s\n", pvc.Name)
			}
		}
	}
	return nil
}

func cleanupPVs(clientset *kubernetes.Clientset) error {
	pvs, err := clientset.CoreV1().PersistentVolumes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to get pv list")
	}

	// finalizers might get pv in a stack state
	for _, pv := range pvs.Items {
		localPath := pv.Spec.Local.Path
		// deleting PVs older than timeoutAfterMinutes+20 minutes
		if time.Since(pv.CreationTimestamp.Time).Minutes() > timeoutAfterMinutes+20 && pv.ObjectMeta.DeletionTimestamp == nil {
			err := clientset.CoreV1().PersistentVolumes().Delete(context.TODO(), pv.Name, metav1.DeleteOptions{})
			if err != nil {
				fmt.Printf("Failed to delete pv %s; ERROR: %s\n", pv.Name, err)
			} else {
				fmt.Printf("Deleted pv %s\n", pv.Name)
			}
			// Image file gets deleted and the space is reclamed on pv deletion
			// However local directory is left with tmpimage file, removing
			// 523M    /var/openebs/local/pvc-xxx
			err = os.RemoveAll(localPath)
			if err != nil {
				fmt.Printf("Failed to delete %s; ERROR: %s\n", localPath, err)
			}
		} else if pv.ObjectMeta.DeletionTimestamp != nil {
			// cleaning pv stack in Terminating state
			pv.ObjectMeta.SetFinalizers(nil)
			p, err := clientset.CoreV1().PersistentVolumes().Update(context.TODO(), &pv, metav1.UpdateOptions{})
			if err != nil {
				fmt.Printf("Failed removing finalizers for pv %s; EROOR: %s\n", p.Name, err)
			} else {
				fmt.Printf("Removed finalizers on %s, localPath: %s\n", p.Name, localPath)
			}
		}
	}
	return nil
}

// CleanUpSecrets removes stale cloud-init configurations stored in secrets and prevents errors on interrupted runs
func cleanupSecrets(clientset *kubernetes.Clientset) error {
	secrets, err := clientset.CoreV1().Secrets(Namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to get secrets list")
	}

	for _, sec := range secrets.Items {
		// Delete stale secrets older than timeoutAfterMinutes+60 minutes
		if strings.HasPrefix(sec.Name, "cloud") && time.Since(sec.CreationTimestamp.Time).Minutes() > timeoutAfterMinutes+60 {
			err := clientset.CoreV1().Secrets(Namespace).Delete(context.TODO(), sec.Name, metav1.DeleteOptions{})
			if err != nil {
				fmt.Printf("Failed to delete secret %s; ERROR: %s\n", sec.Name, err)
			} else {
				fmt.Printf("Deleted stale secret %s\n", sec.Name)
			}
		}
	}
	return nil
}
