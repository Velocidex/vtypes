package vtypes

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/Velocidex/ordereddict"
	"www.velocidex.com/golang/vfilter"
)

type PointerParserOptions struct {
	Type        string            `vfilter:"required,field=type,doc=The underlying type of the choice"`
	TypeOptions *ordereddict.Dict `vfilter:"optional,field=type_options,doc=Any additional options required to parse the type"`
}

type PointerParser struct {
	options PointerParserOptions
	profile *Profile
	parser  Parser
}

func (self *PointerParser) New(profile *Profile, options *ordereddict.Dict) (Parser, error) {
	if options == nil {
		return nil, fmt.Errorf("Pointer parser requires a type in the options")
	}

	result := &PointerParser{profile: profile}
	ctx := context.Background()
	err := ParseOptions(ctx, options, &result.options)
	if err != nil {
		return nil, fmt.Errorf("PointerParser: %v", err)
	}

	parser, err := maybeGetParser(profile,
		result.options.Type, result.options.TypeOptions)
	if err != nil {
		return nil, err
	}
	result.parser = parser

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
	if n == 0 || (err != nil && !errors.Is(err, io.EOF)) {
		return vfilter.Null{}
	}

	address := binary.LittleEndian.Uint64(buf)

	return self.parser.Parse(scope, reader, int64(address))
}
