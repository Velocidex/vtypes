// Implements a binary parsing system.
package vtypes

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode/utf16"
)

// Parsers are objects which know how to parse a particular
// type. Parsers are instantiated once and reused many times. They act
// upon an Object which represents a particular instance of a parser
// in a particular offset.

// Here is an example: A struct foo may have 3 members. There is a
// Struct parser instantiated once which knows how to parse struct foo
// (i.e. all its fiels and their offsets). Once instantiated and
// stored in the Profile, the parser may be reused multiple times to
// parse multiple foo structs - each time, it produces an Object.

// The Object struct contains the offset, and the parser that is used
// to parse it.

type Parser interface {
	SetName(name string) Parser
	DebugString(offset int64, reader io.ReaderAt) string
	ShortDebugString(offset int64, reader io.ReaderAt) string
	Size(offset int64, reader io.ReaderAt) int64
	IsValid(offset int64, reader io.ReaderAt) bool
	ParseArgs(args *json.RawMessage) error
	Copy() Parser
}

type Integerer interface {
	AsInteger(offset int64, reader io.ReaderAt) int64
}

type Stringer interface {
	AsString(offset int64, reader io.ReaderAt) string
}

type Getter interface {
	Get(base Object, field string) Object
	Fields() []string
}

type Iterator interface {
	Value(base Object) Object
	Next(base Object) Object
}

type Object interface {
	Name() string
	AsInteger() int64
	AsString() string
	Get(field string) Object
	Reader() io.ReaderAt
	Offset() int64
	Size() int64
	DebugString() string
	IsValid() bool
	Value() interface{}
	Fields() []string
	Next() Object
	Profile() *Profile
}

type BaseObject struct {
	reader    io.ReaderAt
	offset    int64
	name      string
	type_name string
	parser    Parser
	profile   *Profile
}

func (self *BaseObject) Name() string {
	return self.name
}

func (self *BaseObject) Reader() io.ReaderAt {
	return self.reader
}

func (self *BaseObject) Profile() *Profile {
	return self.profile
}

func (self *BaseObject) Offset() int64 {
	return self.offset
}

func (self *BaseObject) AsInteger() int64 {
	switch self.parser.(type) {
	case Integerer:
		return self.parser.(Integerer).AsInteger(self.offset, self.reader)
	default:
		return 0
	}
}

func (self *BaseObject) AsString() string {
	switch t := self.parser.(type) {
	case Stringer:
		return t.AsString(self.offset, self.reader)
	default:
		return ""
	}
}

func (self *BaseObject) Get(field string) Object {
	if strings.Contains(field, ".") {
		components := strings.Split(field, ".")
		var result Object = self
		for _, component := range components {
			result = result.Get(component)
		}

		return result
	}

	switch t := self.parser.(type) {
	case Getter:
		return t.Get(self, field)
	default:
		return NewErrorObject("Parser does not support Get for " + field)
	}
}

func (self *BaseObject) Next() Object {
	switch t := self.parser.(type) {
	case Iterator:
		return t.Next(self)
	default:
		return NewErrorObject("Parser does not support iteration")
	}
}

func (self *BaseObject) DebugString() string {
	return self.parser.DebugString(self.offset, self.reader)
}

func (self *BaseObject) Size() int64 {
	return self.parser.Size(self.offset, self.reader)
}

func (self *BaseObject) IsValid() bool {
	return self.parser.IsValid(self.offset, self.reader)
}

func (self *BaseObject) Value() interface{} {
	switch t := self.parser.(type) {
	case Stringer:
		return self.AsString()
	case Integerer:
		return self.AsInteger()
	case Iterator:
		return t.Value(self)
	default:
		return self
	}
}

func (self *BaseObject) Fields() []string {
	switch t := self.parser.(type) {
	case Getter:
		return t.Fields()
	default:
		return []string{}
	}
}

func (self *BaseObject) MarshalJSON() ([]byte, error) {
	res := make(map[string]interface{})
	for _, field := range self.Fields() {
		res[field] = self.Get(field).Value()
	}
	buf, err := json.Marshal(res)
	return buf, err
}

// When an operation fails we return an error object. The error object
// can continue to be used in all operations and it will just carry
// itself over safely. This means that callers do not need to check
// for errors all the time:

// a.Get("field").Next().Get("field") -> ErrorObject
type ErrorObject struct {
	message string
	err     error
}

func NewError(err error) *ErrorObject {
	return &ErrorObject{"", err}
}

func NewErrorObject(message string) *ErrorObject {
	return &ErrorObject{message: message}
}

func (self ErrorObject) Error() string {
	if self.err != nil {
		return self.err.Error()
	}
	return self.message
}

func (self *ErrorObject) Name() string {
	return "Error: " + self.message
}

func (self *ErrorObject) Reader() io.ReaderAt {
	return nil
}

func (self *ErrorObject) Offset() int64 {
	return 0
}

func (self *ErrorObject) Get(field string) Object {
	return self
}

func (self *ErrorObject) Next() Object {
	return self
}

func (self *ErrorObject) AsInteger() int64 {
	return 0
}

func (self *ErrorObject) AsString() string {
	return ""
}

func (self *ErrorObject) DebugString() string {
	return fmt.Sprintf("Error: %s", self.message)
}

func (self *ErrorObject) Size() int64 {
	return 0
}

func (self *ErrorObject) IsValid() bool {
	return false
}

func (self *ErrorObject) Profile() *Profile {
	return &Profile{}
}

func (self *ErrorObject) Value() interface{} {
	return errors.New(self.message)
}

func (self *ErrorObject) Fields() []string {
	return []string{}
}

// Baseclass for parsers.
type BaseParser struct {
	Name      string
	size      int64
	type_name string
}

func (self BaseParser) Copy() Parser {
	return &self
}

func (self *BaseParser) SetName(name string) Parser {
	self.Name = name
	return self
}

func (self *BaseParser) DebugString(offset int64, reader io.ReaderAt) string {
	return fmt.Sprintf("[%s] @ %#0x", self.type_name, offset)
}

func (self *BaseParser) ShortDebugString(offset int64, reader io.ReaderAt) string {
	return fmt.Sprintf("[%s] @ %#0x", self.type_name, offset)
}

func (self *BaseParser) Size(offset int64, reader io.ReaderAt) int64 {
	return self.size
}

func (self *BaseParser) IsValid(offset int64, reader io.ReaderAt) bool {
	buf := make([]byte, self.size)
	_, err := reader.ReadAt(buf, offset)
	if err != nil {
		return false
	}
	return true
}

// If a derived parser takes args. process them here.
func (self *BaseParser) ParseArgs(args *json.RawMessage) error {
	return nil
}

// Parse various sizes of ints.
type IntParser struct {
	*BaseParser
	converter func(buf []byte) int64
}

func (self IntParser) Copy() Parser {
	return &IntParser{
		BaseParser: self.BaseParser.Copy().(*BaseParser),
		converter:  self.converter,
	}
}

func (self *IntParser) DebugString(offset int64, reader io.ReaderAt) string {
	return fmt.Sprintf("[%s] %#0x",
		self.type_name, self.AsInteger(offset, reader))
}

func (self *IntParser) ShortDebugString(offset int64, reader io.ReaderAt) string {
	return fmt.Sprintf("%#0x", self.AsInteger(offset, reader))
}

func (self *IntParser) AsString(offset int64, reader io.ReaderAt) string {
	return fmt.Sprintf("%d", self.AsInteger(offset, reader))
}

func (self *IntParser) AsInteger(offset int64, reader io.ReaderAt) int64 {
	buf := make([]byte, 8)

	n, err := reader.ReadAt(buf, offset)
	if n == 0 || err != nil {
		return 0
	}
	return self.converter(buf)
}

func NewIntParser(type_name string, converter func(buf []byte) int64) *IntParser {
	return &IntParser{&BaseParser{
		type_name: type_name,
	}, converter}
}

// Parses strings.
type StringParserOptions struct {
	Length   *int64
	Term     string
	Encoding string
}

type StringParser struct {
	*BaseParser
	options *StringParserOptions
}

func NewStringParser(type_name string) *StringParser {
	return &StringParser{
		&BaseParser{type_name: type_name},
		&StringParserOptions{}}
}

func (self StringParser) Copy() Parser {
	return &StringParser{
		BaseParser: self.BaseParser.Copy().(*BaseParser),
		options:    &StringParserOptions{},
	}
}

func (self *StringParser) AsString(offset int64, reader io.ReaderAt) string {
	read_length := 1024
	if self.options.Length != nil {
		read_length = int(*self.options.Length)
	}
	buf := make([]byte, read_length)

	term := self.options.Term
	if term == "" {
		term = "\x00"
	}

	n, _ := reader.ReadAt(buf, offset)
	result := buf[:n]
	idx := bytes.Index(result, []byte(term))
	if idx >= 0 {
		result = result[:idx]
	}

	if self.options.Encoding == "utf16" {
		order := binary.LittleEndian
		u16s := []uint16{}

		for i, j := 0, len(result); i < j; i += 2 {
			if len(result) < i+2 {
				break
			}
			u16s = append(u16s, order.Uint16(result[i:]))
		}

		return string(utf16.Decode(u16s))
	}

	return string(result)
}

func (self *StringParser) Size(offset int64, reader io.ReaderAt) int64 {
	return int64(len(self.AsString(offset, reader)))
}

func (self *StringParser) DebugString(offset int64, reader io.ReaderAt) string {
	return "[string '" + self.AsString(offset, reader) + "']"
}

func (self *StringParser) ShortDebugString(offset int64, reader io.ReaderAt) string {
	return self.AsString(offset, reader)
}

func (self *StringParser) ParseArgs(args *json.RawMessage) error {
	return json.Unmarshal(*args, &self.options)
}

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

type StructParser struct {
	*BaseParser
	fields map[string]*ParseAtOffset
}

func (self StructParser) Copy() Parser {
	// Structs do not have options.
	return &StructParser{
		BaseParser: self.BaseParser.Copy().(*BaseParser),
		fields:     self.fields,
	}
}

func (self *StructParser) Get(base Object, field string) Object {
	parser, pres := self.fields[field]
	if pres {
		return parser.Get(base, field)
	}

	return NewErrorObject("Field " + field + " not known.")
}

func (self *StructParser) Fields() []string {
	var result []string
	for k := range self.fields {
		result = append(result, k)
	}

	return result
}

func indent(input string) string {
	var indented []string
	for _, line := range strings.Split(input, "\n") {
		indented = append(indented, "  "+line)
	}

	return strings.Join(indented, "\n")
}

func (self *StructParser) DebugString(offset int64, reader io.ReaderAt) string {
	result := []string{}

	for _, parser := range self.fields {
		result = append(result, indent(parser.DebugString(offset, reader)))
	}
	sort.Strings(result)
	return fmt.Sprintf("[%s] @ %#0x\n", self.type_name, offset) +
		strings.Join(result, "\n")
}

func (self *StructParser) ShortDebugString(offset int64, reader io.ReaderAt) string {
	return fmt.Sprintf("[%s] @ %#0x\n", self.type_name, offset)
}

func (self *StructParser) AddParser(field string, parser *ParseAtOffset) {
	self.fields[field] = parser
}

func NewStructParser(type_name string, size int64) *StructParser {
	result := &StructParser{
		&BaseParser{type_name: type_name, size: size},
		make(map[string]*ParseAtOffset),
	}

	return result
}

type ParseAtOffset struct {
	// Field offset within the struct.
	offset int64
	name   string

	// The name of the parser to use and the params - will be
	// dynamically resolved on first access.
	type_name string
	params    *json.RawMessage

	profile *Profile

	// A local cache of the resolved parser for this field.
	parser Parser
}

func (self ParseAtOffset) Copy() Parser {
	return &ParseAtOffset{
		offset:    self.offset,
		name:      self.name,
		type_name: self.type_name,
		params:    self.params,
		profile:   self.profile,
	}
}

func (self *ParseAtOffset) Get(base Object, field string) Object {
	parser, pres := self.getParser(self.type_name)
	if !pres {
		return NewErrorObject(fmt.Sprintf(
			"Type '%s' not found", self.type_name))
	}

	result := &BaseObject{
		name:      field,
		type_name: self.type_name,
		offset:    base.Offset() + self.offset,
		reader:    base.Reader(),
		parser:    parser,
		profile:   base.Profile(),
	}
	return result
}

func (self *ParseAtOffset) Fields() []string {
	parser, pres := self.getParser(self.type_name)
	if pres {
		getter, ok := parser.(Getter)
		if ok {
			return getter.Fields()
		}
	}

	return []string{}
}

func (self *ParseAtOffset) DebugString(offset int64, reader io.ReaderAt) string {
	parser, pres := self.getParser(self.type_name)
	if !pres {
		return fmt.Sprintf("%s: Type '%s' not found.",
			self.name, self.type_name)
	}
	return fmt.Sprintf(
		"%#03x  %s  %s", self.offset, self.name,
		parser.DebugString(self.offset+offset, reader))
}

func (self *ParseAtOffset) ShortDebugString(offset int64, reader io.ReaderAt) string {
	parser, pres := self.getParser(self.type_name)
	if !pres {
		return fmt.Sprintf("Type '%s' not found", self.type_name)
	}

	return parser.ShortDebugString(self.offset+offset, reader)
}

func (self *ParseAtOffset) SetName(name string) Parser {
	self.name = name
	return self
}

func (self *ParseAtOffset) Size(offset int64, reader io.ReaderAt) int64 {
	parser, pres := self.getParser(self.type_name)
	if !pres {
		return 0
	}

	return parser.Size(self.offset+offset, reader)
}

func (self *ParseAtOffset) IsValid(offset int64, reader io.ReaderAt) bool {
	parser, pres := self.getParser(self.type_name)
	if !pres {
		return false
	}

	return parser.IsValid(self.offset+offset, reader)
}

func (self *ParseAtOffset) getParser(name string) (Parser, bool) {
	// Get parser from the cache if possible.
	if self.parser != nil {
		return self.parser, true
	}

	parser, pres := self.profile.GetParser(self.type_name)
	if !pres {
		return nil, false
	}

	// Prepare a new parser based on the params.
	self.parser = parser.Copy()
	self.parser.ParseArgs(self.params)
	self.profile = nil

	return self.parser, true
}

func (self *ParseAtOffset) ParseArgs(args *json.RawMessage) error {
	self.params = args
	return nil
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
