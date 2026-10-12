package handlers

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/replicatedhq/kurl-testgrid/tgapi/pkg/logger"
	"github.com/replicatedhq/kurl-testgrid/tgapi/pkg/testinstance"
)

type GetInstanceCancelledResponse struct {
	Cancelled bool `json:"cancelled"`
}

// GetInstanceCancelled reports whether an instance has been cancelled.
// Runner hosts check this mid-execution to stop creating VMs for cancelled
// instances instead of waiting for the reconciliation loop to delete them.
func GetInstanceCancelled(w http.ResponseWriter, r *http.Request) {
	instanceID := mux.Vars(r)["instanceId"]

	cancelled, err := testinstance.IsCancelled(instanceID)
	if err != nil {
		logger.Error(err)
		JSON(w, 500, nil)
		return
	}

	JSON(w, 200, GetInstanceCancelledResponse{Cancelled: cancelled})
}
