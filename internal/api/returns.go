package api

import (
	"encoding/json"
	"log"
	"net/http"

	returnpkg "github.com/checkstream/checkstream/internal/return_"
)

// ReturnsHandler handles return/reversal HTTP requests.
type ReturnsHandler struct {
	returnSvc *returnpkg.Service
}

// NewReturnsHandler creates a new ReturnsHandler.
func NewReturnsHandler(returnSvc *returnpkg.Service) *ReturnsHandler {
	return &ReturnsHandler{returnSvc: returnSvc}
}

// ProcessReturn handles POST /returns.
func (h *ReturnsHandler) ProcessReturn(w http.ResponseWriter, r *http.Request) {
	var req returnpkg.ReturnRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.TransferID == "" {
		writeError(w, http.StatusBadRequest, "transfer_id is required")
		return
	}

	result, err := h.returnSvc.ProcessReturn(&req)
	if err != nil {
		log.Printf("returns: %v", err)
		// Check for not found vs other errors
		if result == nil {
			writeError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to process return")
		return
	}

	writeJSON(w, http.StatusOK, result)
}
