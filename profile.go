//
package vtypes

import (
	"errors"
	"fmt"
	"io"

	"github.com/Velocidex/ordereddict"
	"github.com/Velocidex/yaml"

	"www.velocidex.com/golang/vfilter"
)

type FieldDefinition struct {
	Name string
	// Offset within the struct
	Offset int64

	// Alternatively offset may be given as an expression.
	OffsetExpression string

	// Name of the type of parser in this field.
	Type string

	// Options to the type
	Options *ordereddict.Dict
}

type StructDefinition struct {
	Name           string
	Size           int
	SizeExpression string
	Fields         []*FieldDefinition
}

type Profile struct {
	types map[string]Parser
}

func NewProfile() *Profile {
	result := Profile{
		types: make(map[string]Parser),
	}

	return &result
}

func (self *Profile) AddParser(type_name string, parser Parser) {
	self.types[type_name] = parser
}

func (self *Profile) GetParser(name string, options *ordereddict.Dict) (Parser, error) {
	parser, pres := self.types[name]
	if !pres {
		return nil, errors.New("Parser not found")
	}
	return parser.New(self, options)
}

func (self *Profile) ObjectSize(scope vfilter.Scope,
	name string, reader io.ReaderAt, offset int64) int {
	parser, pres := self.types[name]
	if pres {
		sizer, ok := parser.(Sizer)
		if ok {
			return sizer.Size()
		}
	}

	return 0
}

// Build the profile from definitions given in the vtypes language.
func (self *Profile) ParseStructDefinitions(definitions string) (err error) {
	var profile_definitions []*StructDefinition

	err = yaml.Unmarshal([]byte(definitions), &profile_definitions)
	if err != nil {
		return err
	}

	for _, struct_def := range profile_definitions {
		struct_parser := NewStructParser(struct_def.Name, struct_def.Size)
		self.types[struct_def.Name] = struct_parser

		// Try to parse it as a VQL Lambda
		if struct_def.SizeExpression != "" {
			struct_parser.size_expression, err = vfilter.ParseLambda(
				struct_def.SizeExpression)
			if err != nil {
				return fmt.Errorf("struct definition %v size expression '%v': %w",
					struct_def.Name, struct_def.SizeExpression, err)
			}
		}

		for _, field_def := range struct_def.Fields {
			// Install a parser now to maintain
			// field ordering but do not include
			// delegate parser yet
			temp_parser := &ParseAtOffset{
				offset: field_def.Offset,
			}
			struct_parser.AddField(field_def.Name, temp_parser)

			if field_def.OffsetExpression != "" {
				temp_parser.offset_expression, err = vfilter.ParseLambda(
					field_def.OffsetExpression)
				if err != nil {
					return fmt.Errorf("struct %v field offset '%v': %w",
						struct_def.Name, field_def.OffsetExpression, err)
				}
			}

			// Get the parser by name
			parser, pres := self.types[field_def.Type]
			if pres {
				temp_parser.parser, err = parser.New(self, field_def.Options)
				if err != nil {
					return fmt.Errorf("struct %v field '%v': %w",
						struct_def.Name, field_def.Name, err)
				}
			} else {

				// Delay the creation of the parser until we
				// have added all the structs in case the
				// parser name refers to a struct which has
				// not been defined yet.
				defer func(field_def *FieldDefinition, temp_parser *ParseAtOffset) {
					if err != nil {
						return
					}

					parser, pres := self.types[field_def.Type]
					if !pres {
						err = fmt.Errorf(
							"Reference to undefined type %v in %v.%v",
							field_def.Type, struct_def.Name,
							field_def.Name)
						return
					}
					temp_parser.parser, _ = parser.New(self, field_def.Options)
				}(field_def, temp_parser)
			}
		}

	}
	return nil
}

// Create a new object of the specified type by instantiating the
// named parser on the reader at the specified offset.

// For example:
// type_name = "Array"
// options = { "Target": "int"}
func (self *Profile) Parse(scope vfilter.Scope, type_name string,
	reader io.ReaderAt, offset int64) (interface{}, error) {
	parser, pres := self.types[type_name]
	if !pres {
		return nil, errors.New(
			fmt.Sprintf("Type name %s is not known.", type_name))
	}

	return parser.Parse(scope, reader, offset), nil
}
