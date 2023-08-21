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

	return time.Unix(value/self.factor, value%self.factor).UTC()
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
	if err != nil || parser == nil {
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

	return time.Unix((value/self.factor/10000000)-11644473600, 0).UTC()
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

	// https://docs.microsoft.com/en-us/windows/win32/api/winbase/nf-winbase-dosdatetimetofiletime
	fat_date := date_int & 0xFFFF
	fat_time := date_int >> 16

	// Bits 9-15
	year := 1980 + (fat_date >> 9)

	// Bits 5-8
	month := (fat_date >> 5) & ((1 << 4) - 1)

	// Bits 0-4
	day := fat_date & ((1 << 5) - 1)

	// Bits 11 - 15
	hour := (fat_time >> 11)

	// Bits 5-10
	min := (fat_time >> 5) & ((1 << 6) - 1)

	// Bits 0-4 divided by 2
	sec := (fat_time & ((1 << 5) - 1)) * 2

	return time.Date(int(year), time.Month(month), int(day),
		int(hour), int(min), int(sec), 0, time.UTC).UTC()
}
