package ledger

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const (
	TypeMovement = "MOVEMENT"
	MemoFree     = "FREE"
	SubTypeDeposit = "DEPOSIT"
	TransferTypeCheck = "CHECK"
	CurrencyUSD  = "USD"
)

// Entry represents a ledger entry.
type Entry struct {
	ID                  string  `json:"id"`
	TransferID          string  `json:"transfer_id"`
	ToAccountID         string  `json:"to_account_id"`
	FromAccountID       string  `json:"from_account_id"`
	Type                string  `json:"type"`
	Memo                string  `json:"memo"`
	SubType             string  `json:"sub_type"`
	TransferType        string  `json:"transfer_type"`
	Currency            string  `json:"currency"`
	Amount              float64 `json:"amount"`
	SourceApplicationID string  `json:"source_application_id"`
	ContributionType    string  `json:"contribution_type,omitempty"`
	CreatedAt           string  `json:"created_at"`
	IsReversal          int     `json:"is_reversal"`
	ReversalFee         float64 `json:"reversal_fee"`
}

// Service handles ledger entry creation.
type Service struct {
	db *sql.DB
}

// NewService creates a new ledger service.
func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// CreateMovementEntry creates a MOVEMENT ledger entry within an optional transaction.
func (s *Service) CreateMovementEntry(tx *sql.Tx, toAccountID, fromAccountID, transferID, contributionType string, amount float64) (*Entry, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	entry := &Entry{
		ID:                  uuid.New().String(),
		TransferID:          transferID,
		ToAccountID:         toAccountID,
		FromAccountID:       fromAccountID,
		Type:                TypeMovement,
		Memo:                MemoFree,
		SubType:             SubTypeDeposit,
		TransferType:        TransferTypeCheck,
		Currency:            CurrencyUSD,
		Amount:              amount,
		SourceApplicationID: transferID,
		ContributionType:    contributionType,
		CreatedAt:           now,
		IsReversal:          0,
		ReversalFee:         0,
	}

	query := `
		INSERT INTO ledger_entries
			(id, transfer_id, to_account_id, from_account_id, type, memo, sub_type, transfer_type,
			 currency, amount, source_application_id, contribution_type, created_at, is_reversal, reversal_fee)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	args := []interface{}{
		entry.ID, entry.TransferID, entry.ToAccountID, entry.FromAccountID,
		entry.Type, entry.Memo, entry.SubType, entry.TransferType,
		entry.Currency, entry.Amount, entry.SourceApplicationID, entry.ContributionType,
		entry.CreatedAt, entry.IsReversal, entry.ReversalFee,
	}

	var err error
	if tx != nil {
		_, err = tx.Exec(query, args...)
	} else {
		_, err = s.db.Exec(query, args...)
	}
	if err != nil {
		return nil, fmt.Errorf("create movement entry: %w", err)
	}
	return entry, nil
}

// CreateReversalEntry creates a reversal ledger entry (debits investor, credits omnibus).
func (s *Service) CreateReversalEntry(tx *sql.Tx, toAccountID, fromAccountID, transferID string, amount, fee float64) (*Entry, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	entry := &Entry{
		ID:                  uuid.New().String(),
		TransferID:          transferID,
		ToAccountID:         toAccountID,
		FromAccountID:       fromAccountID,
		Type:                TypeMovement,
		Memo:                MemoFree,
		SubType:             SubTypeDeposit,
		TransferType:        TransferTypeCheck,
		Currency:            CurrencyUSD,
		Amount:              amount,
		SourceApplicationID: transferID,
		CreatedAt:           now,
		IsReversal:          1,
		ReversalFee:         fee,
	}

	query := `
		INSERT INTO ledger_entries
			(id, transfer_id, to_account_id, from_account_id, type, memo, sub_type, transfer_type,
			 currency, amount, source_application_id, contribution_type, created_at, is_reversal, reversal_fee)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	args := []interface{}{
		entry.ID, entry.TransferID, entry.ToAccountID, entry.FromAccountID,
		entry.Type, entry.Memo, entry.SubType, entry.TransferType,
		entry.Currency, entry.Amount, entry.SourceApplicationID, "",
		entry.CreatedAt, entry.IsReversal, entry.ReversalFee,
	}

	var err error
	if tx != nil {
		_, err = tx.Exec(query, args...)
	} else {
		_, err = s.db.Exec(query, args...)
	}
	if err != nil {
		return nil, fmt.Errorf("create reversal entry: %w", err)
	}
	return entry, nil
}

// ListEntries returns all ledger entries, optionally filtered by transfer_id.
func (s *Service) ListEntries(transferID string) ([]*Entry, error) {
	query := `
		SELECT id, transfer_id, to_account_id, from_account_id, type, memo, sub_type, transfer_type,
		       currency, amount, source_application_id, COALESCE(contribution_type,''), created_at,
		       is_reversal, reversal_fee
		FROM ledger_entries`
	args := []interface{}{}
	if transferID != "" {
		query += " WHERE transfer_id = ?"
		args = append(args, transferID)
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list entries: %w", err)
	}
	defer rows.Close()

	var entries []*Entry
	for rows.Next() {
		e := &Entry{}
		if err := rows.Scan(
			&e.ID, &e.TransferID, &e.ToAccountID, &e.FromAccountID,
			&e.Type, &e.Memo, &e.SubType, &e.TransferType,
			&e.Currency, &e.Amount, &e.SourceApplicationID, &e.ContributionType,
			&e.CreatedAt, &e.IsReversal, &e.ReversalFee,
		); err != nil {
			return nil, fmt.Errorf("scan entry: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// GetAccountBalance calculates the current balance for an account.
func (s *Service) GetAccountBalance(accountID string) (float64, error) {
	// Credits (deposits TO this account)
	var credits float64
	err := s.db.QueryRow(`SELECT COALESCE(SUM(amount),0) FROM ledger_entries WHERE to_account_id = ? AND is_reversal = 0`, accountID).Scan(&credits)
	if err != nil {
		return 0, fmt.Errorf("sum credits: %w", err)
	}

	// Debits (reversals FROM this account)
	var debits float64
	err = s.db.QueryRow(`SELECT COALESCE(SUM(amount + reversal_fee),0) FROM ledger_entries WHERE from_account_id = ? AND is_reversal = 1`, accountID).Scan(&debits)
	if err != nil {
		return 0, fmt.Errorf("sum debits: %w", err)
	}

	return credits - debits, nil
}
