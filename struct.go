package vtypes

import (
	"io"
	"strings"

	"github.com/Velocidex/ordereddict"
	"www.velocidex.com/golang/vfilter"
)

type StructParser struct {
	type_name string
	size      int

	size_expression *vfilter.Lambda

	// Maintain the order of the fields.
	fields      map[string]Parser
	field_names []string
}

// StructParser does not take options
func (self *StructParser) New(profile *Profile, options *ordereddict.Dict) (Parser, error) {
	return self, nil
}

func (self *StructParser) Size() int {
	return self.size
}

func (self *StructParser) AddField(field_name string, parser *ParseAtOffset) {
	self.fields[field_name] = parser
	self.field_names = append(self.field_names, field_name)
}

func (self *StructParser) Parse(
	scope vfilter.Scope,
	reader io.ReaderAt, offset int64) interface{} {

	ScopeDebug(scope, "Instantiating struct %v on %v\n", self.type_name, offset)

	obj := &StructObject{
		parser: self,
		reader: reader,
		offset: offset,
	}

	// All dependencies will use this as the current struct
	subscope := scope.Copy()
	defer subscope.Close()

	subscope.AppendVars(ordereddict.NewDict().Set("this", obj))
	obj.scope = subscope

	return obj
}

func NewStructParser(type_name string, size int) *StructParser {
	result := &StructParser{
		type_name: type_name,
		size:      size,
		fields:    make(map[string]Parser),
	}

	return result
}

// A parser that parses its delegate at a particular offset
type ParseAtOffset struct {
	// Field offset within the struct.
	offset            int64
	offset_expression *vfilter.Lambda

	type_name string

	// Delegate parser
	parser Parser
}

func (self *ParseAtOffset) New(profile *Profile, options *ordereddict.Dict) (Parser, error) {
	return self, nil
}

func (self *ParseAtOffset) getOffset(scope vfilter.Scope) int64 {
	if self.offset_expression == nil {
		return self.offset
	}

	return EvalLambdaAsInt64(self.offset_expression, scope)
}

// NOTE: offset is the offset to the start of the struct.
func (self *ParseAtOffset) Parse(scope vfilter.Scope,
	reader io.ReaderAt, offset int64) interface{} {

	if IsNil(self.parser) {
		return vfilter.Null{}
	}

	// Get the field offset from the start of the struct.
	field_offset := self.getOffset(scope)

	// Apply the field parser on the combined offset.
	return self.parser.Parse(scope, reader, offset+field_offset)
}

// A Lazy object representing the struct
type StructObject struct {
	parser *StructParser
	reader io.ReaderAt
	offset int64

	// The subscope in which to evaluate expressions. In this
	// subscope "this" is assigned to this StructObject.
	scope vfilter.Scope

	// Cache the output of Get()
	cache map[string]interface{}

	parent *StructObject
}

func (self *StructObject) Start() int64 {
	return self.offset
}

func (self *StructObject) End() int64 {
	return self.offset + int64(self.Size())
}

func (self *StructObject) Get(field string) (interface{}, bool) {
	if self.cache == nil {
		self.cache = make(map[string]interface{})
	}

	hit, pres := self.cache[field]
	if pres {
		return hit, true
	}

	parser, pres := self.parser.fields[field]
	if !pres {
		return vfilter.Null{}, false
	}

	res := parser.Parse(self.scope, self.reader, self.offset)
	switch t := res.(type) {
	case *StructObject:
		t.parent = self

	case *ArrayObject:
		t.SetParent(self)
	}

	self.cache[field] = res
	return res, true
}

// Get the size of the struct - it can either be fixed, or derived
// using a lambda expression.
func (self *StructObject) Size() int {
	if self.parser.size_expression != nil {
		return int(EvalLambdaAsInt64(self.parser.size_expression, self.scope))
	}

	return self.parser.size
}

func (self *StructObject) Parent() vfilter.Any {
	if self.parent == nil {
		return vfilter.Null{}
	}
	return self.parent
}

func (self *StructObject) MarshalJSON() ([]byte, error) {
	result := ordereddict.NewDict()
	for _, field_name := range self.parser.field_names {
		if strings.HasPrefix(field_name, "__") {
			continue
		}
		value, ok := self.Get(field_name)
		if ok && value != self && value != self.parent {
			result.Set(field_name, value)
		}
	}
	res, err := result.MarshalJSON()
	return res, err
}
