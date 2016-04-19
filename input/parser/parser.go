package parser

import (
	"errors"
	"expvar"
	"strings"
)

var (
	err            error
	stats          = expvar.NewMap("parser")
	fmtsByStandard = []string{"rfc5424", "ecma404"}
	fmtsByName     = []string{"syslog", "json"}
	unxConvErr     = errors.New("unix timestamp conversion error")
)

func IsFmt(format string) bool {
	for _, f := range append(fmtsByStandard, fmtsByName...) {
		if f == format {
			return true
		}
	}
	return false
}

type Parser struct {
	Fmt     string
	Raw     []byte
	Result  map[string]interface{}
	rfc5424 *Rfc5424
	ecma404 *Ecma404
	rfc3339 *Rfc3339
}

func NewParser(f string) *Parser {
	p := &Parser{}
	p.detectFmt(strings.TrimSpace(strings.ToLower(f)))
	p.newEcma404Parser()
	p.newRfc5424Parser()
	p.newRfc3339Parser()
	return p
}

func (p *Parser) detectFmt(f string) {
	for i, v := range fmtsByName {
		if f == v {
			p.Fmt = fmtsByStandard[i]
			return
		}
	}
	for _, v := range fmtsByStandard {
		if f == v {
			p.Fmt = v
			return
		}
	}
	stats.Add("invalidParserFormat", 1)
	p.Fmt = "ecma404"
	return
}

func (p *Parser) Parse(b []byte) bool {
	p.Result = map[string]interface{}{}
	p.Raw = b
	if p.Fmt == "ecma404" {
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
