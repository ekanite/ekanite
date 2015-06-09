package input

import "time"

// Event is a log message, with a reception timestamp and sequence number.
type Event struct {
	Text          string          // Delimited log line
	Parsed        *RFC5424Message // If non-nil, contains parsed fields
	ReceptionTime time.Time       // Time log line was received
	Sequence      int64           // Provides order of reception
	SourceIP      string          // Sender's IP address

	referenceTime time.Time // Memomized reference time
}

// NewEvent retuns a new Event.
func NewEvent() *Event {
	return &Event{}
}

// ReferenceTime returns the reference time of an event.
func (e *Event) ReferenceTime() time.Time {
	if e.referenceTime.IsZero() {
		if e.Parsed == nil {
			e.referenceTime = e.ReceptionTime
		} else if refTime, err := time.Parse(time.RFC3339, e.Parsed.Timestamp); err != nil {
			e.referenceTime = e.ReceptionTime
		} else {
			e.referenceTime = refTime
		}

	}
	return e.referenceTime
}
