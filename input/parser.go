package input

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	ok                  bool
	FORMATS_BY_STANDARD = []string{"rfc5424", "ecma404"}
	FORMATS_BY_NAME     = []string{"syslog", "json"}
)

type Input struct {
	Format   string
	Parsed   map[string]interface{}
	Unparsed []byte
	Matcher  *regexp.Regexp
}

type Timestamp struct {
	Unparsed []string
	Formated []int64
	Parsed   string
}

func NewParser(f string) Input {

	i := Input{}
	i.Format = i.findFormat(strings.TrimSpace(strings.ToLower(f)))

	if i.Format == "rfc5424" {

		leading := `(?s)`
		pri := `<([0-9]{1,3})>`
		ver := `([0-9])`
		ts := `([^ ]+)`
		host := `([^ ]+)`
		app := `([^ ]+)`
		pid := `(-|[0-9]{1,5})`
		id := `([\w-]+)`
		msg := `(.+$)`

		i.Matcher = regexp.MustCompile(leading + pri + ver + `\s` + ts + `\s` + host + `\s` + app + `\s` + pid + `\s` + id + `\s` + msg)

	}

	return i

}

func (i *Input) findFormat(f string) string {

	// try to return if given format matches one of the supported formats using its common name
	for i, v := range FORMATS_BY_NAME {

		if f == v {

			return FORMATS_BY_STANDARD[i]

		}

	}

	// try to return if given format matches one of the supported formats using the name of its standard
	for _, v := range FORMATS_BY_STANDARD {

		if f == v {

			return v

		}

	}

	// returns using "ecma404" as the default input format
	stats.Add("invalidParserFormat", 1)
	return "ecma404"

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

	m := i.Matcher.FindStringSubmatch(string(i.Unparsed))

	if m == nil || len(m) != 9 {

		stats.Add("rfc5424Unparsed", 1)
		return false

	}

	stats.Add("rfc5424Parsed", 1)

	pri, _ := strconv.Atoi(m[1])
	ver, _ := strconv.Atoi(m[2])

	var pid int

	if m[6] != "-" {

		pid, _ = strconv.Atoi(m[6])

	}

	i.Parsed = map[string]interface{}{
		"priority":   pri,
		"version":    ver,
		"timestamp":  m[3],
		"host":       m[4],
		"app":        m[5],
		"pid":        pid,
		"message_id": m[7],
		"message":    m[8],
	}

	return true

}

func (i *Input) parseJson() bool {

	i.Parsed = map[string]interface{}{}

	if err := json.Unmarshal(i.Unparsed, &i.Parsed); err != nil {

		stats.Add("ecma404Unparsed", 1)
		return false

	}

	stats.Add("ecma404Parsed", 1)
	return i.parseTimestamp()

}

func (i *Input) parseTimestamp() bool {

	if _, ok := i.Parsed["timestamp"]; !ok {

		stats.Add("ecma404MissingTimestamp", 1)
		return false

	}

	ok, i.Parsed["timestamp"] = NewTiemstamp().Parse(i.Parsed["timestamp"].(string))

	if !ok {

		return false

	}

	return true

}

var err error

func NewTiemstamp() *Timestamp {

	return &Timestamp{}

}

func (t *Timestamp) Parse(ts string) (bool, string) {

	t.Unparsed = strings.Split(ts, ".")

	if ok := t.format(); !ok {

		return false, t.Parsed

	}

	t.Parsed = time.Unix(t.Formated[0], t.Formated[1]).Format(time.RFC3339)

	return true, t.Parsed

}

func (t *Timestamp) format() bool {

	for _, ts := range t.Unparsed {

		p, err := strconv.ParseInt(ts, 10, 64)

		if err != nil {

			stats.Add("UnformatedTimestampPart", 1)
			return false

		}

		t.Formated = append(t.Formated, p)

	}

	t.ensureLength()
	return true

}

func (t *Timestamp) ensureLength() {

	if len(t.Formated) == 1 {

		t.Formated = append(t.Formated, 0)

	}

}
