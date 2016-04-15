package delimiter

import (
	"bytes"
	"errors"
	"strconv"
)

const (
	LenBuffEnd = ":"
	ValBuffEnd = ";"
	NoResult   = false
)

var (
	err error
)

// A Delimiter detects when message lines start.
type Delimiter struct {
	Result      string
	lenBuff     bytes.Buffer
	valBuff     bytes.Buffer
	valBuffLen  int
	valBuffMode bool
	ignoreMode  bool
	brokenMode  bool
}

// NewDelimiter returns an initialized Delimiter.
func NewDelimiter() *Delimiter {
	return &Delimiter{
		lenBuff: *bytes.NewBuffer([]byte{}),
		valBuff: *bytes.NewBuffer([]byte{}),
	}
}

// Returns rather a new result is available
// and the first occurring error (if any occurred).
func (d *Delimiter) Push(b byte) (bool, error) {
	if d.brokenMode {
		return NoResult, errors.New("broken")
	}
	return d.processByte(b)
}

// Restes the instance close to its initial state.
func (d *Delimiter) Reset() {
	d.useLenBuff()
}

// Checks rather a byte must be processed as "length byte"
// or as "value byte".
func (d *Delimiter) processByte(b byte) (bool, error) {
	if d.valBuffMode {
		return d.processValByte(b)
	}
	return d.processLenByte(b)
}

// Writes the passed byte to the "length buffer",
// unless the passed byte is the end of the "length buffer".
func (d *Delimiter) processLenByte(b byte) (bool, error) {
	if b == LenBuffEnd[0] {
		return NoResult, d.useValBuff()
	}
	if err = d.lenBuff.WriteByte(b); err != nil {
		d.brokenMode = true
		return NoResult, errors.New("length-buffer-incomplete")
	}
	return NoResult, nil
}

// Writes the passed byte to the "value buffer",
// unless the "value buffer length" is equal to 0.
func (d *Delimiter) processValByte(b byte) (bool, error) {
	if d.valBuffLen == 0 {
		d.useLenBuff()
		return true, nil
	}
	d.valBuffLen--
	if d.ignoreMode {
		return NoResult, nil
	}
	// If an error occurs, while writing to the buffer,
	// the current "value buffer" gets ignored.
	if err = d.valBuff.WriteByte(b); err != nil {
		d.ignoreMode = true
		return NoResult, errors.New("value-buffer-incomplete")
	}
	return NoResult, nil
}

// Overwrites the old result and resets values.
func (d *Delimiter) useLenBuff() {
	if d.ignoreMode {
		d.Result = ""
		d.ignoreMode = false
	} else {
		d.Result = d.valBuff.String()
	}
	d.valBuff.Reset()
	d.valBuffMode = false
}

// Converts the "length buffer" value to an integer,
// representing the "value buffer length" and resets values.
func (d *Delimiter) useValBuff() error {
	if d.valBuffLen, err = strconv.Atoi(d.lenBuff.String()); err != nil {
		d.brokenMode = true
		return errors.New("length-buffer-conversion-error")
	}
	d.lenBuff.Reset()
	d.valBuffMode = true
	return nil
}
