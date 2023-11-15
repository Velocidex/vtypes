package vtypes

import (
	"fmt"
	"io"

	"github.com/Velocidex/ordereddict"
	"www.velocidex.com/golang/vfilter"
)

type BitField struct {
	StartBit int64  `json:"start_bit"`
	EndBit   int64  `json:"end_bit"`
	Type     string `json:"type"`

	parser Parser
}

func (self *BitField) New(profile *Profile, options *ordereddict.Dict) (Parser, error) {
	parser_type, pres := options.GetString("type")
	if !pres {
		return nil, fmt.Errorf("BitField parser requires a type in the options")
	}

	parser, err := profile.GetParser(parser_type, ordereddict.NewDict())
	if err != nil {
		return nil, fmt.Errorf("BitField parser requires a type in the options: %w", err)
	}

	start_bit, pres := options.GetInt64("start_bit")
	if !pres || start_bit < 0 {
		start_bit = 0
	}

	end_bit, pres := options.GetInt64("end_bit")
	if !pres || end_bit > 64 {
		end_bit = 64
	}

	return &BitField{
		StartBit: start_bit,
		EndBit:   end_bit,
		parser:   parser,
	}, nil
}

func (self *BitField) Parse(
	scope vfilter.Scope, reader io.ReaderAt, offset int64) interface{} {

	result := int64(0)
	value, ok := to_int64(self.parser.Parse(scope, reader, offset))
	if !ok {
		return 0
	}
	for i := self.StartBit; i < self.EndBit; i++ {
		result |= value & (1 << uint8(i))
	}

	return result >> self.StartBit
}
