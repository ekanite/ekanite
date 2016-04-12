package input

import (
	"bytes"
	"testing"
)

var (
	ecma404_valid []string = []string{
		// gelf message (except timestamp value is string)
		`{"version": "1.1", "host": "example.org", "short_message": "A short message that helps you identify what is going on", "full_message": "Backtrace here\n\nmore stuff", "timestamp": "1095379198.75", "level": 1, "_user_id": 9001, "_some_info": "foo", "_some_env_var": "bar"}`,
	}
	ecma404_invalid []string = []string{
		// missing closing bracket
		`{"version": "1.1", "host": "example.org", "short_message": "A short message that helps you identify what is going on", "full_message": "Backtrace here\n\nmore stuff", "timestamp": "1095379198.75", "level": 1, "_user_id": 9001, "_some_info": "foo", "_some_env_var": "bar"`,
		// missing timestamp
		`{"version": "1.1", "host": "example.org", "short_message": "A short message that helps you identify what is going on", "full_message": "Backtrace here\n\nmore stuff", "level": 1, "_user_id": 9001, "_some_info": "foo", "_some_env_var": "bar"}`,
	}
	rfc5424_valid []string = []string{
		`<34>1 2003-10-11T22:14:15.003Z mymachine.example.com su - ID47 - BOM'su root' failed for lonvick on /dev/pts/8`,
	}
	rfc5424_invalid []string = []string{
		// missing PRI
		`1 2003-10-11T22:14:15.003Z mymachine.example.com su - ID47 - BOM'su root' failed for lonvick on /dev/pts/8`,
		// missing TIMESTAMP
		`<34>1 mymachine.example.com su - ID47 - BOM'su root' failed for lonvick on /dev/pts/8`,
	}
)

func Test_Formats(t *testing.T) {

	var p Input

	mismatched := func(rtrnd string, intnd string, intndA string) {

		if intndA != "" {

			t.Fatalf("Parser format %v does not match the intended format %v.\n", rtrnd, intnd)

		} else {

			t.Fatalf("Parser format %v does not match the indended format %v, which is equal to %v.\n", rtrnd, intndA, intnd)

		}

	}

	for i, f := range FORMATS_BY_NAME {

		p = NewParser(f)

		if p.Format != FORMATS_BY_STANDARD[i] {

			mismatched(p.Format, f, FORMATS_BY_STANDARD[i])

		}

	}

	for _, f := range FORMATS_BY_STANDARD {

		p = NewParser(f)

		if p.Format != f {

			mismatched(p.Format, f, "")

		}

	}

	p = NewParser("unknown-format")

	if p.Format != "ecma404" {

		mismatched(p.Format, "ecma404", "")

	}

}

func Test_Ecma404(t *testing.T) {

	var ok bool
	p := NewParser("json")

	for _, m := range ecma404_valid {

		ok, _ = p.Parse(bytes.NewBufferString(m).Bytes())

		if !ok {

			t.Fatalf("Parser should be able to parse: %v\n", m)

		}

	}

	for _, m := range ecma404_invalid {

		ok, _ = p.Parse(bytes.NewBufferString(m).Bytes())

		if ok {

			t.Fatalf("Parser should not be able to parse: %v\n", m)

		}

	}

}

func Test_RFC5424(t *testing.T) {

	var ok bool
	p := NewParser("syslog")

	for _, m := range rfc5424_valid {

		ok, _ = p.Parse(bytes.NewBufferString(m).Bytes())

		if !ok {

			t.Fatalf("Parser should be able to parse: %v\n", m)

		}

	}

	for _, m := range rfc5424_invalid {

		ok, _ = p.Parse(bytes.NewBufferString(m).Bytes())

		if ok {

			t.Fatalf("Parser should not be able to parse: %v\n", m)

		}

	}

}
