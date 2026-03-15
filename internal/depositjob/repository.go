package depositjob

import (
	"database/sql"
	"fmt"
	"time"
)

// Status values for deposit_jobs.
const (
	StatusPending = "pending"
	StatusRunning = "running"
	StatusDone    = "done"
	StatusFailed  = "failed"
)

// Job represents a deposit processing job.
type Job struct {
	ID          int64
	TransferID  string
	Status      string
	Scenario    string
	Source      string
	ErrorMessage string
	CreatedAt   string
	UpdatedAt   string
}

// Repository manages deposit_jobs.
type Repository struct {
	db    *sql.DB
	nowFn func() time.Time
}

// NewRepository creates a new deposit job repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db, nowFn: time.Now}
}

// Add enqueues a job for the given transfer. Idempotent: ignores if transfer_id already exists (pending/running).
func (r *Repository) Add(transferID, scenario, source string) error {
	now := r.nowFn().UTC().Format(time.RFC3339)
	_, err := r.db.Exec(`
		INSERT INTO deposit_jobs (transfer_id, status, scenario, source, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		transferID, StatusPending, scenario, source, now, now,
	)
	if err != nil {
		return fmt.Errorf("add deposit job: %w", err)
	}
	return nil
}

// ClaimNext claims one pending job (status -> running) and returns it. Returns (nil, false) if none.
func (r *Repository) ClaimNext() (*Job, bool, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, false, fmt.Errorf("begin claim: %w", err)
	}
	defer tx.Rollback()

	var j Job
	err = tx.QueryRow(`
		SELECT id, transfer_id, status, COALESCE(scenario,''), COALESCE(source,''), COALESCE(error_message,''), created_at, updated_at
		FROM deposit_jobs WHERE status = ? ORDER BY id ASC LIMIT 1`,
		StatusPending,
	).Scan(&j.ID, &j.TransferID, &j.Status, &j.Scenario, &j.Source, &j.ErrorMessage, &j.CreatedAt, &j.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("select pending job: %w", err)
	}

	now := r.nowFn().UTC().Format(time.RFC3339)
	_, err = tx.Exec(`UPDATE deposit_jobs SET status = ?, updated_at = ? WHERE id = ?`, StatusRunning, now, j.ID)
	if err != nil {
		return nil, false, fmt.Errorf("claim job: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, false, fmt.Errorf("commit claim: %w", err)
	}
	j.Status = StatusRunning
	j.UpdatedAt = now
	return &j, true, nil
}

// Complete marks a job as done.
func (r *Repository) Complete(id int64) error {
	now := r.nowFn().UTC().Format(time.RFC3339)
	_, err := r.db.Exec(`UPDATE deposit_jobs SET status = ?, updated_at = ? WHERE id = ?`, StatusDone, now, id)
	if err != nil {
		return fmt.Errorf("complete job: %w", err)
	}
	return nil
}

// Fail marks a job as failed and stores the error message.
func (r *Repository) Fail(id int64, errMsg string) error {
	now := r.nowFn().UTC().Format(time.RFC3339)
	_, err := r.db.Exec(`UPDATE deposit_jobs SET status = ?, error_message = ?, updated_at = ? WHERE id = ?`, StatusFailed, errMsg, now, id)
	if err != nil {
		return fmt.Errorf("fail job: %w", err)
	}
	return nil
}

// GetByTransferID returns the job for the given transfer, if any.
func (r *Repository) GetByTransferID(transferID string) (*Job, error) {
	var j Job
	err := r.db.QueryRow(`
		SELECT id, transfer_id, status, COALESCE(scenario,''), COALESCE(source,''), COALESCE(error_message,''), created_at, updated_at
		FROM deposit_jobs WHERE transfer_id = ?`, transferID,
	).Scan(&j.ID, &j.TransferID, &j.Status, &j.Scenario, &j.Source, &j.ErrorMessage, &j.CreatedAt, &j.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get job by transfer: %w", err)
	}
	return &j, nil
}
