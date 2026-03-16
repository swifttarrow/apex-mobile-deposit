package transfer

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Repository handles persistence for transfers.
type Repository struct {
	db    *sql.DB
	nowFn func() time.Time
}

// NewRepository creates a new transfer repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{
		db:    db,
		nowFn: time.Now,
	}
}

// SetNowFunc overrides the clock source used for persisted timestamps.
func (r *Repository) SetNowFunc(nowFn func() time.Time) {
	if nowFn == nil {
		r.nowFn = time.Now
		return
	}
	r.nowFn = nowFn
}

// CreateTransfer inserts a new transfer with state Requested.
func (r *Repository) CreateTransfer(accountID string, amount float64) (*Transfer, error) {
	now := r.nowFn().UTC().Format(time.RFC3339)
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
	now := r.nowFn().UTC().Format(time.RFC3339)
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

// GetTransfersByIDs returns transfers for the given IDs. Missing IDs are omitted; order is not guaranteed.
func (r *Repository) GetTransfersByIDs(ids []string) ([]*Transfer, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	query := `
		SELECT id, account_id, amount, state,
		       COALESCE(vendor_response,''), COALESCE(front_image_path,''), COALESCE(back_image_path,''),
		       COALESCE(micr_data,''), COALESCE(ocr_amount,0), COALESCE(entered_amount,0),
		       COALESCE(transaction_id,''), COALESCE(contribution_type,''),
		       COALESCE(settlement_batch_id,''), COALESCE(settlement_ack_at,''),
		       created_at, updated_at
		FROM transfers WHERE id IN (` + strings.Join(placeholders, ",") + `)`
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("get transfers by ids: %w", err)
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

// CountTransfers returns the total number of transfers, optionally filtered by state.
func (r *Repository) CountTransfers(state State) (int, error) {
	query := "SELECT COUNT(*) FROM transfers"
	args := []interface{}{}
	if state != "" {
		query += " WHERE state = ?"
		args = append(args, string(state))
	}
	var n int
	err := r.db.QueryRow(query, args...).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count transfers: %w", err)
	}
	return n, nil
}

// ListTransfersPaginated returns transfers with limit and offset, optionally filtered by state, newest first.
func (r *Repository) ListTransfersPaginated(limit, offset int, state State) ([]*Transfer, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
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
	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

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

// ListSettledTransfersSince returns Completed transfers with settlement_ack_at > since (for settlement reports).
func (r *Repository) ListSettledTransfersSince(since time.Time) ([]*Transfer, error) {
	sinceStr := since.UTC().Format(time.RFC3339)
	query := `
		SELECT id, account_id, amount, state,
		       COALESCE(vendor_response,''), COALESCE(front_image_path,''), COALESCE(back_image_path,''),
		       COALESCE(micr_data,''), COALESCE(ocr_amount,0), COALESCE(entered_amount,0),
		       COALESCE(transaction_id,''), COALESCE(contribution_type,''),
		       COALESCE(settlement_batch_id,''), COALESCE(settlement_ack_at,''),
		       created_at, updated_at
		FROM transfers
		WHERE state = ? AND settlement_ack_at > ?
		ORDER BY settlement_ack_at ASC`
	rows, err := r.db.Query(query, string(StateCompleted), sinceStr)
	if err != nil {
		return nil, fmt.Errorf("list settled since: %w", err)
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

// SettlementBatchSummary is a row returned by ListSettlementBatches.
type SettlementBatchSummary struct {
	BatchID     string
	SettlementAckAt string
	Count       int
	TotalAmount float64
}

// CountCompletedTransfersNotInReport returns the count and total amount of Completed transfers
// that are not yet in a manually generated report (settlement_batch_id not starting with "report-").
func (r *Repository) CountCompletedTransfersNotInReport() (count int, totalAmount float64, err error) {
	query := `
		SELECT COUNT(*), COALESCE(SUM(amount), 0)
		FROM transfers
		WHERE state = ? AND (settlement_batch_id = '' OR settlement_batch_id NOT LIKE 'report-%')`
	err = r.db.QueryRow(query, string(StateCompleted)).Scan(&count, &totalAmount)
	if err != nil {
		return 0, 0, fmt.Errorf("count unreported settlements: %w", err)
	}
	return count, totalAmount, nil
}

// ListSettlementBatches returns distinct settlement batches (Completed transfers grouped by settlement_batch_id), newest first.
func (r *Repository) ListSettlementBatches() ([]SettlementBatchSummary, error) {
	query := `
		SELECT settlement_batch_id, MIN(settlement_ack_at) AS ack_at, COUNT(*) AS cnt, COALESCE(SUM(amount), 0) AS total
		FROM transfers
		WHERE state = ? AND settlement_batch_id != ''
		GROUP BY settlement_batch_id
		ORDER BY ack_at DESC`
	rows, err := r.db.Query(query, string(StateCompleted))
	if err != nil {
		return nil, fmt.Errorf("list settlement batches: %w", err)
	}
	defer rows.Close()
	var out []SettlementBatchSummary
	for rows.Next() {
		var s SettlementBatchSummary
		err := rows.Scan(&s.BatchID, &s.SettlementAckAt, &s.Count, &s.TotalAmount)
		if err != nil {
			return nil, fmt.Errorf("scan settlement batch: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ListTransfersBySettlementBatch returns all Completed transfers for the given settlement batch ID, ordered by settlement_ack_at.
func (r *Repository) ListTransfersBySettlementBatch(batchID string) ([]*Transfer, error) {
	query := `
		SELECT id, account_id, amount, state,
		       COALESCE(vendor_response,''), COALESCE(front_image_path,''), COALESCE(back_image_path,''),
		       COALESCE(micr_data,''), COALESCE(ocr_amount,0), COALESCE(entered_amount,0),
		       COALESCE(transaction_id,''), COALESCE(contribution_type,''),
		       COALESCE(settlement_batch_id,''), COALESCE(settlement_ack_at,''),
		       created_at, updated_at
		FROM transfers
		WHERE state = ? AND settlement_batch_id = ?
		ORDER BY settlement_ack_at ASC`
	rows, err := r.db.Query(query, string(StateCompleted), batchID)
	if err != nil {
		return nil, fmt.Errorf("list transfers by batch: %w", err)
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

// ListTransfersByAccount returns transfers for an account, ordered by created_at DESC, with pagination.
// If status is non-empty, filters by that state. limit=0 means no limit.
func (r *Repository) ListTransfersByAccount(accountID string, limit, offset int, status State) ([]*Transfer, error) {
	return r.ListTransfersByAccounts([]string{accountID}, limit, offset, status)
}

// ListTransfersByAccounts returns transfers for any of the given accounts, ordered by created_at DESC.
// If accountIDs is empty, returns nil. If status is non-empty, filters by that state. limit=0 means no limit.
func (r *Repository) ListTransfersByAccounts(accountIDs []string, limit, offset int, status State) ([]*Transfer, error) {
	if len(accountIDs) == 0 {
		return nil, nil
	}
	placeholders := make([]string, len(accountIDs))
	args := make([]interface{}, 0, len(accountIDs)+1)
	for i, id := range accountIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}
	query := `
		SELECT id, account_id, amount, state,
		       COALESCE(vendor_response,''), COALESCE(front_image_path,''), COALESCE(back_image_path,''),
		       COALESCE(micr_data,''), COALESCE(ocr_amount,0), COALESCE(entered_amount,0),
		       COALESCE(transaction_id,''), COALESCE(contribution_type,''),
		       COALESCE(settlement_batch_id,''), COALESCE(settlement_ack_at,''),
		       created_at, updated_at
		FROM transfers WHERE account_id IN (` + strings.Join(placeholders, ",") + `)`
	if status != "" {
		query += " AND state = ?"
		args = append(args, string(status))
	}
	query += " ORDER BY created_at DESC"
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list transfers by accounts: %w", err)
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

// SumPostedAmountByAccountYear returns the sum of amounts for an account in the given year
// for transfers in FundsPosted or Completed state (contribution YTD).
func (r *Repository) SumPostedAmountByAccountYear(accountID string, year int) (float64, error) {
	yearStr := fmt.Sprintf("%d", year)
	var sum sql.NullFloat64
	err := r.db.QueryRow(`
		SELECT COALESCE(SUM(amount), 0) FROM transfers
		WHERE account_id = ? AND (state = ? OR state = ?)
		  AND strftime('%Y', created_at) = ?`,
		accountID, string(StateFundsPosted), string(StateCompleted), yearStr).Scan(&sum)
	if err != nil {
		return 0, fmt.Errorf("sum posted amount by account year: %w", err)
	}
	if !sum.Valid {
		return 0, nil
	}
	return sum.Float64, nil
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
