package event

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/atask/atask/internal/domain"
)

func TestStreamManager_SSE(t *testing.T) {
	bus := NewBus()
	sm := NewStreamManager(bus)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/events?topics=task.*", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()

	// Publish an event after 50ms delay in a goroutine
	go func() {
		time.Sleep(50 * time.Millisecond)
		e := domain.NewDomainEvent(domain.TaskCompleted, "task", "123", "actor1", nil)
		e.ID = 42
		bus.Publish(&e)
	}()

	sm.ServeHTTP(rr, req)

	body := rr.Body.String()

	if !strings.Contains(body, "event: task.completed") {
		t.Errorf("expected body to contain 'event: task.completed', got:\n%s", body)
	}
	if !strings.Contains(body, "id: 42") {
		t.Errorf("expected body to contain 'id: 42', got:\n%s", body)
	}
}

// sseEvent represents a parsed Server-Sent Event.
type sseEvent struct {
	Event string
	Data  map[string]any
	ID    string
}

// readSSEEvents reads SSE events from an HTTP response body until the expected
// count is reached or the timeout expires.
func readSSEEvents(t *testing.T, resp *http.Response, count int, timeout time.Duration) []sseEvent {
	t.Helper()

	var events []sseEvent
	scanner := bufio.NewScanner(resp.Body)

	done := make(chan struct{})
	go func() {
		defer close(done)
		var current sseEvent
		for scanner.Scan() {
			line := scanner.Text()
			switch {
			case strings.HasPrefix(line, "event: "):
				current.Event = strings.TrimPrefix(line, "event: ")
			case strings.HasPrefix(line, "data: "):
				var data map[string]any
				if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &data); err == nil {
					current.Data = data
				}
			case strings.HasPrefix(line, "id: "):
				current.ID = strings.TrimPrefix(line, "id: ")
			case line == "":
				if current.Event != "" {
					events = append(events, current)
					current = sseEvent{}
					if len(events) >= count {
						return
					}
				}
			}
		}
	}()

	select {
	case <-done:
	case <-time.After(timeout):
		t.Log("SSE read timed out")
	}

	return events
}

// TestStreamManager_SSE_Integration tests the full SSE flow using a real HTTP
// server: connect, receive multiple events in order, verify payload contents.
func TestStreamManager_SSE_Integration(t *testing.T) {
	bus := NewBus()
	sm := NewStreamManager(bus)

	server := httptest.NewServer(sm)
	defer server.Close()

	ctx := t.Context()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"?topics=task.*", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("SSE connect: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("expected Content-Type text/event-stream, got %s", ct)
	}

	// Publish 3 events after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)

		e1 := domain.NewDomainEvent(domain.TaskCreated, "task", "task-1", "user-1", map[string]any{"title": "Buy groceries"})
		e1.ID = 1
		bus.Publish(&e1)

		time.Sleep(20 * time.Millisecond)

		e2 := domain.NewDomainEvent(domain.TaskCompleted, "task", "task-1", "user-1", nil)
		e2.ID = 2
		bus.Publish(&e2)

		time.Sleep(20 * time.Millisecond)

		e3 := domain.NewDomainEvent(domain.TaskCreated, "task", "task-2", "user-1", map[string]any{"title": "Walk the dog"})
		e3.ID = 3
		bus.Publish(&e3)
	}()

	events := readSSEEvents(t, resp, 3, 2*time.Second)

	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}

	// Event 1: task.created with entity_id and title
	if events[0].Event != "task.created" {
		t.Errorf("event[0]: expected task.created, got %s", events[0].Event)
	}
	if events[0].Data["entity_id"] != "task-1" {
		t.Errorf("event[0]: expected entity_id=task-1, got %v", events[0].Data["entity_id"])
	}
	if events[0].Data["title"] != "Buy groceries" {
		t.Errorf("event[0]: expected title=Buy groceries, got %v", events[0].Data["title"])
	}
	if events[0].ID != "1" {
		t.Errorf("event[0]: expected id=1, got %s", events[0].ID)
	}

	// Event 2: task.completed — must include entity_id even with no custom payload
	if events[1].Event != "task.completed" {
		t.Errorf("event[1]: expected task.completed, got %s", events[1].Event)
	}
	if events[1].Data["entity_id"] != "task-1" {
		t.Errorf("event[1]: expected entity_id=task-1, got %v", events[1].Data["entity_id"])
	}
	if events[1].Data["entity_type"] != "task" {
		t.Errorf("event[1]: expected entity_type=task, got %v", events[1].Data["entity_type"])
	}
	if events[1].ID != "2" {
		t.Errorf("event[1]: expected id=2, got %s", events[1].ID)
	}

	// Event 3: second task.created
	if events[2].Event != "task.created" {
		t.Errorf("event[2]: expected task.created, got %s", events[2].Event)
	}
	if events[2].Data["entity_id"] != "task-2" {
		t.Errorf("event[2]: expected entity_id=task-2, got %v", events[2].Data["entity_id"])
	}
	if events[2].Data["title"] != "Walk the dog" {
		t.Errorf("event[2]: expected title=Walk the dog, got %v", events[2].Data["title"])
	}
}

// TestStreamManager_SSE_TopicFiltering verifies that events not matching the
// subscribed topic pattern are not delivered to the SSE client.
func TestStreamManager_SSE_TopicFiltering(t *testing.T) {
	bus := NewBus()
	sm := NewStreamManager(bus)

	server := httptest.NewServer(sm)
	defer server.Close()

	ctx := t.Context()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"?topics=project.*", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("SSE connect: %v", err)
	}
	defer resp.Body.Close()

	go func() {
		time.Sleep(50 * time.Millisecond)

		// Task event — should NOT be received
		e1 := domain.NewDomainEvent(domain.TaskCreated, "task", "task-1", "user-1", nil)
		e1.ID = 1
		bus.Publish(&e1)

		time.Sleep(20 * time.Millisecond)

		// Project event — should be received
		e2 := domain.NewDomainEvent(domain.ProjectCreated, "project", "proj-1", "user-1", map[string]any{"title": "My Project"})
		e2.ID = 2
		bus.Publish(&e2)
	}()

	events := readSSEEvents(t, resp, 1, 2*time.Second)

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Event != "project.created" {
		t.Errorf("expected project.created, got %s", events[0].Event)
	}
	if events[0].Data["entity_id"] != "proj-1" {
		t.Errorf("expected entity_id=proj-1, got %v", events[0].Data["entity_id"])
	}
}

// TestStreamManager_SSE_ClientDisconnect verifies that publishing events after
// a client disconnects does not panic or hang.
func TestStreamManager_SSE_ClientDisconnect(t *testing.T) {
	bus := NewBus()
	sm := NewStreamManager(bus)

	server := httptest.NewServer(sm)
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"?topics=task.*", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("SSE connect: %v", err)
	}

	// Disconnect the client
	cancel()
	resp.Body.Close()

	// Publish after disconnect — must not panic
	time.Sleep(50 * time.Millisecond)
	e := domain.NewDomainEvent(domain.TaskCreated, "task", "task-1", "user-1", nil)
	e.ID = 1
	bus.Publish(&e)

	time.Sleep(50 * time.Millisecond)
}
