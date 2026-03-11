package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// responseRecorder captures the response for caching.
type responseRecorder struct {
	http.ResponseWriter
	body       *bytes.Buffer
	statusCode int
}

func (r *responseRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

// WithIdempotency wraps a handler with idempotency key support.
func WithIdempotency(db *sql.DB, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("X-Idempotency-Key")
		if key == "" {
			// Generate a new idempotency key if none provided
			key = uuid.New().String()
		}

		// Check if we've seen this key before
		var body string
		var statusCode int
		err := db.QueryRow(
			`SELECT response_body, status_code FROM idempotency_keys WHERE key = ?`,
			key,
		).Scan(&body, &statusCode)

		if err == nil {
			// Replay cached response
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Idempotency-Replayed", "true")
			w.WriteHeader(statusCode)
			io.WriteString(w, body)
			return
		}

		// Record the response
		rec := &responseRecorder{
			ResponseWriter: w,
			body:           &bytes.Buffer{},
			statusCode:     http.StatusOK,
		}

		next(rec, r)

		// Cache the response
		now := time.Now().UTC().Format(time.RFC3339)
		_, saveErr := db.Exec(
			`INSERT OR IGNORE INTO idempotency_keys (key, response_body, status_code, created_at) VALUES (?, ?, ?, ?)`,
			key, rec.body.String(), rec.statusCode, now,
		)
		if saveErr != nil {
			log.Printf("idempotency: failed to save key %s: %v", key, saveErr)
		}
	}
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writeJSON: %v", err)
	}
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
