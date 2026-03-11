package funding

import (
	"testing"

	"github.com/checkstream/checkstream/internal/transfer"
)

// mockRepo is a mock transfer lookup.
type mockRepo struct {
	transfers map[string]*transfer.Transfer
}

func (m *mockRepo) GetTransferByTransactionID(txnID string) (*transfer.Transfer, error) {
	t, ok := m.transfers[txnID]
	if !ok {
		return nil, nil
	}
	return t, nil
}

func newTestService() *Service {
	cfg := NewConfig()
	repo := &mockRepo{transfers: map[string]*transfer.Transfer{
		"TXN-existing": {ID: "transfer-old", TransactionID: "TXN-existing"},
	}}
	return NewService(cfg, repo)
}

func TestCheckLimit_Under(t *testing.T) {
	svc := newTestService()
	if err := svc.CheckLimit(1000.00); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestCheckLimit_AtLimit(t *testing.T) {
	svc := newTestService()
	if err := svc.CheckLimit(5000.00); err != nil {
		t.Errorf("expected no error at limit, got %v", err)
	}
}

func TestCheckLimit_Over(t *testing.T) {
	svc := newTestService()
	if err := svc.CheckLimit(5001.00); err == nil {
		t.Error("expected error over limit")
	}
}

func TestCheckDuplicate_New(t *testing.T) {
	svc := newTestService()
	if err := svc.CheckDuplicate("TXN-new"); err != nil {
		t.Errorf("expected no error for new txn, got %v", err)
	}
}

func TestCheckDuplicate_Existing(t *testing.T) {
	svc := newTestService()
	if err := svc.CheckDuplicate("TXN-existing"); err == nil {
		t.Error("expected error for duplicate txn")
	}
}

func TestCheckEligibility_Valid(t *testing.T) {
	svc := newTestService()
	if err := svc.CheckEligibility("ACC-001"); err != nil {
		t.Errorf("expected eligible account, got %v", err)
	}
}

func TestCheckEligibility_Invalid(t *testing.T) {
	svc := newTestService()
	if err := svc.CheckEligibility("UNKNOWN-999"); err == nil {
		t.Error("expected ineligible account error")
	}
}

func TestContributionDefault(t *testing.T) {
	cfg := NewConfig()
	if got := cfg.GetContributionDefault("ACC-RETIRE-001"); got != "individual" {
		t.Errorf("expected individual, got %s", got)
	}
	if got := cfg.GetContributionDefault("ACC-001"); got != "individual" {
		t.Errorf("expected individual, got %s", got)
	}
}
