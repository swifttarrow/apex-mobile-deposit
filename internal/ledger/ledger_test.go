package ledger

import (
	"testing"

	"github.com/checkstream/checkstream/internal/db"
)

func TestCreateMovementEntry(t *testing.T) {
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	svc := NewService(database)
	entry, err := svc.CreateMovementEntry(nil, "ACC-001", "OMNIBUS-001", "transfer-1", "individual", 150.00)
	if err != nil {
		t.Fatalf("create entry: %v", err)
	}
	if entry.ID == "" {
		t.Error("expected non-empty ID")
	}
	if entry.Amount != 150.00 {
		t.Errorf("expected 150.00, got %f", entry.Amount)
	}
	if entry.IsReversal != 0 {
		t.Error("expected not a reversal")
	}
}

func TestCreateReversalEntry(t *testing.T) {
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	svc := NewService(database)
	entry, err := svc.CreateReversalEntry(nil, "OMNIBUS-001", "ACC-001", "transfer-1", 150.00, 30.00)
	if err != nil {
		t.Fatalf("create reversal: %v", err)
	}
	if entry.IsReversal != 1 {
		t.Error("expected reversal flag")
	}
	if entry.ReversalFee != 30.00 {
		t.Errorf("expected fee 30.00, got %f", entry.ReversalFee)
	}
}

func TestGetAccountBalance(t *testing.T) {
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	svc := NewService(database)
	// Deposit
	if _, err := svc.CreateMovementEntry(nil, "ACC-001", "OMNIBUS-001", "transfer-1", "individual", 500.00); err != nil {
		t.Fatalf("create entry: %v", err)
	}

	balance, err := svc.GetAccountBalance("ACC-001")
	if err != nil {
		t.Fatalf("get balance: %v", err)
	}
	if balance != 500.00 {
		t.Errorf("expected 500.00, got %f", balance)
	}
}
