// Implements a binary parsing system.
package vtypes

import (
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

// Used by parsers or wrappers who have fixed size
type Sizer interface {
	Size() int
}

// Applies on a parser which needs to instantiate to figure out the
// size. i.e. the size depends on the data read (e.g. a string).
type InstanceSizer interface {
	InstanceSize(scope vfilter.Scope, reader io.ReaderAt, offset int64) int
}

// Allows psuedo elements to reveal their own value.
type Valuer interface {
	Value() interface{}
}

// Return the start and end of the object
type Starter interface {
	Start() int64
}

type Ender interface {
	End() int64
}
