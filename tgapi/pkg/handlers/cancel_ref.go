package handlers

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/replicatedhq/kurl-testgrid/tgapi/pkg/logger"
	"github.com/replicatedhq/kurl-testgrid/tgapi/pkg/testinstance"
	"go.uber.org/zap"
)

type CancelRefResponse struct {
	PendingCancelled int64 `json:"pendingCancelled"`
	RunningCancelled int64 `json:"runningCancelled"`
}

// CancelRef cancels all pending and running instances of a run. Pending
// instances are skipped (removed from the runner queue) and running instances
// are marked cancelled; runner hosts poll ListCancelled and delete the
// associated VMs within one reconciliation interval.
func CancelRef(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}

	refID := mux.Vars(r)["refId"]

	logger.Debug("cancelRef",
		zap.String("refId", refID))

	pendingCancelled, runningCancelled, err := testinstance.CancelByRef(r.Context(), refID)
	if err != nil {
		logger.Error(err)
		JSON(w, 500, nil)
		return
	}

	logger.Debug("cancelled ref",
		zap.String("refId", refID),
		zap.Int64("pendingCancelled", pendingCancelled),
		zap.Int64("runningCancelled", runningCancelled))

	JSON(w, 200, CancelRefResponse{
		PendingCancelled: pendingCancelled,
		RunningCancelled: runningCancelled,
	})
}
