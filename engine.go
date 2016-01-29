package ekanite

import (
	"expvar"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ekanite/ekanite/input"
)

const (
	DefaultNumShards       = 16
	DefaultIndexDuration   = 24 * time.Hour
	DefaultRetentionPeriod = 24 * time.Hour

	RetentionCheckInterval = time.Hour
)

var (
	stats = expvar.NewMap("engine")
)

// EventIndex is the interface a system than can index events must implement.
type EventIndexer interface {
	Index(events []*Event) error
}

// Batcher accepts "input events", and once it has a certain number, or a certain amount
// of time has passed, sends those as indexable Events to an Indexer. It also supports a
// maximum number of unprocessed Events it will keep pending. Once this limit is reached,
// it will not accept anymore until outstanding Events are processed.
type Batcher struct {
	indexer  EventIndexer
	size     int
	duration time.Duration

	c chan *input.Event
}

// NewBatcher returns a Batcher for EventIndexer e, a batching size of sz, a maximum duration
// of dur, and a maximum outstanding count of max.
func NewBatcher(e EventIndexer, sz int, dur time.Duration, max int) *Batcher {
	return &Batcher{
		indexer:  e,
		size:     sz,
		duration: dur,
		c:        make(chan *input.Event, max),
	}
}

// Start starts the batching process.
func (b *Batcher) Start(errChan chan<- error) error {
	go func() {
		batch := make([]*Event, 0, b.size)
		timer := time.NewTimer(b.duration)
		timer.Stop() // Stop any first firing.

		send := func() {
			err := b.indexer.Index(batch)
			if err != nil {
				stats.Add("batchIndexedError", 1)
				return
			}
			stats.Add("batchIndexed", 1)
			stats.Add("eventsIndexed", int64(len(batch)))
			if errChan != nil {
				errChan <- err
			}
			batch = make([]*Event, 0, b.size)
		}

		for {
			select {
			case event := <-b.c:
				idxEvent := &Event{
					event,
				}
				batch = append(batch, idxEvent)
				if len(batch) == 1 {
					timer.Reset(b.duration)
				}
				if len(batch) == b.size {
					timer.Stop()
					send()
				}
			case <-timer.C:
				stats.Add("batchTimeout", 1)
				send()
			}
		}
	}()

	return nil
}

// C returns the channel on the batcher to which events should be sent.
func (b *Batcher) C() chan<- *input.Event {
	return b.c
}

// Engine is the component that performs all indexing.
type Engine struct {
	path            string        // Path to all indexed data
	NumShards       int           // Number of shards to use when creating an index.
	IndexDuration   time.Duration // Duration of created indexes.
	RetentionPeriod time.Duration // How long after Index end-time to hang onto data.

	mu      sync.RWMutex
	indexes Indexes

	open bool
	done chan struct{}
	wg   sync.WaitGroup

	Logger *log.Logger
}

// NewEngine returns a new indexing engine, which will use any data located at path.
func NewEngine(path string) *Engine {
	return &Engine{
		path:            path,
		NumShards:       DefaultNumShards,
		IndexDuration:   DefaultIndexDuration,
		RetentionPeriod: DefaultRetentionPeriod,
		done:            make(chan struct{}),
		Logger:          log.New(os.Stderr, "[engine] ", log.LstdFlags),
	}
}

// Open opens the engine.
func (e *Engine) Open() error {
	if err := os.MkdirAll(e.path, 0755); err != nil {
		return err
	}
	d, err := os.Open(e.path)
	if err != nil {
		return fmt.Errorf("failed to open engine: %s", err.Error())
	}

	fis, err := d.Readdir(0)
	if err != nil {
		return err
	}

	// Open all indexes.
	for _, fi := range fis {
		if !fi.IsDir() || strings.HasPrefix(fi.Name(), ".") {
			continue
		}
		indexPath := filepath.Join(e.path, fi.Name())
		i, err := OpenIndex(indexPath)
		if err != nil {
			log.Printf("engine failed to open at index %s: %s", indexPath, err.Error())
			return err
		}
		log.Printf("engine opened index with %d shard(s) at %s", len(i.Shards), indexPath)
		e.indexes = append(e.indexes, i)
		sort.Sort(e.indexes)
	}

	e.wg.Add(1)
	go e.runRetentionEnforcement()

	e.open = true
	return nil
}

// Close closes the engine.
func (e *Engine) Close() error {
	if !e.open {
		return nil
	}

	for _, i := range e.indexes {
		if err := i.Close(); err != nil {
			return err
		}
	}

	close(e.done)
	e.wg.Wait()

	e.open = false
	return nil
}

// Total returns the total number of documents indexed.
func (e *Engine) Total() (uint64, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var total uint64
	for _, i := range e.indexes {
		t, err := i.Total()
		if err != nil {
			return 0, err
		}
		total += t
	}
	return total, nil
}

// runRetentionEnforcement periodically run retention enforcement.
func (e *Engine) runRetentionEnforcement() {
	defer e.wg.Done()
	for {
		select {
		case <-e.done:
			return

		case <-time.After(RetentionCheckInterval):
			e.Logger.Print("retention enforcement commencing")
			stats.Add("retentionEnforcementRun", 1)
			e.enforceRetention()
		}
	}
}

// enforceRetention removes indexes which have aged out.
func (e *Engine) enforceRetention() {
	e.mu.Lock()
	defer e.mu.Unlock()

	filtered := e.indexes[:0]
	for _, i := range e.indexes {
		if i.Expired(time.Now().UTC(), e.RetentionPeriod) {
			if err := DeleteIndex(i); err != nil {
				e.Logger.Printf("retention enforcement failed to delete index %s: %s", i.path, err.Error())
			} else {
				e.Logger.Printf("retention enforcement deleted index %s", i.path)
				stats.Add("retentionEnforcementDeletions", 1)
			}
		} else {
			filtered = append(filtered, i)
		}
	}
	e.indexes = filtered
	return
}

// indexForReferenceTime returns an index suitable for indexing an event
// for the given reference time. Must be called under RLock.
func (e *Engine) indexForReferenceTime(t time.Time) *Index {
	for _, i := range e.indexes {
		if i.Contains(t) {
			return i
		}
	}
	return nil
}

// createIndex creates an index with a given start and end time and adds the
// created index to the Engine's store. It must be called under lock.
func (e *Engine) createIndex(startTime, endTime time.Time) (*Index, error) {
	// There cannot be two indexes with the same start time, since this would mean
	// two indexes with the same path. So if an index already exists with the requested
	// start time, use that index's end time as the start time.
	var idx *Index
	for _, i := range e.indexes {
		if i.startTime == startTime {
			idx = i
			break
		}
	}
	if idx != nil {
		startTime = idx.endTime // XXX This could still align with another start time! Needs some sort of loop.
		assert(!startTime.After(endTime), "new start time after end time")
	}

	i, err := NewIndex(e.path, startTime, endTime, e.NumShards)
	if err != nil {
		return nil, err
	}
	e.indexes = append(e.indexes, i)
	sort.Sort(e.indexes)

	e.Logger.Printf("index %s created with %d shards, start time: %s, end time: %s",
		i.Path(), e.NumShards, i.StartTime(), i.EndTime())
	return i, nil
}

// createIndexForReferenceTime creates an index suitable for indexing an event at the given
// reference time.
func (e *Engine) createIndexForReferenceTime(rt time.Time) (*Index, error) {
	start := rt.Truncate(e.IndexDuration).UTC()
	end := start.Add(e.IndexDuration).UTC()
	return e.createIndex(start, end)
}

// Index indexes a batch of Events. It blocks until all processing has completed.
func (e *Engine) Index(events []*Event) error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var wg sync.WaitGroup

	// De-multiplex the batch into sub-batches, one sub-batch for each Index.
	subBatches := make(map[*Index][]Document, 0)

	for _, ev := range events {
		index := e.indexForReferenceTime(ev.ReferenceTime())
		if index == nil {
			func() {
				// Take a RWLock, check again, and create a new index if necessary.
				// Doing this in a function makes lock management foolproof.
				e.mu.RUnlock()
				defer e.mu.RLock()
				e.mu.Lock()
				defer e.mu.Unlock()

				index = e.indexForReferenceTime(ev.ReferenceTime())
				if index == nil {
					var err error
					index, err = e.createIndexForReferenceTime(ev.ReferenceTime())
					if err != nil || index == nil {
						panic(fmt.Sprintf("failed to create index for %s: %s", ev.ReferenceTime(), err))
					}
				}
			}()
		}

		if _, ok := subBatches[index]; !ok {
			subBatches[index] = make([]Document, 0)
		}
		subBatches[index] = append(subBatches[index], ev)
	}

	// Index each batch in parallel.
	for index, subBatch := range subBatches {
		wg.Add(1)
		go func(i *Index, b []Document) {
			defer wg.Done()
			i.Index(b)
		}(index, subBatch)
	}
	wg.Wait()
	return nil
}

// Search performs a search.
func (e *Engine) Search(query string) (<-chan string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	stats.Add("queriesRx", 1)

	// Buffer channel to control how many docs are sent back. XXX Will this allow
	// the client to control? Possibly.
	c := make(chan string, 1)

	go func() {
		// Sequentially search each index, starting with the earliest in time.
		// This could be done in parallel but more sorting would be required.

		for i := len(e.indexes) - 1; i >= 0; i-- {
			e.Logger.Printf("searching index %s", e.indexes[i].Path())
			ids, err := e.indexes[i].Search(query)
			if err != nil {
				e.Logger.Println("error performing search:", err.Error())
				break
			}
			for _, id := range ids {
				b, err := e.indexes[i].Document(id)
				if err != nil {
					e.Logger.Println("error getting document:", err.Error())
					break
				}
				stats.Add("docsIDsRetrived", 1)
				c <- string(b) // There is excessive byte-slice-to-strings here.
			}
		}
		close(c)
	}()

	return c, nil
}

// Path returns the path to the indexed data directory.
func (e *Engine) Path() string {
	return e.path
}

// assert will panic with a given formatted message if the given condition is false.
func assert(condition bool, msg string, v ...interface{}) {
	if !condition {
		panic(fmt.Sprintf("assert failed: "+msg, v...))
	}
}
