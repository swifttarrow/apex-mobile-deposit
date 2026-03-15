package api

import (
	"log"
	"net/http"
	"time"

	"github.com/checkstream/checkstream/internal/settlement"
)

// SettlementHandler handles settlement-related HTTP requests.
type SettlementHandler struct {
	engine *settlement.Engine
	nowFn  func() time.Time
}

// NewSettlementHandler creates a new SettlementHandler.
func NewSettlementHandler(engine *settlement.Engine) *SettlementHandler {
	return &SettlementHandler{
		engine: engine,
		nowFn:  time.Now,
	}
}

// SetNowFunc overrides the clock source used by the handler.
func (h *SettlementHandler) SetNowFunc(nowFn func() time.Time) {
	if nowFn == nil {
		h.nowFn = time.Now
		return
	}
	h.nowFn = nowFn
}

// SettlementHealth handles GET /health/settlement for monitoring missing or delayed settlement files.
func (h *SettlementHandler) SettlementHealth(w http.ResponseWriter, r *http.Request) {
	now := h.nowFn()
	unsettledCount, afterEOD, err := h.engine.SettlementHealth(now)
	if err != nil {
		log.Printf("settlement health: %v", err)
		writeError(w, http.StatusInternalServerError, "settlement health check failed")
		return
	}
	status := "ok"
	warning := ""
	if afterEOD && unsettledCount > 0 {
		warning = "unsettled_funds_posted"
		// Optional: return 503 to alert; we use 200 with warning for flexibility
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":                   status,
		"after_eod_cutoff":         afterEOD,
		"unsettled_funds_posted":   unsettledCount,
		"warning":                  warning,
	})
}

// Trigger handles POST /settlement/trigger.
func (h *SettlementHandler) Trigger(w http.ResponseWriter, r *http.Request) {
	afterEOD := settlement.IsAfterEOD(h.nowFn())
	batch, err := h.engine.GenerateSettlementFile()
	if err != nil {
		log.Printf("settlement trigger: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to generate settlement file")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":          "settlement triggered",
		"batch_id":         batch.BatchID,
		"total_count":      batch.TotalCount,
		"total_amount":     batch.TotalAmount,
		"created_at":       batch.CreatedAt,
		"after_eod_cutoff": afterEOD,
	})
}
