package handlers

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/replicatedhq/kurl-testgrid/tgapi/pkg/logger"
	"github.com/replicatedhq/kurl-testgrid/tgapi/pkg/testinstance"
	"go.uber.org/zap"
)

func InstanceSonobuoyResults(w http.ResponseWriter, r *http.Request) {
	log.Printf("DEBUG: Start %s %s handler %d", r.Method, r.URL.Path, r.ContentLength)
	defer log.Printf("DEBUG: End %s %s handler %d", r.Method, r.URL.Path, r.ContentLength)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error(err)
		JSON(w, 500, nil)
		return
	}

	instanceID := mux.Vars(r)["instanceId"]

	logger.Debug("instanceSonobuoyResults",
		zap.String("instanceId", instanceID))

	if err := testinstance.SetInstanceSonobuoyResults(instanceID, body); err != nil {
		logger.Error(err)
		JSON(w, 500, nil)
		return
	}

	isSuccess := false
	scanner := bufio.NewScanner(bytes.NewReader(body))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Status:") {
			isSuccess = strings.HasSuffix(line, "passed")
			if !isSuccess {
				break
			}
		}
	}

	failureReason := ""
	if !isSuccess {
		failureReason = "sonobuoy_results"
	}

	if err := testinstance.SetInstanceFinishedAndSuccess(instanceID, isSuccess, failureReason); err != nil {
		logger.Error(err)
		JSON(w, 500, nil)
		return
	}

	JSON(w, 200, nil)
}
