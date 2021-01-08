package vtypes

import (
	"fmt"
	"io"

	"github.com/Velocidex/ordereddict"
	"www.velocidex.com/golang/vfilter"
)

type EnumerationParser struct {
	choices *ordereddict.Dict
	parser  Parser
}

func (self *EnumerationParser) New(profile *Profile, options *ordereddict.Dict) (Parser, error) {
	parser_type, pres := options.GetString("type")
	if !pres {
		return nil, fmt.Errorf("Enumeration parser requires a type in the options")
	}

	parser, err := profile.GetParser(parser_type, ordereddict.NewDict())
	if err != nil {
		return nil, fmt.Errorf("Enumeration parser requires a type in the options: %w", err)
	}

	choices, pres := options.Get("choices")
	if !pres {
		choices = ordereddict.NewDict()
	}

	choices_dict, ok := choices.(*ordereddict.Dict)
	if !ok {
		return nil, fmt.Errorf("Enumeration parser requires choices to be a mapping between numbers and strings")
	}

	return &EnumerationParser{
		choices: choices_dict,
		parser:  parser,
	}, nil
}

func (self *EnumerationParser) Parse(
	scope vfilter.Scope, reader io.ReaderAt, offset int64) interface{} {

	value := fmt.Sprintf("%v", self.parser.Parse(scope, reader, offset))
	string_value, pres := self.choices.Get(value)
	if !pres {
		string_value = value
	}
	return string_value
}
