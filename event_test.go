package ekanite

import (
	"testing"
	"time"

	"github.com/otoolep/ekanite/input"
)

// TestEvent_New tests simple Event creation.
func TestEvent_New(t *testing.T) {
	ev := NewEvent()
	if ev == nil {
		t.Fatalf("failed to create new event")
	}
}

// TestEvent_NewUnparsed tests generation of an Event with unparsed data.
func TestEvent_NewUnparsed(t *testing.T) {
	text := "this is a log line"
	now := time.Now()
	ev := &Event{
		&input.Event{
			Text:          text,
			ReceptionTime: now,
			Sequence:      1234,
			SourceIP:      "192.1.2.3",
		},
	}

	if string(ev.Source()) != ev.Text {
		t.Errorf("wrong Event source return, exp: %s, got %s", text, string(ev.Source()))
	}
	if ev.ReferenceTime() != now {
		t.Errorf("wrong Event reference time, exp: %s, got %s", now, ev.ReferenceTime())
	}
}
