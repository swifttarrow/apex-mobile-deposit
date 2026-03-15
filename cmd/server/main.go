package main

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/checkstream/checkstream/internal/api"
	"github.com/checkstream/checkstream/internal/auth"
	appclock "github.com/checkstream/checkstream/internal/clock"
	"github.com/checkstream/checkstream/internal/db"
	"github.com/checkstream/checkstream/internal/funding"
	"github.com/checkstream/checkstream/internal/ledger"
	"github.com/checkstream/checkstream/internal/operator"
	returnpkg "github.com/checkstream/checkstream/internal/return_"
	"github.com/checkstream/checkstream/internal/settlement"
	"github.com/checkstream/checkstream/internal/transfer"
	"github.com/checkstream/checkstream/internal/vendor"
)

//go:embed all:web/scenarios
var scenarioFS embed.FS

//go:embed all:web/operator
var operatorFS embed.FS

//go:embed all:web/mobile
var mobileFS embed.FS

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "checkstream.db"
	}

	database, err := db.Open(dsn)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer database.Close()

	// Initialize repositories and services
	transferRepo := transfer.NewRepository(database)
	vendorStub := vendor.NewStub("config/scenarios.json")
	ledgerSvc := ledger.NewService(database)
	fundingCfg := funding.NewConfig()
	fundingSvc := funding.NewService(fundingCfg, transferRepo)
	operatorRepo := operator.NewRepository(database)
	if err := operatorRepo.SeedTestOperators(); err != nil {
		log.Printf("warning: seed test operators: %v", err)
	}
	settlementEngine := settlement.NewEngine(database, transferRepo, ledgerSvc)
	travelClock := appclock.NewTravelClock()
	transferRepo.SetNowFunc(travelClock.Now)
	operatorRepo.SetNowFunc(travelClock.Now)
	settlementEngine.SetNowFunc(travelClock.Now)
	returnSvc := returnpkg.NewService(database, transferRepo, ledgerSvc)

	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"service": "checkdepot",
			"version": "1.0.0",
		})
	})

	// Register vendor stub route
	mux.HandleFunc("POST /vendor/validate", vendorStub.HandleValidate)

	// Register deposit routes
	depositHandler := api.NewDepositHandler(transferRepo, vendorStub, ledgerSvc, fundingSvc, fundingCfg, operatorRepo, database)
	mux.HandleFunc("POST /deposits", api.WithIdempotency(database, depositHandler.Create))
	mux.HandleFunc("GET /deposits", depositHandler.List)
	mux.HandleFunc("GET /deposits/{id}", depositHandler.Get)

	// Operator auth routes (no auth required)
	authHandler := api.NewAuthHandler(operatorRepo)
	mux.HandleFunc("POST /operator/login", authHandler.Login)
	mux.HandleFunc("POST /operator/guest", authHandler.Guest)
	mux.HandleFunc("POST /operator/logout", authHandler.Logout)
	mux.HandleFunc("GET /operator/me", authHandler.Me)

	// Operator routes (require login)
	operatorHandler := api.NewOperatorHandler(operatorRepo, transferRepo, ledgerSvc, fundingCfg)
	mux.HandleFunc("GET /operator/queue", auth.RequireOperator(operatorHandler.Queue))
	mux.HandleFunc("GET /operator/audit", auth.RequireOperator(operatorHandler.Audit))
	mux.HandleFunc("POST /operator/approve", auth.RequireOperator(operatorHandler.Approve))
	mux.HandleFunc("POST /operator/reject", auth.RequireOperator(operatorHandler.Reject))
	mux.HandleFunc("GET /operator/actions/{transfer_id}", auth.RequireOperator(operatorHandler.Actions))

	// Settlement routes (require operator login)
	settlementHandler := api.NewSettlementHandler(settlementEngine)
	settlementHandler.SetNowFunc(travelClock.Now)
	mux.HandleFunc("POST /settlement/trigger", auth.RequireOperator(settlementHandler.Trigger))

	// Test-only time travel controls in operator portal.
	clockHandler := api.NewClockHandler(travelClock)
	mux.HandleFunc("GET /operator/clock", auth.RequireOperator(clockHandler.Get))
	mux.HandleFunc("POST /operator/clock", auth.RequireOperator(clockHandler.Update))

	// Returns routes
	returnsHandler := api.NewReturnsHandler(returnSvc)
	mux.HandleFunc("POST /returns", returnsHandler.ProcessReturn)

	// Ledger routes
	ledgerHandler := api.NewLedgerHandler(database)
	mux.HandleFunc("GET /ledger", ledgerHandler.List)
	mux.HandleFunc("GET /accounts/{id}/balance", ledgerHandler.Balance)

	// Sandbox UI (embedded) — scenario showcase at /sandbox
	sandboxRoot, _ := fs.Sub(scenarioFS, "web/scenarios")
	sandboxHandler := func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/sandbox" || path == "/sandbox/" {
			path = "/sandbox/index.html"
		}
		name := strings.TrimPrefix(path, "/sandbox/")
		if name == "" {
			name = "index.html"
		}
		b, err := fs.ReadFile(sandboxRoot, name)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		switch {
		case strings.HasSuffix(name, ".js"):
			w.Header().Set("Content-Type", "application/javascript")
		case strings.HasSuffix(name, ".css"):
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		default:
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		}
		w.Write(b)
	}
	mux.HandleFunc("GET /sandbox", sandboxHandler)
	mux.HandleFunc("GET /sandbox/", sandboxHandler)

	// Operator UI (embedded) — root page and /operator
	operatorRoot, _ := fs.Sub(operatorFS, "web/operator")
	operatorPageHandler := func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		var name string
		switch {
		case path == "/" || path == "" || path == "/operator" || path == "/operator/":
			name = "index.html"
		case path == "/operator/review-queue",
			path == "/operator/settlement",
			path == "/operator/audit-log",
			path == "/operator/accounts",
			path == "/operator/settings":
			name = "index.html"
		case path == "/operator/login" || path == "/login":
			name = "login.html"
		default:
			name = strings.TrimPrefix(path, "/operator/")
			if name == path {
				name = strings.TrimPrefix(path, "/")
			}
			if name == "" {
				name = "index.html"
			}
		}
		b, err := fs.ReadFile(operatorRoot, name)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		switch {
		case strings.HasSuffix(name, ".js"):
			w.Header().Set("Content-Type", "application/javascript")
		case strings.HasSuffix(name, ".css"):
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		default:
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		}
		w.Write(b)
	}
	mux.HandleFunc("GET /", operatorPageHandler)
	mux.HandleFunc("GET /operator", operatorPageHandler)
	mux.HandleFunc("GET /operator/", operatorPageHandler)
	mux.HandleFunc("GET /operator/login", operatorPageHandler)
	mux.HandleFunc("GET /login", operatorPageHandler)

	// Mobile UI (embedded)
	mobileRoot, _ := fs.Sub(mobileFS, "web/mobile")
	mobileHandler := func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/mobile" || path == "/mobile/" {
			path = "/mobile/index.html"
		}
		name := strings.TrimPrefix(path, "/mobile/")
		if name == "" {
			name = "index.html"
		}
		b, err := fs.ReadFile(mobileRoot, name)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		switch {
		case strings.HasSuffix(name, ".js"):
			w.Header().Set("Content-Type", "application/javascript")
		case strings.HasSuffix(name, ".css"):
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		case strings.HasSuffix(name, ".png"):
			w.Header().Set("Content-Type", "image/png")
		case strings.HasSuffix(name, ".jpg"), strings.HasSuffix(name, ".jpeg"):
			w.Header().Set("Content-Type", "image/jpeg")
		default:
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		}
		w.Write(b)
	}
	mux.HandleFunc("GET /mobile", mobileHandler)
	mux.HandleFunc("GET /mobile/", mobileHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port
	log.Printf("Checkdepot server listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
