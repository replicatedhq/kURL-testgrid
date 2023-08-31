package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/replicatedhq/kurl-testgrid/tgapi/pkg/logger"
	"github.com/replicatedhq/kurl-testgrid/tgapi/pkg/testinstance"
)

type CreateJoinCommandRequest struct {
	PrimaryJoin   string `json:"primaryJoin"`
	SecondaryJoin string `json:"secondaryJoin"`
}

type GetJoinCommandResponse struct {
	PrimaryJoin   string `json:"primaryJoin"`
	SecondaryJoin string `json:"secondaryJoin"`
}

func AddNodeJoinCommand(w http.ResponseWriter, r *http.Request) {
	joinCommandRequest := CreateJoinCommandRequest{}
	if err := json.NewDecoder(r.Body).Decode(&joinCommandRequest); err != nil {
		logger.Error(err)
		JSON(w, 400, nil)
		return
	}

	instanceID := mux.Vars(r)["instanceId"]

	if err := testinstance.AddNodeJoinCommand(instanceID, joinCommandRequest.PrimaryJoin, joinCommandRequest.SecondaryJoin); err != nil {
		logger.Error(err)
		JSON(w, 500, nil)
		return
	}

	JSON(w, 200, nil)
}

func GetNodeJoinCommand(w http.ResponseWriter, r *http.Request) {
	instanceID := mux.Vars(r)["instanceId"]

	primaryJoin, secondaryJoin, err := testinstance.GetNodeJoinCommand(instanceID)
	if err != nil {
		logger.Error(err)
		JSON(w, 500, err)
		return
	}
	joinCommandResponse := GetJoinCommandResponse{}
	joinCommandResponse.PrimaryJoin = primaryJoin
	joinCommandResponse.SecondaryJoin = secondaryJoin
	JSON(w, 200, joinCommandResponse)
}
