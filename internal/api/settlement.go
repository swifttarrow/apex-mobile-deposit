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
}

// NewSettlementHandler creates a new SettlementHandler.
func NewSettlementHandler(engine *settlement.Engine) *SettlementHandler {
	return &SettlementHandler{engine: engine}
}

// Trigger handles POST /settlement/trigger.
func (h *SettlementHandler) Trigger(w http.ResponseWriter, r *http.Request) {
	afterEOD := settlement.IsAfterEOD(time.Now())
	batch, err := h.engine.GenerateSettlementFile()
	if err != nil {
		log.Printf("settlement trigger: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to generate settlement file")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":         "settlement triggered",
		"batch_id":        batch.BatchID,
		"total_count":     batch.TotalCount,
		"total_amount":    batch.TotalAmount,
		"created_at":      batch.CreatedAt,
		"after_eod_cutoff": afterEOD,
	})
}
