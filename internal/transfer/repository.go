package transfer

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Repository handles persistence for transfers.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new transfer repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// CreateTransfer inserts a new transfer with state Requested.
func (r *Repository) CreateTransfer(accountID string, amount float64) (*Transfer, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	t := &Transfer{
		ID:        uuid.New().String(),
		AccountID: accountID,
		Amount:    amount,
		State:     StateRequested,
		CreatedAt: now,
		UpdatedAt: now,
	}

	_, err := r.db.Exec(`
		INSERT INTO transfers (id, account_id, amount, state, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		t.ID, t.AccountID, t.Amount, string(t.State), t.CreatedAt, t.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create transfer: %w", err)
	}
	return t, nil
}

// GetTransfer retrieves a transfer by ID.
func (r *Repository) GetTransfer(id string) (*Transfer, error) {
	row := r.db.QueryRow(`
		SELECT id, account_id, amount, state,
		       COALESCE(vendor_response,''), COALESCE(front_image_path,''), COALESCE(back_image_path,''),
		       COALESCE(micr_data,''), COALESCE(ocr_amount,0), COALESCE(entered_amount,0),
		       COALESCE(transaction_id,''), COALESCE(contribution_type,''),
		       COALESCE(settlement_batch_id,''), COALESCE(settlement_ack_at,''),
		       created_at, updated_at
		FROM transfers WHERE id = ?`, id)

	t := &Transfer{}
	err := row.Scan(
		&t.ID, &t.AccountID, &t.Amount, &t.State,
		&t.VendorResponse, &t.FrontImagePath, &t.BackImagePath,
		&t.MICRData, &t.OCRAmount, &t.EnteredAmount,
		&t.TransactionID, &t.ContributionType,
		&t.SettlementBatchID, &t.SettlementAckAt,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get transfer: %w", err)
	}
	return t, nil
}

// UpdateTransferState updates the transfer state and all mutable fields.
func (r *Repository) UpdateTransferState(t *Transfer) error {
	now := time.Now().UTC().Format(time.RFC3339)
	t.UpdatedAt = now
	_, err := r.db.Exec(`
		UPDATE transfers SET
			state = ?,
			vendor_response = ?,
			front_image_path = ?,
			back_image_path = ?,
			micr_data = ?,
			ocr_amount = ?,
			entered_amount = ?,
			transaction_id = ?,
			contribution_type = ?,
			settlement_batch_id = ?,
			settlement_ack_at = ?,
			updated_at = ?
		WHERE id = ?`,
		string(t.State),
		t.VendorResponse,
		t.FrontImagePath,
		t.BackImagePath,
		t.MICRData,
		t.OCRAmount,
		t.EnteredAmount,
		t.TransactionID,
		t.ContributionType,
		t.SettlementBatchID,
		t.SettlementAckAt,
		now,
		t.ID,
	)
	if err != nil {
		return fmt.Errorf("update transfer state: %w", err)
	}
	return nil
}

// ListTransfers returns all transfers, optionally filtered by state.
func (r *Repository) ListTransfers(state State) ([]*Transfer, error) {
	query := `
		SELECT id, account_id, amount, state,
		       COALESCE(vendor_response,''), COALESCE(front_image_path,''), COALESCE(back_image_path,''),
		       COALESCE(micr_data,''), COALESCE(ocr_amount,0), COALESCE(entered_amount,0),
		       COALESCE(transaction_id,''), COALESCE(contribution_type,''),
		       COALESCE(settlement_batch_id,''), COALESCE(settlement_ack_at,''),
		       created_at, updated_at
		FROM transfers`
	args := []interface{}{}
	if state != "" {
		query += " WHERE state = ?"
		args = append(args, string(state))
	}
	query += " ORDER BY created_at DESC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list transfers: %w", err)
	}
	defer rows.Close()

	var transfers []*Transfer
	for rows.Next() {
		t := &Transfer{}
		err := rows.Scan(
			&t.ID, &t.AccountID, &t.Amount, &t.State,
			&t.VendorResponse, &t.FrontImagePath, &t.BackImagePath,
			&t.MICRData, &t.OCRAmount, &t.EnteredAmount,
			&t.TransactionID, &t.ContributionType,
			&t.SettlementBatchID, &t.SettlementAckAt,
			&t.CreatedAt, &t.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan transfer: %w", err)
		}
		transfers = append(transfers, t)
	}
	return transfers, rows.Err()
}

// GetTransferByTransactionID looks up a transfer by its vendor transaction_id.
func (r *Repository) GetTransferByTransactionID(txnID string) (*Transfer, error) {
	row := r.db.QueryRow(`
		SELECT id, account_id, amount, state,
		       COALESCE(vendor_response,''), COALESCE(front_image_path,''), COALESCE(back_image_path,''),
		       COALESCE(micr_data,''), COALESCE(ocr_amount,0), COALESCE(entered_amount,0),
		       COALESCE(transaction_id,''), COALESCE(contribution_type,''),
		       COALESCE(settlement_batch_id,''), COALESCE(settlement_ack_at,''),
		       created_at, updated_at
		FROM transfers WHERE transaction_id = ? LIMIT 1`, txnID)

	t := &Transfer{}
	err := row.Scan(
		&t.ID, &t.AccountID, &t.Amount, &t.State,
		&t.VendorResponse, &t.FrontImagePath, &t.BackImagePath,
		&t.MICRData, &t.OCRAmount, &t.EnteredAmount,
		&t.TransactionID, &t.ContributionType,
		&t.SettlementBatchID, &t.SettlementAckAt,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get transfer by txn id: %w", err)
	}
	return t, nil
}
