package handlers

import (
	"net/http"

	"github.com/replicatedhq/kurl-testgrid/tgapi/pkg/logger"
	"github.com/replicatedhq/kurl-testgrid/tgapi/pkg/testinstance"
)

type CancelledInstance struct {
	ID string `json:"id"`
}

type ListCancelledResponse struct {
	Instances []CancelledInstance `json:"instances"`
}

// ListCancelled returns instances cancelled within the reconciliation window.
// Runner hosts poll this and delete local VMs/PVCs/secrets whose instance has
// been cancelled.
func ListCancelled(w http.ResponseWriter, r *http.Request) {
	ids, err := testinstance.ListRecentlyCancelledIDs(r.Context())
	if err != nil {
		logger.Error(err)
		JSON(w, 500, nil)
		return
	}

	listCancelledResponse := ListCancelledResponse{
		Instances: []CancelledInstance{},
	}
	for _, id := range ids {
		listCancelledResponse.Instances = append(listCancelledResponse.Instances, CancelledInstance{ID: id})
	}

	JSON(w, 200, listCancelledResponse)
}
