package handlers

import (
	"github.com/rs/zerolog/log"
	"sync"
)

type AsyncReader[T any] struct {
	values []T
	wg     sync.WaitGroup
}

func NewAsyncReader[T any](ch <-chan T) *AsyncReader[T] {
	cr := &AsyncReader[T]{
		values: []T{},
		wg:     sync.WaitGroup{},
	}

	cr.wg.Add(1)
	go cr.await(ch)

	return cr
}

func (cr *AsyncReader[T]) await(channel <-chan T) {
	defer cr.wg.Done()

	log.Trace().Msg("[AsyncReader.await] reading channel..")
	for v := range channel {
		cr.values = append(cr.values, v)
	}

	log.Trace().Int("len", len(cr.values)).Msg("[AsyncReader.await] values found")
}

func (cr *AsyncReader[T]) Read() []T {
	cr.wg.Wait()

	result := make([]T, len(cr.values))
	copy(result, cr.values)

	return result
}
