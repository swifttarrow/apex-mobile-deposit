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
	"github.com/checkstream/checkstream/internal/db"
	"github.com/checkstream/checkstream/internal/funding"
	"github.com/checkstream/checkstream/internal/ledger"
	"github.com/checkstream/checkstream/internal/operator"
	"github.com/checkstream/checkstream/internal/settlement"
	returnpkg "github.com/checkstream/checkstream/internal/return_"
	"github.com/checkstream/checkstream/internal/transfer"
	"github.com/checkstream/checkstream/internal/vendor"
)

//go:embed all:web/scenarios
var scenarioFS embed.FS

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
	settlementEngine := settlement.NewEngine(database, transferRepo, ledgerSvc)
	returnSvc := returnpkg.NewService(database, transferRepo, ledgerSvc)

	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"service": "checkstream",
			"version": "1.0.0",
		})
	})

	// Register vendor stub route
	mux.HandleFunc("POST /vendor/validate", vendorStub.HandleValidate)

	// Register deposit routes
	depositHandler := api.NewDepositHandler(transferRepo, vendorStub, ledgerSvc, fundingSvc, fundingCfg, database)
	mux.HandleFunc("POST /deposits", api.WithIdempotency(database, depositHandler.Create))
	mux.HandleFunc("GET /deposits/{id}", depositHandler.Get)

	// Operator routes
	operatorHandler := api.NewOperatorHandler(operatorRepo, transferRepo, ledgerSvc, fundingCfg)
	mux.HandleFunc("GET /operator/queue", operatorHandler.Queue)
	mux.HandleFunc("POST /operator/approve", operatorHandler.Approve)
	mux.HandleFunc("POST /operator/reject", operatorHandler.Reject)

	// Settlement routes
	settlementHandler := api.NewSettlementHandler(settlementEngine)
	mux.HandleFunc("POST /settlement/trigger", settlementHandler.Trigger)

	// Returns routes
	returnsHandler := api.NewReturnsHandler(returnSvc)
	mux.HandleFunc("POST /returns", returnsHandler.ProcessReturn)

	// Ledger routes
	ledgerHandler := api.NewLedgerHandler(database)
	mux.HandleFunc("GET /ledger", ledgerHandler.List)
	mux.HandleFunc("GET /accounts/{id}/balance", ledgerHandler.Balance)

	// Scenario showcase UI (embedded)
	scenarioRoot, _ := fs.Sub(scenarioFS, "web/scenarios")
	scenarioHandler := func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/scenarios" || path == "/scenarios/" {
			path = "/scenarios/index.html"
		}
		name := strings.TrimPrefix(path, "/scenarios/")
		if name == "" {
			name = "index.html"
		}
		b, err := fs.ReadFile(scenarioRoot, name)
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
	mux.HandleFunc("GET /scenarios", scenarioHandler)
	mux.HandleFunc("GET /scenarios/", scenarioHandler)

	log.Println("Checkstream server listening on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
