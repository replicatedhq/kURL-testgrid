package handlers

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/replicatedhq/kurl-testgrid/tgapi/pkg/logger"
	"github.com/replicatedhq/kurl-testgrid/tgapi/pkg/testinstance"
)

type GetInstanceLogsResponse struct {
	Logs string `json:"logs"`
}

func GetInstanceLogs(w http.ResponseWriter, r *http.Request) {
	log.Printf("DEBUG: Start %s %s handler", r.Method, r.URL.Path)
	defer log.Printf("DEBUG: End %s %s handler", r.Method, r.URL.Path)

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}

	logs, err := testinstance.GetLogs(mux.Vars(r)["instanceId"])
	if err != nil {
		logger.Error(err)
		JSON(w, 500, nil)
		return
	}

	getInstanceLogsResponse := GetInstanceLogsResponse{}
	getInstanceLogsResponse.Logs = logs

	JSON(w, 200, getInstanceLogsResponse)
}
