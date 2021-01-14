package vtypes

import (
	"fmt"
	"io"
	"time"

	"github.com/Velocidex/ordereddict"
	"www.velocidex.com/golang/vfilter"
)

type EpochTimestamp struct {
	parser Parser
	factor int64
}

func (self *EpochTimestamp) New(profile *Profile, options *ordereddict.Dict) (Parser, error) {
	parser_type := "uint64"
	factor := int64(1)
	pres := false
	if options != nil {
		parser_type, pres = options.GetString("type")
		if !pres {
			return nil, fmt.Errorf("EpochTimestamp parser requires a type in the options")
		}

		factor, pres = options.GetInt64("factor")
		if !pres {
			factor = 1
		}
	}

	parser, err := profile.GetParser(parser_type, ordereddict.NewDict())
	if err != nil {
		return nil, fmt.Errorf("EpochTimestamp parser requires a type in the options: %w", err)
	}

	return &EpochTimestamp{
		parser: parser,
		factor: factor,
	}, nil
}

func (self *EpochTimestamp) Parse(
	scope vfilter.Scope, reader io.ReaderAt, offset int64) interface{} {
	value, ok := to_int64(self.parser.Parse(scope, reader, offset))
	if !ok {
		return vfilter.Null{}
	}

	return time.Unix(value/self.factor, value%self.factor)
}

type WinFileTime struct {
	parser Parser
	factor int64
}

func (self *WinFileTime) New(profile *Profile, options *ordereddict.Dict) (Parser, error) {
	parser_type := "uint64"
	factor := int64(1)
	pres := false
	if options != nil {
		parser_type, pres = options.GetString("type")
		if !pres {
			return nil, fmt.Errorf("WinfileTime parser requires a type in the options")
		}

		factor, pres = options.GetInt64("factor")
		if !pres {
			factor = 1
		}
	}

	parser, err := profile.GetParser(parser_type, ordereddict.NewDict())
	if err != nil {
		return nil, fmt.Errorf("WinfileTime parser requires a type in the options: %w", err)
	}

	return &WinFileTime{
		parser: parser,
		factor: factor,
	}, nil
}

func (self *WinFileTime) Parse(
	scope vfilter.Scope, reader io.ReaderAt, offset int64) interface{} {
	value, ok := to_int64(self.parser.Parse(scope, reader, offset))
	if !ok {
		return vfilter.Null{}
	}

	return time.Unix((value/self.factor/10000000)-11644473600, 0)
}
