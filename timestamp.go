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

type FatTimestamp struct {
	profile *Profile
}

func (self *FatTimestamp) New(profile *Profile, options *ordereddict.Dict) (Parser, error) {
	return &FatTimestamp{profile: profile}, nil
}

func (self *FatTimestamp) Parse(
	scope vfilter.Scope, reader io.ReaderAt, offset int64) interface{} {

	parser, err := self.profile.GetParser("uint32", nil)
	if err != nil {
		return vfilter.Null{}
	}

	date_int, ok := to_int64(parser.Parse(scope, reader, offset))
	if !ok {
		return vfilter.Null{}
	}

	// Dos times are stored as 2 uint16 numbers - first the date
	// then the time so swap them.
	date_int = ((date_int & 0xFFFF) << 16) + (date_int >> 16)

	year := 1980 + (date_int >> 25)
	month := (date_int >> 21) & ((1 << 4) - 1)
	day := (date_int >> 16) & ((1 << 6) - 1)
	hour := (date_int >> 11) & ((1 << 6) - 1)
	min := (date_int >> 5) & ((1 << 7) - 1)
	sec := (date_int) & ((1 << 6) - 1)

	return time.Date(int(year), time.Month(month), int(day),
		int(hour), int(min), int(sec), 0, time.UTC)
}
