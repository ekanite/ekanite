package parser

import (
	"regexp"
	"strconv"
)

type RFC5424 struct {
	matcher *regexp.Regexp
}

var rfc5424Stats = func(key string, delta int64){}

func (p *RFC5424) Stats(callback func(key string, delta int64)) {
	rfc5424Stats = callback
}

func (p *RFC5424) CompileMatcher() {
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

func (p *RFC5424) Parse(raw []byte, result *map[string]interface{}) {
	m := p.matcher.FindStringSubmatch(string(raw))
	if m == nil || len(m) != 9 {
		rfc5424Stats("rfc5424Unparsed", 1)
		return
	}
	rfc5424Stats("rfc5424Parsed", 1)
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
