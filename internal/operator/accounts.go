package operator

import (
	"database/sql"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// OperatorAccount represents an operator user who can log in and review deposits.
type OperatorAccount struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	DisplayName  string `json:"display_name"`
	Email        string `json:"email"`
	PasswordHash string `json:"-"`
	CreatedAt    string `json:"created_at"`
}

// GetOperatorByUsername returns an operator by username, or nil if not found.
func (r *Repository) GetOperatorByUsername(username string) (*OperatorAccount, error) {
	row := r.db.QueryRow(`
		SELECT id, username, password_hash, display_name, email, created_at
		FROM operators WHERE username = ?`, username)
	o := &OperatorAccount{}
	err := row.Scan(&o.ID, &o.Username, &o.PasswordHash, &o.DisplayName, &o.Email, &o.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get operator by username: %w", err)
	}
	return o, nil
}

// GetOperatorByID returns an operator by ID, or nil if not found.
func (r *Repository) GetOperatorByID(id string) (*OperatorAccount, error) {
	row := r.db.QueryRow(`
		SELECT id, username, password_hash, display_name, email, created_at
		FROM operators WHERE id = ?`, id)
	o := &OperatorAccount{}
	err := row.Scan(&o.ID, &o.Username, &o.PasswordHash, &o.DisplayName, &o.Email, &o.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get operator by id: %w", err)
	}
	return o, nil
}

// VerifyPassword checks if the given password matches the operator's hash.
func (o *OperatorAccount) VerifyPassword(password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(o.PasswordHash), []byte(password)) == nil
}

// SeedTestOperators inserts 5 test operator accounts if the operators table is empty.
// Password for all: "password"
func (r *Repository) SeedTestOperators() error {
	var count int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM operators`).Scan(&count); err != nil {
		return fmt.Errorf("seed check: %w", err)
	}
	if count > 0 {
		return nil // already seeded
	}

	testAccounts := []struct {
		id, username, displayName, email string
	}{
		{"op-1", "joe", "Joe Doe", "joe@checkdepot.com"},
		{"op-2", "jane", "Jane Smith", "jane@checkdepot.com"},
		{"op-3", "bob", "Bob Wilson", "bob@checkdepot.com"},
		{"op-4", "alice", "Alice Chen", "alice@checkdepot.com"},
		{"op-5", "charlie", "Charlie Davis", "charlie@checkdepot.com"},
	}

	// bcrypt hash of "password" (cost 10)
	hash, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	hashStr := string(hash)
	now := time.Now().UTC().Format(time.RFC3339)

	for _, a := range testAccounts {
		_, err := r.db.Exec(`
			INSERT INTO operators (id, username, password_hash, display_name, email, created_at)
			VALUES (?, ?, ?, ?, ?, ?)`,
			a.id, a.username, hashStr, a.displayName, a.email, now,
		)
		if err != nil {
			return fmt.Errorf("insert operator %s: %w", a.username, err)
		}
	}
	return nil
}
