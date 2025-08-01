package vtypes

import (
	"context"
	"fmt"
	"io"
	"strconv"

	"github.com/Velocidex/ordereddict"
	"www.velocidex.com/golang/vfilter"
)

type EnumerationParserOptions struct {
	Type        string            `vfilter:"required,field=type,doc=The underlying type of the choice"`
	TypeOptions *ordereddict.Dict `vfilter:"optional,field=type_options,doc=Any additional options required to parse the type"`
	Choices     *ordereddict.Dict `vfilter:"optional,field=choices,doc=A mapping between numbers and strings."`
	Map         *ordereddict.Dict `vfilter:"optional,field=map,doc=A mapping between strings and numbers."`

	choices map[int64]string
}

type EnumerationParser struct {
	options EnumerationParserOptions
	profile *Profile
	parser  Parser

	invalid_parser bool
}

func (self *EnumerationParser) New(profile *Profile, options *ordereddict.Dict) (Parser, error) {
	if options == nil {
		return nil, fmt.Errorf("Enumeration parser requires an options dict")
	}

	result := &EnumerationParser{profile: profile}
	ctx := context.Background()
	err := ParseOptions(ctx, options, &result.options)
	if err != nil {
		return nil, err
	}

	result.options.choices = make(map[int64]string)

	if result.options.Choices != nil {
		for _, k := range result.options.Choices.Keys() {
			v, _ := result.options.Choices.Get(k)
			i, err := strconv.ParseInt(k, 0, 64)
			if err != nil {
				return nil, fmt.Errorf("Enumeration parser requires choices to be a mapping between numbers and strings (not %v)", k)
			}

			v_str, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("Enumeration parser requires choices to be a mapping between numbers and strings")
			}

			result.options.choices[i] = v_str
		}
	}

	if result.options.Map != nil {
		for _, k := range result.options.Map.Keys() {
			v, _ := result.options.Map.Get(k)
			v_int, ok := to_int64(v)
			if !ok {
				return nil, fmt.Errorf("Enumeration parser requires map to be a mapping between strings and numbers")
			}

			result.options.choices[v_int] = k
		}
	}

	// Get the parser now so we can catch errors in sub parser
	// definitions
	parser, err := maybeGetParser(profile,
		result.options.Type, result.options.TypeOptions)
	if err != nil {
		return nil, err
	}

	// Cache the parser for next time.
	result.parser = parser

	return result, nil
}

func (self *EnumerationParser) Parse(
	scope vfilter.Scope, reader io.ReaderAt, offset int64) interface{} {

	if self.invalid_parser {
		return vfilter.Null{}
	}

	if self.parser == nil {
		parser, err := self.profile.GetParser(
			self.options.Type, self.options.TypeOptions)
		if err != nil {
			scope.Log("ERROR:binary_parser: EnumerationParser: %v", err)
			self.invalid_parser = true
			return vfilter.Null{}
		}

		// Cache the parser for next time.
		self.parser = parser
	}

	value, ok := to_int64(self.parser.Parse(scope, reader, offset))
	if !ok {
		return vfilter.Null{}
	}

	string_value, pres := self.options.choices[value]
	if !pres {
		string_value = fmt.Sprintf("%#x", value)
	}
	return string_value
}
