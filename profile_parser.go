package vtypes

import (
	"errors"
	"fmt"
	"io"

	"github.com/Velocidex/ordereddict"
	"www.velocidex.com/golang/vfilter"
)

type ProfileParserOptions struct {
	Type        string
	TypeOptions *ordereddict.Dict
	Offset      *vfilter.Lambda
}

type ProfileParser struct {
	options ProfileParserOptions
	profile *Profile
	parser  Parser
}

func (self *ProfileParser) New(profile *Profile, options *ordereddict.Dict) (Parser, error) {
	var pres bool

	if options == nil {
		return nil, fmt.Errorf("Profile parser requires a type in the options")
	}

	result := &ProfileParser{profile: profile}

	result.options.Type, pres = options.GetString("type")
	if !pres {
		return nil, errors.New("Profile parser must specify the type in options")
	}

	topts, pres := options.Get("type_options")
	if pres {
		topts_dict, ok := topts.(*ordereddict.Dict)
		if ok {
			result.options.TypeOptions = topts_dict
		}
	}

	offset_expression, _ := options.GetString("offset")
	if offset_expression != "" {
		var err error
		result.options.Offset, err = vfilter.ParseLambda(offset_expression)
		if err != nil {
			return nil, fmt.Errorf("Profile offset '%v': %w",
				offset_expression, err)
		}
	}

	if result.options.Offset == nil {
		return nil, fmt.Errorf("Profile offset must be specified.")
	}

	return result, nil
}

func (self *ProfileParser) Parse(
	scope vfilter.Scope,
	reader io.ReaderAt, offset int64) interface{} {
	if self.parser == nil {
		parser, err := self.profile.GetParser(
			self.options.Type, self.options.TypeOptions)
		if err != nil {
			return vfilter.Null{}
		}

		// Cache the parser for next time.
		self.parser = parser
	}

	// Take the offset from the expression.
	offset = EvalLambdaAsInt64(self.options.Offset, scope)
	return self.parser.Parse(scope, reader, offset)
}
