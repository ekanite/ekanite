package input

import (
	"strconv"
	"strings"
	"time"
)

var err error

type Timestamp struct {
	Unparsed []string
	Fromated []int64
	Parsed   time.Time
}

func NewTiemstamp() *Timestamp {

	return &Timestamp{}

}

func (t *Timestamp) Parse(ts string) (bool, time.Time) {

	t.Unparsed = strings.Split(ts, ".")

	if ok := t.format(); !ok {

		return false, t.Parsed

	}

	u := time.Unix(t.Fromated[0], t.Fromated[1]).String()

	t.Parsed, err = time.Parse(time.RFC3339, u)

	if err != nil {

		stats.Add("UnparsedTimestamp", 1)
		return false, t.Parsed

	}

	return true, t.Parsed

}

func (t *Timestamp) format() bool {

	for _, ts := range t.Unparsed {

		p, err := strconv.ParseInt(ts, 10, 64)

		if err != nil {

			stats.Add("UnformatedTimestampPart", 1)
			return false

		}

		t.Fromated = append(t.Fromated, p)

	}

	t.ensureLength()
	return true

}

func (t *Timestamp) ensureLength() {

	if len(t.Fromated) == 1 {

		t.Fromated = append(t.Fromated, 0)

	}

}
