// Implements a binary parsing system.
package vtypes

import (
	"fmt"
	"io"

	"github.com/Velocidex/ordereddict"
	"www.velocidex.com/golang/vfilter"
)

// Parsers are objects which know how to parse a particular
// type. Parsers are instantiated once and reused many times.
type Parser interface {
	Parse(scope vfilter.Scope, reader io.ReaderAt, offset int64) interface{}

	// Given options, this returns a new configured parser
	New(profile *Profile, options *ordereddict.Dict) (Parser, error)
}

type Sizer interface {
	Size() int
}

// Return the start and end of the object
type Starter interface {
	Start() int64
}

type Ender interface {
	End() int64
}

// Parse various sizes of ints.
type IntParser struct {
	type_name string
	size      int
	converter func(buf []byte) interface{}
}

// IntParser does not take options
func (self *IntParser) New(profile *Profile, options *ordereddict.Dict) (Parser, error) {
	return self, nil
}

func (self *IntParser) Size() int {
	return self.size
}

func (self *IntParser) DebugString(scope vfilter.Scope, offset int64, reader io.ReaderAt) string {
	return fmt.Sprintf("[%s] %#0x",
		self.type_name, self.Parse(scope, reader, offset))
}

func (self *IntParser) Parse(scope vfilter.Scope, reader io.ReaderAt, offset int64) interface{} {
	buf := make([]byte, 8)

	n, err := reader.ReadAt(buf, offset)
	if n == 0 || err != nil {
		return 0
	}
	return self.converter(buf)
}

func NewIntParser(type_name string, size int, converter func(buf []byte) interface{}) *IntParser {
	return &IntParser{
		type_name: type_name,
		size:      size,
		converter: converter,
	}
}

/*
type FlagsParserOptions struct {
	Target string
	Bitmap map[string]int
}

type FlagsParser struct {
	*BaseParser
	options *FlagsParserOptions
	profile *Profile
	parser  Parser
}

func NewFlagsParser(type_name string, profile *Profile) *FlagsParser {
	return &FlagsParser{
		BaseParser: &BaseParser{
			type_name: type_name,
		},
		options: &FlagsParserOptions{},
		profile: profile,
	}
}

func (self FlagsParser) Copy() Parser {
	return &FlagsParser{
		BaseParser: self.BaseParser.Copy().(*BaseParser),
		options:    &FlagsParserOptions{},
		profile:    self.profile,
	}
}

func (self *FlagsParser) getParser() (Parser, bool) {
	if self.parser != nil {
		return self.parser, true
	}

	target := "unsigned int"
	if self.options.Target != "" {
		target = self.options.Target
	}

	parser, ok := self.profile.GetParser(target)
	if !ok {
		return nil, false
	}

	self.parser = parser.Copy()
	self.profile = nil

	return self.parser, true
}

func (self *FlagsParser) Get(base Object, field string) Object {
	bitmap, pres := self.options.Bitmap[field]
	if pres {
		parser, pres := self.getParser()
		if pres {
			integer, ok := parser.(Integerer)
			if ok {
				value := integer.AsInteger(
					base.Offset(), base.Reader())

				if value&(1<<uint8(bitmap)) != 0 {
					return base
				}
				return NewErrorObject("Not set.")
			}
		}
	}
	return NewErrorObject("No such value.")
}

func (self *FlagsParser) Fields() []string {
	result := []string{}
	for k, _ := range self.options.Bitmap {
		result = append(result, k)
	}

	return result
}

func (self *FlagsParser) AsInteger(offset int64, reader io.ReaderAt) int64 {
	parser, pres := self.getParser()
	if !pres {
		return -1
	}

	integer, ok := parser.(Integerer)
	if ok {
		return integer.AsInteger(offset, reader)
	}

	return -1
}

func (self *FlagsParser) AsString(offset int64, reader io.ReaderAt) string {
	parser, pres := self.getParser()
	if !pres {
		return ""
	}

	integer, ok := parser.(Integerer)
	if ok {
		result := []string{}
		value := integer.AsInteger(offset, reader)
		for k, v := range self.options.Bitmap {
			if value&(1<<uint16(v)) != 0 {
				result = append(result, k)
			}
		}

		sort.Strings(result)
		return strings.Join(result, ", ")
	}

	return ""
}

func (self *FlagsParser) Size(offset int64, reader io.ReaderAt) int64 {
	return int64(len(self.AsString(offset, reader)))
}

func (self *FlagsParser) DebugString(offset int64, reader io.ReaderAt) string {
	return fmt.Sprintf("[flags %0x '%s']", self.AsInteger(offset, reader),
		self.AsString(offset, reader))
}

func (self *FlagsParser) ShortDebugString(offset int64, reader io.ReaderAt) string {
	return self.AsString(offset, reader)
}

func (self *FlagsParser) ParseArgs(args *json.RawMessage) error {
	return json.Unmarshal(*args, &self.options)
}

type EnumerationOptions struct {
	Choices map[string]string
	Target  string
}

type Enumeration struct {
	*BaseParser
	profile *Profile
	parser  Parser
	options *EnumerationOptions
}

func NewEnumeration(type_name string, profile *Profile) *Enumeration {
	return &Enumeration{&BaseParser{
		type_name: type_name,
	}, profile, nil, &EnumerationOptions{}}
}

func (self Enumeration) Copy() Parser {
	return &Enumeration{
		BaseParser: self.BaseParser.Copy().(*BaseParser),
		profile:    self.profile,
		options:    &EnumerationOptions{},
	}
}

func (self *Enumeration) getParser() (Parser, bool) {
	if self.parser != nil {
		return self.parser, true
	}

	target := "unsigned int"
	if self.options.Target != "" {
		target = self.options.Target
	}

	parser, ok := self.profile.GetParser(target)
	if !ok {
		return nil, false
	}

	self.parser = parser
	self.profile = nil
	return self.parser, true
}

func (self *Enumeration) AsInteger(offset int64, reader io.ReaderAt) int64 {
	parser, _ := self.getParser()
	integer, ok := parser.(Integerer)
	if ok {
		return integer.AsInteger(offset, reader)
	}

	return -1
}

func (self *Enumeration) AsString(offset int64, reader io.ReaderAt) string {
	parser, pres := self.getParser()
	if !pres {
		return ""
	}

	integer, ok := parser.(Integerer)
	if ok {
		string_int := fmt.Sprintf("%d", integer.AsInteger(offset, reader))
		name, pres := self.options.Choices[string_int]
		if pres {
			return name
		}
		return string_int
	}

	return ""
}

func (self *Enumeration) DebugString(offset int64, reader io.ReaderAt) string {
	return self.AsString(offset, reader)
}

func (self *Enumeration) ShortDebugString(offset int64, reader io.ReaderAt) string {
	return self.AsString(offset, reader)
}

func (self *Enumeration) Size(offset int64, reader io.ReaderAt) int64 {
	parser, ok := self.getParser()
	if ok {
		return parser.Size(offset, reader)
	}

	return 0
}

func (self *Enumeration) IsValid(offset int64, reader io.ReaderAt) bool {
	parser, ok := self.getParser()
	if ok {
		return parser.IsValid(offset, reader)
	}

	return false
}

func (self *Enumeration) ParseArgs(args *json.RawMessage) error {
	return json.Unmarshal(*args, &self.options)
}

type BitFieldOptions struct {
	StartBit float64 `json:"start_bit"`
	EndBit   float64 `json:"end_bit"`
	Target   string  `json:"target"`
}

type BitField struct {
	*BaseParser
	profile *Profile
	parser  Parser
	options *BitFieldOptions
}

func NewBitField(type_name string, profile *Profile) *BitField {
	return &BitField{&BaseParser{
		type_name: type_name,
	}, profile, nil, &BitFieldOptions{}}
}

func (self BitField) Copy() Parser {
	return &BitField{
		BaseParser: self.BaseParser.Copy().(*BaseParser),
		profile:    self.profile,
		options:    &BitFieldOptions{},
	}
}

func (self *BitField) getParser() (Parser, bool) {
	if self.parser != nil {
		return self.parser, true
	}

	target := "unsigned int"
	if self.options.Target != "" {
		target = self.options.Target
	}

	parser, ok := self.profile.GetParser(target)
	if !ok {
		return nil, false
	}

	self.parser = parser.Copy()
	self.profile = nil

	return self.parser, true
}

func (self *BitField) AsString(offset int64, reader io.ReaderAt) string {
	return self.ShortDebugString(offset, reader)
}

func (self *BitField) AsInteger(offset int64, reader io.ReaderAt) int64 {
	parser, _ := self.getParser()
	integer, ok := parser.(Integerer)
	if ok {
		result := int64(0)
		value := integer.AsInteger(offset, reader)
		for i := self.options.StartBit; i < self.options.EndBit; i++ {
			result |= value & (1 << uint8(i))
		}

		return result
	}

	return 0
}

func (self *BitField) DebugString(offset int64, reader io.ReaderAt) string {
	return fmt.Sprintf("[%s] %#0x",
		self.type_name, self.AsInteger(offset, reader))
}

func (self *BitField) ShortDebugString(offset int64, reader io.ReaderAt) string {
	return fmt.Sprintf("%#0x", self.AsInteger(offset, reader))
}

func (self *BitField) Size(offset int64, reader io.ReaderAt) int64 {
	parser, ok := self.getParser()
	if ok {
		return parser.Size(offset, reader)
	}

	return 0
}

func (self *BitField) IsValid(offset int64, reader io.ReaderAt) bool {
	parser, ok := self.getParser()
	if ok {
		return parser.IsValid(offset, reader)
	}

	return false
}

func (self *BitField) ParseArgs(args *json.RawMessage) error {
	return json.Unmarshal(*args, &self.options)
}
*/
