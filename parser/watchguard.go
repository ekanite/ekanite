package parser

import (
	"regexp"
	"strconv"
)

type Watchguard struct {
	matcher *regexp.Regexp
}

var watchguardStats = func(key string, delta int64){}

func (p *Watchguard) Stats(callback func(key string, delta int64)) {
	watchguardStats = callback
}

//
// Test at regex-golang.appspot.com/assets/html/index.html
//
//([ADFJMNOS][a-z]{2}\s[0-9]{1,2}\s[0-2][0-9]:[0-5]?[0-9]:[0-5]?[0-9])\s([^ ]+)\s([^ ]+)\s\(([^ ]+)\)\s([^ ]+)\smsg_id=\"([0-9\-]+)\"\s([^ ]+)\s([^ ]+)\s([^ ]+)\s(.+$)
//
func (p *Watchguard) Init() {
	leading := `(?s)`
	pri := `<([0-9]{1,3})>`
	local_dtg := `([ADFJMNOS][a-z]{2}\s[0-9]{1,2}\s[0-2][0-9]:[0-5]?[0-9]:[0-5]?[0-9])`
	modelName := `([^ ]+)`
	serialno := `([^ ]+)`
	utc_dtg := `([^ ]+)`
	comp := `([^ ]+)`
	msgId := `([0-9\-]+)`
	disposition := `([^ ]+)`
	srcIntf := `([^ ]+)`
	dstIntf := `([^ ]+)`
	msg := `(.+$)`

	p.matcher = regexp.MustCompile(leading + pri + local_dtg + `\s` + modelName + `\s` + serialno + `\s\(` +
		utc_dtg + `\)\s` + comp + `\smsg_id=\"` + msgId + `\"\s` + disposition + `\s` + srcIntf + `\s` +
		dstIntf + `\s` + msg)
}

func (p *Watchguard) Parse(raw []byte, result *map[string]interface{}) {
	m := p.matcher.FindStringSubmatch(string(raw))
	if m == nil || len(m) != 12 {
		watchguardStats("WatchguardM200Unparsed", 1)
		return
	}
	watchguardStats("WatchguardM200Parsed", 1)
	pri, _ := strconv.Atoi(m[1])
	*result = map[string]interface{}{
		"priority":    pri,
		"local_dtg":   m[2],
		"model_name":  m[3],
		"serial_no":   m[4],
		"timestamp":   m[5],
		"component":   m[6],
		"msg_id":      m[7],
		"disposition": m[8],
		"src_intf":    m[9],
		"dst_intf":    m[10],
		"message":     m[11],
	}
}
