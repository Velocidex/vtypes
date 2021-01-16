package vtypes

import (
	"fmt"
	"io"
	"strconv"

	"github.com/Velocidex/ordereddict"
	"www.velocidex.com/golang/vfilter"
)

type EnumerationParser struct {
	choices map[int64]string
	parser  Parser
}

func (self *EnumerationParser) New(profile *Profile, options *ordereddict.Dict) (Parser, error) {
	if options == nil {
		return nil, fmt.Errorf("Enumeration parser requires a type in the options")
	}

	parser_type, pres := options.GetString("type")
	if !pres {
		return nil, fmt.Errorf("Enumeration parser requires a type in the options")
	}

	parser, err := profile.GetParser(parser_type, ordereddict.NewDict())
	if err != nil {
		return nil, fmt.Errorf("Enumeration parser requires a type in the options: %w", err)
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

	return &EnumerationParser{
		choices: mapping,
		parser:  parser,
	}, nil
}

func (self *EnumerationParser) Parse(
	scope vfilter.Scope, reader io.ReaderAt, offset int64) interface{} {

	value, ok := to_int64(self.parser.Parse(scope, reader, offset))
	if !ok {
		return vfilter.Null{}
	}

	string_value, pres := self.choices[value]
	if !pres {
		string_value = fmt.Sprintf("%#x", value)
	}
	return string_value
}
