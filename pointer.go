package vtypes

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/Velocidex/ordereddict"
	"www.velocidex.com/golang/vfilter"
)

type PointerParserOptions struct {
	Type        string
	TypeOptions *ordereddict.Dict
}

type PointerParser struct {
	options PointerParserOptions
	profile *Profile
	parser  Parser
}

func (self *PointerParser) New(profile *Profile, options *ordereddict.Dict) (Parser, error) {
	var pres bool

	if options == nil {
		return nil, fmt.Errorf("Pointer parser requires a type in the options")
	}

	result := &PointerParser{profile: profile}

	result.options.Type, pres = options.GetString("type")
	if !pres {
		return nil, errors.New("Pointer must specify the type in options")
	}

	topts, pres := options.Get("type_options")
	if pres {
		topts_dict, ok := topts.(*ordereddict.Dict)
		if ok {
			result.options.TypeOptions = topts_dict
		}
	}

	return result, nil
}

func (self *PointerParser) Parse(
	scope vfilter.Scope,
	reader io.ReaderAt, offset int64) interface{} {
	if self.parser == nil {
		parser, err := self.profile.GetParser(
			self.options.Type, self.options.TypeOptions)
		if err != nil {
			scope.Log("ERROR:binary_parser: PointerParser: %v", err)
			self.parser = NullParser{}
			return vfilter.Null{}
		}

		// Cache the parser for next time.
		self.parser = parser
	}

	buf := make([]byte, 8)

	n, err := reader.ReadAt(buf, offset)
	if n == 0 || err != nil {
		return vfilter.Null{}
	}

	address := binary.LittleEndian.Uint64(buf)

	return self.parser.Parse(scope, reader, int64(address))
}
