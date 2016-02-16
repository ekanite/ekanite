package input

import (
	"bytes"
	"strconv"
	"strings"
)

const SEPERATOR = ":"

type BufferLength struct {
	Raw    []byte
	Parsed uint64
}

func init() {
	buf := bytes.NewBuffer([]byte(""))
	len := new(BufferLength)
	isBuf := false
}

func Push(b bytes) (bool, string) {
	if isBuffer == true {
		buf.WriteByte(b)
		if uint(buf.Len()) == len.Parsed {
			var buffer string = buff.String()
			ResetBuff()
			return buffer, true
		}
	} else {
		if string(b) == SEPERATOR {
			len.Parsed, _ = strconv.ParseUint(strings.Trim(len.Raw.String(), "\n"), 10, 64)
			isBuf = true
		} else {
			len.Raw.WriteByte(b)
		}
		return nil, false
	}
}

func ResetBuff() {
	buf.Reset()
	len = new(BufferLength)
	isBuf = false
}
