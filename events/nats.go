package events

import (
	"context"
	"github.com/go-errors/errors"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
	"sync"
	"time"
)

type Encoder interface {
	Encode(v interface{}) ([]byte, error)
	Decode(data []byte) (interface{}, error)
}

type SubjectConsumer struct {
	subjects   map[string]*nats.Subscription
	chanStream chan Event
}

type NatsDispatcher struct {
	ctx         context.Context
	mu          sync.RWMutex
	subscribers map[string]SubjectConsumer
	natsConn    *nats.Conn
	handler     Encoder
}

// NewNatsDispatcher creates a new NatsDispatcher with a dedicated goroutine for processing.
func NewNatsDispatcher(ctx context.Context, nc *nats.Conn, handler Encoder) (*NatsDispatcher, error) {
	if nc == nil || handler == nil {
		return nil, errors.New("invalid arguments: nats connection and handler are required")
	}

	return &NatsDispatcher{
		ctx:         ctx,
		subscribers: make(map[string]SubjectConsumer),
		natsConn:    nc,
		handler:     handler,
	}, nil
}

func (d *NatsDispatcher) Subscribe(name string, subjects ...string) (<-chan Event, func()) {
	ch := make(chan Event)

	if err := d.createEventsConsumer(ch, name, subjects); err != nil {
		log.Error().Err(err).Msg("[NatsDispatcher] failed to create consumer")
		close(ch)
		return ch, nil
	}

	unsubscribe := func() {
		d.unsubscribeConsumer(name)
		close(ch)
	}

	return ch, unsubscribe
}

// Publish sends a message to all subscribers of the given subject.
func (d *NatsDispatcher) Publish(subject string, data interface{}) error {
	if d.natsConn.Status() != nats.CONNECTED {
		log.Error().Msg("[NatsDispatcher] NATS client is not connected")
		return errors.New("NATS client is not connected")
	}

	eventData, err := d.handler.Encode(data)
	if err != nil {
		log.Error().Err(err).Msg("[NatsDispatcher] failed to encode event data")
		return err
	}

	if err := d.natsConn.Publish(subject, eventData); err != nil {
		log.Error().Err(err).Msg("[NatsDispatcher] failed to publish event")
		return errors.New(err)
	}

	log.Trace().Str("subject", subject).Int("size", len(eventData)).Msg("[NatsDispatcher] published")

	return nil
}

func (d *NatsDispatcher) createEventsConsumer(ch chan Event, name string, subjects []string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if _, exists := d.subscribers[name]; exists {
		return errors.New("consumer already exists")
	}

	consumer := SubjectConsumer{
		chanStream: ch,
		subjects:   make(map[string]*nats.Subscription),
	}

	for _, subject := range subjects {
		subscription, err := d.natsConn.Subscribe(subject, d.createEventCallback(name, ch))
		if err != nil {
			log.Error().Err(err).Str("subject", subject).Msg("[NatsDispatcher] failed to subscribe")
			return err
		}
		consumer.subjects[subject] = subscription
	}

	d.subscribers[name] = consumer
	return nil
}

func (d *NatsDispatcher) unsubscribeConsumer(name string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	consumer, exists := d.subscribers[name]
	if !exists {
		log.Warn().Str("name", name).Msg("[NatsDispatcher] consumer not found")
		return
	}

	for subject, sub := range consumer.subjects {
		if err := sub.Unsubscribe(); err != nil {
			log.Error().Err(err).Str("subject", subject).Msg("[NatsDispatcher] failed to unsubscribe")
		}
	}

	delete(d.subscribers, name)
}

// Close stops the dispatcher and cleans up resources.
func (d *NatsDispatcher) Close() {
	d.mu.Lock()
	defer d.mu.Unlock()

	for name := range d.subscribers {
		d.unsubscribeConsumer(name)
	}

	if err := d.natsConn.Drain(); err != nil {
		log.Error().Err(err).Msg("[NatsDispatcher] failed to drain NATS connection")
	}
}

func (d *NatsDispatcher) createEventCallback(name string, ch chan Event) func(*nats.Msg) {
	return func(msg *nats.Msg) {
		subject := msg.Subject
		data, err := d.handler.Decode(msg.Data)
		if err != nil {
			log.Error().Err(err).Str("name", name).Str("subject", subject).Msg("[NatsDispatcher] failed to decode event data")
			return
		}

		metadata := map[string]string{
			"timestamp": time.Now().Format(time.RFC3339),
		}

		event := Event{
			Subject:  subject,
			Metadata: metadata,
			Data:     data,
		}

		select {
		case ch <- event:
		case <-d.ctx.Done():
			log.Warn().Str("subject", subject).Msg("[NatsDispatcher] context canceled, dropping event")
		}
	}
}
