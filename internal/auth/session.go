package auth

import (
	"net/http"
	"os"

	"github.com/gorilla/sessions"
)

const (
	sessionName    = "operator_session"
	operatorIDKey  = "operator_id"
	operatorNameKey = "display_name"
)

// Store holds the session store. Use a 32-byte secret for production.
var Store *sessions.CookieStore

func init() {
	secret := os.Getenv("SESSION_SECRET")
	if secret == "" {
		secret = "checkdepot-dev-secret-change-in-production-32b"
	}
	// gorilla/sessions requires 16 or 32 bytes for AES
	if len(secret) < 32 {
		padded := make([]byte, 32)
		copy(padded, secret)
		secret = string(padded)
	} else {
		secret = secret[:32]
	}
	Store = sessions.NewCookieStore([]byte(secret))
	Store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   os.Getenv("SESSION_SECURE") == "true",
	}
}

// GetOperatorID returns the logged-in operator ID from the session, or empty if not logged in.
func GetOperatorID(r *http.Request) (string, error) {
	session, err := Store.Get(r, sessionName)
	if err != nil {
		return "", err
	}
	id, _ := session.Values[operatorIDKey].(string)
	return id, nil
}

// SetOperatorSession stores the operator ID and display name in the session.
func SetOperatorSession(w http.ResponseWriter, r *http.Request, operatorID, displayName string) error {
	session, err := Store.Get(r, sessionName)
	if err != nil {
		return err
	}
	session.Values[operatorIDKey] = operatorID
	session.Values[operatorNameKey] = displayName
	return session.Save(r, w)
}

// ClearSession removes the operator session.
func ClearSession(w http.ResponseWriter, r *http.Request) error {
	session, err := Store.Get(r, sessionName)
	if err != nil {
		return err
	}
	session.Options.MaxAge = -1
	return session.Save(r, w)
}
