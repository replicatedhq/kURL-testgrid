package handlers

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/replicatedhq/kurl-testgrid/tgapi/pkg/logger"
	"github.com/replicatedhq/kurl-testgrid/tgapi/pkg/testinstance"
)

type GetInstanceSonobuoyResultsResponse struct {
	Results string `json:"results"`
}

func GetInstanceSonobuoyResults(w http.ResponseWriter, r *http.Request) {
	log.Printf("DEBUG: Start %s %s handler", r.Method, r.URL.Path)
	defer log.Printf("DEBUG: End %s %s handler", r.Method, r.URL.Path)

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}

	results, err := testinstance.GetSonobuoyResults(mux.Vars(r)["instanceId"])
	if err != nil {
		logger.Error(err)
		JSON(w, 500, nil)
		return
	}

	getInstanceSonobuoyResultsResponse := GetInstanceSonobuoyResultsResponse{}
	getInstanceSonobuoyResultsResponse.Results = results

	JSON(w, 200, getInstanceSonobuoyResultsResponse)
}
