package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
	"github.com/replicatedhq/kurl-testgrid/tgapi/pkg/logger"
	"github.com/replicatedhq/kurl-testgrid/tgapi/pkg/testinstance"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"kubevirt.io/client-go/kubecli"
)

type CancelInstancesResponse struct {
	Cancelled   int      `json:"cancelled"`
	DeletedVMIs []string `json:"deletedVMIs,omitempty"`
	VMIErrors   []string `json:"vmiErrors,omitempty"`
	K8sError    string   `json:"k8sError,omitempty"`
}

// CancelInstances cancels all running (unfinished) instances for a ref and attempts
// to delete their corresponding KubeVirt VMIs in the cluster.
func CancelInstances(w http.ResponseWriter, r *http.Request) {
	refID := mux.Vars(r)["refId"]

	logger.Debug("cancelInstances",
		zap.String("refId", refID))

	// 1. Mark all unfinished instances as cancelled in the database
	ids, err := testinstance.CancelRunningByRef(refID)
	if err != nil {
		logger.Error(err)
		JSON(w, 500, nil)
		return
	}

	resp := CancelInstancesResponse{
		Cancelled: len(ids),
	}

	if len(ids) == 0 {
		JSON(w, 200, resp)
		return
	}

	// 2. Attempt to delete corresponding VMIs in the cluster
	virtClient, k8sErr := getKubevirtClient()
	if k8sErr != nil {
		resp.K8sError = k8sErr.Error()
		JSON(w, 200, resp)
		return
	}

	vmiList, err := virtClient.VirtualMachineInstance("default").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		resp.K8sError = fmt.Sprintf("failed to list vmis: %v", err)
		JSON(w, 200, resp)
		return
	}

	for _, vmi := range vmiList.Items {
		for _, id := range ids {
			if strings.Contains(vmi.Name, id) {
				err := virtClient.VirtualMachineInstance("default").Delete(context.TODO(), vmi.Name, metav1.DeleteOptions{})
				if err != nil {
					resp.VMIErrors = append(resp.VMIErrors, fmt.Sprintf("failed to delete vmi %s: %v", vmi.Name, err))
				} else {
					resp.DeletedVMIs = append(resp.DeletedVMIs, vmi.Name)
				}
				break
			}
		}
	}

	JSON(w, 200, resp)
}

func getKubevirtClient() (kubecli.KubevirtClient, error) {
	var config *rest.Config
	var err error

	// Try in-cluster config first (when running as a pod)
	config, err = rest.InClusterConfig()
	if err != nil {
		// Fall back to local kubeconfig
		kubeconfig := filepath.Join(homeDir(), ".kube", "config")
		if os.Getenv("KUBECONFIG") != "" {
			kubeconfig = os.Getenv("KUBECONFIG")
		}
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}
	}

	return kubecli.GetKubevirtClientFromRESTConfig(config)
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE")
}
