package settlement

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/checkstream/checkstream/internal/ledger"
	"github.com/checkstream/checkstream/internal/transfer"
	"github.com/checkstream/checkstream/internal/trace"
	"github.com/google/uuid"
)

// SettlementFile represents the X9-like settlement batch file.
type SettlementFile struct {
	BatchID     string            `json:"batch_id"`
	CreatedAt   string            `json:"created_at"`
	Transfers   []SettlementEntry `json:"transfers"`
	TotalCount  int               `json:"total_count"`
	TotalAmount float64           `json:"total_amount"`
}

// SettlementEntry is a single transfer in the settlement file.
type SettlementEntry struct {
	TransferID    string  `json:"transfer_id"`
	AccountID     string  `json:"account_id"`
	Amount        float64 `json:"amount"`
	MICRRouting   string  `json:"micr_routing,omitempty"`
	MICRAccount   string  `json:"micr_account,omitempty"`
	CheckNumber   string  `json:"check_number,omitempty"`
	TransactionID string  `json:"transaction_id,omitempty"`
	FrontImageRef string  `json:"front_image_ref,omitempty"`
	BackImageRef  string  `json:"back_image_ref,omitempty"`
}

// Engine generates settlement batches.
type Engine struct {
	db           *sql.DB
	transferRepo *transfer.Repository
	ledgerSvc    *ledger.Service
	nowFn        func() time.Time
}

// NewEngine creates a new settlement engine.
func NewEngine(db *sql.DB, transferRepo *transfer.Repository, ledgerSvc *ledger.Service) *Engine {
	return &Engine{
		db:           db,
		transferRepo: transferRepo,
		ledgerSvc:    ledgerSvc,
		nowFn:        time.Now,
	}
}

// SetNowFunc overrides the clock source used by the engine.
func (e *Engine) SetNowFunc(nowFn func() time.Time) {
	if nowFn == nil {
		e.nowFn = time.Now
		return
	}
	e.nowFn = nowFn
}

// SettlementHealth returns counts and EOD state for monitoring (missing or delayed settlement).
func (e *Engine) SettlementHealth(now time.Time) (unsettledCount int, afterEOD bool, err error) {
	transfers, err := e.transferRepo.ListTransfers(transfer.StateFundsPosted)
	if err != nil {
		return 0, false, err
	}
	return len(transfers), IsAfterEOD(now), nil
}

// ReportSummary is a single row in the list of previous settlement reports (batches).
type ReportSummary struct {
	BatchID        string  `json:"batch_id"`
	SettlementAckAt string `json:"settlement_ack_at"`
	Count          int     `json:"count"`
	TotalAmount    float64 `json:"total_amount"`
}

// Status holds settlement overview for the operator UI.
type Status struct {
	UnsettledCount  int     `json:"unsettled_count"`
	UnsettledAmount float64 `json:"unsettled_amount"`
	SettledCount    int     `json:"settled_count"`
	SettledAmount   float64 `json:"settled_amount"`
	LastReportAt    string  `json:"last_report_at,omitempty"` // RFC3339 when last report was generated
}

// Status returns counts and amounts for FundsPosted (pending settlement) and Completed (settled).
func (e *Engine) Status() (*Status, error) {
	unsettled, err := e.transferRepo.ListTransfers(transfer.StateFundsPosted)
	if err != nil {
		return nil, err
	}
	settled, err := e.transferRepo.ListTransfers(transfer.StateCompleted)
	if err != nil {
		return nil, err
	}
	var unsettledAmt, settledAmt float64
	for _, t := range unsettled {
		unsettledAmt += t.Amount
	}
	for _, t := range settled {
		settledAmt += t.Amount
	}
	lastReport, _ := e.readLastReportAt()
	var lastReportStr string
	if !lastReport.IsZero() {
		lastReportStr = lastReport.Format(time.RFC3339)
	}
	return &Status{
		UnsettledCount:  len(unsettled),
		UnsettledAmount: unsettledAmt,
		SettledCount:    len(settled),
		SettledAmount:   settledAmt,
		LastReportAt:    lastReportStr,
	}, nil
}

// ListReports returns all previous settlement batches (reports), newest first.
func (e *Engine) ListReports() ([]ReportSummary, error) {
	batches, err := e.transferRepo.ListSettlementBatches()
	if err != nil {
		return nil, err
	}
	out := make([]ReportSummary, len(batches))
	for i := range batches {
		out[i] = ReportSummary{
			BatchID:         batches[i].BatchID,
			SettlementAckAt:  batches[i].SettlementAckAt,
			Count:           batches[i].Count,
			TotalAmount:     batches[i].TotalAmount,
		}
	}
	return out, nil
}

// GetReport returns a full settlement report (batch) by batch ID, or nil if not found.
func (e *Engine) GetReport(batchID string) (*SettlementFile, error) {
	transfers, err := e.transferRepo.ListTransfersBySettlementBatch(batchID)
	if err != nil {
		return nil, err
	}
	if len(transfers) == 0 {
		return nil, nil
	}
	createdAt := ""
	if transfers[0].SettlementAckAt != "" {
		createdAt = transfers[0].SettlementAckAt
	}
	var totalAmount float64
	entries := make([]SettlementEntry, 0, len(transfers))
	for _, t := range transfers {
		entry := SettlementEntry{
			TransferID:    t.ID,
			AccountID:     t.AccountID,
			Amount:        t.Amount,
			TransactionID: t.TransactionID,
			FrontImageRef: fmt.Sprintf("%s/front", t.ID),
			BackImageRef:  fmt.Sprintf("%s/back", t.ID),
		}
		if t.MICRData != "" {
			var micr struct {
				Routing     string `json:"routing"`
				Account     string `json:"account"`
				CheckNumber string `json:"checkNumber"`
			}
			if err := json.Unmarshal([]byte(t.MICRData), &micr); err == nil {
				entry.MICRRouting = micr.Routing
				entry.MICRAccount = micr.Account
				entry.CheckNumber = micr.CheckNumber
			}
		}
		entries = append(entries, entry)
		totalAmount += t.Amount
	}
	return &SettlementFile{
		BatchID:     batchID,
		CreatedAt:   createdAt,
		Transfers:   entries,
		TotalCount:  len(entries),
		TotalAmount: totalAmount,
	}, nil
}

// RunSettlement runs settlement (FundsPosted → Completed) without writing a file. Used by the cron.
func (e *Engine) RunSettlement() (*SettlementFile, error) {
	return e.generateSettlementFile(false)
}

// GenerateSettlementFile creates a settlement batch for all FundsPosted transfers and optionally writes it to disk.
func (e *Engine) GenerateSettlementFile(writeFile bool) (*SettlementFile, error) {
	return e.generateSettlementFile(writeFile)
}

// generateSettlementFile creates a settlement batch for all FundsPosted transfers; writeFile controls disk write.
func (e *Engine) generateSettlementFile(writeFile bool) (*SettlementFile, error) {
	now := e.nowFn().UTC()
	batchID := uuid.New().String()
	triggerSettlementDate := TriggerSettlementDate(now)

	// Get all FundsPosted transfers
	transfers, err := e.transferRepo.ListTransfers(transfer.StateFundsPosted)
	if err != nil {
		return nil, fmt.Errorf("list funds posted: %w", err)
	}

	if len(transfers) == 0 {
		log.Println("settlement: no FundsPosted transfers to settle")
		return &SettlementFile{
			BatchID:   batchID,
			CreatedAt: now.Format(time.RFC3339),
			Transfers: []SettlementEntry{},
		}, nil
	}

	batch := &SettlementFile{
		BatchID:   batchID,
		CreatedAt: now.Format(time.RFC3339),
	}

	var totalAmount float64
	ackAt := now.Format(time.RFC3339)

	for _, t := range transfers {
		createdAt, parseErr := time.Parse(time.RFC3339, t.CreatedAt)
		if parseErr == nil {
			transferSettlementDate := SettlementDateForDeposit(createdAt)
			if transferSettlementDate.After(triggerSettlementDate) {
				// This transfer belongs to a future settlement business day.
				continue
			}
		} else {
			log.Printf("settlement: unable to parse created_at for transfer %s: %v", t.ID, parseErr)
		}

		entry := SettlementEntry{
			TransferID:    t.ID,
			AccountID:     t.AccountID,
			Amount:        t.Amount,
			TransactionID: t.TransactionID,
			// Image references for check detail records (format: transfer_id/image_type)
			FrontImageRef: fmt.Sprintf("%s/front", t.ID),
			BackImageRef:  fmt.Sprintf("%s/back", t.ID),
		}

		// Parse MICR data if available
		if t.MICRData != "" {
			var micr struct {
				Routing     string `json:"routing"`
				Account     string `json:"account"`
				CheckNumber string `json:"checkNumber"`
			}
			if err := json.Unmarshal([]byte(t.MICRData), &micr); err == nil {
				entry.MICRRouting = micr.Routing
				entry.MICRAccount = micr.Account
				entry.CheckNumber = micr.CheckNumber
			}
		}

		batch.Transfers = append(batch.Transfers, entry)
		totalAmount += t.Amount

		// Update transfer: set batch ID, ack time, transition to Completed
		t.SettlementBatchID = batchID
		t.SettlementAckAt = ackAt

		if err := t.Transition(transfer.StateCompleted); err != nil {
			log.Printf("settlement: transition transfer %s: %v", t.ID, err)
			continue
		}
		if err := e.transferRepo.UpdateTransferState(t); err != nil {
			log.Printf("settlement: update transfer %s: %v", t.ID, err)
		}
		trace.DepositTrace(t.ID, t.AccountID, "settlement_status", map[string]interface{}{"status": "completed", "batch_id": batchID})
	}

	batch.TotalCount = len(batch.Transfers)
	batch.TotalAmount = totalAmount

	if writeFile {
		if err := os.MkdirAll("settlements", 0755); err != nil {
			return nil, fmt.Errorf("create settlements dir: %w", err)
		}
		filename := filepath.Join("settlements", fmt.Sprintf("settlement-%s.json", batchID))
		data, err := json.MarshalIndent(batch, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal settlement: %w", err)
		}
		if err := os.WriteFile(filename, data, 0644); err != nil {
			return nil, fmt.Errorf("write settlement file: %w", err)
		}
		log.Printf("settlement: batch %s written to %s (%d transfers, $%.2f)", batchID, filename, batch.TotalCount, batch.TotalAmount)
	}
	return batch, nil
}

const lastReportAtFile = "settlements/.last_report_at"

// ReportSinceLastReport returns a report of all Completed transfers settled after the last report time,
// and updates the last report timestamp. First call includes all settled transfers ever.
func (e *Engine) ReportSinceLastReport() (report *SettlementFile, lastReportAt time.Time, err error) {
	since, err := e.readLastReportAt()
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("read last report at: %w", err)
	}
	transfers, err := e.transferRepo.ListSettledTransfersSince(since)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("list settled since: %w", err)
	}
	now := e.nowFn().UTC()
	report = &SettlementFile{
		BatchID:    fmt.Sprintf("report-%s", now.Format("20060102-150405")),
		CreatedAt:  now.Format(time.RFC3339),
		Transfers:  make([]SettlementEntry, 0, len(transfers)),
		TotalCount: len(transfers),
	}
	var totalAmount float64
	for _, t := range transfers {
		entry := SettlementEntry{
			TransferID:    t.ID,
			AccountID:     t.AccountID,
			Amount:        t.Amount,
			TransactionID: t.TransactionID,
			FrontImageRef: fmt.Sprintf("%s/front", t.ID),
			BackImageRef:  fmt.Sprintf("%s/back", t.ID),
		}
		if t.MICRData != "" {
			var micr struct {
				Routing     string `json:"routing"`
				Account     string `json:"account"`
				CheckNumber string `json:"checkNumber"`
			}
			if err := json.Unmarshal([]byte(t.MICRData), &micr); err == nil {
				entry.MICRRouting = micr.Routing
				entry.MICRAccount = micr.Account
				entry.CheckNumber = micr.CheckNumber
			}
		}
		report.Transfers = append(report.Transfers, entry)
		totalAmount += t.Amount
	}
	report.TotalAmount = totalAmount
	if err := e.writeLastReportAt(now); err != nil {
		return nil, time.Time{}, fmt.Errorf("write last report at: %w", err)
	}
	return report, now, nil
}

// LastReportAt returns the stored last-report timestamp (zero if never).
func (e *Engine) LastReportAt() (time.Time, error) {
	return e.readLastReportAt()
}

func (e *Engine) readLastReportAt() (time.Time, error) {
	b, err := os.ReadFile(lastReportAtFile)
	if err != nil {
		if os.IsNotExist(err) {
			return time.Time{}, nil
		}
		return time.Time{}, err
	}
	t, err := time.Parse(time.RFC3339, string(bytes.TrimSpace(b)))
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}

func (e *Engine) writeLastReportAt(t time.Time) error {
	if err := os.MkdirAll("settlements", 0755); err != nil {
		return err
	}
	return os.WriteFile(lastReportAtFile, []byte(t.UTC().Format(time.RFC3339)), 0644)
}
