package input

import (
	"testing"
)

/*
 * SyslogDelimiter tests.
 */

func Test_SyslogDelimiter(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected []string
	}{
		{
			name:     "simple",
			line:     "<11>1 sshd is down\n<22>1 sshd is up\n<67>2 password accepted",
			expected: []string{"<11>1 sshd is down", "<22>1 sshd is up"},
		},
		{
			name:     "leading",
			line:     "password accepted for user root<12>1 sshd is down\n<145>1 sshd is up\n<67>2 password accepted",
			expected: []string{"<12>1 sshd is down", "<145>1 sshd is up"},
		},
		{
			name:     "CRLF",
			line:     "<12>1 sshd is down\r\n<145>1 sshd is up\r\n<67>2 password accepted",
			expected: []string{"<12>1 sshd is down", "<145>1 sshd is up"},
		},
		{
			name:     "stacktrace",
			line:     "<12>1 sshd is down\n<145>1 OOM on line 42, dummy.java\n\tclass_loader.jar\n<67>2 password accepted",
			expected: []string{"<12>1 sshd is down", "<145>1 OOM on line 42, dummy.java\n\tclass_loader.jar"},
		},
		{
			name:     "embedded",
			line:     "<12>1 sshd is <down>\n<145>1 sshd is up<33>4\n<67>2 password accepted",
			expected: []string{"<12>1 sshd is <down>", "<145>1 sshd is up<33>4"},
		},
	}

	for _, tt := range tests {
		d := NewSyslogDelimiter(256)
		events := []string{}

		for _, b := range tt.line {
			event, match := d.Push(byte(b))
			if match {
				events = append(events, event)
			}
		}

		if len(events) != len(tt.expected) {
			t.Errorf("test %s: failed to delimit '%s' as expected", tt.name, tt.line)
		} else {
			for i := 0; i < len(events); i++ {
				if events[i] != tt.expected[i] {
					t.Errorf("test %s: failed to delimit '%s', got %s, expected %s", tt.name, tt.line, events[i], tt.expected)
				}
			}
		}
	}
}

func TestSyslogDelimiter_Vestige(t *testing.T) {
	tests := []struct {
		name          string
		line          string
		expectedEvent string
		expectedMatch bool
	}{
		{
			name:          "vestige zero",
			line:          "",
			expectedEvent: "",
			expectedMatch: false,
		},
		{
			name:          "vestige no match",
			line:          "12\n",
			expectedEvent: "",
			expectedMatch: false,
		},
		{
			name:          "vestige match",
			line:          "<12>3 ",
			expectedEvent: "<12>3 ",
			expectedMatch: true,
		},
		{
			name:          "vestige rich match",
			line:          "<145>1 OOM on line 42, dummy.java\n\tclass_loader.jar",
			expectedEvent: "<145>1 OOM on line 42, dummy.java\n\tclass_loader.jar",
			expectedMatch: true,
		},
	}

	for _, tt := range tests {
		d := NewSyslogDelimiter(256)
		for _, c := range tt.line {
			d.Push(byte(c))
		}
		e, m := d.Vestige()
		if e != tt.expectedEvent || m != tt.expectedMatch {
			t.Errorf("test %s: vestige test failed, got %s %v, expected %s %v", tt.name, e, m, tt.expectedEvent, tt.expectedMatch)
		}
	}
}
