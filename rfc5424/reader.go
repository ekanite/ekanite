package rfc5424

import (
	"bufio"
	"io"
	"strings"
)

type Reader struct {
	r      *bufio.Reader
	buf    []byte
	priLen int
	state  fsmState
}

func NewReader(r io.Reader) *Reader {
	return &Reader{
		r: bufio.NewReader(r),
	}
}

func (r *Reader) ReadLine() (string, error) {
	for {
		b, err := r.r.ReadByte()
		if err != nil {
			return r.line(false), err
		}
		r.buf = append(r.buf, b)

		switch r.state {
		case newline:
			if b == '\n' {
				r.state = priStart
			}
		case priStart:
			if b == '<' {
				r.state = priVal0
			}
		case priVal0:
			if isDigit(b) {
				r.priLen = 1
				r.state = priVal1
			} else {
				// Invalid, reset parser.
				r.state = priStart
			}
		case priVal1:
			if isDigit(b) {
				r.priLen = 2
				r.state = priVal2
			} else if b == '>' {
				r.state = version
			}
		case priVal2:
			if isDigit(b) {
				r.priLen = 3
				r.state = priVal3
			} else if b == '>' {
				r.state = version
			}
		case priVal3:
			if isDigit(b) {
				r.priLen = 4
				r.state = priEnd
			} else if b == '>' {
				r.state = version
			}
		case priEnd:
			if b == '>' {
				r.state = version
			} else {
				// Invalid, reset parser.
				r.state = priStart
			}
		case version:
			if isDigit(b) {
				r.state = postVersion
			} else {
				// Invalid, reset parser.
				r.state = priStart
			}
		case postVersion:
			if b == ' ' {
				return r.line(true), nil
			} else {
				// Invalid, reset parser.
				r.state = priStart
			}
		}
	}
}

func (r *Reader) line(stripDelim bool) string {
	r.state = priStart

	var line string
	if stripDelim {
		line = string(r.buf[:len(r.buf)-r.priLen-4])
	} else {
		line = string(r.buf[:len(r.buf)])
	}
	r.buf = r.buf[len(r.buf)-r.priLen-4:]

	r.priLen = 0
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
