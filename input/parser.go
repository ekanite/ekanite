package input

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/jeromer/syslogparser"
	"github.com/jeromer/syslogparser/rfc3164"
	"github.com/jeromer/syslogparser/rfc5424"
)

var (
	FORMATS_BY_STANDARD = []string{"rfc3164", "rfc5424", "ecma404"}
	FORMATS_BY_NAME     = []string{"syslog-bsd", "syslog", "json"}
)

type Input struct {
	Format string
	Parsed map[string]interface{}
}

func NewParser(f string) Input {
	// remove uppercase letters and leading/trailing white spaces
	f = strings.TrimSpace(strings.ToLower(f))
	// try to return if given format matches one of the supported formats using its common name
	for i, v := range FORMATS_BY_NAME {
		if f == v {
			return Input{Format: FORMATS_BY_STANDARD[i]}
		}
	}
	// try to return if given format matches one of the supported formats using the name of its standard
	for _, v := range FORMATS_BY_STANDARD {
		if f == v {
			return Input{Format: v}
		}
	}
	// returns using "ecma404" as the default input format
	stats.Add("invalid-input-format", 1)
	return Input{Format: "ecma404"}
}

func (i Input) Parse(b []byte) (bool, map[string]interface{}) {
	if i.Format != "ecma404" {
		return parseSyslog(i.Format, b)
	} else {
		return parseJson(b)
	}
}

func parseSyslog(f string, b []byte) (bool, map[string]interface{}) {
	if f == "rfc3164" {
		p := rfc3164.NewParser(b)
		if err := p.Parse(); err != nil {
			stats.Add("rfc3164Unparsed", 1)
			return false, nil
		}
		stats.Add("rfc3164Parsed", 1)
		return true, mapSyslog(p.Dump())
	} else {
		p := rfc5424.NewParser(b)
		if err := p.Parse(); err != nil {
			stats.Add("rfc5424Unparsed", 1)
			return false, nil
		}
		stats.Add("rfc5424Parsed", 1)
		return true, mapSyslog(p.Dump())
	}
}

func mapSyslog(l syslogparser.LogParts) map[string]interface{} {
	r := make(map[string]interface{})
	for k, v := range l {
		r[k] = v
	}
	return r
}

func parseJson(b []byte) (bool, map[string]interface{}) {
	r := make(map[string]interface{})
	if err := json.Unmarshal(b, &r); err != nil {
		stats.Add("ecma404Unparsed", 1)
		return false, nil
	}
	_, ok := r["timestamp"]
	if !ok {
		stats.Add("ecma404MissingTimestamp", 1)
		return false, nil
	}
	t, err := time.Parse(time.RFC3339, r["timestamp"].(string))
	if err != nil {
		stats.Add("ecma404UnparsedTimestamp", 1)
		return false, nil
	}
	r["timestamp"] = t
	stats.Add("ecma404Parsed", 1)
	return false, r
}
