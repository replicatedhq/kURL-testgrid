package handlers

import (
	"io"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/replicatedhq/kurl-testgrid/tgapi/pkg/logger"
	"github.com/replicatedhq/kurl-testgrid/tgapi/pkg/testinstance"
)

func InstanceLogs(w http.ResponseWriter, r *http.Request) {
	log.Printf("DEBUG: Start %s %s handler", r.Method, r.URL.Path)
	defer log.Printf("DEBUG: End %s %s handler", r.Method, r.URL.Path)

	logs, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error(err)
		JSON(w, 500, nil)
		return
	}

	instanceId := mux.Vars(r)["instanceId"]
	if err := testinstance.SetInstanceLogs(instanceId, logs); err != nil {
		logger.Error(err)
		JSON(w, 500, nil)
		return
	}

	JSON(w, 200, nil)
}
