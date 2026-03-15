package settlement

import (
	"os"
	"testing"
	"time"

	"github.com/checkstream/checkstream/internal/db"
	"github.com/checkstream/checkstream/internal/ledger"
	"github.com/checkstream/checkstream/internal/transfer"
)

func mustTime(t *testing.T, value string) time.Time {
	t.Helper()
	ts, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("parse time %q: %v", value, err)
	}
	return ts
}

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

func TestSettlementDateForDeposit_AfterCutoffRollsNextBusinessDay(t *testing.T) {
	// Friday after cutoff should roll to Monday.
	depositAt := mustTime(t, "2024-01-13T00:31:00Z") // 2024-01-12 6:31 PM CT
	settlementDate := SettlementDateForDeposit(depositAt)
	if settlementDate.Format("2006-01-02") != "2024-01-15" {
		t.Fatalf("expected next business day 2024-01-15, got %s", settlementDate.Format("2006-01-02"))
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
	engine.nowFn = func() time.Time { return mustTime(t, "2024-01-16T01:00:00Z") } // 7:00 PM CT

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
	// Ensure the transfer falls before cutoff for the trigger business day.
	if _, err := database.Exec(`UPDATE transfers SET created_at=? WHERE id=?`, "2024-01-16T00:29:00Z", tr.ID); err != nil {
		t.Fatalf("set created_at: %v", err)
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

func TestGenerateSettlementFile_RespectsEODCutoff(t *testing.T) {
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

	// Trigger at 7:00 PM CT (2024-01-16T01:00:00Z). Same-day settlement should
	// include pre-cutoff deposits and defer post-cutoff deposits.
	engine.nowFn = func() time.Time { return mustTime(t, "2024-01-16T01:00:00Z") }

	beforeCutoff := createFundsPostedTransferForSettlementTest(t, transferRepo, "ACC-BEFORE", 101.00)
	afterCutoff := createFundsPostedTransferForSettlementTest(t, transferRepo, "ACC-AFTER", 202.00)

	// 2024-01-15 18:29 CT (before cutoff)
	if _, err := database.Exec(`UPDATE transfers SET created_at=? WHERE id=?`, "2024-01-16T00:29:00Z", beforeCutoff.ID); err != nil {
		t.Fatalf("set created_at before cutoff: %v", err)
	}
	// 2024-01-15 18:31 CT (after cutoff -> next business day)
	if _, err := database.Exec(`UPDATE transfers SET created_at=? WHERE id=?`, "2024-01-16T00:31:00Z", afterCutoff.ID); err != nil {
		t.Fatalf("set created_at after cutoff: %v", err)
	}

	batch, err := engine.GenerateSettlementFile()
	if err != nil {
		t.Fatalf("generate settlement: %v", err)
	}
	if batch.TotalCount != 1 {
		t.Fatalf("expected 1 eligible transfer, got %d", batch.TotalCount)
	}
	if batch.Transfers[0].TransferID != beforeCutoff.ID {
		t.Fatalf("expected before-cutoff transfer %s, got %s", beforeCutoff.ID, batch.Transfers[0].TransferID)
	}

	updatedBefore, err := transferRepo.GetTransfer(beforeCutoff.ID)
	if err != nil {
		t.Fatalf("get before-cutoff transfer: %v", err)
	}
	if updatedBefore.State != transfer.StateCompleted {
		t.Fatalf("expected before-cutoff transfer to be Completed, got %s", updatedBefore.State)
	}

	updatedAfter, err := transferRepo.GetTransfer(afterCutoff.ID)
	if err != nil {
		t.Fatalf("get after-cutoff transfer: %v", err)
	}
	if updatedAfter.State != transfer.StateFundsPosted {
		t.Fatalf("expected after-cutoff transfer to remain FundsPosted, got %s", updatedAfter.State)
	}
}

func createFundsPostedTransferForSettlementTest(t *testing.T, repo *transfer.Repository, accountID string, amount float64) *transfer.Transfer {
	t.Helper()
	tr, err := repo.CreateTransfer(accountID, amount)
	if err != nil {
		t.Fatalf("create transfer: %v", err)
	}
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
	if err := repo.UpdateTransferState(tr); err != nil {
		t.Fatalf("update state: %v", err)
	}
	return tr
}
