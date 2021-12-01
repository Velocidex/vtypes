package vtypes

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"unicode/utf16"

	"github.com/Velocidex/ordereddict"
	"www.velocidex.com/golang/vfilter"
)

type StringParserOptions struct {
	Length           int64
	LengthExpression *vfilter.Lambda
	MaxLength        int64
	Term             string
	TermExpression   *vfilter.Lambda
	Encoding         string
}

type StringParser struct {
	options StringParserOptions
}

func (self *StringParser) New(profile *Profile, options *ordereddict.Dict) (Parser, error) {
	var pres bool
	result := &StringParser{}

	if options == nil {
		options = ordereddict.NewDict()
	}

	// Some defaults
	result.options.Length = -1 // -1 means this is not set
	result.options.MaxLength = 1024

	result.options.Encoding, _ = options.GetString("encoding")
	result.options.Term, pres = options.GetString("term")
	if !pres {
		result.options.Term = "\x00"
	}

	termhex, pres := options.GetString("term_hex")
	if pres {
		term, err := hex.DecodeString(termhex)
		if err != nil {
			return nil, err
		}
		result.options.Term = string(term)
	}

	// Add a termexpression if exist
	termexpression, _ := options.GetString("term_exp")
	if termexpression != "" {
		var err error
		result.options.TermExpression, err = vfilter.ParseLambda(termexpression)
		if err != nil {
			return nil, fmt.Errorf("String parser term expression '%v': %w",
				termexpression, err)
		}
	}

	// Default to 0 length
	length, pres := options.GetInt64("length")
	if pres {
		result.options.Length = length
	}

	max_length, pres := options.GetInt64("max_length")
	if pres {
		result.options.MaxLength = max_length
	}

	// Maybe add a length expression if length is a string.
	expression, _ := options.GetString("length")
	if expression != "" {
		var err error
		result.options.LengthExpression, err = vfilter.ParseLambda(expression)
		if err != nil {
			return nil, fmt.Errorf("String parser length expression '%v': %w",
				expression, err)
		}
	}

	return result, nil
}

func (self *StringParser) getCount(scope vfilter.Scope) int64 {
	result := self.options.Length

	if self.options.LengthExpression != nil {
		// Evaluate the offset expression with the current scope.
		return EvalLambdaAsInt64(self.options.LengthExpression, scope)
	}

	if result > self.options.MaxLength {
		return self.options.MaxLength
	}

	return result
}

func (self *StringParser) Parse(
	scope vfilter.Scope,
	reader io.ReaderAt, offset int64) interface{} {

	result_len := self.getCount(scope)
	if result_len < 0 {
		// length is not specified - max read 1kb.
		result_len = 1024
	}

	buf := make([]byte, result_len)

	n, _ := reader.ReadAt(buf, offset)
	result := buf[:n]

	// If encoding is specified, convert from utf16
	if self.options.Encoding == "utf16" {
		order := binary.LittleEndian
		u16s := []uint16{}

		for i, j := 0, len(result); i < j; i += 2 {
			if len(result) < i+2 {
				break
			}
			u16s = append(u16s, order.Uint16(result[i:]))
		}

		result = []byte(string(utf16.Decode(u16s)))
	}

	// if lamda term_exp configured evaluate and add as a standard term
	if self.options.TermExpression != nil {
		self.options.Term = EvalLambdaAsString(self.options.TermExpression, scope)
	}

	// If a terminator is specified read up to that.
	if self.options.Term != "" {
		idx := bytes.Index(result, []byte(self.options.Term))
		if idx >= 0 {
			result = result[:idx]
		}
	}

	return string(result)
}
