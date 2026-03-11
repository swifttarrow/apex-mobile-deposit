package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/checkstream/checkstream/internal/funding"
	"github.com/checkstream/checkstream/internal/ledger"
	"github.com/checkstream/checkstream/internal/transfer"
	"github.com/checkstream/checkstream/internal/vendor"
)

// DepositRequest is the request body for POST /deposits.
type DepositRequest struct {
	AccountID   string  `json:"account_id"`
	Amount      float64 `json:"amount"`
	FrontImage  string  `json:"front_image"`
	BackImage   string  `json:"back_image"`
}

// DepositHandler handles deposit-related HTTP requests.
type DepositHandler struct {
	transferRepo *transfer.Repository
	vendorStub   *vendor.Stub
	ledgerSvc    *ledger.Service
	fundingSvc   *funding.Service
	fundingCfg   *funding.Config
	db           *sql.DB
}

// NewDepositHandler creates a new DepositHandler.
func NewDepositHandler(
	transferRepo *transfer.Repository,
	vendorStub *vendor.Stub,
	ledgerSvc *ledger.Service,
	fundingSvc *funding.Service,
	fundingCfg *funding.Config,
	db *sql.DB,
) *DepositHandler {
	return &DepositHandler{
		transferRepo: transferRepo,
		vendorStub:   vendorStub,
		ledgerSvc:    ledgerSvc,
		fundingSvc:   fundingSvc,
		fundingCfg:   fundingCfg,
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
	vendorReq := &vendor.VendorRequest{
		AccountID:  req.AccountID,
		Amount:     req.Amount,
		FrontImage: req.FrontImage,
		BackImage:  req.BackImage,
		TransferID: t.ID,
	}
	vendorResp := h.vendorStub.Validate(vendorReq, scenarioOverride)

	// Step 4: Handle vendor response
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
		if err := t.Transition(transfer.StateAnalyzing); err != nil {
			log.Printf("deposit: transition to analyzing: %v", err)
		}
		t.OCRAmount = vendorResp.OCRAmount
		t.EnteredAmount = vendorResp.EnteredAmount
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

	// Step 6: Transition Validating→Analyzing→Approved
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
	// Save state (without transaction_id) — use a temporary blank to avoid premature write
	savedTxnID := t.TransactionID
	t.TransactionID = "" // don't persist yet
	if err := h.transferRepo.UpdateTransferState(t); err != nil {
		log.Printf("deposit: update analyzing state: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to update state")
		return
	}
	t.TransactionID = savedTxnID

	if err := t.Transition(transfer.StateApproved); err != nil {
		log.Printf("deposit: transition to approved: %v", err)
		writeError(w, http.StatusInternalServerError, "state transition error")
		return
	}
	t.TransactionID = "" // still don't persist
	if err := h.transferRepo.UpdateTransferState(t); err != nil {
		log.Printf("deposit: update approved state: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to update state")
		return
	}
	t.TransactionID = vendorTransactionID // restore for business rules check

	// Step 7: Business rules validation
	// ValidateSession — use account ID as a proxy session
	if err := h.fundingSvc.ValidateSession(req.AccountID); err != nil {
		h.rejectTransfer(t, "invalid session")
		writeJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{
			"error":    err.Error(),
			"transfer": t,
		})
		return
	}

	// CheckEligibility
	if err := h.fundingSvc.CheckEligibility(req.AccountID); err != nil {
		h.rejectTransfer(t, "account not eligible")
		writeJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{
			"error":    err.Error(),
			"transfer": t,
		})
		return
	}

	// CheckLimit
	if err := h.fundingSvc.CheckLimit(req.Amount); err != nil {
		h.rejectTransfer(t, "over limit")
		writeJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{
			"error":    err.Error(),
			"transfer": t,
		})
		return
	}

	// CheckDuplicate — only check if we have a transaction ID
	if t.TransactionID != "" {
		if err := h.fundingSvc.CheckDuplicate(t.TransactionID); err != nil {
			if errors.Is(err, funding.ErrDuplicate) {
				h.rejectTransfer(t, "duplicate")
				writeJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{
					"error":    err.Error(),
					"transfer": t,
				})
				return
			}
		}
	}

	// Step 9: Apply contribution default
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

	writeJSON(w, http.StatusCreated, t)
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
