package input

import "encoding/json"

type Ecma404 struct{}

func (P *Parser) newEcma404Parser() *Ecma404 {
	return &Ecma404{}
}

func (j *Ecma404) parse(raw []byte, result *map[string]interface{}) {
	if err := json.Unmarshal(raw, result); err != nil {
		stats.Add("ecma404Unparsed", 1)
		return
	}
	stats.Add("ecma404Parsed", 1)
}
