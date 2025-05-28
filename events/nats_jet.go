package events

import (
	"context"
	"github.com/go-errors/errors"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog/log"
	"sync"
	"time"
)

type StreamConsumer struct {
	natsStream jetstream.ConsumeContext
	chanStream chan Event
}

type NatsJetDispatcher struct {
	ctx         context.Context
	mu          sync.RWMutex
	consumers   map[string]StreamConsumer
	natsConn    *nats.Conn
	natsStream  jetstream.JetStream
	dataHandler Encoder
}

// NewNatsJetDispatcher creates a new NatsJetDispatcher with a dedicated goroutine for processing.
func NewNatsJetDispatcher(ctx context.Context, nc *nats.Conn, handler Encoder) (*NatsJetDispatcher, error) {
	js, err := jetstream.New(nc)
	if err != nil {
		return nil, errors.New(err)
	}

	ed := &NatsJetDispatcher{
		ctx:         ctx,
		consumers:   make(map[string]StreamConsumer),
		natsConn:    nc,
		natsStream:  js,
		dataHandler: handler,
	}

	return ed, nil
}

func (d *NatsJetDispatcher) Subscribe(name string, subjects ...string) (<-chan Event, func()) {
	ch := make(chan Event)

	log.Debug().Str("name", name).Strs("subjects", subjects).Msg("[NatsJetDispatcher] new NATS consumer stream")
	jstream, err := d.natsStream.CreateStream(d.ctx, jetstream.StreamConfig{
		Name:     name,
		Subjects: subjects,
	})

	if err != nil {
		log.Error().Err(err).Msg("[NatsJetDispatcher] failed to create consumer")
		close(ch)
		return ch, nil
	}

	cons, err := jstream.CreateOrUpdateConsumer(d.ctx, jetstream.ConsumerConfig{
		Durable:   "Test",
		AckPolicy: jetstream.AckExplicitPolicy,
	})

	if err != nil {
		log.Error().Err(err).Msg("[NatsJetDispatcher] failed to create consumer")
		close(ch)
		return ch, nil
	}

	// register even callback
	cc, err := cons.Consume(d.createEventCallback(name, ch))

	d.mu.Lock()
	d.consumers[name] = StreamConsumer{
		natsStream: cc,
		chanStream: ch,
	}
	d.mu.Unlock()

	unsubscribe := func() {
		d.mu.Lock()
		defer d.mu.Unlock()

		consumer, ok := d.consumers[name]
		if ok && consumer.chanStream == ch {
			delete(d.consumers, name)

			consumer.natsStream.Stop()
			close(consumer.chanStream)
		}
	}

	return ch, unsubscribe
}

// Publish sends a message to all subscribers of the given subject.
func (d *NatsJetDispatcher) Publish(subject string, data interface{}) error {
	if d.natsConn.Status() != nats.CONNECTED {
		log.Error().Msg("[NatsJetDispatcher] NATS client is not connected")
		return errors.New("NATS client is not connected")
	}

	eventData, err := d.dataHandler.Encode(data)
	if err != nil {
		log.Error().Err(err).Msg("[NatsJetDispatcher] failed to encode event data")
		return err
	}

	if _, err := d.natsStream.Publish(d.ctx, subject, eventData); err != nil {
		log.Error().Err(err).Msg("[NatsJetDispatcher] failed to publish event")
		return errors.New(err)
	}

	return nil
}

// Close stops the dispatcher and cleans up resources.
func (d *NatsJetDispatcher) close() {
	d.mu.Lock()

	// close NATS connection
	d.natsConn.Close()

	// close all channels
	for name, consumer := range d.consumers {
		delete(d.consumers, name)

		consumer.natsStream.Stop()
		close(consumer.chanStream)
	}

	d.mu.Unlock()
}

func (d *NatsJetDispatcher) createEventCallback(name string, ch chan Event) func(jetstream.Msg) {
	fn := func(msg jetstream.Msg) {
		if err := msg.Ack(); err != nil {
			log.Error().Err(err).Str("name", name).Msg("[NatsJetDispatcher] failed to ack message")
		}

		subject := msg.Subject()
		eventData := msg.Data()

		data, err := d.dataHandler.Decode(eventData)
		if err != nil {
			log.Error().Err(err).
				Str("name", name).
				Str("subject", subject).
				Msg("[NatsJetDispatcher] failed to decode event data")

			return
		}

		event := Event{
			Subject:  subject,
			Metadata: map[string]string{"timestamp": time.Now().Format(time.DateTime)},
			Data:     data,
		}

		select {
		case ch <- event:
		case <-d.ctx.Done():
			log.Warn().Str("subject", subject).Msg("[NatsJetDispatcher] context canceled, dropping event")
		}
	}

	return fn
}
