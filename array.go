package vtypes

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/Velocidex/ordereddict"
	"www.velocidex.com/golang/vfilter"
)

type ArrayParserOptions struct {
	Type               string            `vfilter:"required,field=type,doc=The underlying type of the choice"`
	TypeOptions        *ordereddict.Dict `vfilter:"optional,field=type_options,doc=Any additional options required to parse the type"`
	Count              int64             `vfilter:"optional,lambda=CountExpression,field=count,doc=Number of elements in the array (default 0)"`
	MaxCount           int64             `vfilter:"optional,field=max_count,doc=Maximum number of elements in the array (default 1000)"`
	CountExpression    *vfilter.Lambda
	SentinelExpression *vfilter.Lambda `vfilter:"optional,field=sentinel,doc=A lambda expression that will be used to determine the end of the array"`
}

type ArrayParser struct {
	options ArrayParserOptions
	profile *Profile
	parser  Parser

	invalid_parser bool
}

func (self *ArrayParser) New(profile *Profile, options *ordereddict.Dict) (Parser, error) {
	if options == nil {
		return nil, fmt.Errorf("Array parser requires a type in the options")
	}

	result := &ArrayParser{profile: profile}
	ctx := context.Background()
	err := ParseOptions(ctx, options, &result.options)
	if err != nil {
		return nil, fmt.Errorf("ArrayParser: %v", err)
	}
	if result.options.MaxCount == 0 {
		result.options.MaxCount = 1000
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

func (self *ArrayParser) getCount(scope vfilter.Scope) int64 {
	result := self.options.Count

	if self.options.CountExpression != nil {
		// Evaluate the offset expression with the current scope.
		result = EvalLambdaAsInt64(self.options.CountExpression, scope)
	}

	if result > self.options.MaxCount {
		return self.options.MaxCount
	}

	if result < 0 {
		result = 0
	}
	return result
}

func (self *ArrayParser) Parse(
	scope vfilter.Scope,
	reader io.ReaderAt, offset int64) interface{} {

	result_len := self.getCount(scope)
	result := make([]interface{}, 0, result_len)

	if self.invalid_parser {
		return vfilter.Null{}
	}

	if self.parser == nil {
		parser, err := self.profile.GetParser(
			self.options.Type, self.options.TypeOptions)
		if err != nil {
			scope.Log("ERROR:binary_parser: ArrayParser: %v", err)
			self.invalid_parser = true
			return vfilter.Null{}
		}

		// Cache the parser for next time.
		self.parser = parser
	}

	member_offset := int64(0)
	for i := int64(0); i < result_len; i++ {
		element := self.parser.Parse(
			scope, reader, offset+member_offset)

		// Check for a sentinel value
		if self.options.SentinelExpression != nil {
			ctx := context.Background()
			subscope := scope.Copy()
			sentinel := self.options.SentinelExpression.Reduce(
				ctx, subscope, []vfilter.Any{element})
			subscope.Close()

			if scope.Bool(sentinel) {
				break
			}
		}

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

func (self *ArrayObject) SetParent(parent *StructObject) {
	for _, e := range self.contents {
		switch t := e.(type) {
		case *StructObject:
			t.parent = parent
		}
	}
}

func (self *ArrayObject) Contents() []interface{} {
	res := make([]interface{}, 0, len(self.contents))
	for _, v := range self.contents {
		res = append(res, ValueOf(v))
	}
	return res
}

func (self *ArrayObject) Get(i int64) (interface{}, error) {
	if i < 0 || i > int64(len(self.contents)) {
		return nil, NotFoundError
	}
	return self.contents[i], nil
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
	return json.Marshal(self.Contents())
}
