package ekanite

import (
	"bufio"
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/blevesearch/bleve"
)

const (
	endTimeFileName  = "endtime"
	indexNameLayout  = "20060102_1504"
	maxSearchHitSize = 10000
)

// DocID is a string, with the following configuration. It's 32-characters long, encoding 2
// 64-bit unsigned integers. When sorting DocIDs, the first 16 characters, reading from the
// left hand side represent the most significant 64-bit number. And therefore the next 16
// characters represent the least-significant 64-bit number.
type DocID string
type DocIDs []DocID

func (a DocIDs) Len() int { return len(a) }
func (a DocIDs) Less(i, j int) bool {
	x := a[i]
	y := a[j]

	mustParse := func(s string) uint64 {
		w, err := strconv.ParseUint(s, 16, 64)
		if err != nil {
			panic(fmt.Sprintf("failed to parse 64-bit word: %s", err.Error()))
		}
		return w
	}

	msw0 := mustParse(string(x[0:16]))
	lsw0 := mustParse(string(x[16:32]))
	msw1 := mustParse(string(y[0:16]))
	lsw1 := mustParse(string(y[16:32]))

	if msw0 == msw1 {
		return lsw0 < lsw1
	}
	return msw0 < msw1
}
func (a DocIDs) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// Document specifies the interface required by an object if it is to be indexed.
type Document interface {
	ID() DocID
	Data() interface{}
	Source() []byte
}

// Index represents a collection of shards. It contains data for a specific time range.
type Index struct {
	path      string    // Path to shard data
	startTime time.Time // Start-time inclusive for this index
	endTime   time.Time // End-time exclusive for this index

	Shards []*Shard         // Individual bleve indexes
	Alias  bleve.IndexAlias // All bleve indexes as one reference, for search
}
type Indexes []*Index

// Indexes are ordered by decreasing end time. If two indexes have the same
// start time, then order by decreasing start time. This means that the first
// index in the slice covers the latest time range.
func (i Indexes) Len() int { return len(i) }
func (i Indexes) Less(u, v int) bool {
	if i[u].endTime.After(i[v].endTime) {
		return true
	} else if i[u].endTime.After(i[v].endTime) {
		return false
	} else {
		return i[u].startTime.After(i[v].startTime)
	}
}
func (i Indexes) Swap(u, v int) { i[u], i[v] = i[v], i[u] }

// NewIndex returns an Index for the given start and end time, with the requested shards. It
// returns an error if an index already exists at the path.
func NewIndex(path string, startTime, endTime time.Time, numShards int) (*Index, error) {
	indexName := startTime.UTC().Format(indexNameLayout)
	indexPath := filepath.Join(path, indexName)
	durationPath := filepath.Join(indexPath, endTimeFileName)

	// Create the directory for the index, if it doesn't already exist.
	if _, err := os.Stat(indexPath); err == nil && os.IsNotExist(err) {
		return nil, fmt.Errorf("index already exists at %s", indexPath)
	}
	if err := os.MkdirAll(indexPath, 0755); err != nil {
		return nil, err
	}

	// Insert the file with the duration information.
	f, err := os.Create(durationPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	_, err = f.WriteString(endTime.UTC().Format(indexNameLayout))
	if err != nil {
		return nil, err
	}

	// Create the shards.
	shards := make([]*Shard, 0, numShards)
	for n := 0; n < numShards; n++ {
		s := NewShard(filepath.Join(indexPath, strconv.Itoa(n)))
		if err := s.Open(); err != nil {
			return nil, err
		}
		shards = append(shards, s)
	}

	// Create alias for searching.
	alias := bleve.NewIndexAlias()
	for _, s := range shards {
		alias.Add(s.b)
	}

	// Index is ready to go.
	return &Index{
		path:      indexPath,
		Shards:    shards,
		Alias:     alias,
		startTime: startTime,
		endTime:   endTime,
	}, nil
}

// OpenIndex opens an existing index, at the given path.
func OpenIndex(path string) (*Index, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to access index at %s", path)
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("index %s path is not a directory", path)
	}

	// Get the start time and end time.
	startTime, err := time.Parse(indexNameLayout, fi.Name())
	if err != nil {
		return nil, fmt.Errorf("unable to determine start time of index: %s", err.Error())
	}

	var endTime time.Time
	if f, err := os.Open(filepath.Join(path, endTimeFileName)); err != nil {
		return nil, fmt.Errorf("unable to open end time file for index: %s", err.Error())
	} else {
		defer f.Close()
		r := bufio.NewReader(f)
		if s, err := r.ReadString('\n'); err != nil && err != io.EOF {
			return nil, fmt.Errorf("unable to determine end time of index: %s", err.Error())
		} else {
			endTime, err = time.Parse(indexNameLayout, s)
			if err != nil {
				return nil, fmt.Errorf("unable to parse end time from '%s': %s", s, err.Error())
			}
		}
	}

	// Get an index directory listing.
	d, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	fis, err := d.Readdir(0)
	if err != nil {
		return nil, err
	}

	// Open the shards.
	shards := make([]*Shard, 0)
	for _, fi := range fis {
		if !fi.IsDir() || strings.HasPrefix(fi.Name(), ".") {
			continue
		}
		s := NewShard(filepath.Join(path, fi.Name()))
		if err := s.Open(); err != nil {
			return nil, err
		}
		shards = append(shards, s)
	}

	// Create alias for searching.
	alias := bleve.NewIndexAlias()
	for _, s := range shards {
		alias.Add(s.b)
	}

	// Index is ready to go.
	return &Index{
		path:      path,
		Shards:    shards,
		Alias:     alias,
		startTime: startTime,
		endTime:   endTime,
	}, nil
}

// Path returns the path to storage for the index.
func (i *Index) Path() string { return i.path }

// StartTime returns the inclusive start time of the index.
func (i *Index) StartTime() time.Time { return i.startTime }

// EndTime returns the exclusive end time of the index.
func (i *Index) EndTime() time.Time { return i.endTime }

// Expired returns whether the index has expired at the given time, if the
// retention period is r.
func (i *Index) Expired(t time.Time, r time.Duration) bool {
	return i.endTime.Add(r).Before(t)
}

// Total returns the number of documents in the index.
func (i *Index) Total() (uint64, error) {
	var total uint64
	for _, s := range i.Shards {
		t, err := s.Total()
		if err != nil {
			return 0, err
		}
		total += t
	}
	return total, nil
}

// Contains returns whether the index's time range includes the given
// reference time.
func (i *Index) Contains(t time.Time) bool {
	return (t.Equal(i.startTime) || t.After(i.startTime)) && t.Before(i.endTime)
}

// Index indexes the slice of documents in the index. It takes care of all shard routing.
func (i *Index) Index(documents []Document) error {
	var wg sync.WaitGroup
	shardBatches := make(map[*Shard][]Document, 0)
	for _, d := range documents {
		shard := i.Shard(d.ID())
		shardBatches[shard] = append(shardBatches[shard], d)
	}

	// Index each batch in parallel.
	for shard, batch := range shardBatches {
		wg.Add(1)
		go func(s *Shard, b []Document) {
			defer wg.Done()
			s.Index(b)
		}(shard, batch)
	}
	wg.Wait()
	return nil
}

// Search performs a search of the index using the given query. Returns IDs of documents
// which satisfy all queries. Returns Doc IDs in sorted order, ascending.
func (i *Index) Search(q string) (DocIDs, error) {
	query := bleve.NewQueryStringQuery(q)
	searchRequest := bleve.NewSearchRequest(query)
	searchRequest.Size = maxSearchHitSize
	searchResults, err := i.Alias.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	docIDs := make(DocIDs, 0, len(searchResults.Hits))
	for _, d := range searchResults.Hits {
		docIDs = append(docIDs, DocID(d.ID))
	}
	sort.Sort(docIDs)
	return docIDs, nil
}

// Document returns the source from the index for the given ID.
func (i *Index) Document(id DocID) ([]byte, error) {
	s := i.Shard(id)
	if s == nil {
		return nil, fmt.Errorf("document %s not found", id)
	}
	return s.Document(id)
}

// Close closes the index.
func (i *Index) Close() error {
	for _, s := range i.Shards {
		if err := s.Close(); err != nil {
			return err
		}
	}
	return nil
}

// DeleteIndex deletes the index.
func DeleteIndex(i *Index) error {
	_ = i.Close()
	return os.RemoveAll(i.path)
}

// Shard returns the shard from the index, for the given doc ID.
func (i *Index) Shard(docId DocID) *Shard {
	hasher := fnv.New32a()
	hasher.Write([]byte(docId))
	v := hasher.Sum32() % uint32(len(i.Shards))
	return i.Shards[v]
}

// Shard is a the basic data store for indexed data. Indexing operations are not
// goroutine safe, and only 1 indexing operation should occur at one time.
type Shard struct {
	path string
	b    bleve.Index // Underlying bleve index
}

// NewShard returns a shard using the data at the given path.
func NewShard(path string) *Shard {
	return &Shard{
		path: path,
	}
}

// Opens the shard. If no data exists at the shard's path, an empty shard
// will be created.
func (s *Shard) Open() error {
	_, err := os.Stat(s.path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to check existence of shard")
	} else if os.IsNotExist(err) {
		mapping, err := buildIndexMapping()
		if err != nil {
			return err
		}
		s.b, err = bleve.New(s.path, mapping)
		if err != nil {
			return err
		}
	} else {
		s.b, err = bleve.Open(s.path)
		if err != nil {
			return err
		}
	}
	return nil
}

// Close closes the shard.
func (s *Shard) Close() error {
	if err := s.b.Close(); err != nil {
		return err
	}
	return nil
}

// Index indexes a slice of Documents in the shard.
func (s *Shard) Index(documents []Document) error {
	batch := s.b.NewBatch()

	for _, d := range documents {
		if err := batch.Index(string(d.ID()), d.Data()); err != nil {
			return err // XXX return errors en-masse
		}
		batch.SetInternal([]byte(d.ID()), d.Source())
	}
	if err := s.b.Batch(batch); err != nil {
		return err
	}

	return nil
}

// Total returns the number of events in the shard.
func (s *Shard) Total() (uint64, error) {
	return s.b.DocCount()
}

// Document returns the source from the shard for the given ID.
func (s *Shard) Document(id DocID) ([]byte, error) {
	source, err := s.b.GetInternal([]byte(id))
	if err != nil {
		return nil, err
	}
	return source, nil
}

func buildIndexMapping() (*bleve.IndexMapping, error) {
	var err error

	// Create the index mapping, configure the analyzer, and set as default.
	indexMapping := bleve.NewIndexMapping()
	err = indexMapping.AddCustomTokenizer("ekanite_tk",
		map[string]interface{}{
			"regexp": `[^\W_]+`,
			"type":   `regexp`,
		})
	if err != nil {
		return nil, err
	}
	err = indexMapping.AddCustomAnalyzer("ekanite",
		map[string]interface{}{
			"type":          `custom`,
			"char_filters":  []interface{}{},
			"tokenizer":     `ekanite_tk`,
			"token_filters": []interface{}{`to_lower`},
		})
	if err != nil {
		return nil, err
	}
	indexMapping.DefaultAnalyzer = "ekanite"

	// Create field-specific mappings.

	simpleJustIndexed := bleve.NewTextFieldMapping()
	simpleJustIndexed.Store = false
	simpleJustIndexed.IncludeInAll = true // XXX Move to false when using AST
	simpleJustIndexed.IncludeTermVectors = false

	timeJustIndexed := bleve.NewDateTimeFieldMapping()
	timeJustIndexed.Store = false
	timeJustIndexed.IncludeInAll = false
	timeJustIndexed.IncludeTermVectors = false

	articleMapping := bleve.NewDocumentMapping()

	// Connect field mappings to fields.
	articleMapping.AddFieldMappingsAt("Message", simpleJustIndexed)
	articleMapping.AddFieldMappingsAt("ReferenceTime", timeJustIndexed)
	articleMapping.AddFieldMappingsAt("ReceptionTime", timeJustIndexed)

	// Tell the index about field mappings.
	indexMapping.DefaultMapping = articleMapping

	return indexMapping, nil
}
