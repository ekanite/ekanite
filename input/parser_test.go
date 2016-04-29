package input

import (
	"bytes"
	"reflect"
	"testing"
)

type ParserTestCases []ParserTestCase

type ParserTestCase struct {
	fmt     string
	fail    bool
	tests   []string
	results []map[string]interface{}
}

var parserTests = ParserTestCases{
	ParserTestCase{
		fmt: "syslog",
		tests: []string{
			`<134>1 2003-08-24T05:14:15.000003-07:00 ubuntu sshd 1999 - password accepted`,
			`<33>5 1985-04-12T23:20:50.52Z test.com cron 304 - password accepted`,
			`<1>0 1985-04-12T19:20:50.52-04:00 test.com cron 65535 - password accepted`,
			`<1>0 2003-10-11T22:14:15.003Z test.com cron 65535 msgid1234 password accepted`,
			`<1>0 2003-08-24T05:14:15.000003-07:00 test.com cron 65535 - JVM NPE\nsome_file.java:48\n\tsome_other_file.java:902`,
			`<27>1 2015-03-02T22:53:45-08:00 localhost.localdomain puppet-agent 5334 - mirrorurls.extend(list(self.metalink_data.urls()))`,
			`<29>1 2015-03-03T06:49:08-08:00 localhost.localdomain puppet-agent 51564 - (/Stage[main]/Users_prd/Ssh_authorized_key[1063-username]) Dependency Group[group] has failures: true`,
			`<142>1 2015-03-02T22:23:07-08:00 localhost.localdomain Keepalived_vrrp 21125 - VRRP_Instance(VI_1) ignoring received advertisment...`,
			`<142>1 2015-03-02T22:23:07-08:00 localhost.localdomain Keepalived_vrrp 21125 - HEAD /wp-login.php HTTP/1.1" 200 167 "http://www.philipotoole.com/" "Mozilla/5.0 (Windows NT 6.1) AppleWebKit/537.11 (KHTML, like Gecko) Chrome/23.0.1271.97 Safari/537.11`,
			`<134>0 2015-05-05T21:20:00.493320+00:00 fisher apache-access - - 173.247.206.174 - - [05/May/2015:21:19:52 +0000] "GET /2013/11/ HTTP/1.1" 200 22056 "http://www.philipotoole.com/" "Wget/1.15 (linux-gnu)"`,
		},
		results: []map[string]interface{}{
			map[string]interface{}{
				"priority":   134,
				"version":    1,
				"timestamp":  "2003-08-24T05:14:15.000003-07:00",
				"host":       "ubuntu",
				"app":        "sshd",
				"pid":        1999,
				"message_id": "-",
				"message":    "password accepted",
			},
			map[string]interface{}{
				"priority":   33,
				"version":    5,
				"timestamp":  "1985-04-12T23:20:50.52Z",
				"host":       "test.com",
				"app":        "cron",
				"pid":        304,
				"message_id": "-",
				"message":    "password accepted",
			},
			map[string]interface{}{
				"priority":   1,
				"version":    0,
				"timestamp":  "1985-04-12T19:20:50.52-04:00",
				"host":       "test.com",
				"app":        "cron",
				"pid":        65535,
				"message_id": "-",
				"message":    "password accepted",
			},
			map[string]interface{}{
				"priority":   1,
				"version":    0,
				"timestamp":  "2003-10-11T22:14:15.003Z",
				"host":       "test.com",
				"app":        "cron",
				"pid":        65535,
				"message_id": "msgid1234",
				"message":    "password accepted",
			},
			map[string]interface{}{
				"priority":   1,
				"version":    0,
				"timestamp":  "2003-08-24T05:14:15.000003-07:00",
				"host":       "test.com",
				"app":        "cron",
				"pid":        65535,
				"message_id": "-",
				"message":    `JVM NPE\nsome_file.java:48\n\tsome_other_file.java:902`,
			},
			map[string]interface{}{
				"priority":   27,
				"version":    1,
				"timestamp":  "2015-03-02T22:53:45-08:00",
				"host":       "localhost.localdomain",
				"app":        "puppet-agent",
				"pid":        5334,
				"message_id": "-",
				"message":    "mirrorurls.extend(list(self.metalink_data.urls()))",
			},
			map[string]interface{}{
				"priority":   29,
				"version":    1,
				"timestamp":  "2015-03-03T06:49:08-08:00",
				"host":       "localhost.localdomain",
				"app":        "puppet-agent",
				"pid":        51564,
				"message_id": "-",
				"message":    "(/Stage[main]/Users_prd/Ssh_authorized_key[1063-username]) Dependency Group[group] has failures: true",
			},
			map[string]interface{}{
				"priority":   142,
				"version":    1,
				"timestamp":  "2015-03-02T22:23:07-08:00",
				"host":       "localhost.localdomain",
				"app":        "Keepalived_vrrp",
				"pid":        21125,
				"message_id": "-",
				"message":    "VRRP_Instance(VI_1) ignoring received advertisment...",
			},
			map[string]interface{}{
				"priority":   142,
				"version":    1,
				"timestamp":  "2015-03-02T22:23:07-08:00",
				"host":       "localhost.localdomain",
				"app":        "Keepalived_vrrp",
				"pid":        21125,
				"message_id": "-",
				"message":    `HEAD /wp-login.php HTTP/1.1" 200 167 "http://www.philipotoole.com/" "Mozilla/5.0 (Windows NT 6.1) AppleWebKit/537.11 (KHTML, like Gecko) Chrome/23.0.1271.97 Safari/537.11`,
			},
			map[string]interface{}{
				"priority":   134,
				"version":    0,
				"timestamp":  "2015-05-05T21:20:00.493320+00:00",
				"host":       "fisher",
				"app":        "apache-access",
				"pid":        0,
				"message_id": "-",
				"message":    `173.247.206.174 - - [05/May/2015:21:19:52 +0000] "GET /2013/11/ HTTP/1.1" 200 22056 "http://www.philipotoole.com/" "Wget/1.15 (linux-gnu)"`,
			},
		},
	},
	ParserTestCase{
		fmt:  "syslog",
		fail: true,
		tests: []string{
			`1 2003-10-11T22:14:15.003Z mymachine.example.com su - ID47 - BOM'su root' failed for lonvick on /dev/pts/8`,
			`<34>1 mymachine.example.com su - ID47 - BOM'su root' failed for lonvick on /dev/pts/8`,
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
	for _, tc := range parserTests {
		tc.printTitle(t)
		p := NewParser(tc.fmt)
		for i, v := range tc.tests {
			t.Logf("using %d\n", i+1)
			tc.determFailure(p.Parse(bytes.NewBufferString(v).Bytes()), t)
			if !tc.fail && !reflect.DeepEqual(p.Result, tc.results[i]) {
				t.Logf("%v", p.Result)
				t.Logf("%v", tc.results[i])
				t.Error("\n\nParser result does not match expected result.\n")
			}
		}
	}
}

func (tc *ParserTestCase) printTitle(t *testing.T) {
	var status string
	if !tc.fail {
		status = "success"
	} else {
		status = "failure"
	}
	t.Logf("testing %s (%s)\n", tc.fmt, status)
}

func (tc *ParserTestCase) determFailure(ok bool, t *testing.T) {
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
