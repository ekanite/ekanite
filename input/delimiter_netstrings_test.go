package input

import (
	"bytes"
	"testing"
)

type DelimiterTestCases map[string]DelimiterTestCase

type DelimiterTestCase struct {
	delimiter    *NetstrDelimiter
	raw          string
	results      []string
	resultsIndex int
	errors       []string
	errorsIndex  int
	leftover     string
}

var delimiterTests DelimiterTestCases = DelimiterTestCases{
	"valid": DelimiterTestCase{
		raw: "19:I am a test string.;30:And this is a test string too!;29:You could add plenty of them.;31:And they should all work; fine.;",
		results: []string{
			"I am a test string.",
			"And this is a test string too!",
			"You could add plenty of them.",
			"And they should all work; fine.",
		},
	},
	"invalid length": DelimiterTestCase{
		raw:    "19a:bc",
		errors: []string{"length-buffer-invalid-byte", "broken", "broken", "broken"},
	},
	"missing length": DelimiterTestCase{
		raw:    "I...",
		errors: []string{"length-buffer-invalid-byte", "broken", "broken", "broken"},
	},
	"missing semicolon": DelimiterTestCase{
		raw:     "19:I am a test string.30:A...",
		results: []string{"I am a test string.", ""},
		errors:  []string{"length-buffer-invalid-byte", "broken", "broken"},
	},
	"length too short": DelimiterTestCase{
		raw:      "18:I am a test string.30:A",
		results:  []string{"I am a test string"},
		leftover: "A",
	},
	"length too long": DelimiterTestCase{
		raw:     "20:I am a test string.30:A",
		results: []string{"I am a test string.3"},
		errors:  []string{"length-buffer-conversion-error", "broken"},
	},
	"value too short": DelimiterTestCase{
		raw:     "19:I am a test string30:A",
		results: []string{"I am a test string3"},
		errors:  []string{"length-buffer-conversion-error", "broken"},
	},
	"value too long": DelimiterTestCase{
		raw:      "19:I am a test string..30:A",
		results:  []string{"I am a test string."},
		leftover: "A",
	},
}

// Test_NetstrDelimiter checks, rather each .Push call returns the expected.
func Test_NetstrDelimiter(t *testing.T) {
	for n, tc := range delimiterTests {
		tc.delimiter = NewNetstrDelimiter()
		t.Logf("testing: %v\n", n)
		buff := bytes.NewBufferString(tc.raw)
		for _, b := range buff.Bytes() {
			ok, err := tc.delimiter.Push(b)
			if ok {
				tc.checkResult(t)
				tc.resultsIndex++
			}
			if err != nil {
				tc.checkErr(err, t)
				tc.errorsIndex++
			}
		}
		tc.checkMissingResults(t)
		tc.checkMissingErrors(t)
		tc.checkLeftover(t)
	}
}

func (tc *DelimiterTestCase) checkResult(t *testing.T) {
	if tc.results == nil {
		t.Errorf("\ndelimiter returned unexpected result: '%v'\n", tc.delimiter.Result)
	}
	if tc.results[tc.resultsIndex] != tc.delimiter.Result {
		t.Errorf("\nexpected result: %v (index: %d)\nreturned result:%v\n", tc.results[tc.resultsIndex], tc.resultsIndex, tc.delimiter.Result)
	}
}

func (tc *DelimiterTestCase) checkMissingResults(t *testing.T) {
	if len(tc.results) != tc.resultsIndex {
		t.Errorf("\ndelimiter missed some expected results: %v\n", tc.results[tc.resultsIndex:])
	}
}

func (tc *DelimiterTestCase) checkErr(err error, t *testing.T) {
	if len(tc.errors)-1 < tc.errorsIndex {
		t.Errorf("\ndelimiter returned unexpected error: '%v'\n", err)
		return
	}
	if tc.errors[tc.errorsIndex] != err.Error() {
		t.Errorf("\nexpected error: %v (index: %d)\nreturned error: %v\n", tc.errors[tc.errorsIndex], tc.errorsIndex, err.Error())
	}
}

func (tc *DelimiterTestCase) checkMissingErrors(t *testing.T) {
	if len(tc.errors) != tc.errorsIndex {
		t.Errorf("\ndelimiter missed some expected errors: %v\n", tc.errors[tc.errorsIndex:])
	}
}

func (tc *DelimiterTestCase) checkLeftover(t *testing.T) {
	tc.delimiter.Reset()
	if tc.delimiter.Result != tc.leftover {
		t.Errorf("\nexpected leftover: %v\nreturned leftover: %v\n", tc.leftover, tc.delimiter.Result)
	}
}
