package input

import (
	"fmt"
	"strings"
)

var (
	fmtsByStandard = []string{"rfc5424", "ecma404"}
	fmtsByName     = []string{"syslog", "json"}
)

// ValidFormat returns if the given format matches one of the possible formats.
func ValidFormat(format string) bool {
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
	ecma404 *Ecma404
	rfc3339 *Rfc3339
}

// NewParser returns a new Parser instance.
func NewParser(f string) (*Parser, error) {
	if !ValidFormat(f) {
		return nil, fmt.Errorf("%s is not a valid format", f)
	}

	p := &Parser{}
	p.detectFmt(strings.TrimSpace(strings.ToLower(f)))
	p.newRfc5424Parser()
	p.newEcma404Parser()
	p.newRfc3339Parser()
	return p, nil
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
	if p.fmt == "ecma404" {
		p.ecma404.parse(p.Raw, &p.Result)
		if _, ok := p.Result["timestamp"]; !ok {
			return false
		}
		p.Result["timestamp"], err = p.rfc3339.parse(p.Result["timestamp"].(string))
		if err != nil {
			return false
		}
	} else {
		p.rfc5424.parse(p.Raw, &p.Result)
	}
	if len(p.Result) == 0 {
		return false
	}
	return true
}
