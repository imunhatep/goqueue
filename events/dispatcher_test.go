package events

import (
	"context"
	"github.com/rs/zerolog"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.TraceLevel)
}

func TestEventDispatcher_SubscribeAndPublish(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	dispatcher := NewLocalDispatcher(ctx)
	subject := "test.subject"

	ch, unsubscribe := dispatcher.Subscribe(subject)
	defer unsubscribe()

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		event := <-ch
		assert.Equal(t, subject, event.Subject)
		assert.Equal(t, "test message", event.Data)
	}()

	dispatcher.Publish(subject, "test message")
	wg.Wait()
}

func TestEventDispatcher_PublishWithNoSubscribers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	dispatcher := NewLocalDispatcher(ctx)
	subject := "test.subject"

	// Publish without any subscribers
	dispatcher.Publish(subject, "test message")

	// No assertions needed, just ensure no panic or error
}

func TestEventDispatcher_MultipleSubscribers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	dispatcher := NewLocalDispatcher(ctx)
	subject := "test.subject"

	ch1, unsubscribe1 := dispatcher.Subscribe(subject)
	defer unsubscribe1()
	ch2, unsubscribe2 := dispatcher.Subscribe(subject)
	defer unsubscribe2()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		event := <-ch1
		assert.Equal(t, subject, event.Subject)
		assert.Equal(t, "test message", event.Data)
	}()

	go func() {
		defer wg.Done()
		event := <-ch2
		assert.Equal(t, subject, event.Subject)
		assert.Equal(t, "test message", event.Data)
	}()

	dispatcher.Publish(subject, "test message")
	wg.Wait()
}

func TestEventDispatcher_Unsubscribe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	dispatcher := NewLocalDispatcher(ctx)
	subject := "test.subject"

	ch, unsubscribe := dispatcher.Subscribe(subject)
	unsubscribe()

	var received bool
	go func() {
		select {
		case _, ok := <-ch:
			received = ok
		case <-time.After(100 * time.Millisecond):
			received = false
		}
	}()

	dispatcher.Publish(subject, "test message")
	time.Sleep(200 * time.Millisecond)

	assert.False(t, received)
}

func TestEventDispatcher_Close(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())

	dispatcher := NewLocalDispatcher(ctx)
	subject := "test.subject"

	ch, unsubscribe := dispatcher.Subscribe(subject)
	defer unsubscribe()

	// Close the dispatcher
	cancel()

	// Wait for the dispatcher to close the channel
	time.Sleep(100 * time.Millisecond)

	_, ok := <-ch

	assert.False(t, ok, "The channel should be closed")
}

func TestEventDispatcher_SingleLevelWildcard(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	dispatcher := NewLocalDispatcher(ctx)
	subject := "test.*"

	ch, unsubscribe := dispatcher.Subscribe(subject)
	defer unsubscribe()

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		event := <-ch
		assert.Equal(t, "test message", event.Data)
	}()

	dispatcher.Publish("test.subject", "test message")
	wg.Wait()
}

func TestEventDispatcher_MultiLevelWildcard(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	dispatcher := NewLocalDispatcher(ctx)
	subject := "test.>"

	ch, unsubscribe := dispatcher.Subscribe(subject)
	defer unsubscribe()

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		event := <-ch
		assert.Equal(t, "test message", event.Data)
	}()

	dispatcher.Publish("test.subject.sub", "test message")
	wg.Wait()
}

func TestEventDispatcher_NoMatchForDifferentSubject(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	dispatcher := NewLocalDispatcher(ctx)
	subject := "test.subject"

	ch, unsubscribe := dispatcher.Subscribe(subject)
	defer unsubscribe()

	var received bool
	go func() {
		select {
		case <-ch:
			received = true
		case <-time.After(100 * time.Millisecond):
			received = false
		}
	}()

	dispatcher.Publish("test.other", "test message")
	time.Sleep(200 * time.Millisecond)

	assert.False(t, received)
}
