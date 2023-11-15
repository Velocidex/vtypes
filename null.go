package vtypes

import (
	"io"

	"github.com/Velocidex/ordereddict"
	"www.velocidex.com/golang/vfilter"
)

// A parser that always returns NULL
type NullParser struct{}

func (self NullParser) New(profile *Profile, options *ordereddict.Dict) (Parser, error) {
	return NullParser{}, nil
}

func (self NullParser) Parse(
	scope vfilter.Scope, reader io.ReaderAt, offset int64) interface{} {
	return vfilter.Null{}
}
