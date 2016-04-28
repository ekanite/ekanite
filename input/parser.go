package input

import (
	"strings"
)

var (
	fmtsByStandard = []string{"rfc5424"}
	fmtsByName     = []string{"syslog"}
)

// HasFmt returns if the given format matches one of the possible formats.
func HasFmt(format string) bool {
	for _, f := range append(fmtsByStandard, fmtsByName...) {
		if f == format {
			return true
		}
	}
	return false
}

// A Parser parses the raw input as a map with a timestamp field.
type Parser struct {
	fmt     string
	Raw     []byte
	Result  map[string]interface{}
	rfc5424 *Rfc5424
}

// NewParser returns a new Parser instance.
func NewParser(f string) *Parser {
	p := &Parser{}
	p.detectFmt(strings.TrimSpace(strings.ToLower(f)))
	p.newRfc5424Parser()
	return p
}

// Reads the given format and detects its internal name.
func (p *Parser) detectFmt(f string) {
	for i, v := range fmtsByName {
		if f == v {
			p.fmt = fmtsByStandard[i]
			return
		}
	}
	for _, v := range fmtsByStandard {
		if f == v {
			p.fmt = v
			return
		}
	}
	stats.Add("invalidParserFormat", 1)
	p.fmt = fmtsByStandard[0]
	return
}

// Parse the given byte slice.
func (p *Parser) Parse(b []byte) bool {
	p.Result = map[string]interface{}{}
	p.Raw = b
	p.rfc5424.parse(p.Raw, &p.Result)
	if len(p.Result) == 0 {
		return false
	}
	return true
}
