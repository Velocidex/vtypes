package vtypes

import (
	"encoding/json"
	"fmt"
)

// A parser that holds a logical array of elements.
type ArrayParserOptions struct {
	Target string `vfilter:"required,field=target"`
}

type ArrayParser struct {
	*BaseParser
	counter int64
	profile *Profile
	options *ArrayParserOptions
}

func (self ArrayParser) Copy() Parser {
	return &self
}

func (self *ArrayParser) ParseArgs(args *json.RawMessage) error {
	return json.Unmarshal(*args, &self.options)
}

// Produce the next iteration in the array.
func (self *ArrayParser) Next(base Object) Object {
	result := self.Value(base)
	self.counter += 1
	return result
}

func (self *ArrayParser) Value(base Object) Object {
	parser, pres := self.profile.GetParser(self.options.Target)
	if !pres {
		return NewErrorObject(
			fmt.Sprintf("ArrayParser: Type %s not found",
				self.options.Target))
	}

	return &BaseObject{
		name:      base.Name(),
		type_name: self.options.Target,
		offset: base.Offset() + self.counter*parser.Size(
			base.Offset(), base.Reader()),
		reader: base.Reader(),
		parser: parser,
	}
}

func NewArrayParser(type_name string, name string,
	profile *Profile, options *ArrayParserOptions) *ArrayParser {
	return &ArrayParser{&BaseParser{
		Name:      name,
		type_name: type_name},
		0,
		profile,
		options,
	}
}
