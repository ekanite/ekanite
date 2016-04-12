package rfc5424

import (
	"expvar"
	"regexp"
	"strconv"

	"github.com/ekanite/ekanite/input/types"
)

var stats = expvar.NewMap("rfc404")

// A Parser parses Syslog messages.
type Parser struct {
	regex *regexp.Regexp
}

// Message represents a fully parsed Syslog message.
type Message struct {
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

// Returns an initialized Parser.
func NewParser() *Parser {
	leading := `(?s)`
	pri := `<([0-9]{1,3})>`
	ver := `([0-9])`
	ts := `([^ ]+)`
	host := `([^ ]+)`
	app := `([^ ]+)`
	pid := `(-|[0-9]{1,5})`
	id := `([\w-]+)`
	msg := `(.+$)`

	p := &Parser{}
	r := regexp.MustCompile(leading + pri + ver + `\s` + ts + `\s` + host + `\s` + app + `\s` + pid + `\s` + id + `\s` + msg)
	p.regex = r

	return p
}

// Parse takes a raw message and returns a parsed message. If no match,
// nil is returned.
func (p *Parser) Parse(raw string) types.Message {
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

	return Message{pri, ver, m[3], m[4], m[5], pid, m[7], m[8]}

}

func (m Message) GetTimestamp() string {
	return m.Timestamp
}
