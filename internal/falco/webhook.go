// Package falco: HTTP webhook handler for Falco alerts.
package falco

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

const maxBodyBytes = 1 << 20 // 1 MiB

// WebhookHandler accepts POST /api/v1/alerts, decodes JSON into Event, validates, and sends to Events channel.
type WebhookHandler struct {
	Events chan<- Event
}

// NewWebhookHandler returns a handler that sends decoded events to the given channel.
func NewWebhookHandler(events chan<- Event) *WebhookHandler {
	return &WebhookHandler{Events: events}
}

// ServeHTTP decodes the request body, validates, and sends the event to the handler's channel.
func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	defer r.Body.Close()

	var raw FalcoEventRaw
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		log.Printf("[falco] webhook decode error: %v", err)
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	event := DecodeEvent(raw)
	if err := ValidateEvent(event); err != nil {
		log.Printf("[falco] webhook validation error: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	select {
	case h.Events <- event:
		w.WriteHeader(http.StatusAccepted)
	default:
		http.Error(w, "event queue full", http.StatusServiceUnavailable)
	}
}

// ValidateEvent checks required fields for a Falco event. Returns an error if invalid.
func ValidateEvent(e Event) error {
	if e.Rule == "" {
		return ErrInvalidEvent{Field: "rule", Reason: "empty"}
	}
	if e.Time.IsZero() {
		return ErrInvalidEvent{Field: "time", Reason: "zero or missing"}
	}
	return nil
}

// ErrInvalidEvent is returned when event validation fails.
type ErrInvalidEvent struct {
	Field  string
	Reason string
}

func (e ErrInvalidEvent) Error() string {
	return fmt.Sprintf("invalid event: %s %s", e.Field, e.Reason)
}
