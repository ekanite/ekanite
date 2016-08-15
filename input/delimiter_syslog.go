package input

import (
	"regexp"
	"strings"
)

const (
	// SYSLOG_DELIMITER indicates the start of a syslog line
	SYSLOG_DELIMITER = `<[0-9]{1,3}>[0-9]\s`
)

var syslogRegex *regexp.Regexp
var startRegex *regexp.Regexp
var runRegex *regexp.Regexp

func init() {
	syslogRegex = regexp.MustCompile(SYSLOG_DELIMITER)
	startRegex = regexp.MustCompile(SYSLOG_DELIMITER + `$`)
	runRegex = regexp.MustCompile(`\n` + SYSLOG_DELIMITER)
}

// A SyslogDelimiter detects when Syslog lines start.
type SyslogDelimiter struct {
	buffer []byte
	regex  *regexp.Regexp
}

// NewSyslogDelimiter returns an initialized SyslogDelimiter.
func NewSyslogDelimiter(maxSize int) *SyslogDelimiter {
	s := &SyslogDelimiter{}
	s.buffer = make([]byte, 0, maxSize)
	s.regex = startRegex
	return s
}

// Push a byte into the SyslogDelimiter. If the byte results in a
// a new Syslog message, it'll be flagged via the bool.
func (s *SyslogDelimiter) Push(b byte) (string, bool) {
	s.buffer = append(s.buffer, b)
	delimiter := s.regex.FindIndex(s.buffer)
	if delimiter == nil {
		return "", false
	}

	if s.regex == startRegex {
		// First match -- switch to the regex for embedded lines, and
		// drop any leading characters.
		s.buffer = s.buffer[delimiter[0]:]
		s.regex = runRegex
		return "", false
	}

	dispatch := strings.TrimRight(string(s.buffer[:delimiter[0]]), "\r")
	s.buffer = s.buffer[delimiter[0]+1:]
	return dispatch, true
}

// Vestige returns the bytes which have been pushed to SyslogDelimiter, since
// the last Syslog message was returned, but only if the buffer appears
// to be a valid syslog message.
func (s *SyslogDelimiter) Vestige() (string, bool) {
	delimiter := syslogRegex.FindIndex(s.buffer)
	if delimiter == nil {
		s.buffer = nil
		return "", false
	}
	dispatch := strings.TrimRight(string(s.buffer), "\r\n")
	s.buffer = nil
	return dispatch, true
}
