package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/checkstream/checkstream/internal/db"
	"github.com/checkstream/checkstream/internal/funding"
	"github.com/checkstream/checkstream/internal/ledger"
	"github.com/checkstream/checkstream/internal/operator"
	"github.com/checkstream/checkstream/internal/transfer"
	"github.com/checkstream/checkstream/internal/vendor"
)

func setupOperatorTestMux(t *testing.T) *http.ServeMux {
	t.Helper()
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	transferRepo := transfer.NewRepository(database)
	vendorStub := vendor.NewStub("../../config/scenarios.json")
	ledgerSvc := ledger.NewService(database)
	fundingCfg := funding.NewConfig()
	fundingSvc := funding.NewService(fundingCfg, transferRepo)
	operatorRepo := operator.NewRepository(database)

	depositHandler := NewDepositHandler(transferRepo, vendorStub, ledgerSvc, fundingSvc, fundingCfg, database)
	operatorHandler := NewOperatorHandler(operatorRepo, transferRepo, ledgerSvc, fundingCfg)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /deposits", WithIdempotency(database, depositHandler.Create))
	mux.HandleFunc("GET /deposits/{id}", depositHandler.Get)
	mux.HandleFunc("GET /operator/queue", operatorHandler.Queue)
	mux.HandleFunc("POST /operator/approve", operatorHandler.Approve)
	mux.HandleFunc("POST /operator/reject", operatorHandler.Reject)
	return mux
}

// createFlaggedDeposit submits a MICR fail deposit which lands in Analyzing.
func createFlaggedDeposit(t *testing.T, mux *http.ServeMux) string {
	t.Helper()
	body := map[string]interface{}{
		"account_id":  "ACC-MICR-FAIL",
		"amount":      200.00,
		"front_image": "front",
		"back_image":  "back",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/deposits", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rr.Code, rr.Body.String())
	}
	var result map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &result)
	tr := result["transfer"].(map[string]interface{})
	return tr["id"].(string)
}

func TestOperator_FlaggedInQueue(t *testing.T) {
	mux := setupOperatorTestMux(t)
	id := createFlaggedDeposit(t, mux)

	req := httptest.NewRequest(http.MethodGet, "/operator/queue", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var result map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &result)
	transfers := result["transfers"].([]interface{})
	if len(transfers) == 0 {
		t.Fatal("expected at least one flagged transfer in queue")
	}

	found := false
	for _, tr := range transfers {
		item := tr.(map[string]interface{})
		if item["id"] == id {
			found = true
			// Verify check images and risk scores are present
			if item["front_image_path"] == "" || item["back_image_path"] == "" {
				t.Error("expected front_image_path and back_image_path in queue item")
			}
			if _, ok := item["iq_score"]; !ok {
				t.Error("expected iq_score in queue item")
			}
			if _, ok := item["micr_confidence"]; !ok {
				t.Error("expected micr_confidence in queue item")
			}
		}
	}
	if !found {
		t.Errorf("expected transfer %s in queue", id)
	}
}

func TestOperator_Approve(t *testing.T) {
	mux := setupOperatorTestMux(t)
	id := createFlaggedDeposit(t, mux)

	body := map[string]interface{}{
		"transfer_id": id,
		"operator_id": "op-001",
		"note":        "looks good",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/operator/approve", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var result map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &result)
	if result["state"] != "FundsPosted" {
		t.Errorf("expected FundsPosted, got %v", result["state"])
	}
}

func TestOperator_Reject(t *testing.T) {
	mux := setupOperatorTestMux(t)
	id := createFlaggedDeposit(t, mux)

	body := map[string]interface{}{
		"transfer_id": id,
		"operator_id": "op-001",
		"note":        "suspicious",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/operator/reject", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var result map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &result)
	if result["state"] != "Rejected" {
		t.Errorf("expected Rejected, got %v", result["state"])
	}
}

func TestOperator_ApproveOverLimit(t *testing.T) {
	mux := setupOperatorTestMux(t)
	// Submit flagged deposit with amount over $5000 limit (flagged path skips business rules)
	body := map[string]interface{}{
		"account_id":  "ACC-MICR-FAIL",
		"amount":      6000.00,
		"front_image": "front",
		"back_image":  "back",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/deposits", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202 (flagged), got %d: %s", rr.Code, rr.Body.String())
	}
	var depositResult map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &depositResult)
	tr := depositResult["transfer"].(map[string]interface{})
	id := tr["id"].(string)

	// Operator approve should fail with 422 (over limit)
	approveBody := map[string]interface{}{
		"transfer_id": id,
		"operator_id": "op-001",
		"note":        "attempted approval",
	}
	b, _ = json.Marshal(approveBody)
	req = httptest.NewRequest(http.MethodPost, "/operator/approve", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 (over limit), got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestOperator_ApproveNotFound(t *testing.T) {
	mux := setupOperatorTestMux(t)

	body := map[string]interface{}{
		"transfer_id": "nonexistent",
		"operator_id": "op-001",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/operator/approve", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}
