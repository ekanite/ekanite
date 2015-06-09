package ekanite

import (
	"os"
	"sort"
	"testing"
	"time"
)

type testDoc struct {
	id       DocID
	line     string
	priority string
}

// ID returns a sufficiently long doc ID, embedding the testDoc's ID.
func (t testDoc) ID() DocID {
	return t.id
}
func (t testDoc) Data() interface{} {
	data := struct {
		Line     string
		Priority string
	}{
		Line:     t.line,
		Priority: t.priority,
	}
	return data
}
func (t testDoc) Source() []byte { return []byte(t.line) }

func TestIndex_NewIndex(t *testing.T) {
	path := tempPath()
	defer os.RemoveAll(path)

	start := parseTime("2006-01-02T22:04:00Z")
	i, err := NewIndex(path, start.UTC(), start.Add(time.Hour), 4)
	if err != nil || i == nil {
		t.Fatalf("failed to create new index at %s %s", path, err)
	}
	defer i.Close()

	existOrFail(t, path+"/20060102_2204")
	existOrFail(t, path+"/20060102_2204/endtime")
	containsOrFail(t, path+"/20060102_2204/endtime", "20060102_2304")
	existOrFail(t, path+"/20060102_2204/0")
	existOrFail(t, path+"/20060102_2204/1")
	existOrFail(t, path+"/20060102_2204/2")
	existOrFail(t, path+"/20060102_2204/3")

	tests := []struct {
		timestamp     string
		shouldContain bool
	}{
		{
			timestamp:     "2006-01-02T22:03:59Z",
			shouldContain: false,
		},
		{
			timestamp:     "2006-01-02T22:04:00Z",
			shouldContain: true,
		},
		{
			timestamp:     "2006-01-02T22:04:01Z",
			shouldContain: true,
		},
		{
			timestamp:     "2006-01-02T23:04:00Z",
			shouldContain: false,
		},
		{
			timestamp:     "2006-01-02T23:05:00Z",
			shouldContain: false,
		},
	}
	for n, tt := range tests {
		if tt.shouldContain != i.Contains(parseTime(tt.timestamp)) {
			t.Fatalf("timestamp #%d '%s' failed Contains test", n, tt.timestamp)
		}
	}
}

func TestIndex_Expired(t *testing.T) {
	path := tempPath()
	defer os.RemoveAll(path)

	retentionPeriod := 24 * time.Hour
	startTime := parseTime("2006-01-02T22:04:00Z").UTC()
	endTime := parseTime("2006-01-02T22:04:00Z").Add(time.Hour).UTC()
	i, _ := NewIndex(path, startTime, endTime, 1)
	defer i.Close()

	tests := []struct {
		timestamp time.Time
		expired   bool
	}{
		{
			timestamp: endTime,
			expired:   false,
		},
		{
			timestamp: endTime.Add(time.Hour),
			expired:   false,
		},
		{
			timestamp: endTime.Add(retentionPeriod),
			expired:   false,
		},
		{
			timestamp: endTime.Add(2 * retentionPeriod),
			expired:   true,
		},
	}
	for n, tt := range tests {
		if i.Expired(tt.timestamp, retentionPeriod) != tt.expired {
			t.Errorf("Expired test #%d: index %s has incorrect expired status for time %s",
				n, path, tt.timestamp)
		}
	}
}

func TestIndex_OpenIndex(t *testing.T) {
	path := tempPath()
	defer os.RemoveAll(path)

	start := parseTime("2006-01-04T00:04:00Z")
	startTime := start.UTC()
	endTime := start.Add(time.Hour).UTC()

	n, err := NewIndex(path, startTime, endTime, 4)
	if err != nil {
		t.Fatalf("failed to create new index for Open() test at %s %s", path, err)
	}
	n.Close() // Close it, or it can't be opened.

	i, err := OpenIndex(path + "/20060104_0004")
	if err != nil {
		t.Fatalf("failed to open index for Open() test at %s %s", path, err)
	}
	defer i.Close()

	if i.Path() != path+"/20060104_0004" {
		t.Fatalf("opened index not at expected path, got: %s", i.Path())
	}
	if i.StartTime() != startTime {
		t.Fatalf("start time of opended index is wrong, expected %s, got %s", startTime, i.StartTime())
	}
	if i.EndTime() != endTime {
		t.Fatalf("end time time of opended index is wrong, expected %s, got %s", endTime, i.EndTime())
	}
	if len(i.Shards) != 4 {
		t.Fatalf("wrong number of shards, expected 4, got %d", len(i.Shards))
	}
	if n, err := i.Total(); n != 0 || err != nil {
		t.Fatalf("new index failed doc count check, found %d, %s", n, err.Error())
	}
}

func TestIndex_DeleteIndex(t *testing.T) {
	path := tempPath()
	defer os.RemoveAll(path)

	now := time.Now().UTC()
	i, err := NewIndex(path, now, now.Add(time.Hour), 4)
	if err != nil {
		t.Fatalf("failed to create new index for Open() test at %s %s", path, err)
	}

	if err := DeleteIndex(i); err != nil {
		t.Fatalf("failed to delete index at %s", path)
	}
	notExistOrFail(t, path+"/20060104_0004")
}

func TestIndex_Index(t *testing.T) {
	path := tempPath()
	defer os.RemoveAll(path)
	now := time.Now().UTC()
	i, _ := NewIndex(path, now, now, 2)

	d1 := testDoc{id: DocID("00000000000000000000000000001234"), line: "password accepted for user root"}
	d2 := testDoc{id: DocID("00000000000000000000000000005678"), line: "GET /index.html"}
	d3 := testDoc{id: DocID("00000000000000000000000000009abc"), line: "sshd version 4.0"}

	if err := i.Index([]Document{d1, d2, d3}); err != nil {
		t.Fatalf("failed to index batch into index at %s", path)
	}

	n, err := i.Total()
	if err != nil {
		t.Fatalf("failed to get number of documents in index at %s", path)
	}
	if n != 3 {
		t.Fatalf("wrong number of documents in index at %s", path)
	}
}

func TestIndex_Document(t *testing.T) {
	path := tempPath()
	defer os.RemoveAll(path)
	now := time.Now().UTC()
	i, _ := NewIndex(path, now, now, 2)

	id := DocID("1234")
	source := "password accepted for user root"
	d1 := testDoc{id: DocID(id), line: source}

	if err := i.Index([]Document{d1}); err != nil {
		t.Fatalf("failed to index batch into index at %s", path)
	}

	b, err := i.Document(id)
	if err != nil {
		t.Fatalf(`failed to retrieve document ID "%s": %s`, id, err.Error())
	}
	if string(b) != source {
		t.Fatalf(`source of retrieved document not correct, got: "%s", exp: "%s"`, string(b), source)
	}
}

func TestIndex_IndexSimpleSearch(t *testing.T) {
	path := tempPath()
	defer os.RemoveAll(path)
	now := time.Now().UTC()
	i, _ := NewIndex(path, now, now, 4)

	d1 := testDoc{id: DocID("00000000000000000000000000001234"), line: "auth password accepted for user philip"}
	d2 := testDoc{id: DocID("0000000000000000000000000000ABCD"), line: "auth password invalid for user root"}
	d3 := testDoc{id: DocID("00000000000000000000000000005678"), line: "auth GET /index.html"}
	d4 := testDoc{id: DocID("0000000000000000000000000000DEAD"), line: "pamd POST", priority: "CRITICAL"}
	d5 := testDoc{id: DocID("0000000000000000000000000000BEEF"), line: "pamd PUT", priority: "ERROR"}

	if err := i.Index([]Document{d1, d2, d3, d4, d5}); err != nil {
		t.Fatalf("failed to index batch into index at %s", path)
	}

	tests := []struct {
		Phrase string
		IDs    []string
	}{
		{
			Phrase: "missing",
			IDs:    []string{},
		},
		{
			Phrase: "priority:ERROR",
			IDs:    []string{},
		},
		{
			Phrase: "philip",
			IDs:    []string{"00000000000000000000000000001234"},
		},
		{
			Phrase: "Priority:CRITICAL",
			IDs:    []string{"0000000000000000000000000000DEAD"},
		},
		// This doesn't return all lines because boolean requires at least 1 + search
		{
			Phrase: "-Priority:CRITICAL",
			IDs:    []string{},
		},
		{
			Phrase: "get",
			IDs:    []string{"00000000000000000000000000005678"},
		},
		{
			Phrase: "password",
			IDs:    []string{"00000000000000000000000000001234", "0000000000000000000000000000ABCD"},
		},
		{
			Phrase: "+password +accepted",
			IDs:    []string{"00000000000000000000000000001234"},
		},
		{
			Phrase: "+philip +root",
			IDs:    []string{},
		},
		{
			Phrase: "+password +get",
			IDs:    []string{},
		},
		{
			Phrase: "+password -accepted",
			IDs:    []string{"0000000000000000000000000000ABCD"},
		},
		{
			Phrase: "+password -ACCEPTED",
			IDs:    []string{"0000000000000000000000000000ABCD"},
		},
		{
			Phrase: "auth",
			IDs: []string{
				"00000000000000000000000000001234",
				"0000000000000000000000000000ABCD",
				"00000000000000000000000000005678"},
		},
	}

	f := func(testName string, idx *Index) {
		t.Logf("running '%s'", testName)
		for _, tt := range tests {
			// Perform the search
			IDs, err := idx.Search(tt.Phrase)
			if err != nil {
				t.Errorf("error while searching for '%s': %s", tt.Phrase, err.Error())
			}
			if len(IDs) != len(tt.IDs) {
				t.Errorf("wrong number of hits for search '%s', got %d, expected %d",
					tt.Phrase, len(IDs), len(tt.IDs))
				continue
			}

			// Compare doc IDs.
			exp := make([]string, len(IDs))
			got := make([]string, len(IDs))
			for n := range IDs {
				exp[n] = tt.IDs[n]
				got[n] = string(IDs[n])
			}
			sort.Strings(exp)
			sort.Strings(got)
			for n := range exp {
				if got[n] != exp[n] {
					t.Errorf("search return incorrect doc IDs for search '%s'", tt.Phrase)
					t.Logf("got: %v, exp: %v", got, exp)
					break // No point checking any further.
				}
			}
		}
	}

	// Test searching a newly-created index.
	f("new index search test", i)

	// Close, re-open the index, and try all searches again.
	i.Close()
	i, err := OpenIndex(i.Path())
	if err != nil {
		t.Fatalf("failed to re-open index at %s: %s", i.Path(), err.Error())
	}
	f("re-opened index search test", i)
}

func TestIndex_Shard(t *testing.T) {
	path := tempPath()
	defer os.RemoveAll(path)
	now := time.Now().UTC()
	i, _ := NewIndex(path, now, now.Add(time.Hour), 2)
	s := i.Shard(DocID("00000000000000000000000000001234"))
	if s == nil {
		t.Fatalf("failed to get shard for doc 1234")
	}
	if s != i.Shards[0] && s != i.Shards[1] {
		t.Fatalf("wrong shard returned for doc 1234")
	}

	u := i.Shard(DocID("00000000000000000000000000001234"))
	if u != s {
		t.Fatalf("different shard returned for same doc")
	}
}

func TestShard_NewOpenClose(t *testing.T) {
	path := tempPath()
	defer os.RemoveAll(path)

	s := NewShard(path)
	if s == nil {
		t.Fatalf("failed to create new shard at %s", path)
	}

	if err := s.Open(); err != nil {
		t.Fatalf("failed to open shard at %s: %s", path, err.Error())
	}

	if err := s.Close(); err != nil {
		t.Fatalf("failed to close shard at %s: %s", path, err.Error())
	}

	return
}

func TestShard_Index(t *testing.T) {
	path := tempPath()
	defer os.RemoveAll(path)

	s := NewShard(path)
	s.Open()

	var c uint64
	var err error
	d1 := testDoc{id: DocID("00000000000000000000000000001234"), line: "this is a log line"}
	d2 := testDoc{id: DocID("00000000000000000000000000005678"), line: "this is a log line"}
	d3 := testDoc{id: DocID("00000000000000000000000000009abc"), line: "this is a log line"}

	// Index a single document
	if err := s.Index([]Document{d1}); err != nil {
		t.Fatalf("failed to index single document: %s", err.Error())
	}
	if c, _ := s.Total(); c != 1 {
		t.Fatalf("shard indexed count incorrect, got %d, expected 1", c)
	}
	c, err = s.Total()
	if err != nil {
		t.Fatalf("failed to get shard doc count: %s", err.Error())
	}
	if c != 1 {
		t.Fatalf("shard doc count incorrect, got %d, expected 1", c)
	}

	// Index a couple more.
	if err := s.Index([]Document{d2, d3}); err != nil {
		t.Fatalf("failed to index single document: %s", err.Error())
	}
	if c, _ := s.Total(); c != 3 {
		t.Fatalf("shard indexed count incorrect, got %d, expected 1", c)
	}
	c, err = s.Total()
	if err != nil {
		t.Fatalf("failed to get shard doc count: %s", err.Error())
	}
	if c != 3 {
		t.Fatalf("shard doc count incorrect, got %d, expected 1", c)
	}

	// Test fetching the indexed documents by ID.
	source, err := s.Document(d1.ID())
	if err != nil {
		t.Fatalf("failed to get document %s", d1.ID())
	}
	if string(source) != d1.line {
		t.Fatalf("retrieved document is not identical, got: '%s', expected: '%s'", string(source), d1.line)
	}

	s.Close()
}
