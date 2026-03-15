package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/checkstream/checkstream/internal/auth"
	appclock "github.com/checkstream/checkstream/internal/clock"
	"github.com/checkstream/checkstream/internal/db"
	"github.com/checkstream/checkstream/internal/ledger"
	"github.com/checkstream/checkstream/internal/operator"
	"github.com/checkstream/checkstream/internal/settlement"
	"github.com/checkstream/checkstream/internal/transfer"
)

func TestClockHandler_TestOperatorCanSetFreezeResume(t *testing.T) {
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	operatorRepo := operator.NewRepository(database)
	if err := operatorRepo.SeedTestOperators(); err != nil {
		t.Fatalf("seed operators: %v", err)
	}

	c := appclock.NewTravelClock()
	authHandler := NewAuthHandler(operatorRepo)
	clockHandler := NewClockHandler(c)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /operator/login", authHandler.Login)
	mux.HandleFunc("GET /operator/clock", auth.RequireOperator(clockHandler.Get))
	mux.HandleFunc("POST /operator/clock", auth.RequireOperator(clockHandler.Update))

	loginRes := doJSONRequest(t, mux, http.MethodPost, "/operator/login", map[string]interface{}{
		"username": "joe",
		"password": "password",
	}, nil)
	if loginRes.Code != http.StatusOK {
		t.Fatalf("login failed: %d %s", loginRes.Code, loginRes.Body.String())
	}
	cookies := loginRes.Result().Cookies()

	setRes := doJSONRequest(t, mux, http.MethodPost, "/operator/clock", map[string]interface{}{
		"action": "set",
		"time":   "2024-01-16T01:00:00Z",
	}, cookies)
	if setRes.Code != http.StatusOK {
		t.Fatalf("set failed: %d %s", setRes.Code, setRes.Body.String())
	}

	freezeRes := doJSONRequest(t, mux, http.MethodPost, "/operator/clock", map[string]interface{}{
		"action": "freeze",
	}, cookies)
	if freezeRes.Code != http.StatusOK {
		t.Fatalf("freeze failed: %d %s", freezeRes.Code, freezeRes.Body.String())
	}
	var freeze map[string]interface{}
	_ = json.Unmarshal(freezeRes.Body.Bytes(), &freeze)
	if freeze["mode"] != "frozen" {
		t.Fatalf("expected mode=frozen, got %v", freeze["mode"])
	}

	resumeRes := doJSONRequest(t, mux, http.MethodPost, "/operator/clock", map[string]interface{}{
		"action": "resume",
	}, cookies)
	if resumeRes.Code != http.StatusOK {
		t.Fatalf("resume failed: %d %s", resumeRes.Code, resumeRes.Body.String())
	}
	var resumed map[string]interface{}
	_ = json.Unmarshal(resumeRes.Body.Bytes(), &resumed)
	if resumed["mode"] != "running" {
		t.Fatalf("expected mode=running, got %v", resumed["mode"])
	}
}

func TestClockHandler_GuestForbidden(t *testing.T) {
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	operatorRepo := operator.NewRepository(database)
	authHandler := NewAuthHandler(operatorRepo)
	clockHandler := NewClockHandler(appclock.NewTravelClock())

	mux := http.NewServeMux()
	mux.HandleFunc("POST /operator/guest", authHandler.Guest)
	mux.HandleFunc("GET /operator/clock", auth.RequireOperator(clockHandler.Get))

	guestRes := doJSONRequest(t, mux, http.MethodPost, "/operator/guest", map[string]interface{}{}, nil)
	if guestRes.Code != http.StatusOK {
		t.Fatalf("guest login failed: %d %s", guestRes.Code, guestRes.Body.String())
	}
	cookies := guestRes.Result().Cookies()

	getRes := doJSONRequest(t, mux, http.MethodGet, "/operator/clock", nil, cookies)
	if getRes.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for guest clock access, got %d (%s)", getRes.Code, getRes.Body.String())
	}
}

func TestSettlementTrigger_UsesTravelClock(t *testing.T) {
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	transferRepo := transfer.NewRepository(database)
	ledgerSvc := ledger.NewService(database)
	settlementEngine := settlement.NewEngine(database, transferRepo, ledgerSvc)
	c := appclock.NewTravelClock()
	settlementEngine.SetNowFunc(c.Now)

	settlementHandler := NewSettlementHandler(settlementEngine)
	settlementHandler.SetNowFunc(c.Now)

	operatorRepo := operator.NewRepository(database)
	if err := operatorRepo.SeedTestOperators(); err != nil {
		t.Fatalf("seed operators: %v", err)
	}
	authHandler := NewAuthHandler(operatorRepo)
	clockHandler := NewClockHandler(c)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /operator/login", authHandler.Login)
	mux.HandleFunc("POST /operator/clock", auth.RequireOperator(clockHandler.Update))
	mux.HandleFunc("POST /settlement/trigger", auth.RequireOperator(settlementHandler.Trigger))

	loginRes := doJSONRequest(t, mux, http.MethodPost, "/operator/login", map[string]interface{}{
		"username": "joe",
		"password": "password",
	}, nil)
	cookies := loginRes.Result().Cookies()

	setRes := doJSONRequest(t, mux, http.MethodPost, "/operator/clock", map[string]interface{}{
		"action": "set",
		"time":   "2024-01-16T01:00:00Z",
	}, cookies)
	if setRes.Code != http.StatusOK {
		t.Fatalf("set clock failed: %d %s", setRes.Code, setRes.Body.String())
	}

	triggerRes := doJSONRequest(t, mux, http.MethodPost, "/settlement/trigger", map[string]interface{}{}, cookies)
	if triggerRes.Code != http.StatusOK {
		t.Fatalf("settlement trigger failed: %d %s", triggerRes.Code, triggerRes.Body.String())
	}
	var payload map[string]interface{}
	_ = json.Unmarshal(triggerRes.Body.Bytes(), &payload)
	if payload["after_eod_cutoff"] != true {
		t.Fatalf("expected after_eod_cutoff=true, got %v", payload["after_eod_cutoff"])
	}
}

func doJSONRequest(t *testing.T, mux *http.ServeMux, method, path string, body interface{}, cookies []*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	var b []byte
	if body != nil {
		b, _ = json.Marshal(body)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	return rr
}
