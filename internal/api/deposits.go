package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/checkstream/checkstream/internal/funding"
	"github.com/checkstream/checkstream/internal/ledger"
	"github.com/checkstream/checkstream/internal/operator"
	"github.com/checkstream/checkstream/internal/transfer"
	"github.com/checkstream/checkstream/internal/trace"
	"github.com/checkstream/checkstream/internal/vendor"
)

// DepositRequest is the request body for POST /deposits.
type DepositRequest struct {
	AccountID   string  `json:"account_id"`
	Amount      float64 `json:"amount"`
	FrontImage  string  `json:"front_image"`
	BackImage   string  `json:"back_image"`
	Scenario    string  `json:"scenario,omitempty"` // Overrides vendor stub scenario (e.g. clean_pass, amount_mismatch)
	Source      string  `json:"source,omitempty"`   // Deposit source for logs: mobile, api, demo (default api)
}

// DepositHandler handles deposit-related HTTP requests.
type DepositHandler struct {
	transferRepo *transfer.Repository
	vendorStub   *vendor.Stub
	ledgerSvc    *ledger.Service
	fundingSvc   *funding.Service
	fundingCfg   *funding.Config
	operatorRepo *operator.Repository
	db           *sql.DB
}

// NewDepositHandler creates a new DepositHandler.
func NewDepositHandler(
	transferRepo *transfer.Repository,
	vendorStub *vendor.Stub,
	ledgerSvc *ledger.Service,
	fundingSvc *funding.Service,
	fundingCfg *funding.Config,
	operatorRepo *operator.Repository,
	db *sql.DB,
) *DepositHandler {
	return &DepositHandler{
		transferRepo: transferRepo,
		vendorStub:   vendorStub,
		ledgerSvc:    ledgerSvc,
		fundingSvc:   fundingSvc,
		fundingCfg:   fundingCfg,
		operatorRepo: operatorRepo,
		db:           db,
	}
}

// Create handles POST /deposits.
func (h *DepositHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req DepositRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.AccountID == "" {
		writeError(w, http.StatusBadRequest, "account_id is required")
		return
	}
	if req.Amount <= 0 {
		writeError(w, http.StatusBadRequest, "amount must be positive")
		return
	}
	if req.FrontImage == "" {
		writeError(w, http.StatusBadRequest, "front_image is required")
		return
	}
	if req.BackImage == "" {
		writeError(w, http.StatusBadRequest, "back_image is required")
		return
	}

	// Step 1: Create transfer (Requested)
	t, err := h.transferRepo.CreateTransfer(req.AccountID, req.Amount)
	if err != nil {
		log.Printf("deposit: create transfer: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create transfer")
		return
	}
	t.FrontImagePath = req.FrontImage
	t.BackImagePath = req.BackImage

	// Step 2: Transition to Validating
	if err := t.Transition(transfer.StateValidating); err != nil {
		log.Printf("deposit: transition to validating: %v", err)
		writeError(w, http.StatusInternalServerError, "state transition error")
		return
	}
	if err := h.transferRepo.UpdateTransferState(t); err != nil {
		log.Printf("deposit: update validating state: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to update state")
		return
	}

	// Step 3: Call vendor stub
	scenarioOverride := r.Header.Get("X-Test-Scenario")
	if scenarioOverride == "" && req.Scenario != "" {
		scenarioOverride = req.Scenario
	}
	vendorReq := &vendor.VendorRequest{
		AccountID:  req.AccountID,
		Amount:     req.Amount,
		FrontImage: req.FrontImage,
		BackImage:  req.BackImage,
		TransferID: t.ID,
	}
	vendorResp := h.vendorStub.Validate(vendorReq, scenarioOverride)

	depositSource := req.Source
	if depositSource == "" {
		depositSource = "api"
	}
	// Step 4: Handle vendor response
	trace.DepositTrace(t.ID, req.AccountID, "vendor_response", map[string]interface{}{
		"source":        depositSource,
		"vendor_status": vendorResp.Status,
		"reason":        vendorResp.Reason,
	})
	switch vendorResp.Status {
	case "fail":
		// Image quality failure → Rejected
		if err := t.Transition(transfer.StateRejected); err != nil {
			log.Printf("deposit: transition to rejected (fail): %v", err)
		}
		if err := h.transferRepo.UpdateTransferState(t); err != nil {
			log.Printf("deposit: update rejected state: %v", err)
		}
		writeJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{
			"error":    "image quality check failed",
			"reason":   vendorResp.Reason,
			"message":  vendorResp.Message,
			"transfer": t,
		})
		return

	case "reject":
		// Hard reject (e.g. duplicate from vendor) → Rejected
		if err := t.Transition(transfer.StateRejected); err != nil {
			log.Printf("deposit: transition to rejected (reject): %v", err)
		}
		if err := h.transferRepo.UpdateTransferState(t); err != nil {
			log.Printf("deposit: update rejected state: %v", err)
		}
		writeJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{
			"error":    "check rejected by vendor",
			"reason":   vendorResp.Reason,
			"transfer": t,
		})
		return

	case "flagged":
		// Needs human review → Analyzing
		trace.DepositTrace(t.ID, req.AccountID, "vendor_flagged", map[string]interface{}{"source": depositSource, "state": "Analyzing", "reason": vendorResp.Reason})
		if err := t.Transition(transfer.StateAnalyzing); err != nil {
			log.Printf("deposit: transition to analyzing: %v", err)
		}
		t.OCRAmount = vendorResp.OCRAmount
		t.EnteredAmount = vendorResp.EnteredAmount
		t.VendorResponse = marshalVendorScores(vendorResp)
		if err := h.transferRepo.UpdateTransferState(t); err != nil {
			log.Printf("deposit: update analyzing state: %v", err)
		}
		writeJSON(w, http.StatusAccepted, map[string]interface{}{
			"message":  "check flagged for review",
			"reason":   vendorResp.Reason,
			"transfer": t,
		})
		return

	case "pass":
		// Happy path — continue processing
	}

	// Step 5: Transition Validating→Analyzing
	if err := t.Transition(transfer.StateAnalyzing); err != nil {
		log.Printf("deposit: transition to analyzing (pass): %v", err)
		writeError(w, http.StatusInternalServerError, "state transition error")
		return
	}

	// Store vendor data in memory — do NOT persist transaction_id to DB yet
	// (we need to check for duplicates first; persisting now would cause self-detection)
	if vendorResp.MICR != nil {
		micrJSON, _ := json.Marshal(vendorResp.MICR)
		t.MICRData = string(micrJSON)
	}
	vendorTransactionID := vendorResp.TransactionID
	t.VendorResponse = marshalVendorScores(vendorResp)
	t.TransactionID = "" // don't persist yet
	if err := h.transferRepo.UpdateTransferState(t); err != nil {
		log.Printf("deposit: update analyzing state: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to update state")
		return
	}
	t.TransactionID = vendorTransactionID // restore for business rules check

	// Step 6: Business rules validation — MUST run before transitioning to Approved
	if err := h.fundingSvc.ValidateSession(req.AccountID); err != nil {
		trace.DepositTrace(t.ID, req.AccountID, "business_rules", map[string]interface{}{"source": depositSource, "result": "rejected", "rule": "session"})
		h.rejectTransfer(t, "invalid session")
		writeJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{
			"error":    err.Error(),
			"transfer": t,
		})
		return
	}
	if err := h.fundingSvc.CheckEligibility(req.AccountID); err != nil {
		trace.DepositTrace(t.ID, req.AccountID, "business_rules", map[string]interface{}{"source": depositSource, "result": "rejected", "rule": "eligibility"})
		h.rejectTransfer(t, "account not eligible")
		writeJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{
			"error":    err.Error(),
			"transfer": t,
		})
		return
	}
	if err := h.fundingSvc.CheckLimit(req.Amount); err != nil {
		trace.DepositTrace(t.ID, req.AccountID, "business_rules", map[string]interface{}{"source": depositSource, "result": "rejected", "rule": "limit"})
		h.rejectTransfer(t, "over limit")
		writeJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{
			"error":    err.Error(),
			"transfer": t,
		})
		return
	}
	if t.TransactionID != "" {
		if err := h.fundingSvc.CheckDuplicate(t.TransactionID); err != nil {
			if errors.Is(err, funding.ErrDuplicate) {
				trace.DepositTrace(t.ID, req.AccountID, "business_rules", map[string]interface{}{"source": depositSource, "result": "rejected", "rule": "duplicate"})
				h.rejectTransfer(t, "duplicate")
				writeJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{
					"error":    err.Error(),
					"transfer": t,
				})
				return
			}
		}
	}

	// Step 7: Transition Analyzing→Approved (all rules passed)
	if err := t.Transition(transfer.StateApproved); err != nil {
		log.Printf("deposit: transition to approved: %v", err)
		writeError(w, http.StatusInternalServerError, "state transition error")
		return
	}
	if err := h.transferRepo.UpdateTransferState(t); err != nil {
		log.Printf("deposit: update approved state: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to update state")
		return
	}

	// Step 8: Apply contribution default
	t.ContributionType = h.fundingCfg.GetContributionDefault(req.AccountID)

	// Step 10: Post ledger entry in DB transaction with state update
	omnibusAcct := h.fundingCfg.GetOmnibusAccount(req.AccountID)
	dbTx, err := h.db.Begin()
	if err != nil {
		log.Printf("deposit: begin tx: %v", err)
		writeError(w, http.StatusInternalServerError, "transaction error")
		return
	}

	_, err = h.ledgerSvc.CreateMovementEntry(dbTx, req.AccountID, omnibusAcct, t.ID, t.ContributionType, req.Amount)
	if err != nil {
		dbTx.Rollback()
		log.Printf("deposit: create ledger entry: %v", err)
		writeError(w, http.StatusInternalServerError, "ledger error")
		return
	}

	// Step 11: Transition → FundsPosted
	if err := t.Transition(transfer.StateFundsPosted); err != nil {
		dbTx.Rollback()
		log.Printf("deposit: transition to funds posted: %v", err)
		writeError(w, http.StatusInternalServerError, "state transition error")
		return
	}

	if _, err := dbTx.Exec(`UPDATE transfers SET state=?, contribution_type=?, transaction_id=?, micr_data=?, updated_at=? WHERE id=?`,
		string(t.State), t.ContributionType, t.TransactionID, t.MICRData, t.UpdatedAt, t.ID); err != nil {
		dbTx.Rollback()
		log.Printf("deposit: update funds posted state: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to update state")
		return
	}

	if err := dbTx.Commit(); err != nil {
		log.Printf("deposit: commit tx: %v", err)
		writeError(w, http.StatusInternalServerError, "transaction commit error")
		return
	}

	// Record audit log entry for auto-approved deposits (mobile or API)
	if h.operatorRepo != nil {
		if _, err := h.operatorRepo.RecordAction(t.ID, "auto_approve", "system", "passed validation, no operator review", ""); err != nil {
			log.Printf("deposit: record audit action: %v", err)
		}
	}
	trace.DepositTrace(t.ID, req.AccountID, "funds_posted", map[string]interface{}{"source": depositSource, "state": string(t.State)})

	writeJSON(w, http.StatusCreated, t)
}

// vendorScoresJSON is stored in transfer.vendor_response for operator queue display.
type vendorScoresJSON struct {
	Status         string  `json:"status"`
	Reason         string  `json:"reason,omitempty"`
	IQScore        float64 `json:"iq_score,omitempty"`
	MICRConfidence float64 `json:"micr_confidence,omitempty"`
}

func marshalVendorScores(r *vendor.VendorResponse) string {
	v := vendorScoresJSON{Status: r.Status, Reason: r.Reason, IQScore: r.IQScore, MICRConfidence: r.MICRConfidence}
	b, _ := json.Marshal(v)
	return string(b)
}

// rejectTransfer transitions a transfer to Rejected state.
func (h *DepositHandler) rejectTransfer(t *transfer.Transfer, reason string) {
	if err := t.Transition(transfer.StateRejected); err != nil {
		log.Printf("reject transfer %s (%s): %v", t.ID, reason, err)
		return
	}
	if err := h.transferRepo.UpdateTransferState(t); err != nil {
		log.Printf("update rejected transfer %s: %v", t.ID, err)
	}
}

// List handles GET /deposits?account_id=...&limit=...&offset=...&status=...
func (h *DepositHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	accountID := q.Get("account_id")
	if accountID == "" {
		writeError(w, http.StatusBadRequest, "account_id is required")
		return
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	status := transfer.State(q.Get("status"))

	transfers, err := h.transferRepo.ListTransfersByAccount(accountID, limit, offset, status)
	if err != nil {
		log.Printf("deposit list: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list deposits")
		return
	}
	if transfers == nil {
		transfers = []*transfer.Transfer{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"transfers": transfers,
		"count":     len(transfers),
	})
}

// Get handles GET /deposits/:id.
func (h *DepositHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "transfer id required")
		return
	}

	t, err := h.transferRepo.GetTransfer(id)
	if err != nil {
		log.Printf("deposit get: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get transfer")
		return
	}
	if t == nil {
		writeError(w, http.StatusNotFound, "transfer not found")
		return
	}

	writeJSON(w, http.StatusOK, t)
}
