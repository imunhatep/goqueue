package queue

import (
	"github.com/rs/zerolog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.TraceLevel)
}

func TestNewAsyncReader(t *testing.T) {
	ch := make(chan int, 3)
	reader := NewAsyncReader(ch)

	// write some values
	ch <- 1
	ch <- 2
	ch <- 3
	close(ch)

	values := reader.Read()

	assert.Equal(t, []int{1, 2, 3}, values, "The values read from the channel should match the expected values")
}

func TestAsyncReader_Read2(t *testing.T) {
	ch := make(chan int, 3)

	reader := NewAsyncReader(ch)
	time.Sleep(1 * time.Second) // Ensure the goroutine has time to read from the channel

	ch <- 4
	ch <- 5
	ch <- 6
	close(ch)

	read1 := reader.Read()
	assert.Equal(t, []int{4, 5, 6}, read1, "The values read from the channel should match the expected values")

	read2 := reader.Read()
	assert.Equal(t, []int{4, 5, 6}, read2, "The values read from the channel should match the expected values")
}
