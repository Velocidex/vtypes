package vtypes

import (
	"context"

	"github.com/Velocidex/ordereddict"
	"www.velocidex.com/golang/vfilter"
)

type StructAssociative struct{}

func (self StructAssociative) Applicable(a vfilter.Any, b vfilter.Any) bool {
	switch a.(type) {
	case StructObject, *StructObject:
		_, ok := b.(string)
		if ok {
			return true
		}
	}
	return false
}

func (self StructAssociative) Associative(scope vfilter.Scope,
	a vfilter.Any, b vfilter.Any) (vfilter.Any, bool) {
	lhs, ok := a.(*StructObject)
	if !ok {
		return vfilter.Null{}, false
	}

	rhs, ok := b.(string)
	if !ok {
		return vfilter.Null{}, false
	}

	switch rhs {
	case "SizeOf":
		return lhs.Size(), true

	case "StartOf":
		return lhs.Start(), true

	case "ParentOf":
		return lhs.Parent(), true

	case "EndOf":
		return lhs.End(), true

	default:
		return lhs.Get(rhs)
	}
}

func (self StructAssociative) GetMembers(scope vfilter.Scope, a vfilter.Any) []string {
	lhs, ok := a.(*StructObject)
	if !ok {
		return nil
	}

	return lhs.parser.field_names
}

type ArrayAssociative struct{}

func (self ArrayAssociative) Applicable(a vfilter.Any, b vfilter.Any) bool {
	switch a.(type) {
	case ArrayObject, *ArrayObject:
		_, ok := b.(string)
		if ok {
			return true
		}
	}
	return false
}

func (self ArrayAssociative) Associative(scope vfilter.Scope,
	a vfilter.Any, b vfilter.Any) (vfilter.Any, bool) {
	lhs, ok := a.(*ArrayObject)
	if !ok {
		return vfilter.Null{}, false
	}

	rhs, ok := b.(string)
	if !ok {
		return vfilter.Null{}, false
	}

	switch rhs {
	case "SizeOf":
		return lhs.Size(), true

	case "ContentsOf":
		return lhs.Contents(), true

	case "StartOf":
		return lhs.Start(), true

	case "EndOf":
		return lhs.End(), true

		// Provide a way to access the raw array
	case "Value":
		return lhs.contents, true

	default:
		// Fallback to associative on the underlying array.
		return scope.Associative(lhs.contents, b)
	}
}

func (self ArrayAssociative) GetMembers(scope vfilter.Scope, a vfilter.Any) []string {
	return nil
}

// Arrays also participate in the iterator protocol
type ArrayIterator struct{}

func (self ArrayIterator) Applicable(a vfilter.Any) bool {
	_, ok := a.(*ArrayObject)
	return ok
}

func (self ArrayIterator) Iterate(
	ctx context.Context, scope vfilter.Scope, a vfilter.Any) <-chan vfilter.Row {
	output_chan := make(chan vfilter.Row)

	go func() {
		defer close(output_chan)

		obj, ok := a.(*ArrayObject)
		if !ok {
			return
		}

		for _, item := range obj.contents {
			switch item.(type) {

			// We must emit objects with a valid Associative protocol
			// because this will form the basis for the columns in
			// foreach. These objects are ok to emit directly.
			case *ordereddict.Dict, *StructObject:
			default:
				// Anything else place inside a dict.
				item = ordereddict.NewDict().Set("_value", item)
			}

			select {
			case <-ctx.Done():
				return

			case output_chan <- item:
			}
		}
	}()

	return output_chan

}
