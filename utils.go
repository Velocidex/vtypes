package vtypes

import (
	"context"
	"reflect"
	"strings"

	"www.velocidex.com/golang/vfilter"
)

func to_int64(x interface{}) (int64, bool) {
	switch t := x.(type) {
	case bool:
		if t {
			return 1, true
		} else {
			return 0, true
		}
	case int:
		return int64(t), true
	case uint8:
		return int64(t), true
	case int8:
		return int64(t), true
	case uint16:
		return int64(t), true
	case int16:
		return int64(t), true
	case uint32:
		return int64(t), true
	case int32:
		return int64(t), true
	case uint64:
		return int64(t), true
	case int64:
		return t, true
	case float64:
		return int64(t), true

	case *int:
		return int64(*t), true
	case *uint8:
		return int64(*t), true
	case *int8:
		return int64(*t), true
	case *uint16:
		return int64(*t), true
	case *int16:
		return int64(*t), true
	case *uint32:
		return int64(*t), true
	case *int32:
		return int64(*t), true
	case *uint64:
		return int64(*t), true
	case *int64:
		return int64(*t), true
	case *float64:
		return int64(*t), true

	default:
		return 0, false
	}
}

// Some helpers

func SizeOf(obj interface{}) int {
	sizer, ok := obj.(Sizer)
	if ok {
		return sizer.Size()
	}
	return 0
}

func StartOf(obj interface{}) int64 {
	start, ok := obj.(Starter)
	if ok {
		return start.Start()
	}
	return 0
}

func EndOf(obj interface{}) int64 {
	end, ok := obj.(Ender)
	if ok {
		return end.End()
	}
	return 0
}

func Associative(scope *vfilter.Scope, a vfilter.Any, field string) vfilter.Any {
	var result vfilter.Any = a
	var ok bool

	for _, item := range strings.Split(field, ".") {
		result, ok = scope.Associative(result, item)
		if !ok {
			return vfilter.Null{}
		}
	}
	return result
}

// We need to do this stupid check because Go does not allow
// comparison to nil with interfaces.
func IsNil(v interface{}) bool {
	return v == nil || (reflect.ValueOf(v).Kind() == reflect.Ptr &&
		reflect.ValueOf(v).IsNil())
}

func EvalLambdaAsInt64(expression *vfilter.Lambda, scope *vfilter.Scope) int64 {
	this_obj, pres := scope.Resolve("this")
	if !pres {
		return 0
	}

	result := expression.Reduce(context.Background(), scope, []vfilter.Any{this_obj})
	result_int, _ := to_int64(result)
	return result_int
}
