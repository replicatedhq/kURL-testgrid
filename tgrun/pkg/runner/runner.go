package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/pkg/errors"
	tghandlers "github.com/replicatedhq/kurl-testgrid/tgapi/pkg/handlers"
	"github.com/replicatedhq/kurl-testgrid/tgrun/pkg/runner/helpers"
	"github.com/replicatedhq/kurl-testgrid/tgrun/pkg/runner/types"
	"github.com/replicatedhq/kurl-testgrid/tgrun/pkg/runner/vmi"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Run(singleTest types.SingleRun, uploadProxyURL, tempDir string) error {
	err := execute(singleTest, uploadProxyURL, tempDir)

	if err != nil {
		fmt.Println("execute failed")
		fmt.Println("  ID:", singleTest.ID)
		fmt.Println("  REF:", singleTest.KurlRef)
		fmt.Println("  ERROR:", err)
		if reportError := reportFailed(singleTest, err); reportError != nil {
			return errors.Wrapf(err, "failed to report test failed with error %s", reportError.Error())
		}
		return err
	}

	return nil
}

func reportStarted(singleTest types.SingleRun) error {
	startInstanceRequest := tghandlers.StartInstanceRequest{
		OSName:    singleTest.OperatingSystemName,
		OSVersion: singleTest.OperatingSystemVersion,
		OSImage:   singleTest.OperatingSystemImage,

		Memory: singleTest.Memory,
		CPU:    singleTest.CPU,

		KurlSpec: singleTest.KurlYAML,
		KurlRef:  singleTest.KurlRef,
		KurlURL:  singleTest.KurlURL,
	}

	b, err := json.Marshal(startInstanceRequest)
	if err != nil {
		return errors.Wrap(err, "failed to marshal request")
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/instance/%s/start", singleTest.TestGridAPIEndpoint, singleTest.ID), bytes.NewReader(b))
	if err != nil {
		return errors.Wrap(err, "failed to create request")
	}
	req.Header.Set("Content-Type", "application/json")
	_, err = http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to execute request")
	}

	return nil
}

// reportFailed reports a test failure to the testgrid api
// the error and singleTest contents will be included in the reported logs
func reportFailed(singleTest types.SingleRun, testErr error) error {
	hostname, err := os.Hostname()
	if err != nil {
		fmt.Printf("failed to get hostname: %s\n", err)
		hostname = "unknown"
	}

	errorString := fmt.Sprintf("Test failed to start on %q with error: %s\n\n", hostname, testErr.Error())
	testSpecs, err := json.Marshal(singleTest)
	if err != nil {
		fmt.Printf("failed to marshal singleTest: %s\n", err)
		testSpecs = []byte("failed to marshal singleTest")
	}

	errorString = errorString + string(testSpecs)

	req, err := http.NewRequest("PUT", fmt.Sprintf("%s/v1/instance/%s/logs", singleTest.TestGridAPIEndpoint, singleTest.ID), bytes.NewReader([]byte(errorString)))
	if err != nil {
		return errors.Wrap(err, "failed to create logs request")
	}
	_, err = http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to execute logs request")
	}

	failureRequest := tghandlers.FinishInstanceRequest{
		Success:       false,
		FailureReason: "failed to create VMI",
	}

	b, err := json.Marshal(failureRequest)
	if err != nil {
		return errors.Wrap(err, "failed to marshal request")
	}

	req, err = http.NewRequest("POST", fmt.Sprintf("%s/v1/instance/%s/finish", singleTest.TestGridAPIEndpoint, singleTest.ID), bytes.NewReader(b))
	if err != nil {
		return errors.Wrap(err, "failed to create finish request")
	}
	req.Header.Set("Content-Type", "application/json")
	_, err = http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to execute finish request")
	}

	return nil
}

// pathify OS image by removing non-alphanumeric characters
func urlToPath(url string) string {
	return regexp.MustCompile(`[^a-zA-Z0-9]`).ReplaceAllString(url, "")
}

func execute(singleTest types.SingleRun, uploadProxyURL, tempDir string) error {
	osImagePath := urlToPath(singleTest.OperatingSystemImage)

	_, err := os.Stat(filepath.Join(tempDir, osImagePath))
	if err != nil {
		fmt.Printf("  [downloading from %s]\n", singleTest.OperatingSystemImage)

		// Download the img
		resp, err := http.Get(singleTest.OperatingSystemImage)
		if err != nil {
			return errors.Wrap(err, "failed to get")
		}
		defer resp.Body.Close()

		// Create the file
		out, err := os.Create(filepath.Join(tempDir, osImagePath))
		if err != nil {
			return errors.Wrap(err, "failed to create image file")
		}
		defer out.Close()

		// Write the body to file
		_, err = io.Copy(out, resp.Body)
		if err != nil {
			return errors.Wrap(err, "failed to save vm image")
		}

		fmt.Printf("   [image downloaded]\n")
	} else {
		fmt.Printf("  [using existng image on disk at %s for %s]\n", filepath.Join(tempDir, osImagePath), singleTest.OperatingSystemImage)
	}

	// wait for there to be enough resources for all nodes before creating them
	requiredCPU, requiredMemory := requiredResources(singleTest)
	fmt.Printf("  [waiting for %d CPUs and %dGB to be available]\n", requiredCPU/1000, requiredMemory/1024/1024)
	for {
		availableResources, err := areResourcesAvailable(singleTest)
		if err != nil {
			log.Println(errors.Wrap(err, "failed to check if there are sufficient resources for the cluster"))
		}
		if availableResources {
			break
		}
		time.Sleep(sleepTime)
	}

	// create initial primary node
	if err = vmi.Create(singleTest, vmi.InitPrimaryNode, vmi.InitPrimaryNode, tempDir, osImagePath, uploadProxyURL); err != nil {
		return errors.Wrap(err, "failed to create vm for init primary node")
	}

	// always schedule initial primary before additional nodes
	for {
		canSchedule, err := canScheduleNewVM()
		if err != nil {
			log.Println(errors.Wrap(err, "failed to check if can schedule"))
		}
		if canSchedule {
			break
		}
		time.Sleep(sleepTime)
	}

	// create primary nodes where we exclude the initial primary node so I will start with 1
	for i := 1; i < singleTest.NumPrimaryNodes; i++ {
		primaryNodeName := fmt.Sprintf("%s-%d", vmi.PrimaryNode, i)
		fmt.Println("  [creating primary node", primaryNodeName, "]")
		if err = vmi.Create(singleTest, primaryNodeName, vmi.PrimaryNode, tempDir, osImagePath, uploadProxyURL); err != nil {
			return errors.Wrap(err, "failed to create vm for primary node "+strconv.Itoa(i))
		}
	}
	// create secondary nodes
	for i := 0; i < singleTest.NumSecondaryNodes; i++ {
		secondaryNodeName := fmt.Sprintf("%s-%d", vmi.SecondaryNode, i)
		fmt.Println("  [creating secondary node", secondaryNodeName, "]")
		if err = vmi.Create(singleTest, secondaryNodeName, vmi.SecondaryNode, tempDir, osImagePath, uploadProxyURL); err != nil {
			return errors.Wrap(err, "failed to create vm for secondary node "+strconv.Itoa(i))
		}
	}

	// mark the instance started
	// we do this after the data volume is uploaded
	if err := reportStarted(singleTest); err != nil {
		return errors.Wrap(err, "failed to report test started")
	}

	return nil
}

// determines the total resources used by a test run, and returns the
// mCPU and kb of memory required
func requiredResources(singleTest types.SingleRun) (int64, int64) {
	numNodes := int64(0)
	if singleTest.NumPrimaryNodes == 0 { // there is always at least one primary node
		numNodes = int64(0) + int64(singleTest.NumSecondaryNodes)
	} else {
		numNodes = int64(singleTest.NumPrimaryNodes + singleTest.NumSecondaryNodes)
	}

	testCPU := resource.MustParse(singleTest.CPU)
	testMemory := resource.MustParse(singleTest.Memory)

	requiredCPUs := numNodes * testCPU.MilliValue()
	requiredMemory := numNodes * testMemory.Value()

	return requiredCPUs, requiredMemory
}

// check if we have enough CPU and memory available in the cluster
// this would be totalNodeCount * cpu/memory per node
// we need to check ahead of scheduling so that nodes don't timeout while others have yet to be created
func areResourcesAvailable(singleTest types.SingleRun) (bool, error) {
	clientset, err := helpers.GetClientset()
	if err != nil {
		return false, errors.Wrap(err, "failed to get clientset")
	}

	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return false, errors.Wrap(err, "failed to list nodes")
	}

	availableCPUs := int64(0)
	availableMemory := int64(0)
	for _, node := range nodes.Items {
		availableCPUs += node.Status.Allocatable.Cpu().MilliValue()
		availableMemory += node.Status.Allocatable.Memory().Value()
	}

	requiredCPUs, requiredMemory := requiredResources(singleTest)

	if availableCPUs < requiredCPUs {
		return false, nil
	}

	if availableMemory < requiredMemory {
		return false, nil
	}
	return true, nil
}
