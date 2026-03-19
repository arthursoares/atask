package event

import (
	"strings"
	"sync"

	"github.com/atask/atask/internal/domain"
)

// Handler is a function that handles a domain event.
type Handler func(e *domain.DomainEvent)

type subscription struct {
	id      int
	pattern string
	handler Handler
}

// Bus is an in-process pub/sub bus for domain events with wildcard topic matching.
type Bus struct {
	mu          sync.RWMutex
	subscribers []subscription
	nextID      int
}

// NewBus creates a new Bus instance.
func NewBus() *Bus {
	return &Bus{}
}

// Subscribe registers a handler for events matching the given pattern.
// It returns a subscription ID that can be used to unsubscribe later.
//
// Pattern matching rules:
//   - "*" matches all event types.
//   - "task.*" matches any event type starting with "task.".
//   - "task.completed" matches the exact event type "task.completed".
func (b *Bus) Subscribe(pattern string, handler Handler) int {
	b.mu.Lock()
	defer b.mu.Unlock()

	id := b.nextID
	b.nextID++
	b.subscribers = append(b.subscribers, subscription{
		id:      id,
		pattern: pattern,
		handler: handler,
	})
	return id
}

// Unsubscribe removes the subscription with the given ID.
func (b *Bus) Unsubscribe(id int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	subs := b.subscribers[:0]
	for _, s := range b.subscribers {
		if s.id != id {
			subs = append(subs, s)
		}
	}
	b.subscribers = subs
}

// Publish dispatches the event asynchronously to all matching subscribers.
func (b *Bus) Publish(event *domain.DomainEvent) {
	b.mu.RLock()
	matched := make([]Handler, 0, len(b.subscribers))
	for _, s := range b.subscribers {
		if matchPattern(s.pattern, string(event.Type)) {
			matched = append(matched, s.handler)
		}
	}
	b.mu.RUnlock()

	for _, h := range matched {
		go h(event)
	}
}

// matchPattern reports whether eventType matches the given pattern.
func matchPattern(pattern, eventType string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, ".*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(eventType, prefix)
	}
	return pattern == eventType
}
