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
		if p.fmt != fmtsByStandard[i] {
			mismatched(p.fmt, f, fmtsByStandard[i])
		}
	}
	for _, f := range fmtsByStandard {
		p = NewParser(f)
		if p.fmt != f {
			mismatched(p.fmt, f, "")
		}
	}
	p = NewParser("unknown-format")
	if p.fmt != fmtsByStandard[0] {
		mismatched(p.fmt, fmtsByStandard[0], "")
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
