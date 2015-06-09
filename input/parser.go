package input

import (
	"regexp"
	"strconv"
)

// A RFC5424Parser parses Syslog messages.
type RFC5424Parser struct {
	regex *regexp.Regexp
}

// RFC5424Message represents a fully parsed Syslog RFC5424 message.
type RFC5424Message struct {
	Priority  int    `json:"priority"`
	Version   int    `json:"version"`
	Timestamp string `json:"timestamp"`
	Host      string `json:"host"`
	App       string `json:"app"`
	Pid       int    `json:"pid"`
	MsgId     string `json:"msgid"`
	Message   string `json:"message"`
}

type ApacheCommonFormat struct {
	URL        string
	Referer    string
	Method     string
	StatusCode int
}

// Returns an initialized RFC5424Parser.
func NewRFC5424Parser() *RFC5424Parser {
	leading := `(?s)`
	pri := `<([0-9]{1,3})>`
	ver := `([0-9])`
	ts := `([^ ]+)`
	host := `([^ ]+)`
	app := `([^ ]+)`
	pid := `(-|[0-9]{1,5})`
	id := `([\w-]+)`
	msg := `(.+$)`

	p := &RFC5424Parser{}
	r := regexp.MustCompile(leading + pri + ver + `\s` + ts + `\s` + host + `\s` + app + `\s` + pid + `\s` + id + `\s` + msg)
	p.regex = r

	return p
}

// Parse takes a raw message and returns a parsed message. If no match,
// nil is returned.
func (p *RFC5424Parser) Parse(raw string) *RFC5424Message {
	m := p.regex.FindStringSubmatch(raw)
	if m == nil || len(m) != 9 {
		stats.Add("unparsed", 1)
		return nil
	}
	stats.Add("parsed", 1)

	// Errors are ignored, because the regex shouldn't match if the
	// following ain't numbers.
	pri, _ := strconv.Atoi(m[1])
	ver, _ := strconv.Atoi(m[2])

	var pid int
	if m[6] != "-" {
		pid, _ = strconv.Atoi(m[6])
	}

	return &RFC5424Message{pri, ver, m[3], m[4], m[5], pid, m[7], m[8]}
}
