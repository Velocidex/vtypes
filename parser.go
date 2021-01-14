// Implements a binary parsing system.
package vtypes

import (
	"fmt"
	"io"

	"github.com/Velocidex/ordereddict"
	"www.velocidex.com/golang/vfilter"
)

// Parsers are objects which know how to parse a particular
// type. Parsers are instantiated once and reused many times.
type Parser interface {
	Parse(scope vfilter.Scope, reader io.ReaderAt, offset int64) interface{}

	// Given options, this returns a new configured parser
	New(profile *Profile, options *ordereddict.Dict) (Parser, error)
}

type Sizer interface {
	Size() int
}

// Return the start and end of the object
type Starter interface {
	Start() int64
}

type Ender interface {
	End() int64
}

// Parse various sizes of ints.
type IntParser struct {
	type_name string
	size      int
	converter func(buf []byte) interface{}
}

// IntParser does not take options
func (self *IntParser) New(profile *Profile, options *ordereddict.Dict) (Parser, error) {
	return self, nil
}

func (self *IntParser) Size() int {
	return self.size
}

func (self *IntParser) DebugString(scope vfilter.Scope, offset int64, reader io.ReaderAt) string {
	return fmt.Sprintf("[%s] %#0x",
		self.type_name, self.Parse(scope, reader, offset))
}

func (self *IntParser) Parse(scope vfilter.Scope, reader io.ReaderAt, offset int64) interface{} {
	buf := make([]byte, 8)

	n, err := reader.ReadAt(buf, offset)
	if n == 0 || err != nil {
		return 0
	}
	return self.converter(buf)
}

func NewIntParser(type_name string, size int, converter func(buf []byte) interface{}) *IntParser {
	return &IntParser{
		type_name: type_name,
		size:      size,
		converter: converter,
	}
}
