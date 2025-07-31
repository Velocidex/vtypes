package vtypes

import (
	"errors"
	"fmt"
	"io"

	"github.com/Velocidex/ordereddict"
	"www.velocidex.com/golang/vfilter"
)

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
	if n == 0 || (err != nil && !errors.Is(err, io.EOF)) {
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
