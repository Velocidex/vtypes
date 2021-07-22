//
package vtypes

import (
	"bytes"
	"encoding/json"
	"log"
	"os"
	"testing"

	"github.com/Velocidex/ordereddict"
	"github.com/sebdah/goldie"
	assert "github.com/stretchr/testify/assert"
	"www.velocidex.com/golang/vfilter"
)

var (
	sample = []byte{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
		0x11, 0x12, 0x13,

		// Offset 19 - "hello\x00world\x00"
		0x68, 0x65, 0x6c, 0x6c, 0x6f, 0x00, 0x77, 0x6f, 0x72, 0x6c, 0x64, 0x00,

		// Offset 37 - utf16
		0x68, 0x00, 0x65, 0x00, 0x6c, 0x00, 0x6c, 0x00, 0x6f, 0x00, 0x00, 0x00,
		0x77, 0x00, 0x6f, 0x00, 0x72, 0x00, 0x6c, 0x00, 0x64, 0x00, 0x00, 0x00,
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
        "term_exp": "x=> 'yolo!'"
      }]
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
