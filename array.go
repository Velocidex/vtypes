package vtypes

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/Velocidex/ordereddict"
	"www.velocidex.com/golang/vfilter"
)

type ArrayParserOptions struct {
	Type            string
	TypeOptions     *ordereddict.Dict
	Count           int64
	MaxCount        int64
	CountExpression *vfilter.Lambda
}

type ArrayParser struct {
	options ArrayParserOptions
	profile *Profile
	parser  Parser
}

func (self *ArrayParser) New(profile *Profile, options *ordereddict.Dict) (Parser, error) {
	var pres bool

	if options == nil {
		return nil, fmt.Errorf("Array parser requires a type in the options")
	}

	result := &ArrayParser{profile: profile}

	result.options.Type, pres = options.GetString("type")
	if !pres {
		return nil, errors.New("Array must specify the type in options")
	}

	// Default to 0 length
	result.options.Count, _ = options.GetInt64("count")
	result.options.MaxCount, _ = options.GetInt64("max_count")

	if result.options.MaxCount == 0 {
		result.options.MaxCount = 1000
	}

	// Maybe add a count expression
	expression, _ := options.GetString("count")
	if expression != "" {
		var err error
		result.options.CountExpression, err = vfilter.ParseLambda(expression)
		if err != nil {
			return nil, fmt.Errorf("Array parser count expression '%v': %w",
				expression, err)
		}
	}

	return result, nil
}

func (self *ArrayParser) getCount(scope vfilter.Scope) int64 {
	result := self.options.Count

	if self.options.CountExpression != nil {
		// Evaluate the offset expression with the current scope.
		return EvalLambdaAsInt64(self.options.CountExpression, scope)
	}

	if result > self.options.MaxCount {
		return self.options.MaxCount
	}
	return result
}

func (self *ArrayParser) Parse(
	scope vfilter.Scope,
	reader io.ReaderAt, offset int64) interface{} {

	result_len := self.getCount(scope)
	result := make([]interface{}, 0, result_len)

	if self.parser == nil {
		parser, err := self.profile.GetParser(
			self.options.Type, self.options.TypeOptions)
		if err != nil {
			return vfilter.Null{}
		}

		// Cache the parser for next time.
		self.parser = parser
	}

	member_offset := int64(0)
	for i := int64(0); i < result_len; i++ {
		element := self.parser.Parse(
			scope, reader, offset+member_offset)

		// The parser may know about the element size, or the
		// element itself.
		element_size := SizeOf(self.parser)
		if element_size == 0 {
			element_size = SizeOf(element)
		}

		if element_size == 0 {
			break
		}

		result = append(result, element)

		member_offset += int64(element_size)
	}

	return &ArrayObject{
		contents: result,
		offset:   offset,
		size:     member_offset,
	}
}

type ArrayObject struct {
	contents []interface{}
	offset   int64
	size     int64
}

func (self *ArrayObject) Size() int {
	return int(self.size)
}

func (self *ArrayObject) Start() int64 {
	return self.offset
}

func (self *ArrayObject) End() int64 {
	return self.offset + self.size
}

func (self *ArrayObject) MarshalJSON() ([]byte, error) {
	return json.Marshal(self.contents)
}
