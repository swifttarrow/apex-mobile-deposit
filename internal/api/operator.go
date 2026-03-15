package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/checkstream/checkstream/internal/auth"
	"github.com/checkstream/checkstream/internal/funding"
	"github.com/checkstream/checkstream/internal/ledger"
	"github.com/checkstream/checkstream/internal/operator"
	"github.com/checkstream/checkstream/internal/trace"
	"github.com/checkstream/checkstream/internal/transfer"
)

// OperatorHandler handles operator workflow HTTP requests.
type OperatorHandler struct {
	operatorRepo *operator.Repository
	transferRepo *transfer.Repository
	ledgerSvc    *ledger.Service
	fundingCfg   *funding.Config
	fundingSvc   *funding.Service
}

// NewOperatorHandler creates a new OperatorHandler.
func NewOperatorHandler(
	operatorRepo *operator.Repository,
	transferRepo *transfer.Repository,
	ledgerSvc *ledger.Service,
	fundingCfg *funding.Config,
	fundingSvc *funding.Service,
) *OperatorHandler {
	return &OperatorHandler{
		operatorRepo: operatorRepo,
		transferRepo: transferRepo,
		ledgerSvc:    ledgerSvc,
		fundingCfg:   fundingCfg,
		fundingSvc:   fundingSvc,
	}
}

// QueueItem is a transfer enriched with risk scores for the operator queue.
type QueueItem struct {
	*transfer.Transfer
	IQScore        float64 `json:"iq_score"`
	MICRConfidence float64 `json:"micr_confidence"`
}

// Queue handles GET /operator/queue.
// Query params: date, account, amount_min, amount_max, status, limit, offset.
// Only status=Analyzing returns results (review queue shows flagged deposits).
func (h *OperatorHandler) Queue(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	dateFilter := q.Get("date")
	accountFilter := q.Get("account")
	statusFilter := q.Get("status")
	amountMinStr := q.Get("amount_min")
	amountMaxStr := q.Get("amount_max")

	// Queue only contains Analyzing (flagged) transfers; other status returns empty
	if statusFilter != "" && statusFilter != string(transfer.StateAnalyzing) {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"transfers": []QueueItem{},
			"count":     0,
		})
		return
	}

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
	TransferID              string `json:"transfer_id"`
	Note                    string `json:"note"`
	ContributionTypeOverride string `json:"contribution_type,omitempty"`
}

// Approve handles POST /operator/approve.
func (h *OperatorHandler) Approve(w http.ResponseWriter, r *http.Request) {
	operatorID, err := auth.GetOperatorID(r)
	if err != nil || operatorID == "" {
		writeError(w, http.StatusUnauthorized, "login required")
		return
	}

	var req ApproveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.TransferID == "" {
		writeError(w, http.StatusBadRequest, "transfer_id is required")
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

	// Apply business rules before approving (e.g. deposit/contribution limit for flagged transfers)
	if err := h.fundingSvc.CheckLimit(t.AccountID, t.Amount); err != nil {
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
	if _, err := h.operatorRepo.RecordAction(req.TransferID, "approve", operatorID, req.Note, req.ContributionTypeOverride); err != nil {
		log.Printf("operator approve: record action: %v", err)
	}
	trace.DepositTrace(t.ID, t.AccountID, "operator_action", map[string]interface{}{"action": "approve", "operator_id": operatorID})

	writeJSON(w, http.StatusOK, t)
}

// Audit handles GET /operator/audit.
// Query params: limit, action (all|approved|approve|auto_approve|reject), operator_id
// Enriches each action with transfer_state, settled, and settlement_batch_id for the UI.
func (h *OperatorHandler) Audit(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit := 50
	if l := q.Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	actionFilter := q.Get("action")
	operatorFilter := q.Get("operator_id")

	actions, err := h.operatorRepo.ListRecentActions(limit, actionFilter, operatorFilter)
	if err != nil {
		log.Printf("operator audit: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list audit log")
		return
	}
	if actions == nil {
		actions = []*operator.Action{}
	}

	// Enrich with transfer state and settlement info for each action's transfer
	transferIDs := make([]string, 0, len(actions))
	seen := make(map[string]bool)
	for _, a := range actions {
		if a.TransferID != "" && !seen[a.TransferID] {
			seen[a.TransferID] = true
			transferIDs = append(transferIDs, a.TransferID)
		}
	}
	transfers, _ := h.transferRepo.GetTransfersByIDs(transferIDs)
	transferByID := make(map[string]*transfer.Transfer)
	for _, t := range transfers {
		transferByID[t.ID] = t
	}

	enriched := make([]map[string]interface{}, 0, len(actions))
	for _, a := range actions {
		item := map[string]interface{}{
			"id":                         a.ID,
			"transfer_id":                a.TransferID,
			"action":                     a.Action,
			"operator_id":                a.OperatorID,
			"note":                       a.Note,
			"contribution_type_override": a.ContributionTypeOverride,
			"created_at":                 a.CreatedAt,
		}
		if t := transferByID[a.TransferID]; t != nil {
			item["transfer_state"] = string(t.State)
			item["settled"] = t.State == transfer.StateCompleted
			if t.SettlementBatchID != "" {
				item["settlement_batch_id"] = t.SettlementBatchID
			}
			if t.SettlementAckAt != "" {
				item["settlement_ack_at"] = t.SettlementAckAt
			}
		} else {
			item["transfer_state"] = ""
			item["settled"] = false
		}
		enriched = append(enriched, item)
	}

	// Include distinct operators for filter dropdown (only when not filtering by operator)
	operators := []string{}
	if operatorFilter == "" {
		ops, err := h.operatorRepo.ListAuditOperators()
		if err == nil {
			operators = ops
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"actions":   enriched,
		"operators": operators,
	})
}

// GetTransfer handles GET /operator/transfer/{transfer_id}.
// Returns a single transfer for operator detail view (e.g. from audit log). 404 if not found.
func (h *OperatorHandler) GetTransfer(w http.ResponseWriter, r *http.Request) {
	transferID := r.PathValue("transfer_id")
	if transferID == "" {
		writeError(w, http.StatusBadRequest, "transfer_id is required")
		return
	}
	t, err := h.operatorRepo.GetTransfer(transferID)
	if err != nil {
		log.Printf("operator get transfer: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get transfer")
		return
	}
	if t == nil {
		writeError(w, http.StatusNotFound, "transfer not found")
		return
	}
	item := QueueItem{Transfer: t}
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
	if item.MICRConfidence == 0 && t.OCRAmount > 0 && t.EnteredAmount > 0 {
		item.MICRConfidence = 0.5
	}
	writeJSON(w, http.StatusOK, item)
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
	Note       string `json:"note"`
}

// Reject handles POST /operator/reject.
func (h *OperatorHandler) Reject(w http.ResponseWriter, r *http.Request) {
	operatorID, err := auth.GetOperatorID(r)
	if err != nil || operatorID == "" {
		writeError(w, http.StatusUnauthorized, "login required")
		return
	}

	var req RejectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.TransferID == "" {
		writeError(w, http.StatusBadRequest, "transfer_id is required")
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
	if _, err := h.operatorRepo.RecordAction(req.TransferID, "reject", operatorID, req.Note, ""); err != nil {
		log.Printf("operator reject: record action: %v", err)
	}
	trace.DepositTrace(t.ID, t.AccountID, "operator_action", map[string]interface{}{"action": "reject", "operator_id": operatorID})

	writeJSON(w, http.StatusOK, t)
}
