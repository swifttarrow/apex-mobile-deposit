package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/checkstream/checkstream/internal/db"
	"github.com/checkstream/checkstream/internal/depositjob"
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
	jobRepo := depositjob.NewRepository(database)

	handler := NewDepositHandler(transferRepo, vendorStub, ledgerSvc, fundingSvc, fundingCfg, operatorRepo, jobRepo, database)

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
	handler, mux := setupTestDepositHandler(t)

	rr := postDeposit(t, mux, map[string]interface{}{
		"account_id":  "ACC-001",
		"amount":      150.00,
		"front_image": "base64frontimage",
		"back_image":  "base64backimage",
	}, nil)

	if rr.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d: %s", rr.Code, rr.Body.String())
	}

	var result map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &result)
	tr := result["transfer"].(map[string]interface{})
	if tr["state"] != "Requested" {
		t.Errorf("expected Requested state on accept, got %v", tr["state"])
	}

	if !handler.ProcessOneJob() {
		t.Fatal("expected one job to process")
	}

	id := tr["id"].(string)
	getReq := httptest.NewRequest(http.MethodGet, "/deposits/"+id, nil)
	getRR := httptest.NewRecorder()
	mux.ServeHTTP(getRR, getReq)
	if getRR.Code != http.StatusOK {
		t.Fatalf("get deposit: %d %s", getRR.Code, getRR.Body.String())
	}
	var t2 map[string]interface{}
	json.Unmarshal(getRR.Body.Bytes(), &t2)
	if t2["state"] != "FundsPosted" {
		t.Errorf("expected FundsPosted after process, got %v", t2["state"])
	}
}

func TestDeposit_IQABlur(t *testing.T) {
	handler, mux := setupTestDepositHandler(t)

	rr := postDeposit(t, mux, map[string]interface{}{
		"account_id":  "ACC-IQA-BLUR",
		"amount":      150.00,
		"front_image": "blurryimage",
		"back_image":  "blurryback",
	}, nil)

	if rr.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d: %s", rr.Code, rr.Body.String())
	}

	if !handler.ProcessOneJob() {
		t.Fatal("expected one job to process")
	}

	var result map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &result)
	tr := result["transfer"].(map[string]interface{})
	id := tr["id"].(string)
	getReq := httptest.NewRequest(http.MethodGet, "/deposits/"+id, nil)
	getRR := httptest.NewRecorder()
	mux.ServeHTTP(getRR, getReq)
	var t2 map[string]interface{}
	json.Unmarshal(getRR.Body.Bytes(), &t2)
	if t2["state"] != "Rejected" {
		t.Errorf("expected Rejected state, got %v", t2["state"])
	}
}

func TestDeposit_IQAGlare(t *testing.T) {
	handler, mux := setupTestDepositHandler(t)

	rr := postDeposit(t, mux, map[string]interface{}{
		"account_id":  "ACC-IQA-GLARE",
		"amount":      150.00,
		"front_image": "glareimage",
		"back_image":  "glareback",
	}, nil)

	if rr.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d: %s", rr.Code, rr.Body.String())
	}

	if !handler.ProcessOneJob() {
		t.Fatal("expected one job to process")
	}

	var result map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &result)
	tr := result["transfer"].(map[string]interface{})
	id := tr["id"].(string)
	getReq := httptest.NewRequest(http.MethodGet, "/deposits/"+id, nil)
	getRR := httptest.NewRecorder()
	mux.ServeHTTP(getRR, getReq)
	var t2 map[string]interface{}
	json.Unmarshal(getRR.Body.Bytes(), &t2)
	if t2["state"] != "Rejected" {
		t.Errorf("expected Rejected state, got %v", t2["state"])
	}
}

func TestDeposit_OverLimit(t *testing.T) {
	handler, mux := setupTestDepositHandler(t)

	rr := postDeposit(t, mux, map[string]interface{}{
		"account_id":  "ACC-OVER-LIMIT",
		"amount":      6000.00,
		"front_image": "front",
		"back_image":  "back",
	}, nil)

	if rr.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d: %s", rr.Code, rr.Body.String())
	}

	if !handler.ProcessOneJob() {
		t.Fatal("expected one job to process")
	}

	var result map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &result)
	tr := result["transfer"].(map[string]interface{})
	id := tr["id"].(string)
	getReq := httptest.NewRequest(http.MethodGet, "/deposits/"+id, nil)
	getRR := httptest.NewRecorder()
	mux.ServeHTTP(getRR, getReq)
	var t2 map[string]interface{}
	json.Unmarshal(getRR.Body.Bytes(), &t2)
	if t2["state"] != "Rejected" {
		t.Errorf("expected Rejected state, got %v", t2["state"])
	}
}

func TestDeposit_Flagged_MICRFail(t *testing.T) {
	handler, mux := setupTestDepositHandler(t)

	rr := postDeposit(t, mux, map[string]interface{}{
		"account_id":  "ACC-MICR-FAIL",
		"amount":      150.00,
		"front_image": "front",
		"back_image":  "back",
	}, nil)

	if rr.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d: %s", rr.Code, rr.Body.String())
	}

	if !handler.ProcessOneJob() {
		t.Fatal("expected one job to process")
	}

	var result map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &result)
	tr := result["transfer"].(map[string]interface{})
	id := tr["id"].(string)
	getReq := httptest.NewRequest(http.MethodGet, "/deposits/"+id, nil)
	getRR := httptest.NewRecorder()
	mux.ServeHTTP(getRR, getReq)
	var t2 map[string]interface{}
	json.Unmarshal(getRR.Body.Bytes(), &t2)
	if t2["state"] != "Analyzing" {
		t.Errorf("expected Analyzing state after process, got %v", t2["state"])
	}
}

func TestDeposit_GetStatus(t *testing.T) {
	handler, mux := setupTestDepositHandler(t)

	rr := postDeposit(t, mux, map[string]interface{}{
		"account_id":  "ACC-001",
		"amount":      100.00,
		"front_image": "front",
		"back_image":  "back",
	}, nil)
	if rr.Code != http.StatusAccepted {
		t.Fatalf("deposit accept: %d %s", rr.Code, rr.Body.String())
	}

	var result map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &result)
	tr := result["transfer"].(map[string]interface{})
	id := tr["id"].(string)

	if !handler.ProcessOneJob() {
		t.Fatal("expected one job to process")
	}

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

	if rr1.Code != http.StatusAccepted {
		t.Fatalf("first request failed: %d %s", rr1.Code, rr1.Body.String())
	}
	if rr2.Code != http.StatusAccepted {
		t.Errorf("idempotent replay failed: %d %s", rr2.Code, rr2.Body.String())
	}
	if rr2.Header().Get("X-Idempotency-Replayed") != "true" {
		t.Error("expected X-Idempotency-Replayed header on second request")
	}
	var r1, r2 map[string]interface{}
	json.Unmarshal(rr1.Body.Bytes(), &r1)
	json.Unmarshal(rr2.Body.Bytes(), &r2)
	id1 := r1["transfer"].(map[string]interface{})["id"].(string)
	id2 := r2["transfer"].(map[string]interface{})["id"].(string)
	if id1 != id2 {
		t.Errorf("idempotent replay should return same transfer id: %s vs %s", id1, id2)
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
	jobRepo := depositjob.NewRepository(database)
	handler := NewDepositHandler(transferRepo, vendorStub, ledgerSvc, fundingSvc, fundingCfg, operatorRepo, jobRepo, database)

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
		if rr.Code != http.StatusAccepted {
			b.Fatalf("expected 202, got %d", rr.Code)
		}
	}
}
