package ecma404

import (
	"bytes"
	"strings"
)

// A Delimiter detects when Json lines start.
type Delimiter struct {
	buffer *bytes.Buffer
}

// NewDelimiter returns an initialized Delimiter.
func NewDelimiter() *Delimiter {
	self := &Delimiter{}
	self.buffer = bytes.NewBuffer(nil)
	return self
}

// Push a byte into the Delimiter. If the byte results in a
// a new Json message, it'll be flagged via the bool.
func (self *Delimiter) Push(b byte) (string, bool) {
	self.buffer.WriteByte(b)
	return self.buffer.String(), false
}

// Vestige returns the bytes which have been pushed to Delimiter, since
// the last Json message was returned, but only if the buffer appears
// to be a valid Json message.
func (self *Delimiter) Vestige() (string, bool) {
	dispatch := strings.TrimRight(self.buffer.String(), "\r\n")
	self.buffer.Reset()
	return dispatch, true
}
