package ecma404

import (
	"github.com/ekanite/ekanite/input/types"
)

type Tokenizer struct{}

func (_ Tokenizer) NewDelimiter() types.Delimiter {
	return NewDelimiter()
}

func (_ Tokenizer) NewParser() types.Parser {
	return NewParser()
}
