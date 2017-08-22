package rfc5424

import (
	"io"
	"strings"
	"testing"
)

// Test_NewDelimiter simply tests if a delimiter can be instantiated.
func Test_NewDelimiter(t *testing.T) {
	d := NewDelimiter(nil)
	if d == nil {
		t.Fatal("failed to create simple reader")
	}
}

func Test_DelimiterSingle(t *testing.T) {
	liner := strings.NewReader("<11>1 sshd is down\n<22>1 sshd is up")

	d := NewDelimiter(liner)
	if d == nil {
		t.Fatal("failed to create simple reader")
	}

	line, err := d.ReadLine()
	if err != nil {
		t.Fatalf("failed to read line: %s", err.Error())
	} else if line != "<11>1 sshd is down" {
		t.Fatalf("read line not correct, got %s, exp %s", line, "<11>1 sshd is down")
	}
}

func Test_DelimiterSinglePreceding(t *testing.T) {
	liner := strings.NewReader("xxyyy\n<11>1 sshd is down")

	d := NewDelimiter(liner)
	if d == nil {
		t.Fatal("failed to create simple reader")
	}

	line, err := d.ReadLine()
	if err != nil {
		t.Fatalf("failed to read line: %s", err.Error())
	} else if line != "xxyyy" {
		t.Fatalf("read line not correct, got %s, exp %s", line, "xxyyy")
	}
}

func Test_DelimiterEOF(t *testing.T) {
	liner := strings.NewReader("<11>1 sshd is down\n<22>1 sshd is up")

	d := NewDelimiter(liner)
	if d == nil {
		t.Fatal("failed to create simple reader")
	}

	line, err := d.ReadLine()
	if err != nil {
		t.Fatalf("failed to read line: %s", err.Error())
	} else if line != "<11>1 sshd is down" {
		t.Fatalf("read line not correct, got %s, exp %s", line, "<11>1 sshd is down")
	}

	line, err = d.ReadLine()
	if err != io.EOF {
		t.Fatalf("failed to receive EOF as expected")
	}
	if line != "<22>1 sshd is up" {
		t.Fatalf("returned line not correct after EOF, got %s", line)
	}
}

func Test_DelimiterMulti(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected []string
	}{
		{
			name:     "simple",
			line:     "<11>1 sshd is down\n<2>1 sshd is up\n<67>2 password accepted",
			expected: []string{"<11>1 sshd is down", "<2>1 sshd is up", "<67>2 password accepted"},
		},
		{
			name:     "leading",
			line:     "password accepted for user root\n<12>1 sshd is down\n<145>1 sshd is up\n<67>2 password accepted",
			expected: []string{"password accepted for user root", "<12>1 sshd is down", "<145>1 sshd is up", "<67>2 password accepted"},
		},
		{
			name:     "CRLF",
			line:     "<12>1 sshd is down\r\n<145>1 sshd is up\r\n<67>2 password accepted",
			expected: []string{"<12>1 sshd is down", "<145>1 sshd is up", "<67>2 password accepted"},
		},
		{
			name:     "stacktrace",
			line:     "<12>1 sshd is down\n<145>1 OOM on line 42, dummy.java\n\tclass_loader.jar\n<67>2 password accepted",
			expected: []string{"<12>1 sshd is down", "<145>1 OOM on line 42, dummy.java\n\tclass_loader.jar", "<67>2 password accepted"},
		},
		{
			name:     "embedded",
			line:     "<12>1 sshd is <down>\n<145>1 sshd is up<33>4\n<67>2 password accepted",
			expected: []string{"<12>1 sshd is <down>", "<145>1 sshd is up<33>4", "<67>2 password accepted"},
		},
	}

	for _, tt := range tests {
		d := NewDelimiter(strings.NewReader(tt.line))
		events := []string{}

		for {
			l, err := d.ReadLine()
			if err != nil && err != io.EOF {
				t.Fatalf("error reading lines: %s", err.Error())
			}
			events = append(events, l)

			if err == io.EOF {
				break
			}
		}

		if len(events) != len(tt.expected) {
			t.Errorf("test %s: failed to delimit (count) '%s' as expected", tt.name, tt.line)
		} else {
			for i := 0; i < len(events); i++ {
				if events[i] != tt.expected[i] {
					t.Errorf("test %s: failed to delimit '%s', got %s, expected %s", tt.name, tt.line, events[i], tt.expected[i])
				}
			}
		}
	}
}

func Test_DelimiterReal(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected []string
	}{
		{
			name:     "single",
			line:     `<134>0 2015-08-24T03:33:12.343339+00:00 fisher apache-access - - 68.198.147.33 - - [24/Aug/2015:03:33:07 +0000] "GET /wp-content/uploads/2014/06/grafana-edit.png HTTP/1.1" 200 47090 "http://www.philipotoole.com/influxdb-and-grafana-howto/" "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.9; rv:40.0) Gecko/20100101 Firefox/40.0"`,
			expected: []string{`<134>0 2015-08-24T03:33:12.343339+00:00 fisher apache-access - - 68.198.147.33 - - [24/Aug/2015:03:33:07 +0000] "GET /wp-content/uploads/2014/06/grafana-edit.png HTTP/1.1" 200 47090 "http://www.philipotoole.com/influxdb-and-grafana-howto/" "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.9; rv:40.0) Gecko/20100101 Firefox/40.0"`},
		},
		{
			name: "double",
			line: "<134>0 2015-08-24T03:33:12.343339+00:00 fisher apache-access - - 68.198.147.33 - - [24/Aug/2015:03:33:07 +0000] GET /wp-content/uploads/2014/06/grafana-edit.png HTTP/1.1 200 47090 http://www.philipotoole.com/influxdb-and-grafana-howto/ Mozilla/5.0 (Macintosh; Intel Mac OS X 10.9; rv:40.0) Gecko/20100101 Firefox/40.0\n<134>0 2015-08-24T03:36:12.487253+00:00 fisher apache-access - - 37.59.63.61 - - [24/Aug/2015:03:36:11 +0000] GET /feed/ HTTP/1.1 200 28388 - Ruby",
			expected: []string{
				`<134>0 2015-08-24T03:33:12.343339+00:00 fisher apache-access - - 68.198.147.33 - - [24/Aug/2015:03:33:07 +0000] GET /wp-content/uploads/2014/06/grafana-edit.png HTTP/1.1 200 47090 http://www.philipotoole.com/influxdb-and-grafana-howto/ Mozilla/5.0 (Macintosh; Intel Mac OS X 10.9; rv:40.0) Gecko/20100101 Firefox/40.0`,
				`<134>0 2015-08-24T03:36:12.487253+00:00 fisher apache-access - - 37.59.63.61 - - [24/Aug/2015:03:36:11 +0000] GET /feed/ HTTP/1.1 200 28388 - Ruby`,
			},
		},
	}

	for _, tt := range tests {
		d := NewDelimiter(strings.NewReader(tt.line))
		events := []string{}

		for {
			l, err := d.ReadLine()
			if err != nil && err != io.EOF {
				t.Fatalf("error reading lines: %s", err.Error())
			}
			events = append(events, l)

			if err == io.EOF {
				break
			}
		}

		if len(events) != len(tt.expected) {
			t.Errorf("test %s: failed to delimit (count) '%s' as expected", tt.name, tt.line)
		} else {
			for i := 0; i < len(events); i++ {
				if events[i] != tt.expected[i] {
					t.Errorf("test %s: failed to delimit '%s', got %s, expected %s", tt.name, tt.line, events[i], tt.expected[i])
				}
			}
		}
	}
}
