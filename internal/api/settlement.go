package api

import (
	"encoding/json"
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

// Status handles GET /settlement/status for the operator UI (settled vs unsettled overview).
func (h *SettlementHandler) Status(w http.ResponseWriter, r *http.Request) {
	status, err := h.engine.Status()
	if err != nil {
		log.Printf("settlement status: %v", err)
		writeError(w, http.StatusInternalServerError, "settlement status failed")
		return
	}
	writeJSON(w, http.StatusOK, status)
}

// Trigger handles POST /settlement/trigger. Runs settlement (no file write); reports are generated on-demand via GenerateReport.
func (h *SettlementHandler) Trigger(w http.ResponseWriter, r *http.Request) {
	afterEOD := settlement.IsAfterEOD(h.nowFn())
	batch, err := h.engine.RunSettlement()
	if err != nil {
		log.Printf("settlement trigger: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to run settlement")
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

// GenerateReport handles POST /settlement/report. Returns a report of all settlements since the last report and updates the last-report timestamp.
func (h *SettlementHandler) GenerateReport(w http.ResponseWriter, r *http.Request) {
	report, lastReportAt, err := h.engine.ReportSinceLastReport()
	if err != nil {
		log.Printf("settlement report: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to generate report")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"report_id":     report.BatchID,
		"total_count":   report.TotalCount,
		"total_amount":  report.TotalAmount,
		"created_at":    report.CreatedAt,
		"transfers":     report.Transfers,
		"last_report_at": lastReportAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// LastReport handles GET /settlement/report/last. Returns the last report timestamp for the UI.
func (h *SettlementHandler) LastReport(w http.ResponseWriter, r *http.Request) {
	t, err := h.engine.LastReportAt()
	if err != nil {
		log.Printf("settlement last report: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get last report time")
		return
	}
	var s string
	if !t.IsZero() {
		s = t.Format("2006-01-02T15:04:05Z07:00")
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"last_report_at": s})
}

// ListReports handles GET /settlement/reports. Returns all previous settlement reports (batches), newest first.
func (h *SettlementHandler) ListReports(w http.ResponseWriter, r *http.Request) {
	reports, err := h.engine.ListReports()
	if err != nil {
		log.Printf("settlement list reports: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list reports")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"reports": reports})
}

// GetReport handles GET /settlement/reports/{id}. Returns full report (batch) with transfers for expand in UI.
func (h *SettlementHandler) GetReport(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "report id required")
		return
	}
	report, err := h.engine.GetReport(id)
	if err != nil {
		log.Printf("settlement get report: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get report")
		return
	}
	if report == nil {
		writeError(w, http.StatusNotFound, "report not found")
		return
	}
	writeJSON(w, http.StatusOK, report)
}

// DownloadReport handles GET /settlement/reports/{id}/download. Returns report as JSON file attachment (array of settlements).
func (h *SettlementHandler) DownloadReport(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "report id required")
		return
	}
	report, err := h.engine.GetReport(id)
	if err != nil {
		log.Printf("settlement download report: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get report")
		return
	}
	if report == nil {
		writeError(w, http.StatusNotFound, "report not found")
		return
	}
	// Download as JSON array of settlement entries (transfers)
	data, err := json.MarshalIndent(report.Transfers, "", "  ")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to marshal report")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", `attachment; filename="settlement-report-`+id+`.json"`)
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// DownloadReportX9 handles GET /settlement/reports/{id}/x9. Returns an X9-style text summary file (human-readable).
// Full X9.37 ICL binary format can be added later via moov-io/imagecashletter.
func (h *SettlementHandler) DownloadReportX9(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "report id required")
		return
	}
	report, err := h.engine.GetReport(id)
	if err != nil {
		log.Printf("settlement x9 report: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get report")
		return
	}
	if report == nil {
		writeError(w, http.StatusNotFound, "report not found")
		return
	}
	data := settlement.X9TextSummary(report)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="settlement-`+id+`.x9"`)
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
