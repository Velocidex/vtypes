package vtypes

import (
	"fmt"
	"io"
	"sort"

	"github.com/Velocidex/ordereddict"
	"www.velocidex.com/golang/vfilter"
)

// Accepts option bitmap: name (string) -> bit number
type FlagsOptions struct {
	Type        string
	TypeOptions *ordereddict.Dict
	Bitmap      map[int64]string
	Bits        []int64
}

type Flags struct {
	options FlagsOptions
	profile *Profile
	parser  Parser
}

func (self *Flags) New(profile *Profile, options *ordereddict.Dict) (Parser, error) {
	var pres bool

	if options == nil {
		return nil, fmt.Errorf("Bitmap parser requires a type in the options")
	}

	result := &Flags{profile: profile}

	result.options.Type, pres = options.GetString("type")
	if !pres {
		return nil, fmt.Errorf("Bitmap parser requires a type in the options")
	}

	topts, pres := options.Get("type_options")
	if pres {
		topts_dict, ok := topts.(*ordereddict.Dict)
		if !ok {
			return nil, fmt.Errorf("Enumeration parser options should be a dict")
		}
		result.options.TypeOptions = topts_dict
	}

	bitmap, pres := options.Get("bitmap")
	if !pres {
		bitmap = ordereddict.NewDict()
	}

	bitmap_dict, ok := bitmap.(*ordereddict.Dict)
	if !ok {
		return nil, fmt.Errorf("Bitmap parser requires bitmap to be a mapping between names and the bit number")
	}

	result.options.Bitmap = make(map[int64]string)

	for _, name := range bitmap_dict.Keys() {
		idx_any, _ := bitmap_dict.Get(name)
		idx, ok := to_int64(idx_any)
		if !ok || idx < 0 || idx >= 64 {
			return nil, fmt.Errorf("Bitmap parser requires bitmap bit number between 0 and 64")
		}

		result.options.Bitmap[int64(1)<<idx] = name
		result.options.Bits = append(result.options.Bits, int64(1)<<idx)
	}

	return result, nil
}

func (self *Flags) Parse(
	scope vfilter.Scope, reader io.ReaderAt, offset int64) interface{} {
	result := []string{}

	if self.parser == nil {
		parser, err := self.profile.GetParser(
			self.options.Type, self.options.TypeOptions)
		if err != nil {
			scope.Log("ERROR:binary_parser: Flags: %v", err)
			self.parser = NullParser{}
			return vfilter.Null{}
		}

		// Cache the parser for next time.
		self.parser = parser
	}

	value, ok := to_int64(self.parser.Parse(scope, reader, offset))
	if !ok {
		return result
	}

	for _, idx := range self.options.Bits {
		if idx&value != 0 {
			result = append(result, self.options.Bitmap[idx])
		}
	}

	// Sort result to maintain stable output.
	sort.Strings(result)
	return result
}
