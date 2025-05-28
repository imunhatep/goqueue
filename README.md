# GoQueue Library

GoQueue is a Go library that provides a simple and efficient way to handle fan-out messaging, event dispatching, and asynchronous reading. It includes the following main components:

- `FanOut`: A structure for broadcasting messages to multiple listeners.
- `EventDispatcher`: A structure for publishing and subscribing to events.
- `AsyncReader`: A structure for reading from a channel asynchronously.

## Installation

To install the GoQueue library, use the following command:

```sh
go get github.com/imunhatep/goqueue
```

## Usage

### EventDispatcher

The `EventDispatcher` structure allows you to publish and subscribe to events in-memory. Events are filtered by subject using Go NATS matching wildcards, and subscribers receive events through a channel.

#### Wildcards
```
// Will receive any subject begining with "event."
const SubjectAll = "event.>"

// Will receive "event.object.update.test1" but not "event.update.test1"  
const SubjectUpdateWildcard = "*.*.update.>"
```

#### Example

```go
package main

import (
	"context"
	"fmt"
	"github.com/imunhatep/goqueue/pubsub"
	"time"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dispatcher := pubsub.NewEventDispatcher(ctx)

	ch, unsubscribe := dispatcher.Subscribe("example.subject")
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

### FanOut 
`Fan-out` refers to the practice of starting multiple goroutines to handle incoming tasks. The main idea is to distribute incoming tasks to multiple handlers (goroutines) to ensure that each handler deals with a
manageable number of tasks.

#### Example

```go
package main

import (
	"context"
	"fmt"
	"time"
	"github.com/imunhatep/goqueue"
)

func main() {
	// init metrics, optional
	queue.NewQueueMetrics(queue.QueueMetricsSubsystem)
	
	source := make(chan string)
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fanOut := goqueue.NewFanOut[string](ctx, "exampleQueue", source)

	listener1 := fanOut.Subscribe()
	listener2 := fanOut.Subscribe()

	go func() {
		for msg := range listener1 {
			fmt.Println("Listener 1 received:", msg)
		}
	}()

	go func() {
		for msg := range listener2 {
			fmt.Println("Listener 2 received:", msg)
		}
	}()

	source <- "Hello, World!"
	time.Sleep(1 * time.Second)
}
```

### AsyncReader
The `AsyncReader` structure reads values from channel until channel is closed. This allows multiple readers to share single instance of AsyncReader and get values from a channel.   

#### Example

```go
package main

import (
	"fmt"
	"github.com/imunhatep/goqueue"
)

func main() {
	ch := make(chan int)

	go func() {
		for i := 0; i < 5; i++ {
			ch <- i
		}
		close(ch)
	}()

	reader := goqueue.NewAsyncReader(ch)
	values := reader.Read()

	fmt.Println("Read values:", values)
}
```
