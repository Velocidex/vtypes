package vtypes

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/Velocidex/ordereddict"
	"www.velocidex.com/golang/vfilter"
)

// A ValueParser can either represent a static value, or an
// expression.
type ValueParser struct {
	expression *vfilter.Lambda
	value      interface{}
}

func (self *ValueParser) New(profile *Profile, options *ordereddict.Dict) (Parser, error) {
	if options == nil {
		return nil, fmt.Errorf("Value parser requires a type in the options")
	}

	value, pres := options.Get("value")
	if !pres || IsNil(value) {
		return nil, errors.New("Value parser must specify a value")
	}

	result := &ValueParser{value: value}

	// If the value is a string, it may be a lambda. If it looks
	// like a lambda we parse it here to trap any syntax errors.
	value_str, ok := value.(string)
	if ok && strings.Contains(value_str, "=>") {
		expression, err := vfilter.ParseLambda(value_str)
		if err != nil {
			return nil, err
		}
		result.expression = expression
		result.value = nil
	}

	return result, nil
}

func (self *ValueParser) Parse(scope vfilter.Scope, reader io.ReaderAt, offset int64) interface{} {
	if self.expression != nil {
		subscope := scope.Copy()
		defer subscope.Close()

		this_obj, pres := getThis(subscope)
		if pres {
			return self.expression.Reduce(
				context.Background(), subscope, []vfilter.Any{this_obj})
		}
	}
	if IsNil(self.value) {
		return &vfilter.Null{}
	}
	return self.value
}
