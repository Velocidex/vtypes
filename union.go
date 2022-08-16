package vtypes

import (
	"context"
	"fmt"
	"io"

	"github.com/Velocidex/ordereddict"
	"www.velocidex.com/golang/vfilter"
)

type Union struct {
	Selector     *vfilter.Lambda
	choice_names *ordereddict.Dict
	Choices      map[string]Parser
	profile      *Profile
}

func (self *Union) New(profile *Profile, options *ordereddict.Dict) (Parser, error) {
	if options == nil {
		return nil, fmt.Errorf("Union parser requires options")
	}

	expression, pres := options.GetString("selector")
	if !pres {
		return nil, fmt.Errorf("Union parser requires a lambda selector")
	}

	selector, err := vfilter.ParseLambda(expression)
	if err != nil {
		return nil, fmt.Errorf("Union parser selector expression '%v': %w",
			expression, err)
	}

	choices, pres := options.Get("choices")
	if !pres {
		choices = ordereddict.NewDict()
	}

	choices_dict, ok := choices.(*ordereddict.Dict)
	if !ok {
		return nil, fmt.Errorf("Union parser requires choices to be a mapping between values and strings")
	}

	result := &Union{
		Selector: selector,

		// Map the value to the name of the type
		choice_names: choices_dict,

		// Map the value to the actual parser
		Choices: make(map[string]Parser),

		profile: profile,
	}
	return result, nil
}

func (self *Union) Parse(
	scope vfilter.Scope, reader io.ReaderAt, offset int64) interface{} {

	var value interface{}
	if self.Selector != nil {
		this_obj, pres := scope.Resolve("this")
		if pres {
			value = self.Selector.Reduce(
				context.Background(), scope, []vfilter.Any{this_obj})
		}
	}
	if IsNil(value) {
		return &vfilter.Null{}
	}

	value_str := fmt.Sprintf("%v", value)
	parser, pres := self.Choices[value_str]
	if pres {
		return parser.Parse(scope, reader, offset)
	}

	// Resolve the parser from the profile now.
	parser_name, pres := self.choice_names.GetString(value_str)
	if !pres {
		// Try the default
		parser_name, pres = self.choice_names.GetString("default")
		if !pres {
			// Can not find the type - return null
			return vfilter.Null{}
		}
		parser_name = "default"
	}

	// Resolve the parser from the profile
	parser, err := self.profile.GetParser(parser_name, ordereddict.NewDict())
	if err != nil {
		return vfilter.Null{}
	}

	if value_str != "default" {
		self.Choices[value_str] = parser
	}
	return parser.Parse(scope, reader, offset)
}
