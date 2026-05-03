package response

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"
)

type contextKey string

const traceIDKey contextKey = "traceId"

// Envelope is the unified API response shape.
type Envelope struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data,omitempty"`
	Details   string `json:"details,omitempty"`
	Timestamp string `json:"timestamp"`
	TraceID   string `json:"traceId"`
}

// OK writes a successful API response.
func OK(w http.ResponseWriter, r *http.Request, data any) {
	writeJSON(w, r, http.StatusOK, Envelope{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// Error writes a unified API error response.
func Error(w http.ResponseWriter, r *http.Request, status int, code int, message string, details string) {
	writeJSON(w, r, status, Envelope{
		Code:    code,
		Message: message,
		Details: details,
	})
}

// WithTraceID stores a trace ID in a request context.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// TraceID returns the trace ID from a request context.
func TraceID(ctx context.Context) string {
	value, ok := ctx.Value(traceIDKey).(string)
	if !ok || value == "" {
		return NewTraceID()
	}
	return value
}

// NewTraceID creates a short random trace ID.
func NewTraceID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(bytes[:])
}

func writeJSON(w http.ResponseWriter, r *http.Request, status int, envelope Envelope) {
	envelope.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)
	envelope.TraceID = TraceID(r.Context())

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(true)
	if err := encoder.Encode(envelope); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}
