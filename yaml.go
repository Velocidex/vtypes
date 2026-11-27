package vtypes

import (
	"errors"
	"fmt"

	"github.com/Velocidex/ordereddict"
	"go.yaml.in/yaml/v4"
)

func (self *StructDefinition) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.SequenceNode || len(node.Content) < 3 {
		return errors.New("expected a struct definition [name, size, fields]")
	}

	name := resolveAlias(node.Content[0])
	if name.ShortTag() != "!!str" {
		return errors.New("Name should be a string")
	}
	result := StructDefinition{Name: name.Value}
	size, expr, err := numberOrExpression(node.Content[1])
	if err != nil {
		return fmt.Errorf("%s: size: %w", result.Name, err)
	}
	result.Size, result.SizeExpression = int(size), expr

	fields := resolveAlias(node.Content[2])
	if fields.Kind != yaml.SequenceNode {
		return errors.New("Fields should be a list of field definitions")
	}
	for _, entry := range fields.Content {
		field, err := decodeYAMLField(resolveAlias(entry))
		if err != nil {
			return fmt.Errorf("%s: %w", result.Name, err)
		}
		result.Fields = append(result.Fields, field)
	}
	*self = result
	return nil
}

func decodeYAMLField(node *yaml.Node) (*FieldDefinition, error) {
	if node.Kind != yaml.SequenceNode || (len(node.Content) != 3 && len(node.Content) != 4) {
		return nil, errors.New("expected a field definition [name, offset, type, options?]")
	}
	parts := node.Content
	name := resolveAlias(parts[0])
	if name.ShortTag() != "!!str" {
		return nil, errors.New("field name should be a string")
	}
	field := &FieldDefinition{Name: name.Value}
	var err error
	field.Offset, field.OffsetExpression, err = numberOrExpression(parts[1])
	if err != nil {
		return nil, fmt.Errorf("field %s offset: %w", field.Name, err)
	}
	fieldType := resolveAlias(parts[2])
	if fieldType.ShortTag() != "!!str" {
		return nil, fmt.Errorf("field %s type should be a string", field.Name)
	}
	field.Type = fieldType.Value
	if len(parts) == 4 {
		options := resolveAlias(parts[3])
		if options.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("field %s options should be a map", field.Name)
		}
		field.Options = ordereddict.NewDict()
		if hasSpecialKeys(options, make(map[*yaml.Node]bool)) {
			// Let the YAML loader check the complete graph before following aliases.
			var decoded map[interface{}]interface{}
			if err = options.Load(&decoded, yaml.WithV2Defaults()); err == nil {
				field.Options, err = decodeProfileOptions(options)
			}
		} else {
			err = field.Options.UnmarshalYAML(options)
		}
		if err != nil {
			return nil, fmt.Errorf("field %s options: %w", field.Name, err)
		}
	}
	return field, nil
}

func resolveAlias(node *yaml.Node) *yaml.Node {
	if node.Kind == yaml.AliasNode {
		return node.Alias
	}
	return node
}

func numberOrExpression(node *yaml.Node) (int64, string, error) {
	node = resolveAlias(node)
	if node.Kind == yaml.ScalarNode {
		var value interface{}
		if err := node.Load(&value, yaml.WithV2Defaults()); err != nil {
			return 0, "", err
		}
		if number, ok := to_int64(value); ok {
			return number, "", nil
		}
		if expression, ok := value.(string); ok {
			return 0, expression, nil
		}
	}
	return 0, "", errors.New("expected a number or expression")
}

// Resolve keys at this mapping level. ordereddict handles ordinary values,
// while the YAML loader supplies effective entries from aliases and merges.
func decodeProfileOptions(node *yaml.Node) (*ordereddict.Dict, error) {
	var entries map[interface{}]yaml.Node
	if err := node.Load(&entries, yaml.WithV2Defaults()); err != nil {
		return nil, err
	}
	mapping := *node
	mapping.Content = append([]*yaml.Node(nil), node.Content...)
	for i := 0; i < len(mapping.Content); i += 2 {
		mapping.Content[i] = resolveAlias(mapping.Content[i])
	}
	dict := ordereddict.NewDict()
	if err := dict.UnmarshalYAML(&mapping); err != nil {
		return nil, err
	}
	for k, entry := range entries {
		key, ok := k.(string)
		if !ok {
			return nil, errors.New("keys should be strings")
		}
		value := resolveAlias(&entry)
		if _, present := dict.Get(key); present && !hasSpecialKeys(value, make(map[*yaml.Node]bool)) {
			continue
		}
		var decoded interface{}
		var err error
		if value.Kind == yaml.MappingNode {
			decoded, err = decodeProfileOptions(value)
		} else {
			// v2 left lists as native Go values, including numeric map keys.
			err = value.Load(&decoded, yaml.WithV2Defaults())
		}
		if err != nil {
			return nil, err
		}
		dict.Set(key, decoded)
	}
	return dict, nil
}

func hasSpecialKeys(node *yaml.Node, seen map[*yaml.Node]bool) bool {
	node = resolveAlias(node)
	if seen[node] {
		return false
	}
	seen[node] = true
	for i, child := range node.Content {
		if node.Kind == yaml.MappingNode && i%2 == 0 && child.Tag != "!!str" && child.Tag != "!!merge" {
			return true
		}
		if hasSpecialKeys(child, seen) {
			return true
		}
	}
	return false
}
