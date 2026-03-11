package funding

import (
	"errors"
	"fmt"

	"github.com/checkstream/checkstream/internal/transfer"
)

var (
	ErrOverLimit      = errors.New("deposit amount exceeds limit")
	ErrDuplicate      = errors.New("duplicate transaction detected")
	ErrIneligible     = errors.New("account not eligible for check deposit")
	ErrInvalidSession = errors.New("invalid or missing session")
)

// TransferLookup is the interface for looking up transfers by transaction ID.
type TransferLookup interface {
	GetTransferByTransactionID(txnID string) (*transfer.Transfer, error)
}

// Service provides funding validation logic.
type Service struct {
	cfg  *Config
	repo TransferLookup
}

// NewService creates a new funding service.
func NewService(cfg *Config, repo TransferLookup) *Service {
	return &Service{cfg: cfg, repo: repo}
}

// CheckLimit validates that the deposit amount does not exceed the configured limit.
func (s *Service) CheckLimit(amount float64) error {
	if amount > s.cfg.DepositLimit {
		return fmt.Errorf("%w: %.2f > %.2f", ErrOverLimit, amount, s.cfg.DepositLimit)
	}
	return nil
}

// CheckDuplicate validates that the transaction ID hasn't been seen before.
// If transactionID is empty, no check is performed.
func (s *Service) CheckDuplicate(transactionID string) error {
	if transactionID == "" {
		return nil
	}
	existing, err := s.repo.GetTransferByTransactionID(transactionID)
	if err != nil {
		return fmt.Errorf("duplicate check: %w", err)
	}
	if existing != nil {
		return fmt.Errorf("%w: %s already processed as transfer %s", ErrDuplicate, transactionID, existing.ID)
	}
	return nil
}

// ValidateSession validates that a session token is present.
// In this stub implementation, any non-empty session is valid.
func (s *Service) ValidateSession(sessionToken string) error {
	if sessionToken == "" {
		return ErrInvalidSession
	}
	return nil
}

// CheckEligibility validates that the account is eligible for check deposit.
// An account is eligible if it appears in the omnibus map.
func (s *Service) CheckEligibility(accountID string) error {
	if s.cfg.GetOmnibusAccount(accountID) == "" {
		return fmt.Errorf("%w: account %s not found in omnibus map", ErrIneligible, accountID)
	}
	return nil
}
