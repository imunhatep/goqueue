# GoQueue Library

GoQueue is a Go library that provides a event brokers for In Memory, and NATS queue handling. Handlers for reading chan messages till closed. It includes the following main components:
- `events.LocalDispatcher`: A structure for publishing and subscribing to events in memory.
- `events.NatsDispatcher`: A structure for publishing and subscribing to events in NATS.io.
- `events.NatsJetDispatcher`: A structure for publishing and subscribing to events in NATS.io JetStreams.
- `handler.AsyncReader`: A structure for reading from a channel asynchronously.
- `handler.GobEncoder`: A structure for encoding/decoding events in NATS.io dispatchers.

## Installation

To install the GoQueue library, use the following command:

```sh
go get github.com/imunhatep/goqueue
```

## Usage

### Dispatchers

The `LocaDispatcher` structure allows you to publish and subscribe to events in-memory. Events are filtered by subject using Go NATS matching wildcards, and subscribers receive events through a channel.

#### Wildcards
```
// Will receive any subject begining with "event."
const SubjectAll = "event.>"

// Will receive "event.object.update.test1" but not "event.update.test1"  
const SubjectUpdateWildcard = "*.*.update.>"
```

#### Example 1

```go
package main

import (
	"context"
	"fmt"
	"github.com/imunhatep/goqueue/events"
	"time"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dispatcher := events.NewEventDispatcher(ctx)

	ch, unsubscribe := dispatcher.Subscribe("subscriber name", "example.subject")
	defer unsubscribe()

	go func() {
		for event := range ch {
			fmt.Println("Received event:", event)
		}
	}()

	dispatcher.Publish("example.subject", "Hello, Event!")
	time.Sleep(1 * time.Second)
}
```

#### Example 2

NATS.io dispatcher allows you to publish and subscribe to events using NATS.io. It supports both regular NATS and JetStream.

```go
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

````

### AsyncReader
The `AsyncReader` structure reads values from channel until channel is closed. This allows multiple readers to share single instance of AsyncReader and get values from a channel.   

#### Example

```go
package main

import (
	"fmt"
	"github.com/imunhatep/handlers"
)

func main() {
	ch := make(chan int)

	go func() {
		for i := 0; i < 5; i++ {
			ch <- i
		}
		close(ch)
	}()

	reader := handlers.NewAsyncReader(ch)
	values := reader.Read()

	fmt.Println("Read values:", values)
}
```
