package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/checkstream/checkstream/internal/auth"
	"github.com/checkstream/checkstream/internal/db"
	"github.com/checkstream/checkstream/internal/funding"
	"github.com/checkstream/checkstream/internal/ledger"
	"github.com/checkstream/checkstream/internal/operator"
	returnpkg "github.com/checkstream/checkstream/internal/return_"
	"github.com/checkstream/checkstream/internal/settlement"
	"github.com/checkstream/checkstream/internal/transfer"
	"github.com/checkstream/checkstream/internal/vendor"
)

func setupFullMux(t *testing.T) *http.ServeMux {
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
	fundingSvc := funding.NewServiceWithContributionLookup(fundingCfg, transferRepo, transferRepo)
	operatorRepo := operator.NewRepository(database)
	if err := operatorRepo.SeedTestOperators(); err != nil {
		t.Fatalf("seed operators: %v", err)
	}
	settlementEngine := settlement.NewEngine(database, transferRepo, ledgerSvc)
	returnSvc := returnpkg.NewService(database, transferRepo, ledgerSvc)

	depositHandler := NewDepositHandler(transferRepo, vendorStub, ledgerSvc, fundingSvc, fundingCfg, operatorRepo, database)
	authHandler := NewAuthHandler(operatorRepo)
	operatorHandler := NewOperatorHandler(operatorRepo, transferRepo, ledgerSvc, fundingCfg, fundingSvc)
	settlementHandler := NewSettlementHandler(settlementEngine)
	returnsHandler := NewReturnsHandler(returnSvc)
	ledgerHandler := NewLedgerHandler(database)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /deposits", WithIdempotency(database, depositHandler.Create))
	mux.HandleFunc("GET /deposits/{id}", depositHandler.Get)
	mux.HandleFunc("POST /operator/login", authHandler.Login)
	mux.HandleFunc("GET /operator/queue", auth.RequireOperator(operatorHandler.Queue))
	mux.HandleFunc("POST /operator/approve", auth.RequireOperator(operatorHandler.Approve))
	mux.HandleFunc("POST /operator/reject", auth.RequireOperator(operatorHandler.Reject))
	mux.HandleFunc("POST /settlement/trigger", auth.RequireOperator(settlementHandler.Trigger))
	mux.HandleFunc("POST /returns", returnsHandler.ProcessReturn)
	mux.HandleFunc("GET /ledger", ledgerHandler.List)
	mux.HandleFunc("GET /accounts/{id}/balance", ledgerHandler.Balance)
	return mux
}

func doPost(t *testing.T, mux *http.ServeMux, path string, body interface{}, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	return rr
}

func loginAndGetCookies(t *testing.T, mux *http.ServeMux) []*http.Cookie {
	t.Helper()
	rr := doPost(t, mux, "/operator/login", map[string]interface{}{"username": "joe", "password": "password"}, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("login failed: %d %s", rr.Code, rr.Body.String())
	}
	return rr.Result().Cookies()
}

func doPostWithCookies(t *testing.T, mux *http.ServeMux, path string, body interface{}, cookies []*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	return rr
}

func doGet(t *testing.T, mux *http.ServeMux, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	return rr
}

// Scenario 1: Clean pass (ACC-001)
func TestScenario_CleanPass(t *testing.T) {
	mux := setupFullMux(t)
	rr := doPost(t, mux, "/deposits", map[string]interface{}{
		"account_id":  "ACC-001",
		"amount":      150.00,
		"front_image": "front",
		"back_image":  "back",
	}, nil)
	if rr.Code != http.StatusCreated {
		t.Fatalf("scenario 1 (clean pass): expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var result map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &result)
	if result["state"] != "FundsPosted" {
		t.Errorf("expected FundsPosted, got %v", result["state"])
	}
}

// Scenario 2: IQA blur fail (ACC-IQA-BLUR)
func TestScenario_IQABlur(t *testing.T) {
	mux := setupFullMux(t)
	rr := doPost(t, mux, "/deposits", map[string]interface{}{
		"account_id":  "ACC-IQA-BLUR",
		"amount":      100.00,
		"front_image": "front",
		"back_image":  "back",
	}, nil)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("scenario 2 (IQA blur): expected 422, got %d: %s", rr.Code, rr.Body.String())
	}
}

// Scenario 3: IQA glare fail (ACC-IQA-GLARE)
func TestScenario_IQAGlare(t *testing.T) {
	mux := setupFullMux(t)
	rr := doPost(t, mux, "/deposits", map[string]interface{}{
		"account_id":  "ACC-IQA-GLARE",
		"amount":      100.00,
		"front_image": "front",
		"back_image":  "back",
	}, nil)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("scenario 3 (IQA glare): expected 422, got %d: %s", rr.Code, rr.Body.String())
	}
}

// Scenario 4: MICR fail → flagged → operator approves
func TestScenario_MICRFail_OperatorApprove(t *testing.T) {
	mux := setupFullMux(t)
	rr := doPost(t, mux, "/deposits", map[string]interface{}{
		"account_id":  "ACC-MICR-FAIL",
		"amount":      200.00,
		"front_image": "front",
		"back_image":  "back",
	}, nil)
	if rr.Code != http.StatusAccepted {
		t.Fatalf("scenario 4 (MICR fail): expected 202, got %d: %s", rr.Code, rr.Body.String())
	}
	var result map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &result)
	tr := result["transfer"].(map[string]interface{})
	id := tr["id"].(string)

	// Operator approves
	cookies := loginAndGetCookies(t, mux)
	rr2 := doPostWithCookies(t, mux, "/operator/approve", map[string]interface{}{
		"transfer_id": id,
		"note":        "manually verified",
	}, cookies)
	if rr2.Code != http.StatusOK {
		t.Fatalf("scenario 4: approve failed: %d: %s", rr2.Code, rr2.Body.String())
	}
	var approved map[string]interface{}
	json.Unmarshal(rr2.Body.Bytes(), &approved)
	if approved["state"] != "FundsPosted" {
		t.Errorf("expected FundsPosted, got %v", approved["state"])
	}
}

// Scenario 5: MICR fail → flagged → operator rejects
func TestScenario_MICRFail_OperatorReject(t *testing.T) {
	mux := setupFullMux(t)
	rr := doPost(t, mux, "/deposits", map[string]interface{}{
		"account_id":  "ACC-MICR-FAIL",
		"amount":      200.00,
		"front_image": "front",
		"back_image":  "back",
	}, nil)
	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rr.Code)
	}
	var result map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &result)
	tr := result["transfer"].(map[string]interface{})
	id := tr["id"].(string)

	cookies := loginAndGetCookies(t, mux)
	rr2 := doPostWithCookies(t, mux, "/operator/reject", map[string]interface{}{
		"transfer_id": id,
		"note":        "suspicious",
	}, cookies)
	if rr2.Code != http.StatusOK {
		t.Fatalf("reject failed: %d: %s", rr2.Code, rr2.Body.String())
	}
	var rejected map[string]interface{}
	json.Unmarshal(rr2.Body.Bytes(), &rejected)
	if rejected["state"] != "Rejected" {
		t.Errorf("expected Rejected, got %v", rejected["state"])
	}
}

// Scenario 6: Amount mismatch → flagged for review
func TestScenario_AmountMismatch(t *testing.T) {
	mux := setupFullMux(t)
	rr := doPost(t, mux, "/deposits", map[string]interface{}{
		"account_id":  "ACC-MISMATCH",
		"amount":      1500.00,
		"front_image": "front",
		"back_image":  "back",
	}, nil)
	if rr.Code != http.StatusAccepted {
		t.Fatalf("scenario 6 (amount mismatch): expected 202, got %d: %s", rr.Code, rr.Body.String())
	}
	var result map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &result)
	if result["reason"] != "amount_mismatch" {
		t.Errorf("expected reason amount_mismatch, got %v", result["reason"])
	}
}

// Scenario 7: Duplicate detected (ACC-DUP-001)
func TestScenario_Duplicate(t *testing.T) {
	mux := setupFullMux(t)
	rr := doPost(t, mux, "/deposits", map[string]interface{}{
		"account_id":  "ACC-DUP-001",
		"amount":      100.00,
		"front_image": "front",
		"back_image":  "back",
	}, nil)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("scenario 7 (duplicate): expected 422, got %d: %s", rr.Code, rr.Body.String())
	}
	var result map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &result)
	if result["reason"] != "duplicate" {
		t.Errorf("expected reason duplicate, got %v", result["reason"])
	}
}

// Scenario 8: Over deposit limit
func TestScenario_OverLimit(t *testing.T) {
	mux := setupFullMux(t)
	rr := doPost(t, mux, "/deposits", map[string]interface{}{
		"account_id":  "ACC-OVER-LIMIT",
		"amount":      5001.00,
		"front_image": "front",
		"back_image":  "back",
	}, nil)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("scenario 8 (over limit): expected 422, got %d: %s", rr.Code, rr.Body.String())
	}
}

// Scenario 9: Return/reversal flow
func TestScenario_Return(t *testing.T) {
	// Use temp dir for settlements
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	mux := setupFullMux(t)

	// Submit clean deposit
	rr := doPost(t, mux, "/deposits", map[string]interface{}{
		"account_id":  "ACC-001",
		"amount":      500.00,
		"front_image": "front",
		"back_image":  "back",
	}, nil)
	if rr.Code != http.StatusCreated {
		t.Fatalf("deposit failed: %d: %s", rr.Code, rr.Body.String())
	}
	var result map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &result)
	id := result["id"].(string)

	// Process return
	rr2 := doPost(t, mux, "/returns", map[string]interface{}{
		"transfer_id": id,
		"reason":      "insufficient funds",
	}, nil)
	if rr2.Code != http.StatusOK {
		t.Fatalf("return failed: %d: %s", rr2.Code, rr2.Body.String())
	}
	var returnResult map[string]interface{}
	json.Unmarshal(rr2.Body.Bytes(), &returnResult)
	tr := returnResult["transfer"].(map[string]interface{})
	if tr["state"] != "Returned" {
		t.Errorf("expected Returned, got %v", tr["state"])
	}
	if returnResult["reversal_fee"].(float64) != 30.00 {
		t.Errorf("expected fee 30.00, got %v", returnResult["reversal_fee"])
	}
}

// Scenario 10: Settlement trigger
func TestScenario_Settlement(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	mux := setupFullMux(t)

	// Submit deposit
	rr := doPost(t, mux, "/deposits", map[string]interface{}{
		"account_id":  "ACC-001",
		"amount":      300.00,
		"front_image": "front",
		"back_image":  "back",
	}, nil)
	if rr.Code != http.StatusCreated {
		t.Fatalf("deposit failed: %d: %s", rr.Code, rr.Body.String())
	}

	// Trigger settlement
	cookies := loginAndGetCookies(t, mux)
	rr2 := doPostWithCookies(t, mux, "/settlement/trigger", nil, cookies)
	if rr2.Code != http.StatusOK {
		t.Fatalf("settlement failed: %d: %s", rr2.Code, rr2.Body.String())
	}
	var settlementResult map[string]interface{}
	json.Unmarshal(rr2.Body.Bytes(), &settlementResult)
	if _, ok := settlementResult["total_count"].(float64); !ok {
		t.Errorf("expected numeric total_count, got %T", settlementResult["total_count"])
	}
}

// Scenario 11: Retirement account with contribution type
func TestScenario_RetirementAccount(t *testing.T) {
	mux := setupFullMux(t)
	rr := doPost(t, mux, "/deposits", map[string]interface{}{
		"account_id":  "ACC-RETIRE-001",
		"amount":      1000.00,
		"front_image": "front",
		"back_image":  "back",
	}, nil)
	if rr.Code != http.StatusCreated {
		t.Fatalf("scenario 11 (retirement): expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var result map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &result)
	if result["contribution_type"] != "individual" {
		t.Errorf("expected contribution_type=individual, got %v", result["contribution_type"])
	}
}
