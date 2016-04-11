package input

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/ekanite/ekanite/input/syslog"
	"github.com/ekanite/ekanite/input/syslog/rfc3164"
	"github.com/ekanite/ekanite/input/syslog/rfc5424"
)

var (
	FORMATS_BY_STANDARD = []string{"rfc3164", "rfc5424", "ecma404"}
	FORMATS_BY_NAME     = []string{"syslog-bsd", "syslog", "json"}
)

type Input struct {
	Format   string
	Parsed   map[string]interface{}
	Unparsed []byte
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

func (i *Input) Parse(b []byte) (bool, map[string]interface{}) {

	var ok bool
	i.Unparsed = b

	if i.Format != "ecma404" {

		ok = i.parseSyslog()

	} else {

		ok = i.parseJson()
	}

	return ok, i.Parsed

}

func (i *Input) parseSyslog() bool {

	if i.Format == "rfc3164" {

		p := rfc3164.NewParser(i.Unparsed)

		if err := p.Parse(); err != nil {

			stats.Add("rfc3164Unparsed", 1)
			return false

		}

		stats.Add("rfc3164Parsed", 1)
		i.mapSyslog(p.Dump())
		return true

	} else {

		p := rfc5424.NewParser(i.Unparsed)

		if err := p.Parse(); err != nil {

			stats.Add("rfc5424Unparsed", 1)
			return false

		}

		stats.Add("rfc5424Parsed", 1)
		i.mapSyslog(p.Dump())
		return true

	}

}

func (i *Input) mapSyslog(l syslogparser.LogParts) {

	i.Parsed = map[string]interface{}{}

	for k, v := range l {

		i.Parsed[k] = v

	}

}

func (i *Input) parseJson() bool {

	i.Parsed = map[string]interface{}{}

	if err := json.Unmarshal(i.Unparsed, &i.Parsed); err != nil {

		stats.Add("ecma404Unparsed", 1)
		return false

	}

	_, ok := i.Parsed["timestamp"]

	if !ok {

		stats.Add("ecma404MissingTimestamp", 1)
		return false

	}

	t, err := time.Parse(time.RFC3339, i.Parsed["timestamp"].(string))

	if err != nil {

		stats.Add("ecma404UnparsedTimestamp", 1)
		return false

	}

	i.Parsed["timestamp"] = t
	stats.Add("ecma404Parsed", 1)
	return false

}
