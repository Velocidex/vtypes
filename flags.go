package vtypes

import (
	"context"
	"fmt"
	"io"
	"sort"

	"github.com/Velocidex/ordereddict"
	"www.velocidex.com/golang/vfilter"
)

// Accepts option bitmap: name (string) -> bit number
type FlagsOptions struct {
	Type        string            `vfilter:"required,field=type,doc=The underlying type of the choice"`
	TypeOptions *ordereddict.Dict `vfilter:"optional,field=type_options,doc=Any additional options required to parse the type"`
	Bitmap      *ordereddict.Dict `vfilter:"required,field=bitmap,doc=A mapping between names and the bit number"`

	bits   []int64
	bitmap map[int64]string
}

type Flags struct {
	options FlagsOptions
	profile *Profile
	parser  Parser
}

func (self *Flags) New(profile *Profile, options *ordereddict.Dict) (Parser, error) {
	if options == nil {
		return nil, fmt.Errorf("Bitmap parser requires a type in the options")
	}

	result := &Flags{profile: profile}
	ctx := context.Background()
	err := ParseOptions(ctx, options, &result.options)
	if err != nil {
		return nil, fmt.Errorf("FlagsParser: %v", err)
	}
	result.options.bitmap = make(map[int64]string)

	for _, name := range result.options.Bitmap.Keys() {
		idx_any, _ := result.options.Bitmap.Get(name)
		idx, ok := to_int64(idx_any)
		if !ok || idx < 0 || idx >= 64 {
			return nil, fmt.Errorf(
				"Bitmap parser requires bitmap bit number between 0 and 64")
		}

		result.options.bitmap[int64(1)<<idx] = name
		result.options.bits = append(result.options.bits, int64(1)<<idx)
	}

	// Type must be available at definition time because flag fields
	// can not operate on custome types.
	result.parser, err = profile.GetParser(
		result.options.Type, result.options.TypeOptions)
	return result, err
}

func (self *Flags) Parse(
	scope vfilter.Scope, reader io.ReaderAt, offset int64) interface{} {
	result := []string{}

	if self.parser == nil {
		return vfilter.Null{}
	}

	value, ok := to_int64(self.parser.Parse(scope, reader, offset))
	if !ok {
		return result
	}

	for _, idx := range self.options.bits {
		if idx&value != 0 {
			result = append(result, self.options.bitmap[idx])
		}
	}

	// Sort result to maintain stable output.
	sort.Strings(result)
	return result
}
