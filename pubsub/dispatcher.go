package pubsub

import (
	"context"
	"strings"
	"sync"
	"time"
)

type Event struct {
	Subject  string            `json:"subject"`            // The exact subject name
	Data     interface{}       `json:"data"`               // The actual event data
	Metadata map[string]string `json:"metadata,omitempty"` // Optional metadata like timestamps, IDs, etc.
}

type EventDispatcher struct {
	mu          sync.RWMutex
	subscribers map[string][]chan Event
	publishCh   chan Event
}

// NewEventDispatcher creates a new EventDispatcher with a dedicated goroutine for processing.
func NewEventDispatcher(ctx context.Context) *EventDispatcher {
	ed := &EventDispatcher{
		subscribers: make(map[string][]chan Event),
		publishCh:   make(chan Event),
	}

	go ed.run(ctx)

	return ed
}

func (ed *EventDispatcher) run(ctx context.Context) {
	for {
		select {
		case event := <-ed.publishCh:
			ed.mu.RLock()
			for sub, chans := range ed.subscribers {
				if matchSubject(sub, event.Subject) {
					for _, ch := range chans {
						go func(c chan Event) {
							c <- event
						}(ch)
					}
				}
			}
			ed.mu.RUnlock()

		case <-ctx.Done():
			ed.close()
			return
		}
	}
}

func (ed *EventDispatcher) Subscribe(subject string) (<-chan Event, func()) {
	ch := make(chan Event)

	ed.mu.Lock()
	ed.subscribers[subject] = append(ed.subscribers[subject], ch)
	ed.mu.Unlock()

	unsubscribe := func() {
		ed.mu.Lock()
		defer ed.mu.Unlock()

		chans := ed.subscribers[subject]
		for i, sub := range chans {
			if sub == ch {
				ed.subscribers[subject] = append(chans[:i], chans[i+1:]...)
				close(ch)
				break
			}
		}

		if len(ed.subscribers[subject]) == 0 {
			delete(ed.subscribers, subject)
		}
	}

	return ch, unsubscribe
}

// Publish sends a message to all subscribers of the given subject.
func (ed *EventDispatcher) Publish(subject string, data interface{}) {
	// predefined meta, should be an argument
	metadata := map[string]string{
		"timestamp": time.Now().Format(time.DateTime),
	}

	event := Event{
		Subject:  subject,
		Data:     data,
		Metadata: metadata,
	}

	ed.publishCh <- event
}

// Close stops the dispatcher and cleans up resources.
func (ed *EventDispatcher) close() {
	ed.mu.Lock()
	for subject, chans := range ed.subscribers {
		for _, ch := range chans {
			close(ch)
		}
		delete(ed.subscribers, subject)
	}
	ed.mu.Unlock()
}

// matchSubject checks if a subject matches the subscription pattern.
func matchSubject(pattern, subject string) bool {
	patternTokens := strings.Split(pattern, ".")
	subjectTokens := strings.Split(subject, ".")

	for i := 0; i < len(patternTokens); i++ {
		if i >= len(subjectTokens) {
			// Pattern has more tokens than subject, but no multi-level wildcard to match
			return patternTokens[i] == ">"
		}

		switch patternTokens[i] {
		case "*":
			// Single-level wildcard, matches exactly one token
			continue
		case ">":
			// Multi-level wildcard, matches everything after this point
			return true
		default:
			if patternTokens[i] != subjectTokens[i] {
				return false
			}
		}
	}

	// Match if the pattern ends at the same point as the subject
	return len(patternTokens) == len(subjectTokens)
}
