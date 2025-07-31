package vtypes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/Velocidex/ordereddict"
	"github.com/sebdah/goldie"
	assert "github.com/stretchr/testify/assert"
	"www.velocidex.com/golang/vfilter"
)

var (
	sample = []byte{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c,
		0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13,

		// Offset 19 - "hello\x00world\x00"
		0x68, 0x65, 0x6c, 0x6c, 0x6f, 0x00, 0x77, 0x6f, 0x72, 0x6c, 0x64, 0x00,

		// Offset 31 - utf16
		0x68, 0x00, 0x65, 0x00, 0x6c, 0x00, 0x6c, 0x00, 0x6f, 0x00, 0x00, 0x00,
		0x77, 0x00, 0x6f, 0x00, 0x72, 0x00, 0x6c, 0x00, 0x64, 0x00, 0x00, 0x00,

		// Offset 55 - uint32 timestamp
		0x61, 0x9c, 0x53, 0x65, 0x00, 0x00, 0x00, 0x00,

		// Offset 63 - uint64 timestamp in millisec
		0x40, 0x1a, 0xe7, 0x0c, 0x1f, 0x0a, 0x06, 0x00,

		// Offset 71 - uint64 timestamp in millisec Bitfield from bit 4
		0x00, 0xa4, 0x71, 0xce, 0xf0, 0xa1, 0x60, 0x00,

		// Offset 79 - uint64 WinFileTime
		0x81, 0x00, 0x61, 0x4e, 0x15, 0x17, 0xda, 0x01,

		// Offset 87 - uint64 WinFileTime from 4th bit
		0x10, 0x08, 0x10, 0xe6, 0x54, 0x71, 0xa1, 0x1d,

		// offset 95 - E58E26 -> 624485
		0xe5, 0x8e, 0x26,
	}
)

func TestIntegerParser(t *testing.T) {
	reader := bytes.NewReader(sample)
	profile := NewProfile()
	AddModel(profile)

	scope := vfilter.NewScope()
	obj, err := profile.Parse(scope, "unsigned long long", reader, 0)
	assert.NoError(t, err)

	// 578437695752307201
	assert.Equal(t, uint64(0x0807060504030201), obj)
}

func TestLeb128Parser(t *testing.T) {
	reader := bytes.NewReader(sample)
	profile := NewProfile()
	AddModel(profile)

	scope := vfilter.NewScope()
	obj, err := profile.Parse(scope, "leb128", reader, 95)
	assert.NoError(t, err)

	obj_val, ok := obj.(VarInt)
	assert.True(t, ok)

	assert.Equal(t, uint64(624485), obj_val.Value())
}

func TestStructParser(t *testing.T) {
	profile := NewProfile()
	AddModel(profile)

	scope := MakeScope()
	scope.SetLogger(log.New(os.Stderr, " ", 0))

	definition := `
[
  ["TestStruct", "x => x.Field1 + 5", [
     ["Field1", 2, "uint8"],
     ["Field2", 4, "Second"],
     ["X", 0, "Value", {"value": "x=>x"}],
     ["Field3", 0, "unsigned long long"],
     ["Field4", "x => x.Field1", "Second"]
  ]],

  ["Second", 5, [
      ["SecondField1", 2, "uint8"]
  ]]
]
`

	err := profile.ParseStructDefinitions(definition)
	assert.NoError(t, err)

	// Parse TestStruct over the reader
	reader := bytes.NewReader(sample)
	obj, err := profile.Parse(scope, "TestStruct", reader, 0)
	assert.NoError(t, err)

	// Field1 is at offset 2 has value 0x03
	assert.Equal(t, uint64(3), Associative(scope, obj, "Field1"))

	// Object size is calculated as x.Field1 + 5  ... 8
	assert.Equal(t, 8, SizeOf(obj))

	// Field4's offset is calculated as x=>x.Field1
	// i.e. 3. SecondField1 has a relative offset of 2, therefore
	// absolute offset of 3 + 2 = 5 -> value = 0x06
	assert.Equal(t, uint64(6), Associative(scope, obj, "Field4.SecondField1"))

	serialized, err := json.MarshalIndent(obj, "", " ")
	assert.NoError(t, err)

	goldie.Assert(t, "TestStructParser", serialized)
}

func TestArrayParserError(t *testing.T) {
	profile := NewProfile()
	AddModel(profile)

	scope := MakeScope()
	scope.SetLogger(log.New(os.Stderr, " ", 0))

	definition := `
[
  ["TestStruct", 0, [
     ["Field1", 2, "Array", {
        "max_count": "x=>x.Length",
        "type": "uint8"
     }],
  ]
]]
`
	err := profile.ParseStructDefinitions(definition)
	assert.Error(t, err)

	assert.Contains(t, err.Error(), "Array max_count must be an int not string")
}

func TestArrayParser(t *testing.T) {
	profile := NewProfile()
	AddModel(profile)

	scope := MakeScope()
	scope.SetLogger(log.New(os.Stderr, " ", 0))

	definition := `
[
  ["TestStruct", 0, [
     ["Length", 1, "uint8"],
     ["Field1", 2, "Array", {
        "count": "x=>x.Length",
        "type": "uint8"
     }],
     ["Field2", 1, "Array", {
        "count": 2,
        "type": "Second"
     }],

     # Field with sentinel - note that the container struct is
     # accessible with the this variable.
     ["FieldSentinel", 0, "Array", {
        count: 100,
        type: "uint8",
        sentinel: "x=> x=this.Length + 2",
     }]
  ]],

  ["Second", 5, [
      ["SecondField1", 2, "uint8"]
  ]]

]
`

	err := profile.ParseStructDefinitions(definition)
	assert.NoError(t, err)

	// Parse TestStruct over the reader
	reader := bytes.NewReader(sample)
	obj, err := profile.Parse(scope, "TestStruct", reader, 0)
	assert.NoError(t, err)

	// Length is at offset 1 value 2
	assert.Equal(t, uint64(2), Associative(scope, obj, "Length"))

	// Field1 is has length of 2 and starts at offset 3
	assert.Equal(t, []interface{}{uint64(3), uint64(4)},
		Associative(scope, obj, "Field1.Value"))

	// Field2 is an array of structs (each 5 bytes) starting at offset 1.
	assert.Equal(t, []vfilter.Any{uint64(4), uint64(9)},
		Associative(scope, obj, "Field2.SecondField1"))

	serialized, err := json.MarshalIndent(obj, "", " ")
	assert.NoError(t, err)

	goldie.Assert(t, "TestArrayParser", serialized)
}

func TestStringParser(t *testing.T) {
	profile := NewProfile()
	AddModel(profile)

	scope := MakeScope()
	scope.SetLogger(log.New(os.Stderr, " ", 0))

	definition := `
[
  ["TestStruct", 0, [
     ["Length", 2, "uint8"],
     ["OverflowLength", 0, "uint64"],
     ["Field1", 19, "String"],
     ["Field2", 19, "String", {
        "length": 11,
        "term": ""
      }],
     ["Field3", 31, "String", {
        "length": 22,
        "term": "",
        "encoding": "utf16"
      }],
     ["Field4", 19, "String", {
        "length": "x=>x.Length"
      }],
     ["Field5", 31, "String", {
        "length": "x=>x.Length * 2",
        "encoding": "utf16"
      }],
     ["Field6", 31, "String", {
        "term_exp": "x=> '\u0000world'",
        "encoding": "utf16"
      }],

     # A length of zero is a valid length for the empty string.
     ["Field7", 31, "String", {
        "length": "x=>0",
        "encoding": "utf16"
     }],

     # When length is not specified, we default to 1kb but still honor the term.
     ["Field8", 31, "String", {
        "encoding": "utf16"
     }],

     ["OverflowString", 31, "String", {
        "length": "x=>x.OverflowLength",
        "max_length": 5,
        "term": "",
     }],
  ]]
]
`

	err := profile.ParseStructDefinitions(definition)
	assert.NoError(t, err)

	// Parse TestStruct over the reader
	reader := bytes.NewReader(sample)
	obj, err := profile.Parse(scope, "TestStruct", reader, 0)
	assert.NoError(t, err)

	// Length is at offset 2 value 3
	assert.Equal(t, uint64(3), Associative(scope, obj, "Length"))

	// Field1 is default string - null terminated utf8.
	assert.Equal(t, "hello", Associative(scope, obj, "Field1"))

	// Field2 is length string - not null terminated utf8.
	assert.Equal(t, "hello\x00world", Associative(scope, obj, "Field2"))

	// Field3 is length string - not null terminated utf16 (note length is byte length).
	assert.Equal(t, "hello\x00world", Associative(scope, obj, "Field3"))

	// Field4 is length string, length specified by expression
	// depends on Length field of struct (3).
	assert.Equal(t, "hel", Associative(scope, obj, "Field4"))

	// Field5 is length string, length specified by expression
	// depends on Length field of struct (3) times 2 (due to utf16).
	assert.Equal(t, "hel", Associative(scope, obj, "Field5"))

	assert.Equal(t, "hello", Associative(scope, obj, "Field6"))

	// Even though the length is huge, the max_length ensure string is
	// clamped at something reasonable.
	over_flow := Associative(scope, obj, "OverflowString")
	assert.Equal(t, 5, len(over_flow.(string)))

	serialized, err := json.MarshalIndent(obj, "", " ")
	assert.NoError(t, err)

	goldie.Assert(t, "TestStringParser", serialized)
}

// This is a fairly complex parser so it makes an excellent test.
func TestPowershellParser(t *testing.T) {
	profile := NewProfile()
	AddModel(profile)

	scope := MakeScope()
	scope.SetLogger(log.New(os.Stderr, " ", 0))
	//scope.AppendVars(ordereddict.NewDict().Set("DEBUG_VTYPES", 1))

	definition := `
[
  ["Header", 0, [
    ["Signature", 0, "String", {"length": 13}],
    ["CountOfEntries", 14, "uint32"],
    ["Entries", 18, "Array", {"type": "Entry", "count": "x=>x.CountOfEntries"}]
  ]],

  ["Entry", "x=>x.Func.SizeOf + x.ModuleLength + 20", [
    ["Offset", 0, "Value", {"value": "x=>x.StartOf"}],
    ["EntryLength", 0, "Value", {"value": "x=>x.Func.EndOf - x.StartOf + 4"}],
    ["TimestampTicks", 0, "uint64"],
    ["ModuleLength", 8, "uint32"],
    ["ModuleName", 12, "String", {"length": "x => x.ModuleLength"}],
    ["CommandCount", "x=>x.ModuleLength + 12", "uint32"],
    ["Func", "x => x.ModuleLength + 16", "Array",
           {"type": "FunctionInfo", "count": "x=>x.CommandCount"}],
    ["CountOfTypes", "x=>x.Func.EndOf", "uint32"]
  ]],

  ["FunctionInfo", "x => x.NameLen + 8", [
    ["NameLen", 0, "uint32"],
    ["Name", 4, "String", {"length": "x => x.NameLen"}],
    ["Count", "x => x.NameLen + 4", "uint32"]
  ]]
]
`

	err := profile.ParseStructDefinitions(definition)
	assert.NoError(t, err)

	reader, err := os.Open("test_data/ModuleAnalysisCache")
	assert.NoError(t, err)

	obj, err := profile.Parse(scope, "Header", reader, 0)
	assert.NoError(t, err)

	serialized, err := json.MarshalIndent(obj, "", " ")
	assert.NoError(t, err)

	goldie.Assert(t, "TestPowershellParser", serialized)
}

func TestUnion(t *testing.T) {
	profile := NewProfile()
	AddModel(profile)
	scope := MakeScope()
	scope.SetLogger(log.New(os.Stderr, " ", 0))

	definition := `
[
  ["Header", 0, [
    ["Field", 0, "uint8"],
    ["Union", 4, "Union", {
       "selector": "x=>x.Field",
       "choices": {
          "1": "Struct1",
          "2": "Struct2",
       }
    }]
  ]],
  ["Struct1", 0, [
    ["Field1", 0, "uint8"],
  ]],
  ["Struct2", 0, [
    ["Field2", 4, "uint32"],
  ]],
]
`
	err := profile.ParseStructDefinitions(definition)
	assert.NoError(t, err)

	reader := bytes.NewReader(sample)
	result := ordereddict.NewDict()

	obj, err := profile.Parse(scope, "Header", reader, 0)
	assert.NoError(t, err)
	result.Set("@offset 0", obj)

	obj, err = profile.Parse(scope, "Header", reader, 1)
	assert.NoError(t, err)
	result.Set("@offset 1", obj)

	obj, err = profile.Parse(scope, "Header", reader, 2)
	assert.NoError(t, err)
	result.Set("@offset 2", obj)

	serialized, err := json.MarshalIndent(result, "", " ")
	assert.NoError(t, err)

	goldie.Assert(t, "TestUnion", serialized)
}

func TestEnumerationParser(t *testing.T) {
	profile := NewProfile()
	AddModel(profile)

	scope := MakeScope()
	scope.SetLogger(log.New(os.Stderr, " ", 0))

	definition := `
[
  ["TestStruct", 0, [
     ["FirstByte", 0, "Enumeration", {
        type: "uint8",
        map: {
           "ONE": 0x01,
           "TWO": 0x02,
        },
     }],
     ["SecondByte", 1, "Enumeration", {
        type: "uint8",
        choices: {
           "1": "ONE",
           "2": "TWO",
        },
     }],
     ["BitFieldValue", 4, "uint8"],
     ["BitField", 4, "Enumeration", {
        type: "BitField",
        type_options: {
           start_bit: 0,
           end_bit: 1,
           type: "uint8",
        },
        choices: {
           "1": "ONE",
           "2": "TWO",
        },
     }],
  ]]
]
`

	err := profile.ParseStructDefinitions(definition)
	assert.NoError(t, err)

	// Parse TestStruct over the reader
	reader := bytes.NewReader(sample)
	obj, err := profile.Parse(scope, "TestStruct", reader, 0)
	assert.NoError(t, err)

	serialized, err := json.MarshalIndent(obj, "", " ")
	assert.NoError(t, err)

	goldie.Assert(t, "TestEnumerationParser", serialized)
}

func TestBitfieldParser(t *testing.T) {
	profile := NewProfile()
	AddModel(profile)

	scope := MakeScope()
	scope.SetLogger(log.New(os.Stderr, " ", 0))

	definition := `
[
  ["TestStruct", 0, [
     ["Value", 18, "uint8"],
     ["FirstNibble", 18, "BitField", {
        type: "uint8",
        start_bit: 0,
        end_bit: 4,
     }],
     ["SecondNibble", 18, "BitField", {
        type: "uint8",
        start_bit: 4,
        end_bit: 8,
     }],
  ]]
]
`

	err := profile.ParseStructDefinitions(definition)
	assert.NoError(t, err)

	// Parse TestStruct over the reader
	reader := bytes.NewReader(sample)
	obj, err := profile.Parse(scope, "TestStruct", reader, 0)
	assert.NoError(t, err)

	serialized, err := json.MarshalIndent(obj, "", " ")
	assert.NoError(t, err)

	goldie.Assert(t, "TestBitfieldParser", serialized)
}

func TestFlagsParser(t *testing.T) {
	profile := NewProfile()
	AddModel(profile)

	scope := MakeScope()
	scope.SetLogger(log.New(os.Stderr, " ", 0))

	definition := `
[
  ["TestStruct", 0, [
     ["Value", 18, "uint8"],
     ["Flags", 18, "Flags", {
        type: "uint8",
        bitmap: {
          "FirstBit": 1,
          "SecondBit": 2,
          "ThirdBit": 3,
          "FourthBit": 4,
        }
     }],
     ["FlagsBitfieldSecondNibble", 18, "Flags", {
        type: "BitField",
        type_options: {
            type: "uint8",
            start_bit: 4,
            end_bit: 8,
        },
        bitmap: {
          "FirstBit": 1,
          "SecondBit": 2,
          "ThirdBit": 3,
          "FourthBit": 4,
        }
     }],
  ]]
]
`

	err := profile.ParseStructDefinitions(definition)
	assert.NoError(t, err)

	// Parse TestStruct over the reader
	reader := bytes.NewReader(sample)
	obj, err := profile.Parse(scope, "TestStruct", reader, 0)
	assert.NoError(t, err)

	serialized, err := json.MarshalIndent(obj, "", " ")
	assert.NoError(t, err)

	goldie.Assert(t, "TestFlagsParser", serialized)
}

func TestEpochTimestampParser(t *testing.T) {
	profile := NewProfile()
	AddModel(profile)

	scope := MakeScope()
	scope.SetLogger(log.New(os.Stderr, " ", 0))

	definition := `
[
  ["TestStruct", 0, [
     ["Value", 49, uint32],
     ["Timestamp1", 55, "Timestamp"],
     ["Timestamp2", 63, "Timestamp", {
         factor: 1000000,
     }],
     ["Timestamp3", 71, "Timestamp", {
         type: "BitField",
         type_options: {
            type: "uint64",
            start_bit: 4,
            end_bit: 64,
         } ,
         factor: 1000000,
     }],

     ["WinFileTime", 79, "WinFileTime"],
     ["WinFileTime2", 87, "WinFileTime", {
         type: "BitField",
         type_options: {
            type: "uint64",
            start_bit: 4,
            end_bit: 64,
         } ,
     }],

  ]]
]
`

	err := profile.ParseStructDefinitions(definition)
	assert.NoError(t, err)

	// Parse TestStruct over the reader
	reader := bytes.NewReader(sample)
	obj, err := profile.Parse(scope, "TestStruct", reader, 0)
	assert.NoError(t, err)

	serialized, err := json.MarshalIndent(obj, "", " ")
	assert.NoError(t, err)

	goldie.Assert(t, "TestEpochTimestampParser", serialized)
}

// Make sure errors are reported properly. Errors should only be
// reported for invalid profile definitions since we have no control
// over what data we may encounter. For example if the parsed data
// makes no sense for the type we should not report an error just
// return null. But if the profile definition is invalid then we need
// to report it as an error to the user.
func TestErrors(t *testing.T) {
	profile := NewProfile()
	AddModel(profile)

	scope := MakeScope()
	log_buffer := &strings.Builder{}
	scope.SetLogger(log.New(log_buffer, " ", 0))

	definition := `
[
  ["TestSubStruct", 0, [
     ["ArrayOfUnderfinedStruct", 0, "Array", {
         type: "Undefined",
         count: 100,
     }],
  ]],
  ["TestStruct", 0, [
     ["Flags", 0, "Flags", {
          type: "BitField",
          bitmap: {
             "FirstBit": 1,
          },
     }],
     ["Enumeration", 0, "Enumeration", {
          type: "BitField",
          bitmap: {
             "FirstBit": 1,
          },
     }],
     ["Undefined", 0, "TestSubStruct"],
     ["Undefined2", 0, "TestSubStruct"],
  ]]
]
`
	err := profile.ParseStructDefinitions(definition)
	assert.NoError(t, err)

	// Parse TestStruct over the reader
	reader := bytes.NewReader(sample)
	obj, err := profile.Parse(scope, "TestStruct", reader, 0)
	assert.NoError(t, err)

	serialized, err := json.MarshalIndent(obj, "", " ")
	assert.NoError(t, err)

	golden := string(serialized) + fmt.Sprintf("\n%v\n", log_buffer.String())

	goldie.Assert(t, "TestErrors", []byte(golden))
}
