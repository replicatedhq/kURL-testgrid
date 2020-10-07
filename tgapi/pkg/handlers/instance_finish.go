package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/replicatedhq/kurl/testgrid/tgapi/pkg/logger"
	"github.com/replicatedhq/kurl/testgrid/tgapi/pkg/testinstance"
	"go.uber.org/zap"
)

type FinishInstanceRequest struct {
	Success bool `json:"success"`
}

func FinishInstance(w http.ResponseWriter, r *http.Request) {
	finishInstanceRequest := FinishInstanceRequest{}
	if err := json.NewDecoder(r.Body).Decode(&finishInstanceRequest); err != nil {
		logger.Error(err)
		JSON(w, 500, nil)
		return
	}

	instanceID := mux.Vars(r)["instanceId"]

	logger.Debug("finishInstance",
		zap.String("instanceId", instanceID))

	if err := testinstance.SetInstanceFinishedAndSuccess(instanceID, finishInstanceRequest.Success); err != nil {
		logger.Error(err)
		JSON(w, 500, nil)
		return
	}

	JSON(w, 200, nil)
}
