package input

import (
	"regexp"
	"strconv"
)

// RFC5424 represents a parser for RFC5424-compliant log messages
type RFC5424 struct {
	matcher *regexp.Regexp
}

func (p *Parser) newRFC5424Parser() {
	p.rfc5424 = &RFC5424{}
	p.rfc5424.compileMatcher()
}

func (s *RFC5424) compileMatcher() {
	leading := `(?s)`
	pri := `<([0-9]{1,3})>`
	ver := `([0-9])`
	ts := `([^ ]+)`
	host := `([^ ]+)`
	app := `([^ ]+)`
	pid := `(-|[0-9]{1,5})`
	id := `([\w-]+)`
	msg := `(.+$)`
	s.matcher = regexp.MustCompile(leading + pri + ver + `\s` + ts + `\s` + host + `\s` + app + `\s` + pid + `\s` + id + `\s` + msg)
}

func (s *RFC5424) parse(raw []byte, result *map[string]interface{}) {
	m := s.matcher.FindStringSubmatch(string(raw))
	if m == nil || len(m) != 9 {
		stats.Add("rfc5424Unparsed", 1)
		return
	}
	stats.Add("rfc5424Parsed", 1)
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
