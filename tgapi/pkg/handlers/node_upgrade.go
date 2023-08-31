package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/replicatedhq/kurl-testgrid/tgapi/pkg/logger"
	"github.com/replicatedhq/kurl-testgrid/tgapi/pkg/testinstance"
)

type CreateNodeUpgradeRequest struct {
	Command string `json:"command"`
}

type GetNodeUpgradeResponse struct {
	Command string `json:"command"`
}

func AddNodeUpgradeCommand(w http.ResponseWriter, r *http.Request) {
	joinCommandRequest := CreateNodeUpgradeRequest{}
	if err := json.NewDecoder(r.Body).Decode(&joinCommandRequest); err != nil {
		logger.Error(err)
		JSON(w, 400, nil)
		return
	}

	instanceID := mux.Vars(r)["instanceId"]
	nodeName := mux.Vars(r)["nodeName"]

	if err := testinstance.AddNodeUpgradeCommand(instanceID, nodeName, joinCommandRequest.Command); err != nil {
		logger.Error(err)
		JSON(w, 500, nil)
		return
	}

	JSON(w, 200, nil)
}

func GetNodeUpgradeCommand(w http.ResponseWriter, r *http.Request) {
	instanceID := mux.Vars(r)["instanceId"]
	nodeName := mux.Vars(r)["nodeName"]

	command, err := testinstance.GetNodeUpgradeCommand(instanceID, nodeName)
	if err != nil {
		logger.Error(err)
		JSON(w, 500, err)
		return
	}
	joinCommandResponse := GetNodeUpgradeResponse{}
	joinCommandResponse.Command = command
	JSON(w, 200, joinCommandResponse)
}
