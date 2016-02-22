package types

import (
	"net"
	"time"
)

// Builder specifes the interface all delimiter and parser must implement.
type Builder interface {
	NewDelimiter() Delimiter
	NewParser() Parser
}

// Collector specifies the interface all network collectors must implement.
type Collector interface {
	Start(chan<- *Event) error
	Addr() net.Addr
}

// Delimiter splits multiple input requests into single requests
type Delimiter interface {
	Push(b byte) (string, bool)
	Vestige() (string, bool)
}

// Parser parses the imput request to the correct format
type Parser interface {
	Parse(log string) Message
}

// Message represents the input request, but in the correct format
type Message interface {
	GetTimestamp() string
}

// Event is a log message, with a reception timestamp and sequence number.
type Event struct {
	Text          string    // Delimited log line
	Parsed        Message   // If non-nil, contains parsed fields
	ReceptionTime time.Time // Time log line was received
	Sequence      int64     // Provides order of reception
	SourceIP      string    // Sender's IP address

	referenceTime time.Time // Memomized reference time
}

// NewEvent returns a new Event.
func NewEvent() *Event {
	return &Event{}
}

// ReferenceTime returns the reference time of an event.
func (e *Event) ReferenceTime() time.Time {
	if e.referenceTime.IsZero() {
		if e.Parsed == nil {
			e.referenceTime = e.ReceptionTime
		} else if refTime, err := time.Parse(time.RFC3339, e.Parsed.GetTimestamp()); err != nil {
			e.referenceTime = e.ReceptionTime
		} else {
			e.referenceTime = refTime
		}

	}
	return e.referenceTime
}
