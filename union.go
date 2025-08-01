package vtypes

import (
	"context"
	"fmt"
	"io"

	"github.com/Velocidex/ordereddict"
	"www.velocidex.com/golang/vfilter"
)

type UnionOptions struct {
	Selector *vfilter.Lambda   `vfilter:"required,field=selector,doc=A lambda selector"`
	Choices  *ordereddict.Dict `vfilter:"required,field=choices,doc=A between values and strings"`
	choices  map[string]Parser
}

type Union struct {
	options UnionOptions
	profile *Profile
}

func (self *Union) New(profile *Profile, options *ordereddict.Dict) (Parser, error) {
	if options == nil {
		return nil, fmt.Errorf("Union parser requires options")
	}

	result := &Union{profile: profile}
	ctx := context.Background()
	err := ParseOptions(ctx, options, &result.options)
	if err != nil {
		return nil, fmt.Errorf("Union: %w", err)
	}

	return result, nil
}

func (self *Union) Parse(
	scope vfilter.Scope, reader io.ReaderAt, offset int64) interface{} {

	var value interface{}

	// Initialize the choices late to ensure they are all defined by
	// now.
	if self.options.choices == nil {
		self.options.choices = make(map[string]Parser)

		for _, k := range self.options.Choices.Keys() {
			parser_name, pres := self.options.Choices.GetString(k)
			if !pres {
				continue
			}

			parser, err := self.profile.GetParser(
				parser_name, ordereddict.NewDict())
			if err != nil {
				scope.Log("ERROR:binary_parser: Union: %v", err)
			} else {
				self.options.choices[k] = parser
			}
		}
	}

	subscope := scope.Copy()
	defer subscope.Close()

	this_obj, pres := getThis(subscope)
	if pres {
		value = self.options.Selector.Reduce(
			context.Background(), subscope, []vfilter.Any{this_obj})
	}

	if IsNil(value) {
		return &vfilter.Null{}
	}

	value_str := fmt.Sprintf("%v", value)
	parser, pres := self.options.choices[value_str]
	if pres {
		return parser.Parse(scope, reader, offset)
	}

	// Try the default
	parser, pres = self.options.choices["default"]
	if pres {
		return parser.Parse(scope, reader, offset)
	}

	return &vfilter.Null{}
}
