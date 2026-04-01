package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

const maxJSONBodyBytes = 1 << 20

// RespondJSON writes a JSON response with the given status code.
func RespondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Best-effort; headers already sent.
		_ = err
	}
}

// eventEnvelope is the wire format for RespondEvent.
type eventEnvelope struct {
	Event string `json:"event"`
	Data  any    `json:"data"`
}

// RespondEvent writes a JSON response with event envelope: {"event": "type", "data": ...}.
func RespondEvent(w http.ResponseWriter, status int, eventType string, data any) {
	RespondJSON(w, status, eventEnvelope{Event: eventType, Data: data})
}

// errorBody is the wire format for RespondError.
type errorBody struct {
	Error string `json:"error"`
}

// RespondError writes a JSON error response: {"error": "message"}.
func RespondError(w http.ResponseWriter, status int, message string) {
	RespondJSON(w, status, errorBody{Error: message})
}

// DecodeJSON reads and decodes JSON from the request body into dst.
func DecodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, maxJSONBodyBytes+1))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}

	if dec.InputOffset() > maxJSONBodyBytes {
		return fmt.Errorf("request body exceeds %d bytes", maxJSONBodyBytes)
	}

	if err := dec.Decode(&struct{}{}); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return errors.New("request body must contain a single JSON object")
	}

	return errors.New("request body must contain a single JSON object")
}
