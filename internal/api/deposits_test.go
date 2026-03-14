package api

import (
	"bytes"
	"encoding/json"
	"fmt"
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

func setupTestDepositHandler(t *testing.T) (*DepositHandler, *http.ServeMux) {
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

	handler := NewDepositHandler(transferRepo, vendorStub, ledgerSvc, fundingSvc, fundingCfg, operatorRepo, database)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /deposits", WithIdempotency(database, handler.Create))
	mux.HandleFunc("GET /deposits/{id}", handler.Get)

	return handler, mux
}

func postDeposit(t *testing.T, mux *http.ServeMux, body map[string]interface{}, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/deposits", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	return rr
}

func TestDeposit_HappyPath(t *testing.T) {
	_, mux := setupTestDepositHandler(t)

	rr := postDeposit(t, mux, map[string]interface{}{
		"account_id":  "ACC-001",
		"amount":      150.00,
		"front_image": "base64frontimage",
		"back_image":  "base64backimage",
	}, nil)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var result map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &result)
	if result["state"] != "FundsPosted" {
		t.Errorf("expected FundsPosted state, got %v", result["state"])
	}
}

func TestDeposit_IQABlur(t *testing.T) {
	_, mux := setupTestDepositHandler(t)

	rr := postDeposit(t, mux, map[string]interface{}{
		"account_id":  "ACC-IQA-BLUR",
		"amount":      150.00,
		"front_image": "blurryimage",
		"back_image":  "blurryback",
	}, nil)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d: %s", rr.Code, rr.Body.String())
	}

	var result map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &result)
	if result["reason"] != "blur" {
		t.Errorf("expected reason blur, got %v", result["reason"])
	}
}

func TestDeposit_IQAGlare(t *testing.T) {
	_, mux := setupTestDepositHandler(t)

	rr := postDeposit(t, mux, map[string]interface{}{
		"account_id":  "ACC-IQA-GLARE",
		"amount":      150.00,
		"front_image": "glareimage",
		"back_image":  "glareback",
	}, nil)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestDeposit_OverLimit(t *testing.T) {
	_, mux := setupTestDepositHandler(t)

	rr := postDeposit(t, mux, map[string]interface{}{
		"account_id":  "ACC-OVER-LIMIT",
		"amount":      6000.00,
		"front_image": "front",
		"back_image":  "back",
	}, nil)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestDeposit_Flagged_MICRFail(t *testing.T) {
	_, mux := setupTestDepositHandler(t)

	rr := postDeposit(t, mux, map[string]interface{}{
		"account_id":  "ACC-MICR-FAIL",
		"amount":      150.00,
		"front_image": "front",
		"back_image":  "back",
	}, nil)

	if rr.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d: %s", rr.Code, rr.Body.String())
	}

	var result map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &result)
	tr := result["transfer"].(map[string]interface{})
	if tr["state"] != "Analyzing" {
		t.Errorf("expected Analyzing state, got %v", tr["state"])
	}
}

func TestDeposit_GetStatus(t *testing.T) {
	_, mux := setupTestDepositHandler(t)

	rr := postDeposit(t, mux, map[string]interface{}{
		"account_id":  "ACC-001",
		"amount":      100.00,
		"front_image": "front",
		"back_image":  "back",
	}, nil)
	if rr.Code != http.StatusCreated {
		t.Fatalf("deposit failed: %s", rr.Body.String())
	}

	var created map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &created)
	id := created["id"].(string)

	getReq := httptest.NewRequest(http.MethodGet, "/deposits/"+id, nil)
	getRR := httptest.NewRecorder()
	mux.ServeHTTP(getRR, getReq)

	if getRR.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", getRR.Code)
	}

	var t2 map[string]interface{}
	json.Unmarshal(getRR.Body.Bytes(), &t2)
	if t2["id"] != id {
		t.Errorf("expected id %s, got %v", id, t2["id"])
	}
}

func TestDeposit_NotFound(t *testing.T) {
	_, mux := setupTestDepositHandler(t)

	getReq := httptest.NewRequest(http.MethodGet, "/deposits/nonexistent-id", nil)
	getRR := httptest.NewRecorder()
	mux.ServeHTTP(getRR, getReq)

	if getRR.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", getRR.Code)
	}
}

func TestDeposit_Idempotency(t *testing.T) {
	_, mux := setupTestDepositHandler(t)

	body := map[string]interface{}{
		"account_id":  "ACC-001",
		"amount":      100.00,
		"front_image": "front",
		"back_image":  "back",
	}
	headers := map[string]string{"X-Idempotency-Key": "test-idem-key-123"}

	rr1 := postDeposit(t, mux, body, headers)
	rr2 := postDeposit(t, mux, body, headers)

	if rr1.Code != http.StatusCreated {
		t.Fatalf("first request failed: %d %s", rr1.Code, rr1.Body.String())
	}
	if rr2.Code != http.StatusCreated {
		t.Errorf("idempotent replay failed: %d %s", rr2.Code, rr2.Body.String())
	}
	if rr2.Header().Get("X-Idempotency-Replayed") != "true" {
		t.Error("expected X-Idempotency-Replayed header on second request")
	}
}

func TestDeposit_MissingFields(t *testing.T) {
	_, mux := setupTestDepositHandler(t)

	rr := postDeposit(t, mux, map[string]interface{}{
		"account_id": "ACC-001",
		"amount":     100.00,
		// missing front_image and back_image
	}, nil)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func BenchmarkDeposit_CleanPass(b *testing.B) {
	database, err := db.Open(":memory:")
	if err != nil {
		b.Fatalf("open db: %v", err)
	}
	defer database.Close()

	transferRepo := transfer.NewRepository(database)
	vendorStub := vendor.NewStub("../../config/scenarios.json")
	ledgerSvc := ledger.NewService(database)
	fundingCfg := funding.NewConfig()
	fundingSvc := funding.NewService(fundingCfg, transferRepo)
	operatorRepo := operator.NewRepository(database)
	handler := NewDepositHandler(transferRepo, vendorStub, ledgerSvc, fundingSvc, fundingCfg, operatorRepo, database)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /deposits", WithIdempotency(database, handler.Create))

	body := map[string]interface{}{
		"account_id":  "ACC-001",
		"amount":      150.00,
		"front_image": "base64frontimage",
		"back_image":  "base64backimage",
	}
	bodyBytes, _ := json.Marshal(body)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/deposits", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Idempotency-Key", fmt.Sprintf("bench-%d", i))
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		if rr.Code != http.StatusCreated {
			b.Fatalf("expected 201, got %d", rr.Code)
		}
	}
}
