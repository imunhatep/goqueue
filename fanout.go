package queue

import (
	"context"
	"github.com/rs/zerolog/log"
	"time"
)

type FanOut[T any] struct {
	ctx       context.Context
	name      string
	queueSize int
	metrics   *QueueMetrics

	// broadcaster
	source          <-chan T
	listeners       []chan T
	addListeners    chan (chan T)
	removeListeners chan (<-chan T)
}

func NewFanOut[T any](ctx context.Context, name string, source <-chan T) *FanOut[T] {
	b := &FanOut[T]{
		ctx:       ctx,
		name:      name,
		queueSize: 10,

		// broadcasting
		source:          source,
		listeners:       make([]chan T, 0),
		addListeners:    make(chan chan T),
		removeListeners: make(chan (<-chan T)),
	}

	go b.handleSubscribers()

	return b
}

func (b *FanOut[T]) WithMetrics(metrics *QueueMetrics) *FanOut[T] {
	b.metrics = metrics

	return b
}

func (b *FanOut[T]) WithQueueSize(size int) *FanOut[T] {
	b.queueSize = size

	return b
}

func (b *FanOut[T]) Run() *FanOut[T] {
	go b.listenAndServe()
	return b
}

func (b *FanOut[T]) Subscribe() <-chan T {
	newListener := make(chan T, b.queueSize)
	b.addListeners <- newListener

	return newListener
}

func (b *FanOut[T]) Unsubscribe(channel <-chan T) {
	b.removeListeners <- channel
}

func (b *FanOut[T]) listenAndServe() {
	log.Info().Str("queue", b.name).Msg("[FanOut.listenAndServe] running...")

	for {
		select {
		case <-b.ctx.Done():
			log.Debug().Str("queue", b.name).Msg("[FanOut.listenAndServe] context cancelled")
			return
		case val, ok := <-b.source:
			// metrics
			if b.metrics != nil {
				b.metrics.ObserverReadCount.WithLabelValues(b.name).Inc()
			}

			if !ok {
				log.Error().Str("queue", b.name).Msg("[FanOut.listenAndServe] feeding failed")
				return
			}

			b.push(val)
		}
	}
}

func (b *FanOut[T]) push(val T) {
	for _, listener := range b.listeners {
		if listener == nil {
			continue
		}

		select {
		case <-b.ctx.Done():
			log.Debug().Str("queue", b.name).Msg("[FanOut.push] context cancelled")
			return
		case listener <- val:
			// metrics
			if b.metrics != nil {
				b.metrics.ObserverWriteCount.WithLabelValues(b.name).Inc()
			}

		default:
			log.Warn().Str("queue", b.name).Msg("[FanOut.push] event dismissed, listener chan is full")
			if b.metrics != nil {
				b.metrics.ObserverWriteFullCount.WithLabelValues(b.name).Inc()
			}
		}
	}
}

func (b *FanOut[T]) handleSubscribers() {
	log.Info().Str("queue", b.name).Msg("[FanOut.handleSubscribers] running...")

	defer b.closeListeners()

	for {
		select {
		case <-b.ctx.Done():
			// sleep to give listeners to read from chans
			time.Sleep(5 * time.Second)
			log.Trace().Str("queue", b.name).Msg("[FanOut.handleSubscribers] context cancelled")
			return
		case newListener := <-b.addListeners:
			b.addListener(newListener)
		case listenerToRemove := <-b.removeListeners:
			b.removeListener(listenerToRemove)
		}
	}
}

func (b *FanOut[T]) addListener(newListener chan T) {
	b.listeners = append(b.listeners, newListener)

	// metrics
	if b.metrics != nil {
		b.metrics.ObserverSubscribed.WithLabelValues(b.name).Set(float64(len(b.listeners)))
	}

	log.Debug().Str("queue", b.name).Msg("[FanOut.addListener] subscriber added")
}

func (b *FanOut[T]) removeListener(listenerToRemove <-chan T) {
	for i, ch := range b.listeners {
		if ch == listenerToRemove {
			b.listeners[i] = b.listeners[len(b.listeners)-1]
			b.listeners = b.listeners[:len(b.listeners)-1]
			close(ch)
			break
		}
	}

	// metrics
	if b.metrics != nil {
		b.metrics.ObserverSubscribed.WithLabelValues(b.name).Set(float64(len(b.listeners)))
	}

	log.Debug().Str("queue", b.name).Msg("[FanOut.removeListener] subscriber removed")
}

func (b *FanOut[T]) closeListeners() {
	for _, listener := range b.listeners {
		if listener != nil {
			close(listener)
		}
	}
}
