package main_test

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ekanite/ekanite"
	"github.com/ekanite/ekanite/input"
)

// testSystem represents a single end-to-end system.
type testSystem struct {
	e *testEngine
	b *testBatcher
	s *testServer
	c *testCollector
}

// NewSystem returns a funtioning ingestion, indexing, and search system.
func NewSystem(path string) *testSystem {
	// Clear out any existing data from previous test calls.
	os.RemoveAll(path)

	e := NewEngine(path)
	b := NewBatcher(e)
	s := NewServer("127.0.0.1:0", e)
	c := NewCollector("127.0.0.1:0")

	b.Start(nil)
	c.Start(b.C())
	s.Start()
	return &testSystem{e, b, s, c}
}

// IngestConn returns a TCP connection suitable for ingestion of log events.
func (s *testSystem) IngestConn() net.Conn {
	conn, err := net.Dial("tcp", s.c.Addr().String())
	if err != nil {
		panic("unable to connect to test TCP Collector")
	}
	return conn
}

// Test_EndToEnd ensures a complete system operates as expected.
func Test_EndToEnd(t *testing.T) {
	path := tempPath()
	defer os.RemoveAll(path)

	tests := []struct {
		name     string   // Test name, for ease of use.
		skip     bool     // Skip this test.
		reset    bool     // If true, recreate the system.
		events   []string // Log lines for ingestion. If empty, no events sent.
		query    string   // Search query. Ignored if not set.
		expected []string // Expected results. Ignored if "query" not set.
	}{
		// Simple single-event writes, followed by search.
		{
			name:     "query an empty system",
			reset:    true,
			query:    "server",
			expected: nil,
		},
		{
			name:     "query after single event ingestion",
			events:   []string{"<33>5 1985-04-12T23:20:50.52Z test.com cron 304 - password accepted"},
			query:    "password",
			expected: []string{"<33>5 1985-04-12T23:20:50.52Z test.com cron 304 - password accepted"}, // Only the message, right? Hmmm.
		},
		{
			name:     "query after second event ingestion, single match only",
			events:   []string{"<33>5 1985-04-12T23:20:50.52Z test.com cron 304 - password rejected"},
			query:    "rejected",
			expected: []string{"<33>5 1985-04-12T23:20:50.52Z test.com cron 304 - password rejected"},
		},
		{
			name:  "query only second event ingestion, both should match",
			query: "password",
			expected: []string{
				"<33>5 1985-04-12T23:20:50.52Z test.com cron 304 - password accepted",
				"<33>5 1985-04-12T23:20:50.52Z test.com cron 304 - password rejected",
			},
		},
		{
			name:   "query after third event ingestion, double match only",
			events: []string{"<33>5 1985-04-12T23:20:50.52Z test.com cron 304 - root access denied"},
			query:  "password",
			expected: []string{
				"<33>5 1985-04-12T23:20:50.52Z test.com cron 304 - password accepted",
				"<33>5 1985-04-12T23:20:50.52Z test.com cron 304 - password rejected",
			},
		},
		{
			name:     "query after third event ingestion, should be no match",
			query:    "notfound",
			expected: []string{},
		},

		// More complex writes, followed by simple search. Ensure tokeniser works as expected.
		{
			name:  "query 'GET' which should match 2 events",
			reset: true,
			events: []string{
				"<33>5 1985-04-12T23:20:50.52Z test.com cron 304 - GET /wp-content/uploads/2012/03/steelhead_cloud_accelerator_saas_diagram.jpg",
				"<33>5 1985-04-12T23:21:50.52Z test.com cron 304 - GET /wp-includes/images/smilies/frownie.png HTTP/1.1",
				"<33>5 1985-04-12T23:22:50.52Z test.com cron 304 - POST /log-includes/images/smilies/frownie.png HTTP/1.1",
			},
			query: "GET",
			expected: []string{
				"<33>5 1985-04-12T23:20:50.52Z test.com cron 304 - GET /wp-content/uploads/2012/03/steelhead_cloud_accelerator_saas_diagram.jpg",
				"<33>5 1985-04-12T23:21:50.52Z test.com cron 304 - GET /wp-includes/images/smilies/frownie.png HTTP/1.1",
			},
		},
		{
			name:  "query 'wp' which should match 2 events",
			query: "wp",
			expected: []string{
				"<33>5 1985-04-12T23:20:50.52Z test.com cron 304 - GET /wp-content/uploads/2012/03/steelhead_cloud_accelerator_saas_diagram.jpg",
				"<33>5 1985-04-12T23:21:50.52Z test.com cron 304 - GET /wp-includes/images/smilies/frownie.png HTTP/1.1",
			},
		},
		{
			name:  "query 'content' which should match 1 event",
			query: "content",
			expected: []string{
				"<33>5 1985-04-12T23:20:50.52Z test.com cron 304 - GET /wp-content/uploads/2012/03/steelhead_cloud_accelerator_saas_diagram.jpg",
			},
		},
		{
			name:  "query 'steelhead' which should match 1 event",
			query: "steelhead",
			expected: []string{
				"<33>5 1985-04-12T23:20:50.52Z test.com cron 304 - GET /wp-content/uploads/2012/03/steelhead_cloud_accelerator_saas_diagram.jpg",
			},
		},
	}

	sys := NewSystem(path)
	var expectedCount uint64
	for _, tt := range tests {
		t.Logf("starting test '%s'", tt.name)

		if tt.skip {
			t.Logf("skipping test '%s'", tt.name)
			continue
		}

		if tt.reset {
			sys = NewSystem(path)
			expectedCount = 0
		}

		ingestConn := sys.IngestConn()
		for _, e := range tt.events {
			en := e + "\n"
			n, err := ingestConn.Write([]byte(en))
			if err != nil {
				t.Fatalf("failed to write '%s' to Collector: %s", e, err.Error())
			}
			if n != len(en) {
				t.Fatalf("insufficient bytes written to Collector, exp: %d, wrote: %d", len(e), n)
			}
			expectedCount++
		}

		sys.e.waitForCount(expectedCount)

		results, err := sys.s.Search(tt.query)
		if err != nil {
			t.Errorf("failed to execute search query '%s': %s", tt.query, err.Error())
		}

		if err == nil && tt.expected != nil {
			if len(results) != len(tt.expected) {
				t.Fatalf("wrong number of results received for query, exp %d, got %d", len(tt.expected), len(results))
			}

			// Compare each result.
			for i := range results {
				if tt.expected[i] != results[i] {
					t.Errorf("result %d is wrong\nexp: '%s'\ngot: '%s'", i, tt.expected[i], results[i])
				}
			}
		}
	}
}

// Test_AllInOrder tests that a large number of log messages are returned in the correct order.
func Test_AllInOrder(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	path := tempPath()
	defer os.RemoveAll(path)

	lines := make([]string, 1000)
	for n := 0; n < len(lines); n++ {
		lines[n] = fmt.Sprintf("<33>5 %s test.com cron 304 - password accepted %d", time.Unix(int64(n), 0).UTC().Format(time.RFC3339), n)
	}

	sys := NewSystem(path)
	ingestConn := sys.IngestConn()
	for _, l := range lines {
		ln := l + "\n"
		n, err := ingestConn.Write([]byte(ln))
		if err != nil {
			t.Fatalf("failed to write '%s' to Collector: %s", l, err.Error())
		}
		if n != len(ln) {
			t.Fatalf("insufficient bytes written to Collector, exp: %d, wrote: %d", len(l), n)
		}
	}

	sys.e.waitForCount(uint64(len(lines)))

	query := "password"
	results, err := sys.s.Search(query)
	if err != nil {
		t.Fatalf("failed to execute search query '%s': %s", query, err.Error())
	}

	if len(results) != len(lines) {
		t.Fatalf("wrong number of results received for query, exp %d, got %d", len(lines), len(results))
	}

	// Compare each result.
	for i := range results {
		if lines[i] != results[i] {
			t.Errorf("result %d is wrong, exp: '%s', got: '%s'", i, lines[i], results[i])
		}
	}
}

// Test_AllInOrderShards tests that log messages are returned in the correct order, across shards.
func Test_AllInOrderShards(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	path := tempPath()
	defer os.RemoveAll(path)

	// Lines in order they should be returned in results.
	lines := []string{
		`<134>0 2015-05-05T23:50:17.025568+00:00 fisher apache-access - - 65.98.59.154 - - [05/May/2015:23:50:12 +0000] "GET /wp-login.php HTTP/1.0" 200 206 "-" "-"`,
		`<134>0 2015-05-06T01:24:41.232890+00:00 fisher apache-access - - 104.140.83.221 - - [06/May/2015:01:24:40 +0000] "GET /wp-login.php?action=register HTTP/1.0" 200 206 "http://www.philipotoole.com/" "Opera/9.80 (Windows NT 6.2; Win64; x64) Presto/2.12.388 Version/12.17"`,
		`<134>0 2015-05-06T01:24:41.232895+00:00 fisher apache-access - - 104.140.83.221 - - [06/May/2015:01:24:40 +0000] "GET /wp-login.php?action=register HTTP/1.1" 200 243 "http://www.philipotoole.com/wp-login.php?action=register" "Opera/9.80 (Windows NT 6.2; Win64; x64) Presto/2.12.388 Version/12.17"`,
		`<134>0 2015-05-06T02:47:54.612953+00:00 fisher apache-access - - 184.68.20.22 - - [06/May/2015:02:47:51 +0000] "GET /wp-login.php HTTP/1.1" 200 243 "-" "Mozilla/5.0 (Windows NT 6.1) AppleWebKit/537.1 (KHTML, like Gecko) Chrome/24.0.1309.0 Safari/537.17"`,
		`<134>0 2015-05-06T04:20:49.008609+00:00 fisher apache-access - - 193.104.41.186 - - [06/May/2015:04:20:46 +0000] "POST /wp-login.php HTTP/1.1" 200 206 "-" "Opera 10.00"`,
	}

	sys := NewSystem(path)
	ingestConn := sys.IngestConn()
	for _, l := range lines {
		ln := l + "\n"
		n, err := ingestConn.Write([]byte(ln))
		if err != nil {
			t.Fatalf("failed to write '%s' to Collector: %s", l, err.Error())
		}
		if n != len(ln) {
			t.Fatalf("insufficient bytes written to Collector, exp: %d, wrote: %d", len(l), n)
		}
	}

	sys.e.waitForCount(uint64(len(lines)))

	query := "login"
	results, err := sys.s.Search(query)
	if err != nil {
		t.Fatalf("failed to execute search query '%s': %s", query, err.Error())
	}
	for i := range results {
		if lines[i] != results[i] {
			t.Fatalf("results incorrect, expected:\n%s\n got:\n%s\n", strings.Join(lines, "\n"), strings.Join(results, "\n"))
		}
	}
}

type testEngine struct {
	*ekanite.Engine
}

// NewEngine returns an Engine suitable for end-to-end testing.
func NewEngine(path string) *testEngine {
	e := ekanite.NewEngine(path)
	return &testEngine{
		e,
	}
}

// waitForCount blocks until the number of indexed documents matches 'count'.
func (e *testEngine) waitForCount(count uint64) {
outer:
	for {
		select {
		case <-time.After(10 * time.Millisecond):
			n, err := e.Total()
			if err != nil {
				panic("failed to determine indexed count")
			}
			if n == count {
				break outer
			}
		}
	}
}

type testServer struct {
	*ekanite.Server
}

// NewEngine returns a query Server suitable for end-to-end testing.
func NewServer(addr string, e *testEngine) *testServer {
	return &testServer{ekanite.NewServer(addr, e)}
}

// Search performs the given search and returns the log lines in a slice.
func (s *testServer) Search(query string) ([]string, error) {
	conn, err := net.Dial("tcp", s.Addr().String())
	if err != nil {
		panic("unable to connect to query server")
	}

	n, err := conn.Write([]byte(query + "\n"))
	if err != nil {
		return nil, err
	}
	if n != len(query)+1 {
		return nil, fmt.Errorf("incorrect number of bytes written to query server, exp: %d, wrote: %d", len(query)+1, n)
	}

	var firstNewline bool
	results := make([]string, 0)
	connbuf := bufio.NewReader(conn)
	for {
		result, err := connbuf.ReadString('\n')
		if err != nil {
			return nil, err
		}
		if result == "\n" {
			if firstNewline {
				// No more results.
				break
			}
			firstNewline = true
			continue
		} else {
			firstNewline = false
		}
		results = append(results, strings.Trim(result, "\n"))
	}

	return results, nil
}

type testCollector struct {
	input.Collector
}

// NewCollector returns a new test TCP collector.
func NewCollector(addr string) *testCollector {
	return &testCollector{input.NewCollector("tcp", addr)}
}

type testBatcher struct {
	*ekanite.Batcher
}

// NewBatcher returns a new test batcher which indexes each event immediately.
func NewBatcher(e *testEngine) *testBatcher {
	return &testBatcher{ekanite.NewBatcher(e, 200, 100*time.Millisecond, 10000)}
}

// tempPath provides a path for temporary use.
func tempPath() string {
	f, _ := ioutil.TempFile("", "ekanite_")
	path := f.Name()
	f.Close()
	os.Remove(path)
	return path
}
