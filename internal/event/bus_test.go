package event

import (
	"sync"
	"testing"
	"time"

	"github.com/atask/atask/internal/domain"
)

func makeEvent(eventType domain.EventType) *domain.DomainEvent {
	e := domain.NewDomainEvent(eventType, "task", "123", "actor1", nil)
	return &e
}

func TestBus_SubscribeAndPublish(t *testing.T) {
	b := NewBus()

	var mu sync.Mutex
	var received []*domain.DomainEvent

	b.Subscribe("task.*", func(e *domain.DomainEvent) {
		mu.Lock()
		defer mu.Unlock()
		received = append(received, e)
	})

	b.Publish(makeEvent(domain.TaskCompleted))

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if len(received) != 1 {
		t.Fatalf("expected 1 event, got %d", len(received))
	}
	if received[0].Type != domain.TaskCompleted {
		t.Errorf("expected task.completed, got %s", received[0].Type)
	}
}

func TestBus_WildcardMatching(t *testing.T) {
	b := NewBus()

	var mu sync.Mutex
	var received []*domain.DomainEvent

	b.Subscribe("task.*", func(e *domain.DomainEvent) {
		mu.Lock()
		defer mu.Unlock()
		received = append(received, e)
	})

	b.Publish(makeEvent(domain.TaskCreated))
	b.Publish(makeEvent(domain.ProjectCreated))
	b.Publish(makeEvent(domain.TaskCompleted))

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if len(received) != 2 {
		t.Fatalf("expected 2 events, got %d", len(received))
	}
	for _, e := range received {
		if e.Type == domain.ProjectCreated {
			t.Errorf("project.created should not have been received by task.* subscriber")
		}
	}
}

func TestBus_Unsubscribe(t *testing.T) {
	b := NewBus()

	var mu sync.Mutex
	var received []*domain.DomainEvent

	id := b.Subscribe("task.*", func(e *domain.DomainEvent) {
		mu.Lock()
		defer mu.Unlock()
		received = append(received, e)
	})

	b.Unsubscribe(id)

	b.Publish(makeEvent(domain.TaskCompleted))

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if len(received) != 0 {
		t.Fatalf("expected 0 events after unsubscribe, got %d", len(received))
	}
}
