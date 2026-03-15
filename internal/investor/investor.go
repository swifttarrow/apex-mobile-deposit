package investor

import (
	"database/sql"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Account represents an investor (mobile user) who can log in and make deposits.
// ID is the user_id used for account scoping (e.g. "alice", "bob" in UserAccountIDs).
type Account struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	DisplayName  string `json:"display_name"`
	PasswordHash string `json:"-"`
	CreatedAt    string `json:"created_at"`
}

// InvestorRepo provides investor account persistence.
type InvestorRepo struct {
	db *sql.DB
}

// NewInvestorRepo creates a new investor repository.
func NewInvestorRepo(db *sql.DB) *InvestorRepo {
	return &InvestorRepo{db: db}
}

// GetByUsername returns an investor by username, or nil if not found.
func (r *InvestorRepo) GetByUsername(username string) (*Account, error) {
	row := r.db.QueryRow(`
		SELECT id, username, password_hash, display_name, created_at
		FROM investors WHERE username = ?`, username)
	acc := &Account{}
	err := row.Scan(&acc.ID, &acc.Username, &acc.PasswordHash, &acc.DisplayName, &acc.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get investor by username: %w", err)
	}
	return acc, nil
}

// GetByID returns an investor by ID (user_id), or nil if not found.
func (r *InvestorRepo) GetByID(id string) (*Account, error) {
	row := r.db.QueryRow(`
		SELECT id, username, password_hash, display_name, created_at
		FROM investors WHERE id = ?`, id)
	acc := &Account{}
	err := row.Scan(&acc.ID, &acc.Username, &acc.PasswordHash, &acc.DisplayName, &acc.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get investor by id: %w", err)
	}
	return acc, nil
}

// VerifyPassword checks if the given password matches the investor's hash.
func (a *Account) VerifyPassword(password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(a.PasswordHash), []byte(password)) == nil
}

// SeedTestInvestors inserts test investor accounts if the investors table is empty.
// Passwords: alice / password, bob / password.
func (r *InvestorRepo) SeedTestInvestors() error {
	var count int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM investors`).Scan(&count); err != nil {
		return fmt.Errorf("seed check: %w", err)
	}
	if count > 0 {
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	hashStr := string(hash)
	now := time.Now().UTC().Format(time.RFC3339)

	for _, row := range []struct {
		id, username, displayName string
	}{
		{"alice", "alice", "Alice"},
		{"bob", "bob", "Bob"},
	} {
		_, err := r.db.Exec(`
			INSERT INTO investors (id, username, password_hash, display_name, created_at)
			VALUES (?, ?, ?, ?, ?)`,
			row.id, row.username, hashStr, row.displayName, now,
		)
		if err != nil {
			return fmt.Errorf("insert investor %s: %w", row.username, err)
		}
	}
	return nil
}
