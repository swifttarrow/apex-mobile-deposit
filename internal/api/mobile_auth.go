package api

import (
	"encoding/json"
	"net/http"

	"github.com/checkstream/checkstream/internal/auth"
	"github.com/checkstream/checkstream/internal/investor"
)

// MobileAuthHandler handles login, logout, and session check for mobile/investor users.
type MobileAuthHandler struct {
	investorRepo *investor.InvestorRepo
}

// NewMobileAuthHandler creates a new MobileAuthHandler.
func NewMobileAuthHandler(investorRepo *investor.InvestorRepo) *MobileAuthHandler {
	return &MobileAuthHandler{investorRepo: investorRepo}
}

// MobileLoginRequest is the request body for POST /mobile/login.
type MobileLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// MobileLogin handles POST /mobile/login.
func (h *MobileAuthHandler) MobileLogin(w http.ResponseWriter, r *http.Request) {
	var req MobileLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "username and password are required")
		return
	}

	acc, err := h.investorRepo.GetByUsername(req.Username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "login failed")
		return
	}
	if acc == nil || !acc.VerifyPassword(req.Password) {
		writeError(w, http.StatusUnauthorized, "invalid username or password")
		return
	}

	if err := auth.SetInvestorSession(w, r, acc.ID, acc.DisplayName); err != nil {
		writeError(w, http.StatusInternalServerError, "session error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"user_id":      acc.ID,
		"username":     acc.Username,
		"display_name": acc.DisplayName,
	})
}

// MobileLogout handles POST /mobile/logout.
func (h *MobileAuthHandler) MobileLogout(w http.ResponseWriter, r *http.Request) {
	if err := auth.ClearInvestorSession(w, r); err != nil {
		writeError(w, http.StatusInternalServerError, "logout failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}

// ResolveMobileUserID returns the current mobile user ID from session (if logged in) or X-User-ID header. Used by GET /accounts and GET /deposits.
func ResolveMobileUserID(r *http.Request) string {
	if id, err := auth.GetInvestorID(r); err == nil && id != "" {
		return id
	}
	return r.Header.Get("X-User-ID")
}

// MobileMe handles GET /mobile/me. Returns current investor info or 401.
func (h *MobileAuthHandler) MobileMe(w http.ResponseWriter, r *http.Request) {
	investorID, err := auth.GetInvestorID(r)
	if err != nil || investorID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"unauthorized"}`))
		return
	}

	acc, err := h.investorRepo.GetByID(investorID)
	if err != nil || acc == nil {
		auth.ClearInvestorSession(w, r)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"unauthorized"}`))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"user_id":      acc.ID,
		"username":     acc.Username,
		"display_name": acc.DisplayName,
	})
}
