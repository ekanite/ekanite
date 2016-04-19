package parser

import (
	"strconv"
	"strings"
	"time"
)

type Rfc3339 struct {
	split []string
	fmted []int64
}

func (p *Parser) newRfc3339Parser() {
	p.rfc3339 = &Rfc3339{}
}

func (t *Rfc3339) parse(raw string) (string, error) {
	t.split = strings.Split(raw, ".")
	if err := t.fmt(); err != nil {
		return "", err
	}
	return time.Unix(t.fmted[0], t.fmted[1]).Format(time.RFC3339), nil
}

func (t *Rfc3339) fmt() error {
	for _, v := range t.split {
		p, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			stats.Add("FailedUnixTimestampConversion", 1)
			return unxConvErr
		}
		t.fmted = append(t.fmted, p)
	}
	if len(t.fmted) == 1 {
		t.fmted = append(t.fmted, 0)
	}
	return nil
}
