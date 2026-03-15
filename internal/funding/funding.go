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

// ContributionLookup is the interface for YTD contribution totals (used for retirement limits).
type ContributionLookup interface {
	SumPostedAmountByAccountYear(accountID string, year int) (float64, error)
}

// Service provides funding validation logic.
type Service struct {
	cfg   *Config
	repo  TransferLookup
	ytd   ContributionLookup
	yearFn func() int
}

// NewService creates a new funding service.
func NewService(cfg *Config, repo TransferLookup) *Service {
	return &Service{cfg: cfg, repo: repo, ytd: nil, yearFn: nil}
}

// NewServiceWithContributionLookup creates a funding service that enforces per-account-type contribution limits.
func NewServiceWithContributionLookup(cfg *Config, repo TransferLookup, ytd ContributionLookup) *Service {
	return &Service{cfg: cfg, repo: repo, ytd: ytd, yearFn: nil}
}

// CheckLimit validates that the deposit amount is within limits for the account.
// For standard accounts: single-deposit limit applies. For 401k/IRA etc.: annual contribution limit (YTD + amount) applies.
func (s *Service) CheckLimit(accountID string, amount float64) error {
	annualLimit := s.cfg.GetAnnualContributionLimit(accountID)
	if annualLimit > 0 && s.ytd != nil {
		year := CurrentYear()
		if s.yearFn != nil {
			year = s.yearFn()
		}
		ytd, err := s.ytd.SumPostedAmountByAccountYear(accountID, year)
		if err != nil {
			return fmt.Errorf("contribution check: %w", err)
		}
		if ytd+amount > annualLimit {
			return fmt.Errorf("%w: %.2f + %.2f YTD exceeds annual limit %.2f", ErrOverLimit, amount, ytd, annualLimit)
		}
		// Also cap a single deposit at the general limit for sanity
		if amount > s.cfg.DepositLimit {
			return fmt.Errorf("%w: single deposit %.2f > %.2f", ErrOverLimit, amount, s.cfg.DepositLimit)
		}
		return nil
	}
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
