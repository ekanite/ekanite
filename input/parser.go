package input

import (
	"fmt"
	"github.com/ekanite/ekanite/parser"
	"log"
)

const (
	Rfc5424Standard   = "RFC5424"
	Rfc5424Name       = "syslog"
	WatchguardFirebox = "Watchguard"
	WatchguardName    = "M200"
)

type StatsCollector func(key string, delta int64)

type LogParser interface {
	Parse(raw []byte, result *map[string]interface{})
	CompileMatcher()
}

// A Parser parses the raw input as a map with a timestamp field.
type LogHandler struct {
	Fmt    string
	Raw    []byte
	Result map[string]interface{}
	Parser LogParser
	Stats func(key string, delta int64)
}

func supportedFormats() [][]string {
	return [][]string{{Rfc5424Name, Rfc5424Standard},
		{WatchguardName, WatchguardFirebox}}
}

// ValidFormat returns if the given format matches one of the possible formats.
func ValidFormat(f string) bool {

	l := len(supportedFormats())
	fmts := supportedFormats()

	for i := 0; i < l; i++ {
		if fmts[i][0] == f {
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

	if f == Rfc5424Name {
		p.Parser = &parser.RFC5424{}
	} else if f == WatchguardName {
		p.Parser = &parser.Watchguard{}
	} else {
		panic(fmt.Sprintf("no supported parser for input format %s", f))
	}

	log.Printf("input format parser created for %s", f)
	p.Stats = stats.Add
	p.Parser.CompileMatcher()
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
