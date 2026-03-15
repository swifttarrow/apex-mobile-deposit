package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/checkstream/checkstream/internal/auth"
	"github.com/checkstream/checkstream/internal/clock"
)

// ClockHandler provides test-only controls for app time travel.
type ClockHandler struct {
	clock *clock.TravelClock
}

// NewClockHandler creates a new ClockHandler.
func NewClockHandler(c *clock.TravelClock) *ClockHandler {
	return &ClockHandler{clock: c}
}

type clockUpdateRequest struct {
	Action string `json:"action"` // set | freeze | resume
	Time   string `json:"time"`   // RFC3339; required for action=set
}

func clockStatusPayload(c *clock.TravelClock) map[string]interface{} {
	now := c.Now().UTC()
	frozen := c.IsFrozen()
	mode := "running"
	if frozen {
		mode = "frozen"
	}
	return map[string]interface{}{
		"now":    now.Format(time.RFC3339),
		"frozen": frozen,
		"mode":   mode,
	}
}

func isTestOperatorID(operatorID string) bool {
	// Seeded test operators are op-1..op-5.
	return strings.HasPrefix(operatorID, "op-")
}

func (h *ClockHandler) authorizeTestOperator(w http.ResponseWriter, r *http.Request) bool {
	operatorID, err := auth.GetOperatorID(r)
	if err != nil || operatorID == "" {
		writeError(w, http.StatusUnauthorized, "login required")
		return false
	}
	if !isTestOperatorID(operatorID) {
		writeError(w, http.StatusForbidden, "time travel is limited to test operator accounts")
		return false
	}
	return true
}

// Get handles GET /operator/clock.
func (h *ClockHandler) Get(w http.ResponseWriter, r *http.Request) {
	if !h.authorizeTestOperator(w, r) {
		return
	}
	writeJSON(w, http.StatusOK, clockStatusPayload(h.clock))
}

// Update handles POST /operator/clock.
func (h *ClockHandler) Update(w http.ResponseWriter, r *http.Request) {
	if !h.authorizeTestOperator(w, r) {
		return
	}

	var req clockUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	switch req.Action {
	case "freeze":
		h.clock.Freeze()
	case "resume":
		h.clock.Resume()
	case "set":
		if req.Time == "" {
			writeError(w, http.StatusBadRequest, "time is required for set action")
			return
		}
		ts, err := time.Parse(time.RFC3339, req.Time)
		if err != nil {
			writeError(w, http.StatusBadRequest, "time must be RFC3339")
			return
		}
		h.clock.Set(ts)
	default:
		writeError(w, http.StatusBadRequest, "action must be one of: set, freeze, resume")
		return
	}

	writeJSON(w, http.StatusOK, clockStatusPayload(h.clock))
}
