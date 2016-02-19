package ecma404

import (
	"encoding/json"
	"expvar"
	"strconv"
	"strings"
	"time"

	"github.com/ekanite/ekanite/input/types"
)

var stats = expvar.NewMap("ecma404")

// A Parser parses Json messages.
type Parser struct {
}

// Message represents a fully parsed Json message.
type Message struct {
	data map[string]string
}

// Returns an initialized Parser.
func NewParser() *Parser {
	p := &Parser{}
	return p
}

// Parse takes a raw message and returns a parsed message. If no match,
// nil is rturned.
func (p *Parser) Parse(raw string) types.Message {
	m := Message{data: make(map[string]string)}
	dec := json.NewDecoder(strings.NewReader(raw))
	if err := dec.Decode(&m.data); err != nil {
		stats.Add("unparsed", 1)
		return nil
	}
	_, ok := m.data["timestamp"]
	if !ok {
		stats.Add("unparsed", 1)
		return nil
	}
	stats.Add("parsed", 1)

	return m
}

func (m Message) GetTimestamp() string {
	unixInt, err := strconv.ParseInt(m.data["timestamp"], 10, 64)
	if err != nil {
		panic(err)
	}
	parsedTime := time.Unix(unixInt, 0).Format(time.RFC3339)
	return string(parsedTime)
}
