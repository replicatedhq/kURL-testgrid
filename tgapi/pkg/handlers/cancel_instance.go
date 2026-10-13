package handlers

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/replicatedhq/kurl-testgrid/tgapi/pkg/logger"
	"github.com/replicatedhq/kurl-testgrid/tgapi/pkg/testinstance"
	"go.uber.org/zap"
)

// CancelInstance cancels a single pending or running instance.
// See CancelRef for the state transition semantics.
func CancelInstance(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}

	instanceID := mux.Vars(r)["instanceId"]

	logger.Debug("cancelInstance",
		zap.String("instanceId", instanceID))

	if err := testinstance.CancelInstance(r.Context(), instanceID); err != nil {
		logger.Error(err)
		JSON(w, 500, nil)
		return
	}

	JSON(w, 200, nil)
}
