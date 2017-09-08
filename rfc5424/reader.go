// Package rfc5424 is a state-machine parser of RFC5424-formatted log lines.
package rfc5424

import (
	"io"
	"regexp"
	"strconv"
)

// Reader wraps an io.Reader object and returns valid RFC5424 log messages.
type Reader struct {
	r io.Reader
	d *Delimiter
	p *Parser
}

// NewReader returns a Reader wrapping an io.Reader.
func NewReader(rdr io.Reader) *Reader {
	r := &Reader{
		r: rdr,
		d: NewDelimiter(rdr),
		p: NewParser(),
	}

	return r
}

// ReadLine returns the next valid RFC5424 log message.
func (r *Reader) ReadLine() (string, error) {
	return "", nil
}

// Parser parses an RFC5424-compliant log message.
type Parser struct {
	matcher *regexp.Regexp
}

// NewParser returns an instance of a Parser.
func NewParser() *Parser {
	p := &Parser{}
	p.compileMatcher()
	return p
}

// compileMatcher compiles the regex for RFC5424 log messages. If it fails
// to compile, it panics.
func (p *Parser) compileMatcher() {
	leading := `(?s)`
	pri := `<([0-9]{1,3})>`
	ver := `([0-9])`
	ts := `([^ ]+)`
	host := `([^ ]+)`
	app := `([^ ]+)`
	pid := `(-|[0-9]{1,5})`
	id := `([\w-]+)`
	msg := `(.+$)`
	p.matcher = regexp.MustCompile(leading + pri + ver + `\s` + ts + `\s` + host + `\s` + app + `\s` + pid + `\s` + id + `\s` + msg)
}

// parse attempts to parse the given byte slice.
func (p *Parser) parse(raw []byte, result *map[string]interface{}) {
	m := p.matcher.FindStringSubmatch(string(raw))
	if m == nil || len(m) != 9 {
		return
	}
	pri, _ := strconv.Atoi(m[1])
	ver, _ := strconv.Atoi(m[2])
	var pid int
	if m[6] != "-" {
		pid, _ = strconv.Atoi(m[6])
	}
	*result = map[string]interface{}{
		"priority":   pri,
		"version":    ver,
		"timestamp":  m[3],
		"host":       m[4],
		"app":        m[5],
		"pid":        pid,
		"message_id": m[7],
		"message":    m[8],
	}
}
