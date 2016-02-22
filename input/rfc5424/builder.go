package rfc5424

import (
	"github.com/ekanite/ekanite/input/types"
)

type Builder struct{}

func (_ Builder) NewDelimiter() types.Delimiter {
	return NewDelimiter(256)
}

func (_ Builder) NewParser() types.Parser {
	return NewParser()
}
