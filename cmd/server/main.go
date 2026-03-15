package main

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/checkstream/checkstream/internal/api"
	"github.com/checkstream/checkstream/internal/auth"
	"github.com/checkstream/checkstream/internal/db"
	"github.com/checkstream/checkstream/internal/depositjob"
	"github.com/checkstream/checkstream/internal/funding"
	"github.com/checkstream/checkstream/internal/investor"
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

// wantsHTML returns true when the request Accept header prefers text/html (e.g. browser navigation).
func wantsHTML(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept"), "text/html")
}

// depoListOrPage serves deposit list API (JSON) or operator SPA (HTML) by content negotiation.
func depoListOrPage(apiHandler, pageHandler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if wantsHTML(r) {
			pageHandler(w, r)
			return
		}
		apiHandler(w, r)
	}
}

// depoGetOrPage serves single-deposit API (JSON) or operator SPA (HTML) by content negotiation.
func depoGetOrPage(apiHandler, pageHandler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if wantsHTML(r) {
			pageHandler(w, r)
			return
		}
		apiHandler(w, r)
	}
}

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
	fundingSvc := funding.NewServiceWithContributionLookup(fundingCfg, transferRepo, transferRepo)
	operatorRepo := operator.NewRepository(database)
	if err := operatorRepo.SeedTestOperators(); err != nil {
		log.Printf("warning: seed test operators: %v", err)
	}
	investorRepo := investor.NewInvestorRepo(database)
	if err := investorRepo.SeedTestInvestors(); err != nil {
		log.Printf("warning: seed test investors: %v", err)
	}
	jobRepo := depositjob.NewRepository(database)
	settlementEngine := settlement.NewEngine(database, transferRepo, ledgerSvc)
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

	// Operator UI filesystem and page handler (used by deposit routes for content negotiation)
	operatorRoot, _ := fs.Sub(operatorFS, "web/operator")
	operatorPageHandler := func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		var name string
		switch {
		case path == "/" || path == "":
			name = "index.html"
		case path == "/review-queue",
			path == "/settlement",
			path == "/deposits",
			path == "/ledger":
			name = "index.html"
		case strings.HasPrefix(path, "/deposits/"):
			name = "index.html"
		case path == "/login":
			name = "login.html"
		default:
			name = strings.TrimPrefix(path, "/")
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

	// Register deposit routes
	depositHandler := api.NewDepositHandler(transferRepo, vendorStub, ledgerSvc, fundingSvc, fundingCfg, operatorRepo, jobRepo, database)
	mux.HandleFunc("POST /deposits", api.WithIdempotency(database, depositHandler.Create))
	// GET /deposits and GET /deposits/{id} use content negotiation: JSON -> API, text/html -> operator SPA
	mux.HandleFunc("GET /deposits", depoListOrPage(depositHandler.List, operatorPageHandler))
	mux.HandleFunc("GET /deposits/{id}", depoGetOrPage(depositHandler.Get, operatorPageHandler))

	accountsHandler := api.NewAccountsHandler(fundingCfg, transferRepo)
	mux.HandleFunc("GET /accounts", accountsHandler.List)

	// Mobile/investor auth (login, logout, me)
	mobileAuthHandler := api.NewMobileAuthHandler(investorRepo)
	mux.HandleFunc("POST /mobile/login", mobileAuthHandler.MobileLogin)
	mux.HandleFunc("POST /mobile/logout", mobileAuthHandler.MobileLogout)
	mux.HandleFunc("GET /mobile/me", mobileAuthHandler.MobileMe)

	// Operator auth routes (no auth required)
	authHandler := api.NewAuthHandler(operatorRepo)
	mux.HandleFunc("POST /operator/login", authHandler.Login)
	mux.HandleFunc("POST /operator/guest", authHandler.Guest)
	mux.HandleFunc("POST /operator/logout", authHandler.Logout)
	mux.HandleFunc("GET /operator/me", authHandler.Me)

	// Operator routes (require login)
	operatorHandler := api.NewOperatorHandler(operatorRepo, transferRepo, ledgerSvc, fundingCfg, fundingSvc)
	mux.HandleFunc("GET /operator/queue", auth.RequireOperator(operatorHandler.Queue))
	mux.HandleFunc("GET /operator/transfers", auth.RequireOperator(operatorHandler.ListTransfers))
	mux.HandleFunc("GET /operator/audit", auth.RequireOperator(operatorHandler.Audit))
	mux.HandleFunc("POST /operator/approve", auth.RequireOperator(operatorHandler.Approve))
	mux.HandleFunc("POST /operator/reject", auth.RequireOperator(operatorHandler.Reject))
	mux.HandleFunc("GET /operator/transfer/{transfer_id}", auth.RequireOperator(operatorHandler.GetTransfer))
	mux.HandleFunc("GET /operator/actions/{transfer_id}", auth.RequireOperator(operatorHandler.Actions))

	// Settlement routes (require operator login)
	settlementHandler := api.NewSettlementHandler(settlementEngine)
	mux.HandleFunc("GET /health/settlement", settlementHandler.SettlementHealth)
	mux.HandleFunc("GET /settlement/status", auth.RequireOperator(settlementHandler.Status))
	mux.HandleFunc("GET /settlement/report/last", auth.RequireOperator(settlementHandler.LastReport))
	mux.HandleFunc("POST /settlement/report", auth.RequireOperator(settlementHandler.GenerateReport))
	mux.HandleFunc("POST /settlement/trigger", auth.RequireOperator(settlementHandler.Trigger))
	mux.HandleFunc("GET /settlement/reports", auth.RequireOperator(settlementHandler.ListReports))
	mux.HandleFunc("GET /settlement/reports/{id}/download", auth.RequireOperator(settlementHandler.DownloadReport))
	mux.HandleFunc("GET /settlement/reports/{id}/x9", auth.RequireOperator(settlementHandler.DownloadReportX9))
	mux.HandleFunc("GET /settlement/reports/{id}", auth.RequireOperator(settlementHandler.GetReport))

	// Returns routes
	returnsHandler := api.NewReturnsHandler(returnSvc)
	mux.HandleFunc("POST /returns", returnsHandler.ProcessReturn)

	// Sandbox: process one deposit job (for scenario runner; deposits are async 202)
	mux.HandleFunc("POST /sandbox/process-job", depositHandler.ProcessOneJobHTTP)

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

	// Operator UI (embedded) — page routes (GET /deposits and GET /deposits/{id} registered above with content negotiation)
	mux.HandleFunc("GET /", operatorPageHandler)
	mux.HandleFunc("GET /review-queue", operatorPageHandler)
	mux.HandleFunc("GET /settlement", operatorPageHandler)
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

	// Serve check images at /checks/{filename} for operator portal (same assets as mobile)
	checksHandler := func(w http.ResponseWriter, r *http.Request) {
		filename := strings.TrimPrefix(r.URL.Path, "/checks/")
		if filename == "" || strings.Contains(filename, "/") {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		b, err := fs.ReadFile(mobileRoot, "checks/"+filename)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		switch {
		case strings.HasSuffix(filename, ".png"):
			w.Header().Set("Content-Type", "image/png")
		case strings.HasSuffix(filename, ".jpg"), strings.HasSuffix(filename, ".jpeg"):
			w.Header().Set("Content-Type", "image/jpeg")
		default:
			w.Header().Set("Content-Type", "application/octet-stream")
		}
		w.Write(b)
	}
	mux.HandleFunc("GET /checks/", checksHandler)

	// Run deposit job worker: poll for pending jobs and process them
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			job, ok, err := jobRepo.ClaimNext()
			if err != nil {
				log.Printf("deposit worker: claim: %v", err)
				continue
			}
			if !ok {
				continue
			}
			if err := depositHandler.ProcessDeposit(job.TransferID); err != nil {
				log.Printf("deposit worker: process %s: %v", job.TransferID, err)
				_ = jobRepo.Fail(job.ID, err.Error())
			} else {
				_ = jobRepo.Complete(job.ID)
			}
		}
	}()

	// Run settlement every minute (in-process cron)
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			batch, err := settlementEngine.RunSettlement()
			if err != nil {
				log.Printf("settlement cron: %v", err)
				continue
			}
			if batch.TotalCount > 0 {
				log.Printf("settlement cron: batch %s, %d transfers, $%.2f", batch.BatchID, batch.TotalCount, batch.TotalAmount)
			}
		}
	}()

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
