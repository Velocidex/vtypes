package vtypes

import (
	"errors"
	"fmt"

	"github.com/Velocidex/ordereddict"
)

func (self *StructDefinition) UnmarshalYAML(unmarshal func(v interface{}) error) error {
	var values []interface{}
	err := unmarshal(&values)
	if err != nil {
		return err
	}
	ok := false

	self.Name, ok = values[0].(string)
	if !ok {
		return errors.New("Name should be a string")
	}

	size, ok := to_int64(values[1])
	if ok {
		self.Size = int(size)

	} else {
		self.SizeExpression, ok = values[1].(string)
		if !ok {
			return errors.New("Size should be a string or integer")
		}
	}

	fields, ok := values[2].([]interface{})
	if !ok {
		return errors.New("Fields should be a list of field definitions")
	}

	for _, field_def := range fields {
		field, ok := field_def.([]interface{})
		if !ok {
			return fmt.Errorf("%v: Field Definition should be [name, offset, type, options?]",
				self.Name)
		}

		if len(field) != 3 && len(field) != 4 {
			return fmt.Errorf("%v: Field Definition should be [name, offset, type, options?]",
				self.Name)
		}

		new_field := &FieldDefinition{}
		new_field.Name, ok = field[0].(string)
		if !ok {
			return fmt.Errorf("%v: field name should be a string", self.Name)
		}

		offset, ok := to_int64(field[1])
		if ok {
			new_field.Offset = int64(offset)

		} else {
			new_field.OffsetExpression, ok = field[1].(string)
			if !ok {
				return fmt.Errorf("%v: field %v size should be a string or int",
					self.Name, new_field.Name)
			}
		}

		new_field.Type, ok = field[2].(string)
		if !ok {
			return fmt.Errorf("%v: field %v type should be a string",
				self.Name, new_field.Name)
		}

		if len(field) == 4 {
			option_map, ok := field[3].(map[interface{}]interface{})
			if !ok {
				return fmt.Errorf("%v: field %v options should be a map",
					self.Name, new_field.Name)
			}
			options, err := to_ordereddict(option_map)
			if err != nil {
				return fmt.Errorf("%v: field %v options %v",
					self.Name, new_field.Name, err)
			}
			new_field.Options = options
		}
		self.Fields = append(self.Fields, new_field)
	}

	return nil
}

func to_ordereddict(dict map[interface{}]interface{}) (*ordereddict.Dict, error) {
	var err error
	result := ordereddict.NewDict()
	for k, v := range dict {
		opt_name, ok := k.(string)
		if !ok {
			return nil, errors.New("keys should be strings")
		}
		v_dict, ok := v.(map[interface{}]interface{})
		if ok {
			v, err = to_ordereddict(v_dict)
			if err != nil {
				return nil, err
			}
		}
		result.Set(opt_name, v)
	}

	return result, nil
}
