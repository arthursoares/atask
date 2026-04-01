package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRespondJSON(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}

	w := httptest.NewRecorder()
	RespondJSON(w, http.StatusOK, payload{Name: "test"})

	resp := w.Result()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}

	var got payload
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if got.Name != "test" {
		t.Errorf("expected name %q, got %q", "test", got.Name)
	}
}

func TestRespondJSON_StatusCode(t *testing.T) {
	w := httptest.NewRecorder()
	RespondJSON(w, http.StatusCreated, map[string]string{"id": "abc"})

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}
}

func TestRespondEvent(t *testing.T) {
	type payload struct {
		Value int `json:"value"`
	}

	w := httptest.NewRecorder()
	RespondEvent(w, http.StatusOK, "task.created", payload{Value: 42})

	resp := w.Result()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}

	var envelope map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if _, ok := envelope["event"]; !ok {
		t.Error("expected 'event' key in response envelope")
	}
	if _, ok := envelope["data"]; !ok {
		t.Error("expected 'data' key in response envelope")
	}

	eventType, ok := envelope["event"].(string)
	if !ok {
		t.Fatal("expected 'event' to be a string")
	}
	if eventType != "task.created" {
		t.Errorf("expected event type %q, got %q", "task.created", eventType)
	}

	data, ok := envelope["data"].(map[string]any)
	if !ok {
		t.Fatal("expected 'data' to be an object")
	}
	val, ok := data["value"].(float64)
	if !ok {
		t.Fatal("expected 'data.value' to be a number")
	}
	if val != 42 {
		t.Errorf("expected data.value 42, got %v", val)
	}
}

func TestRespondError(t *testing.T) {
	w := httptest.NewRecorder()
	RespondError(w, http.StatusBadRequest, "invalid input")

	resp := w.Result()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	msg, ok := body["error"]
	if !ok {
		t.Fatal("expected 'error' key in response body")
	}
	if msg != "invalid input" {
		t.Errorf("expected error message %q, got %q", "invalid input", msg)
	}
}

func TestDecodeJSON(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}

	body := `{"name":"hello"}`
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	var dst payload
	if err := DecodeJSON(r, &dst); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if dst.Name != "hello" {
		t.Errorf("expected name %q, got %q", "hello", dst.Name)
	}
}

func TestDecodeJSON_InvalidBody(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("not-json"))
	r.Header.Set("Content-Type", "application/json")

	var dst map[string]any
	if err := DecodeJSON(r, &dst); err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestDecodeJSON_RejectsUnknownFields(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}

	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"hello","extra":true}`))
	r.Header.Set("Content-Type", "application/json")

	var dst payload
	if err := DecodeJSON(r, &dst); err == nil {
		t.Fatal("expected error for unknown fields, got nil")
	}
}

func TestDecodeJSON_RejectsTrailingJSON(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}

	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"hello"}{"name":"again"}`))
	r.Header.Set("Content-Type", "application/json")

	var dst payload
	if err := DecodeJSON(r, &dst); err == nil {
		t.Fatal("expected error for trailing JSON, got nil")
	}
}

func TestDecodeErrorMessage(t *testing.T) {
	t.Run("empty body", func(t *testing.T) {
		if got := decodeErrorMessage(io.EOF); got != "request body must not be empty" {
			t.Fatalf("unexpected message: %q", got)
		}
	})

	t.Run("unknown field", func(t *testing.T) {
		err := errors.New(`json: unknown field "extra"`)
		if got := decodeErrorMessage(err); got != `request body contains unknown field "extra"` {
			t.Fatalf("unexpected message: %q", got)
		}
	})

	t.Run("trailing json", func(t *testing.T) {
		err := errors.New("request body must contain a single JSON object")
		if got := decodeErrorMessage(err); got != "request body must contain a single JSON object" {
			t.Fatalf("unexpected message: %q", got)
		}
	})
}
