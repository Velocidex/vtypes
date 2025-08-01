package vtypes

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/Velocidex/ordereddict"
	"www.velocidex.com/golang/vfilter"
	"www.velocidex.com/golang/vfilter/types"
)

// Structs may tag fields with this name to control parsing.
const tagName = "vfilter"

func getTag(field reflect.StructField) map[string]string {
	options := make(map[string]string)

	tag := field.Tag.Get(tagName)

	// Skip if tag is not defined or ignored
	if tag == "" || tag == "-" {
		return nil
	}

	directives := strings.Split(tag, ",")
	for _, directive := range directives {
		if strings.Contains(directive, "=") {
			components := strings.Split(directive, "=")
			if len(components) >= 2 {
				options[components[0]] = components[1]
			}
		} else {
			options[directive] = "Y"
		}
	}

	return options
}

func ParseOptions(ctx context.Context, args *ordereddict.Dict, target interface{}) error {
	v := reflect.ValueOf(target)
	t := v.Type()

	if t.Kind() == reflect.Ptr {
		v = v.Elem()
		t = v.Type()
	}

	if t.Kind() != reflect.Struct {
		return errors.New("Only structs can be set with ParseOptions()")
	}

	args_spefied := make(map[string]bool)
	for _, k := range args.Keys() {
		args_spefied[k] = true
	}

	for i := 0; i < v.NumField(); i++ {
		// Get the field tag value
		field_types_value := t.Field(i)
		options := getTag(field_types_value)
		if options == nil {
			continue
		}

		// Is the name specified in the tag?
		field_name, pres := options["field"]
		if !pres {
			field_name = field_types_value.Name
		}

		field_value := v.Field(field_types_value.Index[0])
		if !field_value.IsValid() || !field_value.CanSet() {
			return fmt.Errorf("Field %s is unsettable.", field_name)
		}

		_, required := options["required"]
		if required {
			_, pres := args.Get(field_name)
			if !pres {
				return fmt.Errorf("Field %v is required in %T",
					field_name, target)
			}
		}

		field_data, pres := args.Get(field_name)
		if !pres {
			continue
		}
		delete(args_spefied, field_name)

		// Reduce if needed.
		lazy_arg, ok := field_data.(types.LazyExpr)
		if ok {
			field_data = lazy_arg.Reduce(ctx)
		}

		// Does it look like a lambda?
		if isFieldLambda(field_data) {
			// The field tag indicates to store the lambda in an
			// alternative field.
			target_field, pres := options["lambda"]
			if pres {
				lambda, err := vfilter.ParseLambda(field_data.(string))
				if err != nil {
					return fmt.Errorf("Error parsing lambda for field %v: %v",
						field_name, err)
				}

				// Set the other field with the lambda
				lambda_target := v.FieldByName(target_field)
				if !lambda_target.IsValid() || !lambda_target.CanSet() {
					return fmt.Errorf(
						"field %v wants to store lambda in %v but this field does not exist",
						field_name, target_field)
				}

				lambda_target.Set(reflect.ValueOf(lambda))
				continue
			}
		}

		switch field_types_value.Type.String() {

		case "string":
			str, ok := field_data.(string)
			if ok {

				field_value.Set(reflect.ValueOf(str))
			}

		case "int64":
			a, ok := to_int64(field_data)
			if ok {
				field_value.Set(reflect.ValueOf(int64(a)))
				continue
			}
			return fmt.Errorf("field %v: Expecting an integer not %T",
				field_name, field_data)

		case "uint64":
			a, ok := to_int64(field_data)
			if ok {
				field_value.Set(reflect.ValueOf(uint64(a)))
				continue
			}
			return fmt.Errorf("field %v: Expecting an integer not %T",
				field_name, field_data)

		case "bool":
			a, ok := to_int64(field_data)
			if ok {
				field_value.Set(reflect.ValueOf(a > 0))
				continue
			}
			return fmt.Errorf("field %v: Expecting a bool not %T",
				field_name, field_data)

		case "*string":
			str, ok := field_data.(string)
			if ok {
				x := reflect.New(field_value.Type().Elem())
				x.Elem().Set(reflect.ValueOf(str))
				field_value.Set(x)
				continue
			}
			return fmt.Errorf("field %v: Expecting a string not %T",
				field_name, field_data)

		case "*int64":
			a, ok := to_int64(field_data)
			if ok {
				x := reflect.New(field_value.Type().Elem())
				x.Elem().Set(reflect.ValueOf(a))
				field_value.Set(x)
				continue
			}
			return fmt.Errorf("field %v: Expecting a string not %T",
				field_name, field_data)

		case "*ordereddict.Dict":
			dict, ok := field_data.(*ordereddict.Dict)
			if ok {
				field_value.Set(reflect.ValueOf(dict))
				continue
			}

			map_obj, ok := field_data.(map[string]interface{})
			if ok {
				res := ordereddict.NewDict()
				for k, v := range map_obj {
					res.Set(k, v)
				}
				field_value.Set(reflect.ValueOf(res))
				continue
			}

			return fmt.Errorf("field %v: Expecting a mapping not %T",
				field_name, field_data)

		case "*vfilter.Lambda":
			str, ok := field_data.(string)
			if ok {
				lambda, err := vfilter.ParseLambda(str)
				if err != nil {
					return fmt.Errorf("Error parsing lambda for field %v: %v",
						field_name, err)
				}

				field_value.Set(reflect.ValueOf(lambda))
				continue
			}
			return fmt.Errorf("field %v: Expecting a vql lambda not %T",
				field_name, field_data)

		default:
			fmt.Printf("Unable to handle field type %v\n",
				field_types_value.Type.String())

		}
	}

	// Report any unexpected parameters
	if len(args_spefied) > 0 {
		var extras []string
		for k := range args_spefied {
			extras = append(extras, k)
		}
		return fmt.Errorf("Unexpected parameters provided: %v", extras)
	}

	return nil
}

var (
	lambdaRegex = regexp.MustCompile("^[a-zA-Z0-9]+ *=>")
)

func isFieldLambda(value interface{}) bool {
	str, ok := value.(string)
	if !ok {
		return false
	}

	return lambdaRegex.MatchString(str)
}

func maybeGetParser(
	profile *Profile,
	type_name string,
	options *ordereddict.Dict) (Parser, error) {
	// Get the parser now so we can catch errors in sub parser
	// definitions
	parser, err := profile.GetParser(type_name, options)

	// It is fine if the underlying type is not known yet. It may be
	// defined later.
	if err != nil && !errors.Is(err, NotFoundError) {
		return nil, err
	}

	return parser, nil
}
