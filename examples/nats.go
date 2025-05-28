package examples

import (
	"context"
	"github.com/imunhatep/goqueue/events"
	"github.com/imunhatep/goqueue/handlers"
	"log"

	"github.com/nats-io/nats.go"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nc, err := nats.Connect("nats://localhost:4222")
	if err != nil {
		log.Fatal(err)
	}

	dispatcher, err := events.NewNatsDispatcher(ctx, nc, &handlers.GobEncoder{})
	if err != nil {
		log.Fatal(err)
	}

	// Example of publishing an event
	event := events.Event{
		Subject: "example.subject",
		Data:    map[string]string{"key": "value"},
	}

	err = dispatcher.Publish(event.Subject, event)
	if err != nil {
		log.Fatalf("Failed to publish event: %v", err)
	} else {
		log.Printf("Event published successfully: %s", event.Subject)
	}

	// Example of subscribing to an event
	subChan, unsubscribe := dispatcher.Subscribe("subscriber1", "example.subject")
	defer unsubscribe()

	for {
		select {
		case <-ctx.Done():
			log.Println("Context cancelled, exiting...")
			return
		case msg := <-subChan:
			log.Printf("Received event: %s with data: %v", msg.Subject, msg.Data)
		}
	}
}
