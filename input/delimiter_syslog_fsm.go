package input

import (
	"strings"
)

// SyslogDelimiterFSM detects when Syslog lines start. It uses a state
// machine for efficient detection.
type SyslogDelimiterFSM struct {
	buffer       []byte
	state        fsmState
	firstMatched bool
}

// NewSyslogDelimiterFSM returns an initialized SyslogDelimiterFSM.
func NewSyslogDelimiterFSM(maxSize int) *SyslogDelimiterFSM {
	return &SyslogDelimiterFSM{}
}

// Push a byte into the SyslogDelimiterFSM. If the byte results in a
// a new Syslog message, it'll be flagged via the bool.
func (s *SyslogDelimiterFSM) Push(b byte) (string, bool) {
	if !s.firstMatched && b != '<' {
		return "", false
	}
	s.buffer = append(s.buffer, b)

	switch s.state {
	case priStart:
		if b == '<' {
			s.state = priVal0
		}
		return "", false
	case priVal0:
		if isDigit(b) {
			s.state = priVal1
		} else {
			// Invalid, reset parser.
			s.state = priStart
		}
		return "", false
	case priVal1:
		if isDigit(b) {
			s.state = priVal2
		} else if b == '>' {
			s.state = version
		}
		return "", false
	case priVal2:
		if isDigit(b) {
			s.state = priVal3
		} else if b == '>' {
			s.state = version
		}
		return "", false
	case priVal3:
		if isDigit(b) {
			s.state = priEnd
		} else if b == '>' {
			s.state = version
		}
		return "", false
	case priEnd:
		if b == '>' {
			s.state = version
		} else {
			// Invalid, reset parser.
			s.state = priStart
		}
		return "", false
	case version:
		if isDigit(b) {
			s.state = postVersion
		} else {
			// Invalid, reset parser.
			s.state = priStart
		}
		return "", false
	case postVersion:
		if b == ' ' {
			s.state = newline_r
		} else {
			// Invalid, reset parser.
			s.state = priStart
		}
		return "", false
	case newline_r:
		if b == '\n' {
			return s.line(), true
		} else if b == '\r' {
			s.state = newline_n
		} else {
			// Invalid, reset parser.
			s.state = priStart
		}
		return "", false
	case newline_n:
		if b == '\n' {
			return s.line(), true
		}
		s.state = priStart
		return "", false
	}

	return "", false
}

func (s *SyslogDelimiterFSM) line() string {
	s.firstMatched = true
	s.state = priStart

	line := string(s.buffer[0 : len(s.buffer)-1])
	s.buffer = nil

	return strings.TrimRight(line, "\r")
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
	newline_r
	newline_n
)
