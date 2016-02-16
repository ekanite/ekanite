package input

import (
	"regexp"
	"strings"
)

const (
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

// A Delimiter detects when Syslog lines start.
type Delimiter struct {
	buffer []byte
	regex  *regexp.Regexp
}

// NewDelimiter returns an initialized Delimiter.
func NewDelimiter(maxSize int) *Delimiter {
	self := &Delimiter{}
	self.buffer = make([]byte, 0, maxSize)
	self.regex = startRegex
	return self
}

// Push a byte into the Delimiter. If the byte results in a
// a new Syslog message, it'll be flagged via the bool.
func (self *Delimiter) Push(b byte) (string, bool) {
	self.buffer = append(self.buffer, b)
	delimiter := self.regex.FindIndex(self.buffer)
	if delimiter == nil {
		return "", false
	}

	if self.regex == startRegex {
		// First match -- switch to the regex for embedded lines, and
		// drop any leading characters.
		self.buffer = self.buffer[delimiter[0]:]
		self.regex = runRegex
		return "", false
	}

	dispatch := strings.TrimRight(string(self.buffer[:delimiter[0]]), "\r")
	self.buffer = self.buffer[delimiter[0]+1:]
	return dispatch, true
}

// Vestige returns the bytes which have been pushed to Delimiter, since
// the last Syslog message was returned, but only if the buffer appears
// to be a valid syslog message.
func (self *Delimiter) Vestige() (string, bool) {
	delimiter := syslogRegex.FindIndex(self.buffer)
	if delimiter == nil {
		self.buffer = nil
		return "", false
	}
	dispatch := strings.TrimRight(string(self.buffer), "\r\n")
	self.buffer = nil
	return dispatch, true
}
