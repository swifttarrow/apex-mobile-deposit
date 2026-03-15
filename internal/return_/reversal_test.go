package return_

import (
	"os"
	"testing"

	"github.com/checkstream/checkstream/internal/db"
	"github.com/checkstream/checkstream/internal/ledger"
	"github.com/checkstream/checkstream/internal/settlement"
	"github.com/checkstream/checkstream/internal/transfer"
)

func setupReturnTest(t *testing.T) (*Service, *transfer.Repository, *ledger.Service) {
	t.Helper()
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	transferRepo := transfer.NewRepository(database)
	ledgerSvc := ledger.NewService(database)
	svc := NewService(database, transferRepo, ledgerSvc)
	return svc, transferRepo, ledgerSvc
}

func createFundsPostedTransfer(t *testing.T, repo *transfer.Repository, ledgerSvc *ledger.Service) *transfer.Transfer {
	t.Helper()
	tr, err := repo.CreateTransfer("ACC-001", 500.00)
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
	// Create ledger entry
	if _, err := ledgerSvc.CreateMovementEntry(nil, "ACC-001", "OMNIBUS-001", tr.ID, "individual", tr.Amount); err != nil {
		t.Fatalf("create ledger entry: %v", err)
	}
	return tr
}

func TestProcessReturn_FundsPosted(t *testing.T) {
	svc, repo, ledgerSvc := setupReturnTest(t)
	tr := createFundsPostedTransfer(t, repo, ledgerSvc)

	result, err := svc.ProcessReturn(&ReturnRequest{
		TransferID: tr.ID,
		Reason:     "insufficient funds",
	})
	if err != nil {
		t.Fatalf("process return: %v", err)
	}
	if result.Transfer.State != transfer.StateReturned {
		t.Errorf("expected Returned, got %s", result.Transfer.State)
	}
	if result.ReversalFee != ReturnFee {
		t.Errorf("expected fee %.2f, got %.2f", ReturnFee, result.ReversalFee)
	}
}

func TestProcessReturn_NotFound(t *testing.T) {
	svc, _, _ := setupReturnTest(t)
	_, err := svc.ProcessReturn(&ReturnRequest{TransferID: "nonexistent"})
	if err == nil {
		t.Error("expected error for non-existent transfer")
	}
}

func TestProcessReturn_WrongState(t *testing.T) {
	svc, repo, _ := setupReturnTest(t)
	tr, err := repo.CreateTransfer("ACC-001", 100.00)
	if err != nil {
		t.Fatalf("create transfer: %v", err)
	}
	// Transfer is in Requested state, not eligible for return
	_, err = svc.ProcessReturn(&ReturnRequest{TransferID: tr.ID})
	if err == nil {
		t.Error("expected error for wrong state")
	}
}

func TestProcessReturn_CustomFee(t *testing.T) {
	svc, repo, ledgerSvc := setupReturnTest(t)
	tr := createFundsPostedTransfer(t, repo, ledgerSvc)

	result, err := svc.ProcessReturn(&ReturnRequest{
		TransferID: tr.ID,
		Reason:     "stop payment",
		Fee:        50.00,
	})
	if err != nil {
		t.Fatalf("process return: %v", err)
	}
	if result.ReversalFee != 50.00 {
		t.Errorf("expected custom fee 50.00, got %.2f", result.ReversalFee)
	}
}

func TestProcessReturn_Completed(t *testing.T) {
	svc, repo, ledgerSvc := setupReturnTest(t)
	tr := createFundsPostedTransfer(t, repo, ledgerSvc)
	// Move to Completed (e.g. after settlement)
	if err := tr.Transition(transfer.StateCompleted); err != nil {
		t.Fatalf("transition to Completed: %v", err)
	}
	if err := repo.UpdateTransferState(tr); err != nil {
		t.Fatalf("update state: %v", err)
	}

	result, err := svc.ProcessReturn(&ReturnRequest{
		TransferID: tr.ID,
		Reason:     "bounced after settlement",
	})
	if err != nil {
		t.Fatalf("process return from Completed: %v", err)
	}
	if result.Transfer.State != transfer.StateReturned {
		t.Errorf("expected Returned, got %s", result.Transfer.State)
	}
}

func TestProcessReturn_FundsPostedExcludedFromSettlement(t *testing.T) {
	svc, repo, ledgerSvc := setupReturnTest(t)
	tr := createFundsPostedTransfer(t, repo, ledgerSvc)

	if _, err := svc.ProcessReturn(&ReturnRequest{
		TransferID: tr.ID,
		Reason:     "nsf",
	}); err != nil {
		t.Fatalf("process return: %v", err)
	}

	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	engine := settlement.NewEngine(svc.db, repo, ledgerSvc)
	batch, err := engine.GenerateSettlementFile()
	if err != nil {
		t.Fatalf("generate settlement: %v", err)
	}
	if batch.TotalCount != 0 {
		t.Fatalf("expected returned FundsPosted transfer to be excluded, got %d", batch.TotalCount)
	}

	updated, err := repo.GetTransfer(tr.ID)
	if err != nil {
		t.Fatalf("get transfer: %v", err)
	}
	if updated.State != transfer.StateReturned {
		t.Fatalf("expected Returned state, got %s", updated.State)
	}
	if updated.SettlementBatchID != "" || updated.SettlementAckAt != "" {
		t.Fatalf("expected settlement metadata cleared for pre-settlement return")
	}
}
