package api

import (
	"encoding/json"
	"net/http"

	"github.com/checkstream/checkstream/internal/auth"
	"github.com/checkstream/checkstream/internal/operator"
	"github.com/google/uuid"
)

// AuthHandler handles login, logout, and session check for operators.
type AuthHandler struct {
	operatorRepo *operator.Repository
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(operatorRepo *operator.Repository) *AuthHandler {
	return &AuthHandler{operatorRepo: operatorRepo}
}

// LoginRequest is the request body for POST /operator/login.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Login handles POST /operator/login.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "username and password are required")
		return
	}

	op, err := h.operatorRepo.GetOperatorByUsername(req.Username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "login failed")
		return
	}
	if op == nil || !op.VerifyPassword(req.Password) {
		writeError(w, http.StatusUnauthorized, "invalid username or password")
		return
	}

	if err := auth.SetOperatorSession(w, r, op.ID, op.DisplayName); err != nil {
		writeError(w, http.StatusInternalServerError, "session error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"operator_id":  op.ID,
		"username":     op.Username,
		"display_name": op.DisplayName,
		"email":        op.Email,
	})
}

// Guest handles POST /operator/guest. Creates an ephemeral session with a generated operator ID.
func (h *AuthHandler) Guest(w http.ResponseWriter, r *http.Request) {
	operatorID := "guest-" + uuid.New().String()[:8]
	displayName := "Guest"

	if err := auth.SetOperatorSession(w, r, operatorID, displayName); err != nil {
		writeError(w, http.StatusInternalServerError, "session error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"operator_id":  operatorID,
		"username":     "",
		"display_name": displayName,
		"email":        "",
	})
}

// Logout handles POST /operator/logout.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if err := auth.ClearSession(w, r); err != nil {
		writeError(w, http.StatusInternalServerError, "logout failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}

// Me handles GET /operator/me. Returns current operator info or 401.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	operatorID, err := auth.GetOperatorID(r)
	if err != nil || operatorID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"unauthorized"}`))
		return
	}

	// Guest sessions: return synthetic response without DB lookup
	if len(operatorID) >= 6 && operatorID[:6] == "guest-" {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"operator_id":  operatorID,
			"username":     "",
			"display_name": "Guest",
			"email":        "",
		})
		return
	}

	op, err := h.operatorRepo.GetOperatorByID(operatorID)
	if err != nil || op == nil {
		// Session has invalid operator ID
		auth.ClearSession(w, r)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"unauthorized"}`))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"operator_id":  op.ID,
		"username":     op.Username,
		"display_name": op.DisplayName,
		"email":        op.Email,
	})
}
