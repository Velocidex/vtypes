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

func (self *StringParser) getCount(scope vfilter.Scope) int64 {
	var result int64 = 1024

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
	scope vfilter.Scope,
	reader io.ReaderAt, offset int64) interface{} {

	result_len := self.getCount(scope)

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

	if term != "" {
		idx := bytes.Index(result, []byte(term))
		if idx >= 0 {
			result = result[:idx]
		}
	}

	if self.options.Bytes {
		return result
	}

	return string(result)
}
