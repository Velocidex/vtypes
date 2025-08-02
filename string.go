package vtypes

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"unicode/utf16"

	"github.com/Velocidex/ordereddict"
	"www.velocidex.com/golang/vfilter"
)

var (
	defaultTerm = "\x00"
)

type StringParserOptions struct {
	Length           *int64 `vfilter:"optional,lambda=LengthExpression,field=length,doc=Length of the string to read in bytes (Can be a lambda)"`
	LengthExpression *vfilter.Lambda
	MaxLength        int64           `vfilter:"optional,field=max_length,doc=Maximum length that is enforced on the string size"`
	Term             *string         `vfilter:"optional,lambda=TermExpression,field=term,doc=Terminating string (can be an expression)"`
	TermHex          *string         `vfilter:"optional,field=term_hex,doc=A Terminator in hex encoding"`
	TermExpression   *vfilter.Lambda `vfilter:"optional,field=term_exp,doc=A Terminator expression"`
	Encoding         string          `vfilter:"optional,field=encoding,doc=The encoding to use, can be utf8 or utf16"`
	Bytes            bool            `vfilter:"optional,field=byte_string,doc=Terminating string (can be an expression)"`

	utf16 bool
}

type StringParser struct {
	options StringParserOptions
}

func (self *StringParser) New(profile *Profile, options *ordereddict.Dict) (Parser, error) {
	result := &StringParser{}
	ctx := context.Background()
	err := ParseOptions(ctx, options, &result.options)
	if err != nil {
		return nil, fmt.Errorf("StringParser: %v", err)
	}

	if result.options.MaxLength == 0 {
		result.options.MaxLength = 1024
	}

	switch result.options.Encoding {
	case "utf8", "":
	case "utf16":
		result.options.utf16 = true
	default:
		return nil, fmt.Errorf("StringParser: encoding can only be utf8 or utf16")
	}

	if result.options.TermHex != nil {
		term, err := hex.DecodeString(*result.options.TermHex)
		if err != nil {
			return nil, err
		}
		term_str := string(term)
		result.options.Term = &term_str
	}

	return result, nil
}

func (self *StringParser) InstanceSize(
	scope vfilter.Scope,
	reader io.ReaderAt, offset int64) int {

	// The length of the string we are allowed to read.
	result_len := self.getCount(scope)

	buf := make([]byte, result_len)

	n, _ := reader.ReadAt(buf, offset)
	result := buf[:n]

	// If a terminator is specified read up to that.
	term := defaultTerm

	// if lamda term_exp configured evaluate and add as a standard
	// term
	if self.options.TermExpression != nil {
		term = EvalLambdaAsString(
			self.options.TermExpression, scope)
	}

	if self.options.Term != nil {
		term = *self.options.Term
	}

	// We need to bisect the read buffer by the terminator.
	var term_bytes []byte
	step := 1

	if self.options.utf16 {
		term_bytes = UTF16Encode(term)
		step = 2

	} else {
		term_bytes = []byte(term)
	}

	// Truncate to the right place by trying to find the
	// term_bytes. Note that UTF16 comparisons must be aligned to 2
	// bytes.
	if len(term_bytes) > 0 {
		for i := 0; i < len(result); i += step {
			if bytes.HasPrefix(result[i:], term_bytes) {
				// Include the terminator in the size as it is
				// technically part of the string.
				return i + len(term_bytes)
			}
		}
	}

	// Does not include the terminator
	return len(result)
}

func (self *StringParser) getCount(scope vfilter.Scope) int64 {
	var result int64 = 1024

	// If length is not specified, we read 1kb and look for the
	// terminator.
	if self.options.Length != nil {
		result = *self.options.Length
	}

	if self.options.LengthExpression != nil {
		// Evaluate the offset expression with the current scope.
		result = EvalLambdaAsInt64(self.options.LengthExpression, scope)
	}

	if result > self.options.MaxLength {
		return self.options.MaxLength
	}

	return result
}

func (self *StringParser) Parse(
	scope vfilter.Scope, reader io.ReaderAt, offset int64) interface{} {

	result := self._Parse(scope, reader, offset)
	if self.options.Bytes {
		return result
	}

	return string(result)

}

func (self *StringParser) _Parse(
	scope vfilter.Scope, reader io.ReaderAt, offset int64) []byte {

	result_len := self.getCount(scope)

	buf := make([]byte, result_len)

	n, _ := reader.ReadAt(buf, offset)
	result := buf[:n]

	// If a terminator is specified read up to that.
	term := defaultTerm

	// if lamda term_exp configured evaluate and add as a standard
	// term
	if self.options.TermExpression != nil {
		term = EvalLambdaAsString(
			self.options.TermExpression, scope)
	}

	if self.options.Term != nil {
		term = *self.options.Term
	}

	// We need to bisect the read buffer by the terminator.
	var term_bytes []byte
	step := 1

	if self.options.utf16 {
		term_bytes = UTF16Encode(term)
		step = 2

	} else {
		term_bytes = []byte(term)
	}

	// Truncate to the right place by trying to find the
	// term_bytes. Note that UTF16 comparisons must be aligned to 2
	// bytes.
	if len(term_bytes) > 0 {
		for i := 0; i < len(result); i += step {
			if bytes.HasPrefix(result[i:], term_bytes) {
				result = result[:i]
				break
			}
		}
	}

	if self.options.utf16 {
		return []byte(UTF16Decode(result))
	}

	return result
}

func UTF16Encode(in string) []byte {
	buf := bytes.NewBuffer(nil)
	ints := utf16.Encode([]rune(in))
	binary.Write(buf, binary.LittleEndian, &ints)
	return buf.Bytes()
}

func UTF16Decode(in []byte) string {
	ints := make([]uint16, len(in)/2)
	binary.Read(bytes.NewReader([]byte(in)), binary.LittleEndian, &ints)
	return string(utf16.Decode(ints))
}
