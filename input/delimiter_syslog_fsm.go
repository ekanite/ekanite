package input

import (
	"fmt"
	"strings"
)

// SyslogDelimiterFSM detects when Syslog lines start. It uses a state
// machine for efficient detection.
type SyslogDelimiterFSM struct {
	buffer       []byte
	state        fsmState
	firstMatched bool

	priLen int
}

// NewSyslogDelimiterFSM returns an initialized SyslogDelimiterFSM.
func NewSyslogDelimiterFSM(maxSize int) *SyslogDelimiterFSM {
	return &SyslogDelimiterFSM{}
}

// Push a byte into the SyslogDelimiterFSM. If the byte results in a
// a new Syslog message, it'll be flagged via the bool.
func (s *SyslogDelimiterFSM) Push(b byte) (string, bool) {
	s.buffer = append(s.buffer, b)
	fmt.Println(string(s.buffer), s.state)

	switch s.state {
	case priStart:
		if b == '<' {
			s.state = priVal0
		}
	case priVal0:
		if isDigit(b) {
			s.state = priVal1
		} else {
			// Invalid, reset parser.
			s.state = priStart
		}
	case priVal1:
		if isDigit(b) {
			s.priLen = 1
			s.state = priVal2
		} else if b == '>' {
			s.state = version
		}
	case priVal2:
		if isDigit(b) {
			s.priLen = 2
			s.state = priVal3
		} else if b == '>' {
			s.state = version
		}
	case priVal3:
		if isDigit(b) {
			s.priLen = 3
			s.state = priEnd
		} else if b == '>' {
			s.state = version
		}
	case priEnd:
		if b == '>' {
			s.state = version
		} else {
			// Invalid, reset parser.
			s.state = priStart
		}
	case version:
		if isDigit(b) {
			s.state = postVersion
		} else {
			// Invalid, reset parser.
			s.state = priStart
		}
	case postVersion:
		if b == ' ' {
			s.state = newline
		} else {
			// Invalid, reset parser.
			s.state = priStart
		}
	case newline:
		if b == '\n' {
			return s.line()
		}
	}

	return "", false
}

// Vestige returns the bytes which have been pushed to SyslogDelimiter, since
// the last Syslog message was returned, but only if the buffer appears
// to be a valid syslog message.
func (s *SyslogDelimiterFSM) Vestige() (string, bool) {
	return "", false
}

func (s *SyslogDelimiterFSM) line() (string, bool) {
	if !s.firstMatched {
		// Actually, this is the first delimiter we've hit. Just hang onto it,
		// drop the characters preceding, and return to parsing.
		//s.buffer = s.buffer[:len(s.buffer)-s.priLen-4]
		fmt.Println("###1:", string(s.buffer), "###")
		s.firstMatched = true
		s.priLen = 0
		return "", false
	}

	// Return everything in the buffer excluding the delimiter.
	s.state = priStart

	line := string(s.buffer[:len(s.buffer)-1-s.priLen-4])
	s.buffer = s.buffer[len(s.buffer)-1-s.priLen-4:]

	s.priLen = 0
	return strings.TrimRight(line, "\r"), true
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

// fsmState represents the state of the parser and what it is expecting next.
type fsmState int

const (
	priStart fsmState = iota
	priEnd
	priVal0
	priVal1
	priVal2
	priVal3
	version
	postVersion
	newline
)
