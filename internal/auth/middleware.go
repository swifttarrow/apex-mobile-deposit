package auth

import (
	"net/http"
)

// RequireOperator wraps a handler and returns 401 if the user is not logged in.
// The next handler can call GetOperatorID(r) to get the authenticated operator ID.
func RequireOperator(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := GetOperatorID(r)
		if err != nil || id == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"unauthorized","message":"login required"}`))
			return
		}
		next(w, r)
	}
}
