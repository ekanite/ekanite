package input

import (
	"encoding/json"
)

// Parse takes a raw message and returns a parsed message. If no match,
// nil is returned.
func Parse(raw string) map[string]string {
	j := make(map[string]string)
	dec := json.NewDecoder(string.NewReader(raw))
	if err := dec.Decode(&j); err != nil {
		stats.Add("unparsed", 1)
		return nil
	}
	stats.Add("parsed", 1)

	return &j
}
