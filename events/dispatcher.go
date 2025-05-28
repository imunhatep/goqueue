package events

import (
	"context"
	"strings"
	"sync"
	"time"
)

type LocalDispatcher struct {
	mu          sync.RWMutex
	subscribers map[string][]chan Event
	publishCh   chan Event
}

// NewLocalDispatcher creates a new LocalDispatcher with a dedicated goroutine for processing.
func NewLocalDispatcher(ctx context.Context) *LocalDispatcher {
	ed := &LocalDispatcher{
		subscribers: make(map[string][]chan Event),
		publishCh:   make(chan Event),
	}

	go ed.run(ctx)

	return ed
}

func (ed *LocalDispatcher) run(ctx context.Context) {
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

func (ed *LocalDispatcher) Subscribe(name string, subjects ...string) (<-chan Event, func()) {
	ch := make(chan Event)

	ed.mu.Lock()
	for _, subject := range subjects {
		ed.subscribers[subject] = append(ed.subscribers[subject], ch)
	}
	ed.mu.Unlock()

	unsubscribe := func() {
		ed.mu.Lock()
		defer ed.mu.Unlock()

		for _, subject := range subjects {
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
	}

	return ch, unsubscribe
}

// Publish sends a message to all subscribers of the given subject.
func (ed *LocalDispatcher) Publish(subject string, data interface{}) error {
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

	return nil
}

// Close stops the dispatcher and cleans up resources.
func (ed *LocalDispatcher) close() {
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
