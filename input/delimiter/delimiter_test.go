package delimiter

import (
	"bytes"
	"testing"
)

type TestCases map[string]TestCase

type TestCase struct {
	delimiter    *Delimiter
	raw          string
	results      []string
	resultsIndex int
	errors       []string
	errorsIndex  int
	leftover     string
}

var tests TestCases = TestCases{
	"valid": TestCase{
		raw: "19:I am a test string.;30:And this is a test string too!;29:You could add plenty of them.;31:And they should all work; fine.;",
		results: []string{
			"I am a test string.",
			"And this is a test string too!",
			"You could add plenty of them.",
			"And they should all work; fine.",
		},
	},
	"invalid length": TestCase{
		raw:    "19a:bc",
		errors: []string{"length-buffer-invalid-byte", "broken", "broken", "broken"},
	},
	"missing length": TestCase{
		raw:    "I...",
		errors: []string{"length-buffer-invalid-byte", "broken", "broken", "broken"},
	},
	"missing semicolon": TestCase{
		raw:     "19:I am a test string.30:A...",
		results: []string{"I am a test string.", ""},
		errors:  []string{"length-buffer-invalid-byte", "broken", "broken"},
	},
	"length too short": TestCase{
		raw:      "18:I am a test string.30:A",
		results:  []string{"I am a test string"},
		leftover: "A",
	},
	"length too long": TestCase{
		raw:     "20:I am a test string.30:A",
		results: []string{"I am a test string.3"},
		errors:  []string{"length-buffer-conversion-error", "broken"},
	},
	"value too short": TestCase{
		raw:     "19:I am a test string30:A",
		results: []string{"I am a test string3"},
		errors:  []string{"length-buffer-conversion-error", "broken"},
	},
	"value too long": TestCase{
		raw:      "19:I am a test string..30:A",
		results:  []string{"I am a test string."},
		leftover: "A",
	},
}

// Test_Delimiter checks, rather each .Push call returns the expected.
func Test_Delimiter(t *testing.T) {
	for n, tc := range tests {
		tc.delimiter = NewDelimiter()
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

func (tc *TestCase) checkResult(t *testing.T) {
	if tc.results == nil {
		t.Errorf("\ndelimiter returned unexpected result: '%v'\n", tc.delimiter.Result)
	}
	if tc.results[tc.resultsIndex] != tc.delimiter.Result {
		t.Errorf("\nexpected result: %v (index: %d)\nreturned result:%v\n", tc.results[tc.resultsIndex], tc.resultsIndex, tc.delimiter.Result)
	}
}

func (tc *TestCase) checkMissingResults(t *testing.T) {
	if len(tc.results) != tc.resultsIndex {
		t.Errorf("\ndelimiter missed some expected results: %v\n", tc.results[tc.resultsIndex:])
	}
}

func (tc *TestCase) checkErr(err error, t *testing.T) {
	if len(tc.errors)-1 < tc.errorsIndex {
		t.Errorf("\ndelimiter returned unexpected error: '%v'\n", err)
		return
	}
	if tc.errors[tc.errorsIndex] != err.Error() {
		t.Errorf("\nexpected error: %v (index: %d)\nreturned error: %v\n", tc.errors[tc.errorsIndex], tc.errorsIndex, err.Error())
	}
}

func (tc *TestCase) checkMissingErrors(t *testing.T) {
	if len(tc.errors) != tc.errorsIndex {
		t.Errorf("\ndelimiter missed some expected errors: %v\n", tc.errors[tc.errorsIndex:])
	}
}

func (tc *TestCase) checkLeftover(t *testing.T) {
	tc.delimiter.Reset()
	if tc.delimiter.Result != tc.leftover {
		t.Errorf("\nexpected leftover: %v\nreturned leftover: %v\n", tc.leftover, tc.delimiter.Result)
	}
}
