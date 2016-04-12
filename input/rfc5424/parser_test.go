package rfc5424

import "testing"

/*
 * RFC5424 parser tests
 */

func Test_SuccessfulParsing(t *testing.T) {
	p := NewParser()

	tests := []struct {
		message  string
		expected Message
	}{
		{
			message:  "<134>1 2003-08-24T05:14:15.000003-07:00 ubuntu sshd 1999 - password accepted",
			expected: Message{Priority: 134, Version: 1, Timestamp: "2003-08-24T05:14:15.000003-07:00", Host: "ubuntu", App: "sshd", Pid: 1999, MsgId: "-", Message: "password accepted"},
		},
		{
			message:  "<33>5 1985-04-12T23:20:50.52Z test.com cron 304 - password accepted",
			expected: Message{Priority: 33, Version: 5, Timestamp: "1985-04-12T23:20:50.52Z", Host: "test.com", App: "cron", Pid: 304, MsgId: "-", Message: "password accepted"},
		},
		{
			message:  "<1>0 1985-04-12T19:20:50.52-04:00 test.com cron 65535 - password accepted",
			expected: Message{Priority: 1, Version: 0, Timestamp: "1985-04-12T19:20:50.52-04:00", Host: "test.com", App: "cron", Pid: 65535, MsgId: "-", Message: "password accepted"},
		},
		{
			message:  "<1>0 2003-10-11T22:14:15.003Z test.com cron 65535 msgid1234 password accepted",
			expected: Message{Priority: 1, Version: 0, Timestamp: "2003-10-11T22:14:15.003Z", Host: "test.com", App: "cron", Pid: 65535, MsgId: "msgid1234", Message: "password accepted"},
		},
		{
			message:  "<1>0 2003-08-24T05:14:15.000003-07:00 test.com cron 65535 - JVM NPE\nsome_file.java:48\n\tsome_other_file.java:902",
			expected: Message{Priority: 1, Version: 0, Timestamp: "2003-08-24T05:14:15.000003-07:00", Host: "test.com", App: "cron", Pid: 65535, MsgId: "-", Message: "JVM NPE\nsome_file.java:48\n\tsome_other_file.java:902"},
		},
		{
			message:  "<27>1 2015-03-02T22:53:45-08:00 localhost.localdomain puppet-agent 5334 - mirrorurls.extend(list(self.metalink_data.urls()))",
			expected: Message{Priority: 27, Version: 1, Timestamp: "2015-03-02T22:53:45-08:00", Host: "localhost.localdomain", App: "puppet-agent", Pid: 5334, MsgId: "-", Message: "mirrorurls.extend(list(self.metalink_data.urls()))"},
		},
		{
			message:  "<29>1 2015-03-03T06:49:08-08:00 localhost.localdomain puppet-agent 51564 - (/Stage[main]/Users_prd/Ssh_authorized_key[1063-username]) Dependency Group[group] has failures: true",
			expected: Message{Priority: 29, Version: 1, Timestamp: "2015-03-03T06:49:08-08:00", Host: "localhost.localdomain", App: "puppet-agent", Pid: 51564, MsgId: "-", Message: "(/Stage[main]/Users_prd/Ssh_authorized_key[1063-username]) Dependency Group[group] has failures: true"},
		},
		{
			message:  "<142>1 2015-03-02T22:23:07-08:00 localhost.localdomain Keepalived_vrrp 21125 - VRRP_Instance(VI_1) ignoring received advertisment...",
			expected: Message{Priority: 142, Version: 1, Timestamp: "2015-03-02T22:23:07-08:00", Host: "localhost.localdomain", App: "Keepalived_vrrp", Pid: 21125, MsgId: "-", Message: "VRRP_Instance(VI_1) ignoring received advertisment..."},
		},
		{
			message:  `<142>1 2015-03-02T22:23:07-08:00 localhost.localdomain Keepalived_vrrp 21125 - HEAD /wp-login.php HTTP/1.1" 200 167 "http://www.philipotoole.com/" "Mozilla/5.0 (Windows NT 6.1) AppleWebKit/537.11 (KHTML, like Gecko) Chrome/23.0.1271.97 Safari/537.11`,
			expected: Message{Priority: 142, Version: 1, Timestamp: "2015-03-02T22:23:07-08:00", Host: "localhost.localdomain", App: "Keepalived_vrrp", Pid: 21125, MsgId: "-", Message: `HEAD /wp-login.php HTTP/1.1" 200 167 "http://www.philipotoole.com/" "Mozilla/5.0 (Windows NT 6.1) AppleWebKit/537.11 (KHTML, like Gecko) Chrome/23.0.1271.97 Safari/537.11`},
		},
		{
			message:  `<134>0 2015-05-05T21:20:00.493320+00:00 fisher apache-access - - 173.247.206.174 - - [05/May/2015:21:19:52 +0000] "GET /2013/11/ HTTP/1.1" 200 22056 "http://www.philipotoole.com/" "Wget/1.15 (linux-gnu)"`,
			expected: Message{Priority: 134, Version: 0, Timestamp: "2015-05-05T21:20:00.493320+00:00", Host: "fisher", App: "apache-access", Pid: 0, MsgId: "-", Message: `173.247.206.174 - - [05/May/2015:21:19:52 +0000] "GET /2013/11/ HTTP/1.1" 200 22056 "http://www.philipotoole.com/" "Wget/1.15 (linux-gnu)"`},
		},
	}

	for i, tt := range tests {
		m := p.Parse(tt.message)
		if m == nil {
			t.Fatalf("test %d: failed to parse: %s", i, tt.message)
		}
		mm, _ := m.(Message)
		if tt.expected != m {
			t.Errorf("test %d: incorrect parsing of: %v", i, tt.message)
			t.Logf("Priority: %d (match: %v)", mm.Priority, mm.Priority == tt.expected.Priority)
			t.Logf("Version: %d (match: %v)", mm.Version, mm.Version == tt.expected.Version)
			t.Logf("Timestamp: %s (match: %v)", mm.Timestamp, mm.Timestamp == tt.expected.Timestamp)
			t.Logf("Host: %s (match: %v)", mm.Host, mm.Host == tt.expected.Host)
			t.Logf("App: %s (match: %v)", mm.App, mm.App == tt.expected.App)
			t.Logf("PID: %d (match: %v)", mm.Pid, mm.Pid == tt.expected.Pid)
			t.Logf("MsgId: %s (match: %v)", mm.MsgId, mm.MsgId == tt.expected.MsgId)
			t.Logf("Message: %s (match: %v)", mm.Message, mm.Message == tt.expected.Message)
		}
	}
}

func Test_FailedParsing(t *testing.T) {
	p := NewParser()

	tests := []string{
		"<134> 2013-09-04T10:25:52.618085 ubuntu sshd 1999 - password accepted",
		"<33> 7 2013-09-04T10:25:52.618085 test.com cron 304 - password accepted",
		"<33> 7 2013-09-04T10:25:52.618085 test.com cron 304 $ password accepted",
		"<33> 7 2013-09-04T10:25:52.618085 test.com cron 304 - - password accepted",
		"<33>7 2013-09-04T10:25:52.618085 test.com cron not_a_pid - password accepted",
		"5:52.618085 test.com cron 65535 - password accepted",
	}

	for _, message := range tests {
		if p.Parse(message) != nil {
			t.Errorf("parsed '%s', not expected", message)
		}
	}
}

func Benchmark_Parsing(b *testing.B) {
	p := NewParser()
	for n := 0; n < b.N; n++ {
		m := p.Parse(`<134>0 2015-05-05T21:20:00.493320+00:00 fisher apache-access - - 173.247.206.174 - - [05/May/2015:21:19:52 +0000] "GET /2013/11/ HTTP/1.  1" 200 22056 "http://www.philipotoole.com/" "Wget/1.15 (linux-gnu)"`)
		if m == nil {
			panic("message failed to parse during benchmarking")
		}

	}
}
