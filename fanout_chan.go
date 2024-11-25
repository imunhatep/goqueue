package queue

import (
	"context"
	"github.com/imunhatep/gocollection/slice"
	"github.com/rs/zerolog/log"
	"time"
)

type FanOutChan[T any] struct {
	ctx       context.Context
	name      string
	queueSize int
	metrics   *QueueMetrics

	// source chan
	source <-chan (chan T)

	// pubsub
	listeners []chan (<-chan T) // Slice of chan of chans for subscribers

	// subscribers requests
	addListeners    chan (chan (<-chan T))
	removeListeners chan (chan (<-chan T))
}

func NewChanBroadcaster[T any](ctx context.Context, name string, source <-chan (chan T)) *FanOutChan[T] {
	b := &FanOutChan[T]{
		ctx:       ctx,
		name:      name,
		queueSize: 10,

		// source
		source: source,
		// broadcasting
		listeners:       make([]chan (<-chan T), 0),
		addListeners:    make(chan chan (<-chan T)),
		removeListeners: make(chan chan (<-chan T)),
	}

	go b.handleSubscribers()

	return b
}

func (b *FanOutChan[T]) WithMetrics(metrics *QueueMetrics) *FanOutChan[T] {
	b.metrics = metrics

	return b
}

func (b *FanOutChan[T]) WithQueueSize(size int) *FanOutChan[T] {
	b.queueSize = size
	return b
}

func (b *FanOutChan[T]) Run() *FanOutChan[T] {
	go b.handleSource(b.source)

	return b
}

func (b *FanOutChan[T]) handleSource(source <-chan (chan T)) {
	log.Info().Str("queue", b.name).Msg("[FanOutChan.handleSource] running...")

	defer b.closeListeners()

	for {
		select {
		case <-b.ctx.Done():
			log.Trace().Str("queue", b.name).Msg("[FanOutChan.handleSource] context cancelled")
			return
		case ch, ok := <-source: // Read from source channel of channels
			log.Trace().Str("queue", b.name).Msg("[FanOutChan.handleSource] new chan from source")
			if !ok {
				log.Error().Str("queue", b.name).Msg("[FanOutChan.handleSource] feeding failed")
				return
			}
			go b.handleChannel(ch) // Start a goroutine to handle the received channel
		}
	}
}

func (b *FanOutChan[T]) copyChannel(stream <-chan T) []chan T {
	fanOut := []chan T{}

	for _, listener := range b.listeners {
		if listener == nil {
			continue
		}

		streamCopy := make(chan T, cap(stream))
		select {
		case listener <- streamCopy:
			log.Trace().Msg("[FanOutChan.copyChannel] listener stream sent")
		default:
		}
		fanOut = append(fanOut, streamCopy)
	}

	log.Trace().
		Int("listeners", len(b.listeners)).
		Int("fanOut", len(fanOut)).
		Msg("[FanOutChan.copyChannel] chan copied for listeners")

	return fanOut
}

func (b *FanOutChan[T]) handleChannel(stream <-chan T) {
	log.Trace().Int("listeners", len(b.listeners)).Msg("[FanOutChan.handleChannel] handling chan")

	fanOut := b.copyChannel(stream)
	defer slice.Map(fanOut, func(ch chan T) bool {
		log.Trace().Msg("[FanOutChan.handleChannel] closing fanOut chan")
		close(ch)
		return true
	})

	for {
		select {
		case <-b.ctx.Done():
			log.Trace().Str("queue", b.name).Msg("[FanOutChan.handleChannel] context cancelled")
			return
		case val, ok := <-stream:
			if !ok {
				return
			}

			b.handleValue(fanOut, val) // Push the value to listeners
		}
	}
}

func (b *FanOutChan[T]) handleValue(fanOut []chan T, val T) {
	for _, ch := range fanOut {
		select {
		case <-b.ctx.Done():
			log.Trace().Str("queue", b.name).Msg("[FanOutChan.handleValue] context cancelled")
			return
		case ch <- val:
			//log.Trace().Int("fanOut", len(ch)).Msg("[FanOutChan.handleValue] value sent")
			if b.metrics != nil {
				b.metrics.ObserverWriteCount.WithLabelValues(b.name).Inc()
			}
		default:
			log.Warn().Str("queue", b.name).Msg("[FanOutChan.handleValue] event dismissed, listener chan is full")
			if b.metrics != nil {
				b.metrics.ObserverWriteFullCount.WithLabelValues(b.name).Inc()
			}
		}
	}
}

func (b *FanOutChan[T]) addListener(newListener chan (<-chan T)) {
	b.listeners = append(b.listeners, newListener)

	if b.metrics != nil {
		b.metrics.ObserverSubscribed.WithLabelValues(b.name).Set(float64(len(b.listeners)))
	}

	log.Trace().Str("queue", b.name).Msg("[FanOutChan.addListener] subscriber added")
}

func (b *FanOutChan[T]) removeListener(listenerToRemove chan (<-chan T)) {
	for i, ch := range b.listeners {
		if ch == listenerToRemove {
			b.listeners[i] = b.listeners[len(b.listeners)-1]
			b.listeners = b.listeners[:len(b.listeners)-1]
			close(ch)
			break
		}
	}

	if b.metrics != nil {
		b.metrics.ObserverSubscribed.WithLabelValues(b.name).Set(float64(len(b.listeners)))
	}

	log.Trace().Str("queue", b.name).Msg("[FanOutChan.removeListener] subscriber removed")
}

func (b *FanOutChan[T]) closeListeners() {
	for _, listener := range b.listeners {
		if listener != nil {
			close(listener)
		}
	}
}

func (b *FanOutChan[T]) Subscribe() <-chan (<-chan T) {
	log.Info().Str("queue", b.name).Msg("[FanOutChan.Subscribe] request for new listener")

	newListener := make(chan (<-chan T), b.queueSize)
	b.addListeners <- newListener
	return newListener
}

func (b *FanOutChan[T]) Unsubscribe(channel chan (<-chan T)) {
	log.Info().Str("queue", b.name).Msg("[FanOutChan.Unsubscribe] request to remove listener")
	b.removeListeners <- channel
}

func (b *FanOutChan[T]) handleSubscribers() {
	log.Info().Str("queue", b.name).Msg("[FanOutChan.handleSubscribers] running...")

	defer b.closeListeners()

	for {
		select {
		case <-b.ctx.Done():
			// sleep to give listeners to read from chans
			time.Sleep(5 * time.Second)
			log.Trace().Str("queue", b.name).Msg("[FanOutChan.handleSubscribers] context cancelled")
			return
		case newListener := <-b.addListeners:
			b.addListener(newListener)
		case listenerToRemove := <-b.removeListeners:
			b.removeListener(listenerToRemove)
		}
	}
}
