package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/checkstream/checkstream/internal/depositjob"
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
	jobRepo      *depositjob.Repository
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
	jobRepo *depositjob.Repository,
	db *sql.DB,
) *DepositHandler {
	return &DepositHandler{
		transferRepo: transferRepo,
		vendorStub:   vendorStub,
		ledgerSvc:    ledgerSvc,
		fundingSvc:   fundingSvc,
		fundingCfg:   fundingCfg,
		operatorRepo: operatorRepo,
		jobRepo:      jobRepo,
		db:           db,
	}
}

// Create handles POST /deposits. Accepts the deposit and returns 202; processing runs async via deposit_jobs.
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

	// Create transfer (Requested) and persist image paths for async worker
	t, err := h.transferRepo.CreateTransfer(req.AccountID, req.Amount)
	if err != nil {
		log.Printf("deposit: create transfer: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create transfer")
		return
	}
	t.FrontImagePath = req.FrontImage
	t.BackImagePath = req.BackImage
	if err := h.transferRepo.UpdateTransferState(t); err != nil {
		log.Printf("deposit: persist transfer images: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to save deposit")
		return
	}

	scenario := r.Header.Get("X-Test-Scenario")
	if scenario == "" {
		scenario = req.Scenario
	}
	source := req.Source
	if source == "" {
		source = "api"
	}
	if err := h.jobRepo.Add(t.ID, scenario, source); err != nil {
		log.Printf("deposit: enqueue job: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to enqueue deposit")
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"message":  "deposit accepted",
		"transfer": t,
	})
}

// ProcessDeposit runs the deposit pipeline (vendor → rules → ledger) for a transfer. Called by the async worker.
// Transfer must be in Requested state; job must exist for scenario/source.
func (h *DepositHandler) ProcessDeposit(transferID string) error {
	t, err := h.transferRepo.GetTransfer(transferID)
	if err != nil {
		return err
	}
	if t == nil {
		return nil // no transfer, nothing to do
	}
	if t.State != transfer.StateRequested {
		return nil // already processed or in progress
	}

	job, err := h.jobRepo.GetByTransferID(transferID)
	if err != nil || job == nil {
		return err
	}

	scenarioOverride := job.Scenario
	depositSource := job.Source
	if depositSource == "" {
		depositSource = "api"
	}

	// Transition to Validating and persist
	if err := t.Transition(transfer.StateValidating); err != nil {
		return err
	}
	if err := h.transferRepo.UpdateTransferState(t); err != nil {
		return err
	}

	vendorReq := &vendor.VendorRequest{
		AccountID:  t.AccountID,
		Amount:     t.Amount,
		FrontImage: t.FrontImagePath,
		BackImage:  t.BackImagePath,
		TransferID: t.ID,
	}
	vendorResp := h.vendorStub.Validate(vendorReq, scenarioOverride)

	trace.DepositTrace(t.ID, t.AccountID, "vendor_response", map[string]interface{}{
		"source":        depositSource,
		"vendor_status": vendorResp.Status,
		"reason":        vendorResp.Reason,
	})
	switch vendorResp.Status {
	case "fail":
		t.VendorResponse = marshalVendorScores(vendorResp)
		_ = t.Transition(transfer.StateRejected)
		_ = h.transferRepo.UpdateTransferState(t)
		return nil
	case "reject":
		t.VendorResponse = marshalVendorScores(vendorResp)
		_ = t.Transition(transfer.StateRejected)
		_ = h.transferRepo.UpdateTransferState(t)
		return nil
	case "flagged":
		trace.DepositTrace(t.ID, t.AccountID, "vendor_flagged", map[string]interface{}{"source": depositSource, "state": "Analyzing", "reason": vendorResp.Reason})
		_ = t.Transition(transfer.StateAnalyzing)
		t.OCRAmount = vendorResp.OCRAmount
		t.EnteredAmount = vendorResp.EnteredAmount
		t.VendorResponse = marshalVendorScores(vendorResp)
		_ = h.transferRepo.UpdateTransferState(t)
		return nil
	case "pass":
		// fall through
	}

	if err := t.Transition(transfer.StateAnalyzing); err != nil {
		return err
	}
	if vendorResp.MICR != nil {
		micrJSON, _ := json.Marshal(vendorResp.MICR)
		t.MICRData = string(micrJSON)
	}
	vendorTransactionID := vendorResp.TransactionID
	t.VendorResponse = marshalVendorScores(vendorResp)
	t.TransactionID = ""
	if err := h.transferRepo.UpdateTransferState(t); err != nil {
		return err
	}
	t.TransactionID = vendorTransactionID

	if err := h.fundingSvc.ValidateSession(t.AccountID); err != nil {
		trace.DepositTrace(t.ID, t.AccountID, "business_rules", map[string]interface{}{"source": depositSource, "result": "rejected", "rule": "session"})
		h.rejectTransfer(t, "invalid session")
		return nil
	}
	if err := h.fundingSvc.CheckEligibility(t.AccountID); err != nil {
		trace.DepositTrace(t.ID, t.AccountID, "business_rules", map[string]interface{}{"source": depositSource, "result": "rejected", "rule": "eligibility"})
		h.rejectTransfer(t, "account not eligible")
		return nil
	}
	if err := h.fundingSvc.CheckLimit(t.AccountID, t.Amount); err != nil {
		trace.DepositTrace(t.ID, t.AccountID, "business_rules", map[string]interface{}{"source": depositSource, "result": "rejected", "rule": "limit"})
		h.rejectTransfer(t, "over limit")
		return nil
	}
	if t.TransactionID != "" {
		if err := h.fundingSvc.CheckDuplicate(t.TransactionID); err != nil {
			if errors.Is(err, funding.ErrDuplicate) {
				trace.DepositTrace(t.ID, t.AccountID, "business_rules", map[string]interface{}{"source": depositSource, "result": "rejected", "rule": "duplicate"})
				h.rejectTransfer(t, "duplicate")
				return nil
			}
		}
	}

	if err := t.Transition(transfer.StateApproved); err != nil {
		return err
	}
	if err := h.transferRepo.UpdateTransferState(t); err != nil {
		return err
	}
	t.ContributionType = h.fundingCfg.GetContributionDefault(t.AccountID)

	omnibusAcct := h.fundingCfg.GetOmnibusAccount(t.AccountID)
	dbTx, err := h.db.Begin()
	if err != nil {
		return err
	}
	defer dbTx.Rollback()

	if _, err := h.ledgerSvc.CreateMovementEntry(dbTx, t.AccountID, omnibusAcct, t.ID, t.ContributionType, t.Amount); err != nil {
		return err
	}
	if err := t.Transition(transfer.StateFundsPosted); err != nil {
		return err
	}
	if _, err := dbTx.Exec(`UPDATE transfers SET state=?, contribution_type=?, transaction_id=?, micr_data=?, updated_at=? WHERE id=?`,
		string(t.State), t.ContributionType, t.TransactionID, t.MICRData, t.UpdatedAt, t.ID); err != nil {
		return err
	}
	if err := dbTx.Commit(); err != nil {
		return err
	}

	if h.operatorRepo != nil {
		_, _ = h.operatorRepo.RecordAction(t.ID, "auto_approve", "system", "passed validation, no operator review", "")
	}
	trace.DepositTrace(t.ID, t.AccountID, "funds_posted", map[string]interface{}{"source": depositSource, "state": string(t.State)})
	return nil
}

// ProcessOneJob claims one pending deposit job and processes it. Used by tests to run the async worker once.
func (h *DepositHandler) ProcessOneJob() bool {
	job, ok, err := h.jobRepo.ClaimNext()
	if err != nil || !ok {
		return false
	}
	if err := h.ProcessDeposit(job.TransferID); err != nil {
		_ = h.jobRepo.Fail(job.ID, err.Error())
	} else {
		_ = h.jobRepo.Complete(job.ID)
	}
	return true
}

// ProcessOneJobHTTP handles POST /sandbox/process-job. Processes one pending deposit job and returns {"processed": true/false}.
// Used by the sandbox UI so scenarios can run synchronously after POST /deposits (202).
func (h *DepositHandler) ProcessOneJobHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	processed := h.ProcessOneJob()
	writeJSON(w, http.StatusOK, map[string]interface{}{"processed": processed})
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
// account_id is optional; when omitted, returns deposits for all configured accounts.
func (h *DepositHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	accountID := q.Get("account_id")
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 {
		limit = 50
	}
	offset, _ := strconv.Atoi(q.Get("offset"))
	status := transfer.State(q.Get("status"))

	var transfers []*transfer.Transfer
	var err error
	if accountID != "" {
		transfers, err = h.transferRepo.ListTransfersByAccount(accountID, limit, offset, status)
	} else {
		userID := ResolveMobileUserID(r)
		accountIDs := h.fundingCfg.GetAccountIDsForUser(userID)
		transfers, err = h.transferRepo.ListTransfersByAccounts(accountIDs, limit, offset, status)
	}
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
