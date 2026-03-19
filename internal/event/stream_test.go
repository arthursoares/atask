package event

import (
	"context"
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
