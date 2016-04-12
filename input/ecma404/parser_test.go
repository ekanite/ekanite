package ecma404

import (
	"testing"
)

/*
 * ECMA404 parser tests
 */

func Test_SuccessfulParsing(t *testing.T) {
	p := NewParser()

	type testParser struct {
		message  string
		expected Message
	}
	tests := []testParser{
		testParser{
			message: `{"timestamp": "1061727255", "message": "I'm a test log", "x-costum-field" : "{\"name\": \"alice\", \"age\": \"56\"}"}`,
			expected: Message{
				data: map[string]string{
					"timestamp":      "2003-08-24T05:14:15.000003-07:00",
					"message":        "I'm a test log",
					"x-costum-field": `{"name": "alice", "age": "56"}`,
				},
			},
		},
	}

	for i, tt := range tests {
		m := p.Parse(tt.message)
		if m == nil {
			t.Fatalf("test %d: failed to parse: %s", i, tt.message)
		}
		mm, _ := m.(Message)
		for k, v := range tt.expected.data {
			mv, ok := mm.data[k]
			if !ok {
				t.Errorf("Key %q): not in parsed data.", k)
				continue
			}
			if mv != v && k != "timestamp" {
				t.Errorf("(key %q): value %q does not match %q", k, mv, v)
			}
		}
	}
}

func Benchmark_Parsing(b *testing.B) {
	p := NewParser()
	for n := 0; n < b.N; n++ {
		m := p.Parse(`{"timestamp\": \"1061727255\", \"message\": \"I'm a test log\", \"x-costum-field\" : \"{\"name\": \"alice\", \"age\": \"56\"}\"}`)
		if m == nil {
			panic("message failed to parse during benchmarking")
		}

	}
}
