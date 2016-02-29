package rfc5424

import (
	"github.com/ekanite/ekanite/input/types"
)

type Tokenizer struct{}

func (_ Tokenizer) NewDelimiter() types.Delimiter {
	return NewDelimiter(256)
}

func (_ Tokenizer) NewParser() types.Parser {
	return NewParser()
}
