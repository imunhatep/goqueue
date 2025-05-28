package events

type Event struct {
	Subject  string            `json:"subject"`            // The exact subject name
	Data     interface{}       `json:"data"`               // The actual event data
	Metadata map[string]string `json:"metadata,omitempty"` // Optional metadata like timestamps, IDs, etc.
}
