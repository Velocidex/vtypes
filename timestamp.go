package vtypes

import (
	"fmt"
	"io"
	"time"

	"github.com/Velocidex/ordereddict"
	"www.velocidex.com/golang/vfilter"
)

type EpochTimestampOptions struct {
	Type        string
	TypeOptions *ordereddict.Dict
	Factor      int64
}

type EpochTimestamp struct {
	options EpochTimestampOptions
	profile *Profile
	parser  Parser
}

func (self *EpochTimestamp) New(profile *Profile, options *ordereddict.Dict) (Parser, error) {
	var pres bool
	result := &EpochTimestamp{profile: profile}

	result.options.Type, pres = options.GetString("type")
	if !pres {
		result.options.Type = "uint64"
	}

	topts, pres := options.Get("type_options")
	if !pres {
		result.options.TypeOptions = ordereddict.NewDict()

	} else {

		topts_dict, ok := topts.(*ordereddict.Dict)
		if !ok {
			return nil, fmt.Errorf("Timestamp parser options should be a dict")
		}
		result.options.TypeOptions = topts_dict
	}

	result.options.Factor, pres = options.GetInt64("factor")
	if !pres {
		result.options.Factor = 1
	}

	return result, nil
}

func (self *EpochTimestamp) Parse(
	scope vfilter.Scope, reader io.ReaderAt, offset int64) interface{} {

	if self.parser == nil {
		parser, err := self.profile.GetParser(
			self.options.Type, self.options.TypeOptions)
		if err != nil {
			scope.Log("ERROR:binary_parser: EpochTimestamp: %v", err)
			self.parser = NullParser{}
			return vfilter.Null{}
		}

		// Cache the parser for next time.
		self.parser = parser
	}

	value, ok := to_int64(self.parser.Parse(scope, reader, offset))
	if !ok {
		return vfilter.Null{}
	}
	return time.Unix(value/self.options.Factor, value%self.options.Factor).UTC()
}

type WinFileTime struct {
	*EpochTimestamp
}

func (self *WinFileTime) New(profile *Profile, options *ordereddict.Dict) (Parser, error) {
	result, err := self.EpochTimestamp.New(profile, options)
	if err != nil {
		return nil, err
	}

	return &WinFileTime{EpochTimestamp: result.(*EpochTimestamp)}, nil
}

func (self *WinFileTime) Parse(
	scope vfilter.Scope, reader io.ReaderAt, offset int64) interface{} {
	if self.parser == nil {
		parser, err := self.profile.GetParser(
			self.options.Type, self.options.TypeOptions)
		if err != nil {
			scope.Log("ERROR:binary_parser: WinFileTime: %v", err)
			self.parser = NullParser{}
			return vfilter.Null{}
		}

		// Cache the parser for next time.
		self.parser = parser
	}

	value, ok := to_int64(self.parser.Parse(scope, reader, offset))
	if !ok {
		return vfilter.Null{}
	}
	return time.Unix((value/self.options.Factor/10000000)-11644473600, 0).UTC()
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
		scope.Log("ERROR:binary_parser: FatTimestamp: %v", err)
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
