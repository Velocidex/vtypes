package vtypes

import (
	"fmt"
	"io"
	"strconv"

	"github.com/Velocidex/ordereddict"
	"www.velocidex.com/golang/vfilter"
)

type EnumerationParserOptions struct {
	Type        string
	TypeOptions *ordereddict.Dict
	Choices     map[int64]string
}

type EnumerationParser struct {
	options EnumerationParserOptions
	profile *Profile
	parser  Parser
}

func (self *EnumerationParser) New(profile *Profile, options *ordereddict.Dict) (Parser, error) {
	var pres bool

	if options == nil {
		return nil, fmt.Errorf("Enumeration parser requires an options dict")
	}

	result := &EnumerationParser{profile: profile}
	result.options.Type, pres = options.GetString("type")
	if !pres {
		return nil, fmt.Errorf("Enumeration parser requires a type in the options")
	}

	topts, pres := options.Get("type_options")
	if pres {
		topts_dict, ok := topts.(*ordereddict.Dict)
		if !ok {
			return nil, fmt.Errorf("Enumeration parser options should be a dict")
		}
		result.options.TypeOptions = topts_dict
	}

	mapping := make(map[int64]string)

	// Support 2 ways of providing the mapping - choices has ints
	// as keys and map has strings as keys.
	choices, pres := options.Get("choices")
	if pres {
		choices_dict, ok := choices.(*ordereddict.Dict)
		if !ok {
			return nil, fmt.Errorf("Enumeration parser requires choices to be a mapping between numbers and strings")
		}

		for _, k := range choices_dict.Keys() {
			v, _ := choices_dict.Get(k)
			i, err := strconv.ParseInt(k, 0, 64)
			if err != nil {
				return nil, fmt.Errorf("Enumeration parser requires choices to be a mapping between numbers and strings (not %v)", k)
			}

			v_str, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("Enumeration parser requires choices to be a mapping between numbers and strings")
			}

			mapping[i] = v_str
		}
	}

	choices, pres = options.Get("map")
	if pres {
		choices_dict, ok := choices.(*ordereddict.Dict)
		if !ok {
			return nil, fmt.Errorf("Enumeration parser requires map to be a mapping between strings and numbers")
		}
		for _, k := range choices_dict.Keys() {
			v, _ := choices_dict.Get(k)
			v_int, ok := to_int64(v)
			if !ok {
				return nil, fmt.Errorf("Enumeration parser requires map to be a mapping between strings and numbers")
			}

			mapping[v_int] = k
		}
	}

	result.options.Choices = mapping

	return result, nil
}

func (self *EnumerationParser) Parse(
	scope vfilter.Scope, reader io.ReaderAt, offset int64) interface{} {

	if self.parser == nil {
		parser, err := self.profile.GetParser(
			self.options.Type, self.options.TypeOptions)
		if err != nil {
			scope.Log("ERROR:binary_parser: Enumeration: %v", err)
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

	string_value, pres := self.options.Choices[value]
	if !pres {
		string_value = fmt.Sprintf("%#x", value)
	}
	return string_value
}
