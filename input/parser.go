package input

import (
	"fmt"
	"github.com/ekanite/ekanite/parser"
	"log"
)

const (
	RFC5424Standard   = "RFC5424"
	RFC5424Name       = "syslog"
	WatchguardFirebox = "Watchguard"
	WatchguardName    = "M200"
)

type StatsCollector func(key string, delta int64)

type LogParser interface {
	Parse(raw []byte, result *map[string]interface{})
	Init()
}

// A Parser parses the raw input as a map with a timestamp field.
type LogHandler struct {
	Fmt    string
	Raw    []byte
	Result map[string]interface{}
	Parser LogParser
	Stats  func(key string, delta int64)
}

func supportedFormats() [][]string {
	return [][]string{{RFC5424Name, RFC5424Standard},
		{WatchguardName, WatchguardFirebox}}
}

// ValidFormat returns if the given format matches one of the possible formats.
func ValidFormat(f string) bool {
	for _, v := range supportedFormats() {
		if v[0] == f || v[1] == f {
			return true
		}
	}

	return false
}

// NewParser returns a new Parser instance.
func NewParser(f string) (*LogHandler, error) {
	if !ValidFormat(f) {
		return nil, fmt.Errorf("%s is not a valid parser format", f)
	}

	var p = &LogHandler{}

	if f == RFC5424Name || f == RFC5424Standard {
		p.Parser = &parser.RFC5424{}
		p.Fmt = RFC5424Standard
	} else if f == WatchguardName || f == WatchguardFirebox {
		p.Parser = &parser.Watchguard{}
		p.Fmt = WatchguardFirebox
	}

	log.Printf("input format parser created for %s", f)
	p.Stats = stats.Add
	p.Parser.Init()
	return p, nil
}

// Parse the given byte slice.
func (p *LogHandler) Parse(b []byte) bool {
	p.Result = map[string]interface{}{}
	p.Raw = b
	p.Parser.Parse(p.Raw, &p.Result)
	if len(p.Result) == 0 {
		return false
	}
	return true
}
