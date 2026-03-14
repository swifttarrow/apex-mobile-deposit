package operator

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/checkstream/checkstream/internal/transfer"
)

// Action represents an operator action on a transfer.
type Action struct {
	ID                     string `json:"id"`
	TransferID             string `json:"transfer_id"`
	Action                 string `json:"action"`
	OperatorID             string `json:"operator_id"`
	Note                   string `json:"note,omitempty"`
	ContributionTypeOverride string `json:"contribution_type_override,omitempty"`
	CreatedAt              string `json:"created_at"`
}

// Repository handles operator actions and flagged transfer querying.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new operator repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// ListFlaggedTransfers returns transfers in the Analyzing state, with optional filters and pagination.
// limit=0 means no limit.
func (r *Repository) ListFlaggedTransfers(dateFilter, accountFilter string, amountMin, amountMax float64, limit, offset int) ([]*transfer.Transfer, error) {
	query := `
		SELECT id, account_id, amount, state,
		       COALESCE(vendor_response,''), COALESCE(front_image_path,''), COALESCE(back_image_path,''),
		       COALESCE(micr_data,''), COALESCE(ocr_amount,0), COALESCE(entered_amount,0),
		       COALESCE(transaction_id,''), COALESCE(contribution_type,''),
		       COALESCE(settlement_batch_id,''), COALESCE(settlement_ack_at,''),
		       created_at, updated_at
		FROM transfers WHERE state = 'Analyzing'`
	args := []interface{}{}

	if dateFilter != "" {
		query += " AND DATE(created_at) = ?"
		args = append(args, dateFilter)
	}
	if accountFilter != "" {
		query += " AND account_id = ?"
		args = append(args, accountFilter)
	}
	if amountMin > 0 {
		query += " AND amount >= ?"
		args = append(args, amountMin)
	}
	if amountMax > 0 {
		query += " AND amount <= ?"
		args = append(args, amountMax)
	}
	query += " ORDER BY created_at ASC"
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list flagged transfers: %w", err)
	}
	defer rows.Close()

	var transfers []*transfer.Transfer
	for rows.Next() {
		t := &transfer.Transfer{}
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

// GetTransfer retrieves a transfer by ID.
func (r *Repository) GetTransfer(id string) (*transfer.Transfer, error) {
	row := r.db.QueryRow(`
		SELECT id, account_id, amount, state,
		       COALESCE(vendor_response,''), COALESCE(front_image_path,''), COALESCE(back_image_path,''),
		       COALESCE(micr_data,''), COALESCE(ocr_amount,0), COALESCE(entered_amount,0),
		       COALESCE(transaction_id,''), COALESCE(contribution_type,''),
		       COALESCE(settlement_batch_id,''), COALESCE(settlement_ack_at,''),
		       created_at, updated_at
		FROM transfers WHERE id = ?`, id)

	t := &transfer.Transfer{}
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

// ListRecentActions returns the most recent operator actions across all transfers, ordered by created_at DESC.
// actionFilter: "" or "all" = no filter; "approve" = manual approve only; "auto_approve"; "approved" = both approve and auto_approve; "reject"
// operatorFilter: "" = no filter; else filter by operator_id
func (r *Repository) ListRecentActions(limit int, actionFilter, operatorFilter string) ([]*Action, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT id, transfer_id, action, operator_id,
		       COALESCE(note,''), COALESCE(contribution_type_override,''), created_at
		FROM operator_actions`
	args := []interface{}{}
	if actionFilter != "" && actionFilter != "all" {
		switch actionFilter {
		case "approved":
			query += " WHERE action IN ('approve', 'auto_approve')"
		case "approve", "auto_approve", "reject":
			query += " WHERE action = ?"
			args = append(args, actionFilter)
		}
	}
	if operatorFilter != "" {
		if len(args) > 0 {
			query += " AND operator_id = ?"
		} else {
			query += " WHERE operator_id = ?"
		}
		args = append(args, operatorFilter)
	}
	query += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list recent actions: %w", err)
	}
	defer rows.Close()

	var actions []*Action
	for rows.Next() {
		a := &Action{}
		if err := rows.Scan(&a.ID, &a.TransferID, &a.Action, &a.OperatorID, &a.Note, &a.ContributionTypeOverride, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan action: %w", err)
		}
		actions = append(actions, a)
	}
	return actions, rows.Err()
}

// ListAuditOperators returns distinct operator_id values for filter dropdown.
func (r *Repository) ListAuditOperators() ([]string, error) {
	rows, err := r.db.Query(`SELECT DISTINCT operator_id FROM operator_actions ORDER BY operator_id`)
	if err != nil {
		return nil, fmt.Errorf("list operators: %w", err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan operator: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// ListActions returns all operator actions for a transfer, ordered by created_at.
func (r *Repository) ListActions(transferID string) ([]*Action, error) {
	rows, err := r.db.Query(`
		SELECT id, transfer_id, action, operator_id,
		       COALESCE(note,''), COALESCE(contribution_type_override,''), created_at
		FROM operator_actions WHERE transfer_id = ? ORDER BY created_at ASC`, transferID)
	if err != nil {
		return nil, fmt.Errorf("list actions: %w", err)
	}
	defer rows.Close()

	var actions []*Action
	for rows.Next() {
		a := &Action{}
		if err := rows.Scan(&a.ID, &a.TransferID, &a.Action, &a.OperatorID, &a.Note, &a.ContributionTypeOverride, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan action: %w", err)
		}
		actions = append(actions, a)
	}
	return actions, rows.Err()
}

// RecordAction inserts an operator action record.
func (r *Repository) RecordAction(transferID, action, operatorID, note, contributionTypeOverride string) (*Action, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	a := &Action{
		ID:                     uuid.New().String(),
		TransferID:             transferID,
		Action:                 action,
		OperatorID:             operatorID,
		Note:                   note,
		ContributionTypeOverride: contributionTypeOverride,
		CreatedAt:              now,
	}

	_, err := r.db.Exec(`
		INSERT INTO operator_actions (id, transfer_id, action, operator_id, note, contribution_type_override, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.TransferID, a.Action, a.OperatorID, a.Note, a.ContributionTypeOverride, a.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("record action: %w", err)
	}
	return a, nil
}
