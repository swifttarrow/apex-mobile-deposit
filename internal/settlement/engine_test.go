package settlement

import (
	"os"
	"testing"
	"time"

	"github.com/checkstream/checkstream/internal/db"
	"github.com/checkstream/checkstream/internal/ledger"
	"github.com/checkstream/checkstream/internal/transfer"
)

func TestIsAfterEOD(t *testing.T) {
	loc, _ := time.LoadLocation("America/Chicago")

	// 6:31 PM CT - after cutoff
	after := time.Date(2024, 1, 15, 18, 31, 0, 0, loc)
	if !IsAfterEOD(after) {
		t.Error("expected 6:31 PM CT to be after EOD")
	}

	// 6:29 PM CT - before cutoff
	before := time.Date(2024, 1, 15, 18, 29, 0, 0, loc)
	if IsAfterEOD(before) {
		t.Error("expected 6:29 PM CT to be before EOD")
	}

	// exactly 6:30 PM CT - not after
	exact := time.Date(2024, 1, 15, 18, 30, 0, 0, loc)
	if IsAfterEOD(exact) {
		t.Error("expected exactly 6:30 PM CT to not be after EOD")
	}
}

func TestGenerateSettlementFile_Empty(t *testing.T) {
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	// Use a temp dir for settlements
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	transferRepo := transfer.NewRepository(database)
	ledgerSvc := ledger.NewService(database)
	engine := NewEngine(database, transferRepo, ledgerSvc)

	batch, err := engine.GenerateSettlementFile()
	if err != nil {
		t.Fatalf("generate settlement: %v", err)
	}
	if batch.TotalCount != 0 {
		t.Errorf("expected 0 transfers, got %d", batch.TotalCount)
	}
}

func TestGenerateSettlementFile_WithTransfers(t *testing.T) {
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	transferRepo := transfer.NewRepository(database)
	ledgerSvc := ledger.NewService(database)
	engine := NewEngine(database, transferRepo, ledgerSvc)

	// Create a FundsPosted transfer manually
	tr, err := transferRepo.CreateTransfer("ACC-001", 100.00)
	if err != nil {
		t.Fatalf("create transfer: %v", err)
	}
	// Walk through states
	states := []transfer.State{
		transfer.StateValidating,
		transfer.StateAnalyzing,
		transfer.StateApproved,
		transfer.StateFundsPosted,
	}
	for _, s := range states {
		if err := tr.Transition(s); err != nil {
			t.Fatalf("transition to %s: %v", s, err)
		}
	}
	if err := transferRepo.UpdateTransferState(tr); err != nil {
		t.Fatalf("update state: %v", err)
	}

	batch, err := engine.GenerateSettlementFile()
	if err != nil {
		t.Fatalf("generate settlement: %v", err)
	}
	if batch.TotalCount != 1 {
		t.Errorf("expected 1 transfer, got %d", batch.TotalCount)
	}
	if batch.TotalAmount != 100.00 {
		t.Errorf("expected 100.00 total, got %.2f", batch.TotalAmount)
	}

	// Verify transfer transitioned to Completed
	updated, err := transferRepo.GetTransfer(tr.ID)
	if err != nil {
		t.Fatalf("get transfer: %v", err)
	}
	if updated.State != transfer.StateCompleted {
		t.Errorf("expected Completed state, got %s", updated.State)
	}
	if updated.SettlementBatchID == "" {
		t.Error("expected settlement batch ID set")
	}

	// Verify settlement file contains image refs
	if len(batch.Transfers) > 0 {
		entry := batch.Transfers[0]
		if entry.FrontImageRef != tr.ID+"/front" || entry.BackImageRef != tr.ID+"/back" {
			t.Errorf("expected image refs, got front=%q back=%q", entry.FrontImageRef, entry.BackImageRef)
		}
	}
}
