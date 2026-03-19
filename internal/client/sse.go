package client

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// DomainEvent represents a server-sent event from the atask event stream.
type DomainEvent struct {
	ID       int64          `json:"id"`
	Type     string         `json:"type"`
	EntityID string         `json:"entity_id"`
	Payload  map[string]any `json:"payload"`
}

// SubscribeEvents connects to the SSE event stream and returns a channel of DomainEvents.
// It reconnects automatically on EOF or error using the Last-Event-ID header.
// The channel is closed when ctx is cancelled.
func (c *Client) SubscribeEvents(ctx context.Context, topics string) (<-chan DomainEvent, error) {
	ch := make(chan DomainEvent, 256)

	go func() {
		defer close(ch)

		var lastEventID string

		for {
			// Check if context is cancelled before attempting (re)connect.
			if ctx.Err() != nil {
				return
			}

			if err := c.streamEvents(ctx, topics, lastEventID, ch, &lastEventID); err != nil {
				// If the context was cancelled, stop silently.
				if ctx.Err() != nil {
					return
				}
				// Otherwise wait 1s and reconnect.
				select {
				case <-ctx.Done():
					return
				case <-time.After(time.Second):
				}
			}
		}
	}()

	return ch, nil
}

// streamEvents opens one SSE connection, reads events, and forwards them to ch.
// It updates *lastID with each received event ID so the caller can reconnect with it.
func (c *Client) streamEvents(ctx context.Context, topics, lastEventID string, ch chan<- DomainEvent, lastID *string) error {
	url := fmt.Sprintf("%s/events/stream?topics=%s", c.baseURL, topics)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create SSE request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if lastEventID != "" {
		req.Header.Set("Last-Event-ID", lastEventID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("SSE connect: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("SSE server error %d", resp.StatusCode)
	}

	// SSE field accumulators for the current event block.
	var (
		eventType string
		dataLines []string
		idStr     string
	)

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()

		// Blank line signals end of event.
		if line == "" {
			if len(dataLines) > 0 {
				rawData := strings.Join(dataLines, "\n")

				var payload map[string]any
				if err := json.Unmarshal([]byte(rawData), &payload); err != nil {
					// Skip malformed events but keep reading.
					eventType = ""
					dataLines = dataLines[:0]
					idStr = ""
					continue
				}

				var id int64
				if idStr != "" {
					id, _ = strconv.ParseInt(idStr, 10, 64)
					*lastID = idStr
				}

				// Extract entity_id from payload if present.
				entityID, _ := payload["entity_id"].(string)

				evt := DomainEvent{
					ID:       id,
					Type:     eventType,
					EntityID: entityID,
					Payload:  payload,
				}

				select {
				case ch <- evt:
				case <-ctx.Done():
					return nil
				}
			}

			// Reset accumulators.
			eventType = ""
			dataLines = dataLines[:0]
			idStr = ""
			continue
		}

		if strings.HasPrefix(line, "event:") {
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		} else if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		} else if strings.HasPrefix(line, "id:") {
			idStr = strings.TrimSpace(strings.TrimPrefix(line, "id:"))
		}
		// Ignore comment lines (starting with ':') and unknown fields.
	}

	if err := scanner.Err(); err != nil {
		if ctx.Err() != nil {
			return nil
		}
		return fmt.Errorf("SSE read error: %w", err)
	}

	// EOF — signal caller to reconnect.
	return fmt.Errorf("SSE stream closed (EOF)")
}
