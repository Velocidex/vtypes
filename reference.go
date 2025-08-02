package vtypes

import (
	"encoding/json"
	"io"

	"www.velocidex.com/golang/vfilter"
)

// A reference is a wrapper around a struct member which can contain
// metadata about it.

type StructFieldReference struct {
	// Offset to the start of the struct
	offset int64
	reader io.ReaderAt
	scope  vfilter.Scope
	field  string

	parser *ParseAtOffset
}

// The offset within the struct
func (self *StructFieldReference) RelOffset() int64 {
	return self.parser.getOffset(self.scope)
}

func (self *StructFieldReference) Start() int64 {
	return self.offset + self.parser.getOffset(self.scope)
}

func (self *StructFieldReference) Size() int {
	return self.parser.Size(self.scope, self.reader, self.offset)
}

func (self *StructFieldReference) End() int64 {
	return self.Start() + int64(self.Size())
}

func (self *StructFieldReference) Value() interface{} {
	return self.parser.Parse(self.scope, self.reader, self.offset)
}

func (self *StructFieldReference) MarshalJSON() ([]byte, error) {
	return json.Marshal(self.Value())
}
