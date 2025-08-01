package vtypes

import (
	"context"
	"fmt"
	"io"

	"github.com/Velocidex/ordereddict"
	"www.velocidex.com/golang/vfilter"
)

type BitFieldOptions struct {
	StartBit int64  `json:"start_bit" vfilter:"optional,field=start_bit,doc=The start bit in the int to read"`
	EndBit   int64  `json:"end_bit" vfilter:"optional,field=end_bit,doc=The end bit in the int to read"`
	Type     string `json:"type" vfilter:"required,field=type,doc=The underlying type of the bit field"`
}

type BitFieldParser struct {
	options BitFieldOptions

	parser Parser
}

func (self *BitFieldParser) New(profile *Profile, options *ordereddict.Dict) (Parser, error) {
	if options == nil {
		return nil, fmt.Errorf("BitField parser requires an options dict")
	}

	result := &BitFieldParser{}
	ctx := context.Background()
	err := ParseOptions(ctx, options, &result.options)
	if err != nil {
		return nil, err
	}

	if result.options.EndBit == 0 {
		result.options.EndBit = 64
	}

	if result.options.StartBit < 0 || result.options.StartBit > 64 {
		return nil, fmt.Errorf("BitField start_bit should be between 0-64")
	}

	if result.options.EndBit < 0 || result.options.EndBit > 64 {
		return nil, fmt.Errorf("BitField end_bit should be between 0-64")
	}

	if result.options.EndBit <= result.options.StartBit {
		return nil, fmt.Errorf(
			"BitField end_bit (%v) should be larger than start_bit (%v)",
			result.options.EndBit, result.options.StartBit)
	}

	// Type must be available at definition time because bit fields
	// can not operate on custome types.
	result.parser, err = profile.GetParser(result.options.Type, nil)
	return result, err
}

func (self *BitFieldParser) Parse(
	scope vfilter.Scope, reader io.ReaderAt, offset int64) interface{} {

	if self.parser == nil {
		return vfilter.Null{}
	}

	result := int64(0)
	value, ok := to_int64(self.parser.Parse(scope, reader, offset))
	if !ok {
		return 0
	}
	for i := self.options.StartBit; i < self.options.EndBit; i++ {
		result |= value & (1 << uint8(i))
	}

	return result >> self.options.StartBit
}
