package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/checkstream/checkstream/internal/funding"
	"github.com/checkstream/checkstream/internal/ledger"
	"github.com/checkstream/checkstream/internal/operator"
	"github.com/checkstream/checkstream/internal/transfer"
)

// OperatorHandler handles operator workflow HTTP requests.
type OperatorHandler struct {
	operatorRepo *operator.Repository
	transferRepo *transfer.Repository
	ledgerSvc    *ledger.Service
	fundingCfg   *funding.Config
}

// NewOperatorHandler creates a new OperatorHandler.
func NewOperatorHandler(
	operatorRepo *operator.Repository,
	transferRepo *transfer.Repository,
	ledgerSvc *ledger.Service,
	fundingCfg *funding.Config,
) *OperatorHandler {
	return &OperatorHandler{
		operatorRepo: operatorRepo,
		transferRepo: transferRepo,
		ledgerSvc:    ledgerSvc,
		fundingCfg:   fundingCfg,
	}
}

// QueueItem is a transfer enriched with risk scores for the operator queue.
type QueueItem struct {
	*transfer.Transfer
	IQScore        float64 `json:"iq_score"`
	MICRConfidence float64 `json:"micr_confidence"`
}

// Queue handles GET /operator/queue.
func (h *OperatorHandler) Queue(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	dateFilter := q.Get("date")
	accountFilter := q.Get("account")
	amountMinStr := q.Get("amount_min")
	amountMaxStr := q.Get("amount_max")

	var amountMin, amountMax float64
	if amountMinStr != "" {
		amountMin, _ = strconv.ParseFloat(amountMinStr, 64)
	}
	if amountMaxStr != "" {
		amountMax, _ = strconv.ParseFloat(amountMaxStr, 64)
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	transfers, err := h.operatorRepo.ListFlaggedTransfers(dateFilter, accountFilter, amountMin, amountMax, limit, offset)
	if err != nil {
		log.Printf("operator queue: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list flagged transfers")
		return
	}

	items := make([]QueueItem, 0, len(transfers))
	for _, t := range transfers {
		item := QueueItem{Transfer: t}
		// Parse vendor scores from vendor_response JSON
		if t.VendorResponse != "" {
			var scores struct {
				IQScore        float64 `json:"iq_score"`
				MICRConfidence float64 `json:"micr_confidence"`
			}
			if err := json.Unmarshal([]byte(t.VendorResponse), &scores); err == nil {
				item.IQScore = scores.IQScore
				item.MICRConfidence = scores.MICRConfidence
			}
		}
		// Fallback: flagged transfers with amount mismatch have low MICR confidence
		if item.MICRConfidence == 0 && t.OCRAmount > 0 && t.EnteredAmount > 0 {
			item.MICRConfidence = 0.5
		}
		items = append(items, item)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"transfers": items,
		"count":     len(items),
	})
}

// ApproveRequest is the request body for POST /operator/approve.
type ApproveRequest struct {
	TransferID           string `json:"transfer_id"`
	OperatorID           string `json:"operator_id"`
	Note                 string `json:"note"`
	ContributionTypeOverride string `json:"contribution_type,omitempty"`
}

// Approve handles POST /operator/approve.
func (h *OperatorHandler) Approve(w http.ResponseWriter, r *http.Request) {
	var req ApproveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.TransferID == "" {
		writeError(w, http.StatusBadRequest, "transfer_id is required")
		return
	}
	if req.OperatorID == "" {
		writeError(w, http.StatusBadRequest, "operator_id is required")
		return
	}

	t, err := h.operatorRepo.GetTransfer(req.TransferID)
	if err != nil {
		log.Printf("operator approve: get transfer: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get transfer")
		return
	}
	if t == nil {
		writeError(w, http.StatusNotFound, "transfer not found")
		return
	}
	if t.State != transfer.StateAnalyzing {
		writeError(w, http.StatusConflict, "transfer is not in Analyzing state")
		return
	}

	// Apply business rules before approving (e.g. deposit limit for flagged transfers)
	if err := h.fundingCfg.CheckLimit(t.Amount); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	// Apply contribution override if provided
	contributionType := req.ContributionTypeOverride
	if contributionType == "" {
		contributionType = h.fundingCfg.GetContributionDefault(t.AccountID)
	}
	t.ContributionType = contributionType

	// Transition: Analyzing → Approved
	if err := t.Transition(transfer.StateApproved); err != nil {
		log.Printf("operator approve: transition to approved: %v", err)
		writeError(w, http.StatusInternalServerError, "state transition error")
		return
	}
	if err := h.transferRepo.UpdateTransferState(t); err != nil {
		log.Printf("operator approve: update state: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to update state")
		return
	}

	// Post ledger entry
	omnibusAcct := h.fundingCfg.GetOmnibusAccount(t.AccountID)
	if _, err := h.ledgerSvc.CreateMovementEntry(nil, t.AccountID, omnibusAcct, t.ID, contributionType, t.Amount); err != nil {
		log.Printf("operator approve: create ledger entry: %v", err)
		writeError(w, http.StatusInternalServerError, "ledger error")
		return
	}

	// Transition: Approved → FundsPosted
	if err := t.Transition(transfer.StateFundsPosted); err != nil {
		log.Printf("operator approve: transition to funds posted: %v", err)
		writeError(w, http.StatusInternalServerError, "state transition error")
		return
	}
	if err := h.transferRepo.UpdateTransferState(t); err != nil {
		log.Printf("operator approve: update funds posted: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to update state")
		return
	}

	// Record operator action
	if _, err := h.operatorRepo.RecordAction(req.TransferID, "approve", req.OperatorID, req.Note, req.ContributionTypeOverride); err != nil {
		log.Printf("operator approve: record action: %v", err)
	}

	writeJSON(w, http.StatusOK, t)
}

// Actions handles GET /operator/actions/{transfer_id}.
func (h *OperatorHandler) Actions(w http.ResponseWriter, r *http.Request) {
	transferID := r.PathValue("transfer_id")
	if transferID == "" {
		writeError(w, http.StatusBadRequest, "transfer_id is required")
		return
	}
	actions, err := h.operatorRepo.ListActions(transferID)
	if err != nil {
		log.Printf("operator actions: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list actions")
		return
	}
	if actions == nil {
		actions = []*operator.Action{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"actions": actions})
}

// RejectRequest is the request body for POST /operator/reject.
type RejectRequest struct {
	TransferID string `json:"transfer_id"`
	OperatorID string `json:"operator_id"`
	Note       string `json:"note"`
}

// Reject handles POST /operator/reject.
func (h *OperatorHandler) Reject(w http.ResponseWriter, r *http.Request) {
	var req RejectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.TransferID == "" {
		writeError(w, http.StatusBadRequest, "transfer_id is required")
		return
	}
	if req.OperatorID == "" {
		writeError(w, http.StatusBadRequest, "operator_id is required")
		return
	}

	t, err := h.operatorRepo.GetTransfer(req.TransferID)
	if err != nil {
		log.Printf("operator reject: get transfer: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get transfer")
		return
	}
	if t == nil {
		writeError(w, http.StatusNotFound, "transfer not found")
		return
	}
	if t.State != transfer.StateAnalyzing {
		writeError(w, http.StatusConflict, "transfer is not in Analyzing state")
		return
	}

	// Transition: Analyzing → Rejected
	if err := t.Transition(transfer.StateRejected); err != nil {
		log.Printf("operator reject: transition: %v", err)
		writeError(w, http.StatusInternalServerError, "state transition error")
		return
	}
	if err := h.transferRepo.UpdateTransferState(t); err != nil {
		log.Printf("operator reject: update state: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to update state")
		return
	}

	// Record operator action
	if _, err := h.operatorRepo.RecordAction(req.TransferID, "reject", req.OperatorID, req.Note, ""); err != nil {
		log.Printf("operator reject: record action: %v", err)
	}

	writeJSON(w, http.StatusOK, t)
}
