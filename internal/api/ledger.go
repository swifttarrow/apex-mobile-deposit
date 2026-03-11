package api

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/checkstream/checkstream/internal/ledger"
)

// LedgerHandler handles ledger-related HTTP requests.
type LedgerHandler struct {
	db *sql.DB
}

// NewLedgerHandler creates a new LedgerHandler.
func NewLedgerHandler(db *sql.DB) *LedgerHandler {
	return &LedgerHandler{db: db}
}

func (h *LedgerHandler) ledgerSvc() *ledger.Service {
	return ledger.NewService(h.db)
}

// List handles GET /ledger.
func (h *LedgerHandler) List(w http.ResponseWriter, r *http.Request) {
	transferID := r.URL.Query().Get("transfer_id")

	entries, err := h.ledgerSvc().ListEntries(transferID)
	if err != nil {
		log.Printf("ledger list: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list ledger entries")
		return
	}
	if entries == nil {
		entries = []*ledger.Entry{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"entries": entries,
		"count":   len(entries),
	})
}

// Balance handles GET /accounts/:id/balance.
func (h *LedgerHandler) Balance(w http.ResponseWriter, r *http.Request) {
	accountID := r.PathValue("id")
	if accountID == "" {
		writeError(w, http.StatusBadRequest, "account id required")
		return
	}

	balance, err := h.ledgerSvc().GetAccountBalance(accountID)
	if err != nil {
		log.Printf("ledger balance: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get balance")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"account_id": accountID,
		"balance":    balance,
		"currency":   "USD",
	})
}
