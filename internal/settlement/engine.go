package settlement

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/checkstream/checkstream/internal/ledger"
	"github.com/checkstream/checkstream/internal/transfer"
)

// SettlementFile represents the X9-like settlement batch file.
type SettlementFile struct {
	BatchID    string             `json:"batch_id"`
	CreatedAt  string             `json:"created_at"`
	Transfers  []SettlementEntry  `json:"transfers"`
	TotalCount int                `json:"total_count"`
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
}

// NewEngine creates a new settlement engine.
func NewEngine(db *sql.DB, transferRepo *transfer.Repository, ledgerSvc *ledger.Service) *Engine {
	return &Engine{
		db:           db,
		transferRepo: transferRepo,
		ledgerSvc:    ledgerSvc,
	}
}

// GenerateSettlementFile creates a settlement batch for all FundsPosted transfers.
func (e *Engine) GenerateSettlementFile() (*SettlementFile, error) {
	now := time.Now().UTC()
	batchID := uuid.New().String()

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
	}

	batch.TotalCount = len(batch.Transfers)
	batch.TotalAmount = totalAmount

	// Write to settlements/ directory
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
	return batch, nil
}
