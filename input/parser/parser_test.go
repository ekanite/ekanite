package parser

import (
	"bytes"
	"testing"
)

type TestCases []TestCase

type TestCase struct {
	fmt   string
	fail  bool
	tests map[string]string
}

var tests = TestCases{
	TestCase{
		fmt: "json",
		tests: map[string]string{
			"gelf like message": `{"version": "1.1", "host": "example.org", "short_message": "A short message that helps you identify what is going on", "full_message": "Backtrace here\n\nmore stuff", "timestamp": "1095379198.75", "level": 1, "_user_id": 9001, "_some_info": "foo", "_some_env_var": "bar"}`,
		},
	},
	TestCase{
		fmt:  "json",
		fail: true,
		tests: map[string]string{
			"missing closing brackets": `{"version": "1.1", "host": "example.org", "short_message": "A short message that helps you identify what is going on", "full_message": "Backtrace here\n\nmore stuff", "timestamp": "1095379198.75", "level": 1, "_user_id": 9001, "_some_info": "foo", "_some_env_var": "bar"`,
			"missing timestamp":        `{"version": "1.1", "host": "example.org", "short_message": "A short message that helps you identify what is going on", "full_message": "Backtrace here\n\nmore stuff", "level": 1, "_user_id": 9001, "_some_info": "foo", "_some_env_var": "bar"}`,
		},
	},
	TestCase{
		fmt: "syslog",
		tests: map[string]string{
			"syslog like message": `<34>1 2003-10-11T22:14:15.003Z mymachine.example.com su - ID47 - BOM'su root' failed for lonvick on /dev/pts/8`,
		},
	},
	TestCase{
		fmt:  "syslog",
		fail: true,
		tests: map[string]string{
			"missing PRI (priority)": `1 2003-10-11T22:14:15.003Z mymachine.example.com su - ID47 - BOM'su root' failed for lonvick on /dev/pts/8`,
			"missing timestamp":      `<34>1 mymachine.example.com su - ID47 - BOM'su root' failed for lonvick on /dev/pts/8`,
		},
	},
}

func Test_Formats(t *testing.T) {
	var p *Parser
	mismatched := func(rtrnd string, intnd string, intndA string) {
		if intndA != "" {
			t.Fatalf("Parser format %v does not match the intended format %v.\n", rtrnd, intnd)
		} else {
			t.Fatalf("Parser format %v does not match the indended format %v (same as: %v).\n", rtrnd, intndA, intnd)
		}
	}
	for i, f := range fmtsByName {
		p = NewParser(f)
		if p.Fmt != fmtsByStandard[i] {
			mismatched(p.Fmt, f, fmtsByStandard[i])
		}
	}
	for _, f := range fmtsByStandard {
		p = NewParser(f)
		if p.Fmt != f {
			mismatched(p.Fmt, f, "")
		}
	}
	p = NewParser("unknown-format")
	if p.Fmt != "ecma404" {
		mismatched(p.Fmt, "ecma404", "")
	}
}

func Test_Parsing(t *testing.T) {
	for _, tc := range tests {
		tc.printTitle(t)
		p := NewParser(tc.fmt)
		for k, v := range tc.tests {
			t.Logf("using %s:\n", k)
			tc.determFailure(p.Parse(bytes.NewBufferString(v).Bytes()), t)
		}
	}
}

func (tc *TestCase) printTitle(t *testing.T) {
	var status string
	if !tc.fail {
		status = "success"
	} else {
		status = "failure"
	}
	t.Logf("testing %s (%s)\n", tc.fmt, status)
}

func (tc *TestCase) determFailure(ok bool, t *testing.T) {
	if tc.fail {
		if ok {
			t.Error("\n\nParser should fail.\n")
		}
	} else {
		if !ok {
			t.Error("\n\nParser should succeed.\n")
		}
	}
}
