package ecma404

import (
	"github.com/ekanite/ekanite/input/types"
)

type Builder struct{}

func (_ Builder) NewDelimiter() types.Delimiter {
	return NewDelimiter()
}

func (_ Builder) NewParser() types.Parser {
	return NewParser()
}
