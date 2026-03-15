package return_

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/checkstream/checkstream/internal/ledger"
	"github.com/checkstream/checkstream/internal/transfer"
)

const ReturnFee = 30.00

// ReturnRequest is the parameters for processing a return.
type ReturnRequest struct {
	TransferID string  `json:"transfer_id"`
	Reason     string  `json:"reason"`
	Fee        float64 `json:"fee,omitempty"`
}

// ReturnResult is the result of processing a return.
type ReturnResult struct {
	Transfer    *transfer.Transfer `json:"transfer"`
	ReversalFee float64            `json:"reversal_fee"`
	Reason      string             `json:"reason"`
}

// Service handles return/reversal processing.
type Service struct {
	db           *sql.DB
	transferRepo *transfer.Repository
	ledgerSvc    *ledger.Service
}

// NewService creates a new return service.
func NewService(db *sql.DB, transferRepo *transfer.Repository, ledgerSvc *ledger.Service) *Service {
	return &Service{
		db:           db,
		transferRepo: transferRepo,
		ledgerSvc:    ledgerSvc,
	}
}

// ProcessReturn handles a check return: reverses ledger entries, charges fee, transitions state.
func (s *Service) ProcessReturn(req *ReturnRequest) (*ReturnResult, error) {
	t, err := s.transferRepo.GetTransfer(req.TransferID)
	if err != nil {
		return nil, fmt.Errorf("get transfer: %w", err)
	}
	if t == nil {
		return nil, fmt.Errorf("transfer not found: %s", req.TransferID)
	}

	// Only FundsPosted or Completed transfers can be returned
	if t.State != transfer.StateFundsPosted && t.State != transfer.StateCompleted {
		return nil, fmt.Errorf("transfer %s is in state %s, cannot be returned", t.ID, t.State)
	}

	fee := req.Fee
	if fee <= 0 {
		fee = ReturnFee
	}

	// Find the original movement entry to get omnibus account
	entries, err := s.ledgerSvc.ListEntries(t.ID)
	if err != nil {
		return nil, fmt.Errorf("list ledger entries: %w", err)
	}

	var fromAccountID string // investor account
	var toAccountID string   // omnibus account
	for _, e := range entries {
		if e.IsReversal == 0 {
			fromAccountID = e.ToAccountID // original credit went to investor
			toAccountID = e.FromAccountID // original debit from omnibus
			break
		}
	}

	if fromAccountID == "" {
		// Fallback: use transfer account
		fromAccountID = t.AccountID
		toAccountID = "OMNIBUS-001"
	}

	// Create reversal entry in a DB transaction
	dbTx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}

	// Reversal: credit omnibus (toAccountID), debit investor (fromAccountID)
	if _, err := s.ledgerSvc.CreateReversalEntry(dbTx, toAccountID, fromAccountID, t.ID, t.Amount, fee); err != nil {
		dbTx.Rollback()
		return nil, fmt.Errorf("create reversal entry: %w", err)
	}

	originalState := t.State

	// Transition to Returned
	if err := t.Transition(transfer.StateReturned); err != nil {
		dbTx.Rollback()
		return nil, fmt.Errorf("transition to returned: %w", err)
	}

	// If the return arrives before settlement completion, ensure it is not associated
	// with any pending settlement metadata.
	if originalState == transfer.StateFundsPosted {
		t.SettlementBatchID = ""
		t.SettlementAckAt = ""
	}

	if _, err := dbTx.Exec(`UPDATE transfers SET state=?, settlement_batch_id=?, settlement_ack_at=?, updated_at=? WHERE id=?`,
		string(t.State), t.SettlementBatchID, t.SettlementAckAt, t.UpdatedAt, t.ID); err != nil {
		dbTx.Rollback()
		return nil, fmt.Errorf("update transfer state: %w", err)
	}

	if err := dbTx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	// Log notification
	log.Printf("RETURN NOTIFICATION: Transfer %s for account %s has been returned. Reason: %s. Fee: $%.2f",
		t.ID, t.AccountID, req.Reason, fee)

	return &ReturnResult{
		Transfer:    t,
		ReversalFee: fee,
		Reason:      req.Reason,
	}, nil
}
