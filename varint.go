package vtypes

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/Velocidex/ordereddict"
	"www.velocidex.com/golang/vfilter"
)

type VarInt struct {
	base uint64
	size int
}

func (self VarInt) Size() int {
	return self.size
}

func (self VarInt) Value() interface{} {
	return self.base
}

func (self VarInt) MarshalJSON() ([]byte, error) {
	return json.Marshal(self.base)
}

type SVarInt struct {
	VarInt
}

func (self SVarInt) Value() interface{} {
	return int64(self.base)
}

func (self SVarInt) MarshalJSON() ([]byte, error) {
	return json.Marshal(int64(self.base))
}

type Leb128Parser struct{}

func (self *Leb128Parser) New(profile *Profile, options *ordereddict.Dict) (Parser, error) {
	return &Leb128Parser{}, nil
}

func (self *Leb128Parser) DebugString(scope vfilter.Scope, offset int64, reader io.ReaderAt) string {
	return fmt.Sprintf("[Leb128] %#0x", self.Parse(scope, reader, offset))
}

func (self *Leb128Parser) Parse(scope vfilter.Scope, reader io.ReaderAt, offset int64) interface{} {
	// We only support uint64 - max size 64 / 7 = 10  bytes
	buf := make([]byte, 10)

	n, err := reader.ReadAt(buf, offset)
	if n == 0 || (err != nil && !errors.Is(err, io.EOF)) {
		return 0
	}

	var res uint64
	for i := 0; i < len(buf); i++ {
		next := buf[i] & 0x80
		value := uint64(buf[i] & 0x7f)
		res |= value << (i * 7)
		if next == 0 {
			return VarInt{
				base: res,
				size: i + 1,
			}
		}
	}

	return VarInt{
		base: res,
		size: len(buf),
	}
}

type Sleb128Parser struct {
	Leb128Parser
}

func (self *Sleb128Parser) Parse(scope vfilter.Scope, reader io.ReaderAt, offset int64) interface{} {
	res := self.Leb128Parser.Parse(scope, reader, offset)
	res_vi, ok := res.(VarInt)
	if ok {
		return &SVarInt{res_vi}
	}
	return 0
}
