package rfc5424

import (
	"bufio"
	"io"
	"strings"
)

// Delimiter splits incoming data on RFC5424 headers.
type Delimiter struct {
	r      *bufio.Reader
	buf    []byte
	priLen int
	state  fsmState
}

// NewDelimiter returns an instance of a Delimiter.
func NewDelimiter(r io.Reader) *Delimiter {
	return &Delimiter{
		r: bufio.NewReader(r),
	}
}

// ReadLine() returns a line beginning with an RFC5424 header, and
// terminated before the start of the next RFC5424 header.
func (d *Delimiter) ReadLine() (string, error) {
	for {
		b, err := d.r.ReadByte()
		if err != nil {
			return d.line(false), err
		}
		d.buf = append(d.buf, b)

		switch d.state {
		case newline:
			if b == '\n' {
				d.state = priStart
			}
		case priStart:
			if b == '<' {
				d.state = priVal0
			}
		case priVal0:
			if isDigit(b) {
				d.priLen = 1
				d.state = priVal1
			} else {
				// Invalid, reset parser.
				d.state = priStart
			}
		case priVal1:
			if isDigit(b) {
				d.priLen = 2
				d.state = priVal2
			} else if b == '>' {
				d.state = version
			}
		case priVal2:
			if isDigit(b) {
				d.priLen = 3
				d.state = priVal3
			} else if b == '>' {
				d.state = version
			}
		case priVal3:
			if isDigit(b) {
				d.priLen = 4
				d.state = priEnd
			} else if b == '>' {
				d.state = version
			}
		case priEnd:
			if b == '>' {
				d.state = version
			} else {
				// Invalid, reset parser.
				d.state = priStart
			}
		case version:
			if isDigit(b) {
				d.state = postVersion
			} else {
				// Invalid, reset parser.
				d.state = priStart
			}
		case postVersion:
			if b == ' ' {
				return d.line(true), nil
			} else {
				// Invalid, reset parser.
				d.state = priStart
			}
		}
	}
}

func (d *Delimiter) line(stripDelim bool) string {
	d.state = priStart

	var line string
	if stripDelim {
		line = string(d.buf[:len(d.buf)-d.priLen-4])
	} else {
		line = string(d.buf[:len(d.buf)])
	}
	d.buf = d.buf[len(line):]

	d.priLen = 0
	return strings.TrimRight(line, "\r\n")
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

// fsmState represents the state of the parser and what it is expecting next.
type fsmState int

const (
	newline fsmState = iota
	priStart
	priEnd
	priVal0
	priVal1
	priVal2
	priVal3
	version
	postVersion
)
