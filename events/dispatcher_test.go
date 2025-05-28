package events

import (
	"context"
	"github.com/rs/zerolog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.TraceLevel)
}

func TestLocalDispatcher_SubscribePublish(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dispatcher := NewLocalDispatcher(ctx)
	subject := "test.subject"

	// Subscribe to the subject
	ch, unsubscribe := dispatcher.Subscribe("test", subject)
	defer unsubscribe()

	// Publish an event
	data := "test data"
	err := dispatcher.Publish(subject, data)
	assert.NoError(t, err, "Publish should not return an error")

	// Verify the event is received
	select {
	case event := <-ch:
		assert.Equal(t, subject, event.Subject, "Event subject should match")
		assert.Equal(t, data, event.Data, "Event data should match")
	case <-time.After(1 * time.Second):
		t.Fatal("Timed out waiting for event")
	}
}

func TestLocalDispatcher_Unsubscribe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dispatcher := NewLocalDispatcher(ctx)
	subject := "test.subject"

	// Subscribe to the subject
	ch, unsubscribe := dispatcher.Subscribe("test", subject)

	// Unsubscribe
	unsubscribe()

	// Publish an event
	err := dispatcher.Publish(subject, "test data")
	assert.NoError(t, err, "Publish should not return an error")

	// Verify the channel is closed
	select {
	case _, ok := <-ch:
		assert.False(t, ok, "Channel should be closed after unsubscribe")
	case <-time.After(1 * time.Second):
		t.Fatal("Timed out waiting for channel to close")
	}
}

func TestLocalDispatcher_Close(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	dispatcher := NewLocalDispatcher(ctx)
	subject := "test.subject"

	// Subscribe to the subject
	ch, unsubscribe := dispatcher.Subscribe("test", subject)
	defer unsubscribe()

	// Close the dispatcher
	cancel()

	// Verify the channel is closed
	select {
	case _, ok := <-ch:
		assert.False(t, ok, "Channel should be closed after dispatcher is closed")
	case <-time.After(1 * time.Second):
		t.Fatal("Timed out waiting for channel to close")
	}
}
