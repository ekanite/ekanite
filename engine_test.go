package ekanite

import (
	"bufio"
	"io"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/ekanite/ekanite/input/types"
)

type TestIndexer struct {
	BatchesRx int
	EventsRx  int
}

func (t *TestIndexer) Index(b []*Event) error {
	t.BatchesRx++
	t.EventsRx += len(b)
	return nil
}

// TestBatcher_MultiEvent tests that a single event is sent when the batch size is 1.
func TestBatcher_SingleEvent(t *testing.T) {
	e := newInputEvent("", time.Now())
	i := &TestIndexer{}
	b := NewBatcher(i, 1, time.Hour, 0)

	c := make(chan error)
	err := b.Start(c)
	if err != nil {
		t.Fatalf("failed start batcher: %s", err.Error())
	}

	b.C() <- e
	err = <-c
	if err != nil {
		t.Fatalf("failed to send single event: %s", err.Error())
	}

	if i.BatchesRx != 1 || i.EventsRx != 1 {
		t.Fatalf("indexer failed to receive correct number of events: batches: %d, events: %d", i.BatchesRx, i.EventsRx)
	}
}

// TestBatcher_MultiEvent tests that events are sent when the batch size is reached.
func TestBatcher_MultiEvent(t *testing.T) {
	e := newInputEvent("", time.Now())
	i := &TestIndexer{}
	b := NewBatcher(i, 2, time.Hour, 0)

	c := make(chan error)
	err := b.Start(c)
	if err != nil {
		t.Fatalf("failed start batcher: %s", err.Error())
	}

	for n := 0; n < 2; n++ {
		b.C() <- e
	}
	err = <-c
	if err != nil {
		t.Fatalf("failed to send two events: %s", err.Error())
	}

	if i.BatchesRx != 1 || i.EventsRx != 2 {
		t.Fatalf("indexer failed to receive correct number of events: batches: %d, events: %d", i.BatchesRx, i.EventsRx)
	}
}

// TestBatcher_MultiEventMultiBatch tests that multiple batches are sent as expected.
func TestBatcher_MultiEventMultiBatch(t *testing.T) {
	e := newInputEvent("", time.Now())
	i := &TestIndexer{}
	b := NewBatcher(i, 2, time.Hour, 0)

	c := make(chan error, 100) // Buffered channel required.
	err := b.Start(c)
	if err != nil {
		t.Fatalf("failed start batcher: %s", err.Error())
	}

	for n := 0; n < 4; n++ {
		b.C() <- e
	}

	nErr := 0
	for {
		err := <-c
		if err != nil {
			t.Fatalf("failed to send four events: %s", err.Error())
		}
		nErr++
		if nErr == 2 {
			break
		}
	}

	if i.BatchesRx != 2 || i.EventsRx != 4 {
		t.Fatalf("indexer failed to receive correct number of events: batches: %d, events: %d", i.BatchesRx, i.EventsRx)
	}
}

// TestBatcher_Timeout ensures a batch is sent when the timeout expires.
func TestBatcher_Timeout(t *testing.T) {
	e := newInputEvent("", time.Now())
	i := &TestIndexer{}
	b := NewBatcher(i, 250, 100*time.Millisecond, 0)

	c := make(chan error)
	err := b.Start(c)
	if err != nil {
		t.Fatalf("failed start batcher: %s", err.Error())
	}

	b.C() <- e
	err = <-c
	if err != nil {
		t.Fatalf("failed to send event on timeout: %s", err.Error())
	}

	if i.BatchesRx != 1 || i.EventsRx != 1 {
		t.Fatalf("indexer failed to receive correct number of events: batches: %d, events: %d", i.BatchesRx, i.EventsRx)
	}

	// Ensure a second event works.
	b.C() <- e
	err = <-c
	if err != nil {
		t.Fatalf("failed to send second event on timeout: %s", err.Error())
	}

	if i.BatchesRx != 2 || i.EventsRx != 2 {
		t.Fatalf("indexer failed to receive correct number of events: batches: %d, events: %d", i.BatchesRx, i.EventsRx)
	}
}

// TestBatcher_AllModes tests that a combination of batch size and timeouts all work.
func TestBatcher_AllModes(t *testing.T) {
	e := newInputEvent("", time.Now())
	i := &TestIndexer{}
	b := NewBatcher(i, 3, 100*time.Millisecond, 0)

	c := make(chan error, 100) // Lots of room for error responses.
	err := b.Start(c)
	if err != nil {
		t.Fatalf("failed start batcher: %s", err.Error())
	}

	// 2 batches, and 1 timeout.
	for n := 0; n < 6; n++ {
		b.C() <- e
	}
	for n := 0; n < 2; n++ {
		b.C() <- e
	}

	for n := 0; n < 3; n++ {
		err = <-c
	}

	if i.BatchesRx != 3 || i.EventsRx != 8 {
		t.Fatalf("indexer failed to receive correct number of events: batches: %d, events: %d", i.BatchesRx, i.EventsRx)
	}
}

func TestEngine_New(t *testing.T) {
	dataDir := tempPath()
	defer os.RemoveAll(dataDir)

	e := NewEngine(dataDir)
	if e == nil {
		t.Fatal("failed to create new engine")
	}
	if e.Path() != dataDir {
		t.Fatalf("got unexpected path for datadir, got %s, expected %s", e.Path(), dataDir)
	}
	if e.IndexDuration != DefaultIndexDuration || e.NumShards != DefaultNumShards {
		t.Fatal("new engine created with incorrect defaults")
	}
}

func TestEngine_Open(t *testing.T) {
	dataDir := tempPath()
	defer os.RemoveAll(dataDir)

	e := NewEngine(dataDir)
	err := e.Open()
	if e == nil {
		t.Fatalf("failed to open engine: %s", err.Error())
	}
}

func TestEngine_Close(t *testing.T) {
	dataDir := tempPath()
	defer os.RemoveAll(dataDir)

	e := NewEngine(dataDir)
	err := e.Close()
	if e == nil {
		t.Fatalf("failed to close engine: %s", err.Error())
	}
}

// TestEngine_Indexing tests that indexing is OK when an index is comprised of
// different numbers of shards.
func TestEngine_Indexing(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	tests := []struct {
		numShards     int
		indexDuration time.Duration
	}{
		{
			1,
			time.Hour,
		},
		{
			4,
			time.Hour,
		},
		{
			1,
			24 * time.Hour,
		},
		{
			4,
			24 * time.Hour,
		},
		{
			5,
			24 * time.Hour,
		},
		{
			17,
			24 * time.Hour,
		},
	}

	for _, tt := range tests {
		func() {
			dataDir := tempPath()
			defer os.RemoveAll(dataDir)
			e := newEngine(dataDir, tt.numShards, tt.indexDuration)
			defer e.Close()
			testEngine_Index(t, e)
		}()
		func() {
			dataDir := tempPath()
			defer os.RemoveAll(dataDir)
			e := newEngine(dataDir, tt.numShards, tt.indexDuration)
			defer e.Close()
			testEngine_IndexPrime(t, e)
		}()
		func() {
			dataDir := tempPath()
			defer os.RemoveAll(dataDir)
			e := newEngine(dataDir, tt.numShards, tt.indexDuration)
			defer e.Close()
			testEngine_indexForReferenceTime(t, e)
		}()
	}
}

func TestEngine_IndexThenSearch(t *testing.T) {
	dataDir := tempPath()
	defer os.RemoveAll(dataDir)
	e := NewEngine(dataDir)

	line1 := "auth password accepted for user philip"
	ev1 := newIndexableEvent(line1, parseTime("1982-02-05T04:43:00Z"))
	line2 := "auth password accepted for user root"
	ev2 := newIndexableEvent(line2, parseTime("1982-02-05T04:43:01Z"))
	line3 := "auth password rejected for user philip"
	ev3 := newIndexableEvent(line3, parseTime("1982-02-05T04:43:02Z"))

	if err := e.Index([]*types.Event{ev1, ev2, ev3}); err != nil {
		t.Fatalf("failed to index events: %s", err.Error())
	}
	total, err := e.Total()
	if err != nil {
		t.Fatalf("failed to get engine total doc count: %s", err.Error())
	}
	if total != 3 {
		t.Fatalf("engine total doc count, got %d, expected 3", total)
	}

	c, err := e.Search("philip")
	if err != nil {
		t.Fatalf("failed to search for indexed event: %s", err.Error())
	}

	if s, _ := <-c; s != line1 {
		t.Fatalf(`returned source incorrect. got: "%s", exp "%s"`, s, line1)
	}
	if s, _ := <-c; s != line3 {
		t.Fatalf(`returned source incorrect. got: "%s", exp "%s"`, s, line3)
	}

	if _, more := <-c; more {
		t.Fatalf("more documents unexpectedly available")
	}
}

func TestEngine_createIndexForReferenceTime(t *testing.T) {
	dataDir := tempPath()
	defer os.RemoveAll(dataDir)

	e := NewEngine(dataDir)
	if err := e.Open(); err != nil {
		t.Fatalf("failed to open index at %s for indexing test: %s", dataDir, err.Error())
	}
	e.IndexDuration = 2 * time.Hour

	rt := parseTime("1982-02-05T04:43:00Z")
	idx, err := e.createIndexForReferenceTime(rt)
	if err != nil {
		t.Fatalf("failed to create index for reference time %s", rt)
	}
	if idx == nil {
		t.Fatalf("nil index created for reference time %s", rt)
	}

	if idx.startTime != parseTime("1982-02-05T04:00:00Z") || idx.endTime != parseTime("1982-02-05T06:00:00Z") {
		t.Fatalf("index created for reference time %s has wrong limits", rt)
	}
}

func TestEngine_RetentionEnforcement(t *testing.T) {
	dataDir := tempPath()
	defer os.RemoveAll(dataDir)

	e := NewEngine(dataDir)
	if err := e.Open(); err != nil {
		t.Fatalf("failed to open index at %s for indexing test: %s", dataDir, err.Error())
	}
	e.RetentionPeriod = 24 * time.Hour

	now := time.Now().UTC()
	idx, _ := e.createIndex(now.Add(-1*time.Hour), now)
	_, _ = e.createIndex(now.Add(-48*time.Hour), now.Add(-47*time.Hour))

	if len(e.indexes) != 2 {
		t.Fatalf("engine has wrong number of indexes for retention test pre-enforcement")
	}

	e.enforceRetention()
	if len(e.indexes) != 1 {
		t.Fatalf("engine has wrong number of indexes for retention test post-enforcement")
	}
	if e.indexes[0] != idx {
		t.Fatalf("retention enforcement deleted wrong index")
	}
}

func testEngine_indexForReferenceTime(t *testing.T, e *Engine) {
	start1 := parseTime("1982-02-05T04:00:00Z")
	start2 := parseTime("1982-02-05T05:00:00Z")
	start3 := parseTime("1982-02-05T06:00:00Z")
	idx1, err := e.createIndex(start1, start2)
	if err != nil {
		t.Fatalf("failed to create index starting at %s: %s", start1, err.Error())
	}
	if idx1 == nil {
		t.Fatalf("nil index created for %s", start1)
	}

	idx2, err := e.createIndex(start2, start3)
	if err != nil {
		t.Fatalf("failed to create index starting at %s: %s", start2, err.Error())
	}
	if idx2 == nil {
		t.Fatalf("nil index created for %s", start2)
	}

	// Create an index with the same start time as an existing index. This
	// should be allowed, though it doesn't make much sense.
	start4 := parseTime("1982-02-05T00:30:00Z")
	idx3, err := e.createIndex(start3, start4)
	if err != nil {
		t.Fatalf("failed to create index starting at %s: %s", start3, err.Error())
	}
	if idx3 == nil {
		t.Fatalf("nil index created for %s", start3)
	}
	if len(e.indexes) != 3 {
		t.Fatalf("unexpected number of indexes in existence, expected 3, got %d", len(e.indexes))
	}

	tests := []struct {
		timestamp time.Time
		index     *Index
	}{
		{
			start1.Add(-time.Hour).UTC(),
			nil,
		},
		{
			start1,
			idx1,
		},
		{
			start2,
			idx2,
		},
		{
			start2.Add(time.Hour).UTC(),
			nil,
		},
	}

	for n, tt := range tests {
		if i := e.indexForReferenceTime(tt.timestamp); i != tt.index {
			t.Fatalf("Test %d: got wrong index for timestamp %s", n, tt.timestamp)
		}
	}
}

func testEngine_Index(t *testing.T, e *Engine) {
	ev := newIndexableEvent("this is event 1234", parseTime("1982-02-05T04:43:00Z"))

	if err := e.Index([]*Event{ev}); err != nil {
		t.Fatalf("failed to index event %v: %s", ev, err.Error())
	}
	total, err := e.Total()
	if err != nil {
		t.Fatalf("failed to get engine total doc count: %s", err.Error())
	}
	if total != 1 {
		t.Fatalf("engine total doc count, got %d, expected 1", total)
	}
}

// testEngine_IndexPrime tests batch indexing of a batch prime in size. This guarantees
// different size batches, unless the shard count is 1 or the batch size.
func testEngine_IndexPrime(t *testing.T, e *Engine) {
	rt := parseTime("1982-02-05T04:43:00Z")
	batchSize := 97
	batch := make([]*Event, 0, batchSize)
	for b := 0; b < batchSize; b++ {
		ev := newIndexableEvent("this is event 1234", rt)
		rt = rt.Add(time.Second)
		batch = append(batch, ev)
	}

	if err := e.Index(batch); err != nil {
		t.Fatalf("failed to index %d events: %s", batchSize, err.Error())
	}
	total, err := e.Total()
	if err != nil {
		t.Fatalf("failed to get engine total doc count: %s", err.Error())
	}
	if total != uint64(batchSize) {
		t.Fatalf("engine total doc count, got %d, expected %d", total, batchSize)
	}
}

func newEngine(path string, numShards int, indexDuration time.Duration) *Engine {
	e := NewEngine(path)
	e.Open()
	e.NumShards = numShards
	e.IndexDuration = indexDuration
	return e
}

// tempPath provides a path for temporary use.
func tempPath() string {
	f, _ := ioutil.TempFile("", "ekanite_")
	path := f.Name()
	f.Close()
	os.Remove(path)
	return path
}

// existOrFail checks for the existence of the given file path.
func existOrFail(t *testing.T, path string) {
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("error accessing %s: %s", path, err.Error())
	}
}

// notExistOrFail checks for the non-existence of the given file path.
func notExistOrFail(t *testing.T, path string) {
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("NO error accessing %s: %s", path, err.Error())
	}
}

// containsOrFail checks for the existence of the line in the file
// at the given path, or fails.
func containsOrFail(t *testing.T, path, contents string) {
	if f, err := os.Open(path); err != nil {
		t.Fatalf("unable to open %s: %s", path, err.Error())
	} else {
		defer f.Close()
		r := bufio.NewReader(f)
		if s, err := r.ReadString('\n'); err != nil && err != io.EOF {
			t.Fatalf("unable to read line in %s: %s", path, err.Error())
		} else {
			if s != contents {
				t.Fatalf("unexpected contents in %s, got %s, expected %s", path, s, contents)
			}
		}
	}
}

// parseTime parses the given string and returns a time.
func parseTime(timestamp string) time.Time {
	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		panic(err.Error())
	}
	return t
}

func newInputEvent(Line string, refTime time.Time) *types.Event {
	return &types.Event{
		Text:          Line,
		ReceptionTime: refTime,
	}
}

func newIndexableEvent(line string, refTime time.Time) *types.Event {
	return &types.Event{
		&types.Event{
			Text:          line,
			ReceptionTime: refTime,
		},
	}
}
