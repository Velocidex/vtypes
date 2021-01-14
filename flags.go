package vtypes

import (
	"fmt"
	"io"

	"github.com/Velocidex/ordereddict"
	"www.velocidex.com/golang/vfilter"
)

// Accepts option bitmap: name (string) -> bit number

type Flags struct {
	Type   string `json:"type"`
	Bitmap map[int64]string
	bits   []int64
	parser Parser
}

func (self *Flags) New(profile *Profile, options *ordereddict.Dict) (Parser, error) {
	if options == nil {
		return nil, fmt.Errorf("Bitmap parser requires a type in the options")
	}

	parser_type, pres := options.GetString("type")
	if !pres {
		return nil, fmt.Errorf("Bitmap parser requires a type in the options")
	}

	parser, err := profile.GetParser(parser_type, ordereddict.NewDict())
	if err != nil {
		return nil, fmt.Errorf("Bitmap parser requires a type in the options: %w", err)
	}

	bitmap, pres := options.Get("bitmap")
	if !pres {
		bitmap = ordereddict.NewDict()
	}

	bitmap_dict, ok := bitmap.(*ordereddict.Dict)
	if !ok {
		return nil, fmt.Errorf("Bitmap parser requires bitmap to be a mapping between names and the bit number")
	}

	result := &Flags{
		Bitmap: make(map[int64]string),
		parser: parser,
	}

	for _, name := range bitmap_dict.Keys() {
		idx_any, _ := bitmap_dict.Get(name)
		idx, ok := to_int64(idx_any)
		if !ok || idx < 0 || idx >= 64 {
			return nil, fmt.Errorf("Bitmap parser requires bitmap bit number between 0 and 64")
		}

		result.Bitmap[int64(1)<<idx] = name
		result.bits = append(result.bits, int64(1)<<idx)
	}

	return result, nil
}

func (self *Flags) Parse(
	scope vfilter.Scope, reader io.ReaderAt, offset int64) interface{} {
	result := []string{}

	value, ok := to_int64(self.parser.Parse(scope, reader, offset))
	if !ok {
		return result
	}

	for _, idx := range self.bits {
		if idx&value != 0 {
			result = append(result, self.Bitmap[idx])
		}
	}
	return result
}
