# Vtypes parsing subsystem for Golang and VQL

## Design goals

Parsing binary structures is very important for forensic analysis and
DFIR - we encounter binary data in many contexts, such as file
formats, network traffic and more.

Velociraptor uses VQL to provide the flexibility for users to be able
to craft a VQL query in order to retrieve valuable machine state
data. Sometimes we need to parse binary data to answer these
questions.

While binary parsers written in Golang are typically the best options
for speed and memory efficiency, the need to compile a parser into an
executable and push it to the endpoint makes it difficult to implement
adhoc parsers. Ideally we would like to have a parser fully
implemented in VQL, so it can be added to an artifact and pushed to
the endpoint without needing to recompile and rebuild anything.

The parser needs to be data driven and descriptive as to the
underlying data format, but at the same time capable of parsing more
complex binary structures.

We take a lot of inspiration from existing parser frameworks, such as
the Rekall vtypes language (which was implemented in Python), but the
VQL Vtype parsers are much improved, much faster than python and more
flexible.

## Overview

The parser is driven by a json data structure called a "Profile". A
Profile is simply a data driven description of how structs are layed
out and how to parse them.

In order to use the parser, one simply provides a profile definition,
and a file (or datablob) to parse. The parser is given an offset and a
struct to instantiate. Here is an example of VQL that parses a single
struct at offset 10 in the file.

```sql
SELECT parse_binary(profile='[ ["Header": 0, ["Signature", 0, "String", {"length": 10}]]]',
                    filename='/path/to/file', struct='Header')
FROM scope()
```


## Profile description.

Profile descriptions are supposed to be easy to understand and quick
to write. It is a way of describing how to parse a particular binary
type at a high level.

A profile is a list of struct definitions. Each struct definition
contains the name of the struct, its size and a list of field
definitions.

In turn field definitions are a list of the field's name, its offset
(relative to the start of the struct), and its type followed by any
options for that type.

Typically a profile is given as JSON serialized string.

Here is an example:

```json
[
  ["Header", 0, [
    ["Signature", 0, "String", {"length": 13}],
    ["CountOfEntries", 14, "uint32"],
    ["Entries", 18, "Array", {"type": "Entry", "count": "x=>x.CountOfEntries"}]
  ]],
  ...
]
```

In the above example:

1. There is a single struct called Header.

2. The size of the header is not specified (it is 0). The size of a
   struct becomes important when using the struct in an array.

3. The CountOfEntries field starts at offset 14 into the struct and it
   is a uint32.

4. The Entries field starts at offset 18, and contains an array. An
   array is a collection of other items, and so it must be initialized
   with the proper options. In this case the array contains items of
   type "Entry" (which is another struct, not yet defined).

5. The count of the array is the number of items in the array. Here it
   is specified as a lambda function.

Lambda functions are VQL snippets that calculate the value of various
fields at runtime. The Lambda is passed the struct object currently
being parsed, and so can simply express values dependent on the
struct's fields.

In the above example, the count of the array is given as the value of
the field CountOfEntries. This type of construct is very common in
binary structures (where a count or length is specified by another
field in the struct).

Lets continue to view the next definition:

```json
["Entry", "x=>x.ModuleLength + 20", [
  ["Offset", 0, "Value", {"value": "x=>x.StartOf"}],
  ["ModuleLength", 8, "uint32"],
  ...
]],
```

The definition of the Entry struct is given above. The size is also
given by a lambda function, this time, the size of the entries is
derived from the ModuleLength field. Note how in the above definition,
the Entries field is a list of variable sized Entry structs.

## Parsers

Struct fields are parsed out using typed parsers. The name of the
parser is used at the 3rd entry to its definition:

### Simple parsers

These parse primitive types such as int64, uint32 etc.

### Struct parsers

Using the name of a struct definition will cause a StructObject to be
produced. These look a bit like dict objects in that VQL can simply
dereference fields, but fields are parsed lazily (i.e. upon access
only). There are also additional properties available:

1. SizeOf property is the size of the struct (which may be derived
   from a lambda). For example, `x=>x.SizeOf` returns the size of the
   current struct.

2. StartOf and EndOf properties are the offset to the start and end of
   the struct.

### Array parser

An array is a repeated collection of other types. Therefore the array
parser must be initialized with options that specify what the
underlying type is, its count etc.

1. type: The type of the underlying object
2. count: How many items to include in the array (can be lambda)
3. max_count: A hard limit on count (default 1000)

Parsing a field as an array produces an ArrayObject which has the
following properties:

1. SizeOf, StartOf, EndOf properties as above.
2. Value property accessed the underlying array.

You can iterate over an ArrayObject with the `foreach()` plugin:

```vql
SELECT * FROM foreach(row=Header.Entries, query={....})
```

Accessing a member of the foreach will produce an array of each
member. e.g. `Header.Entries.ModuleLength` will just produce a list of
length.

### String parser

Strings are very common to parse. The string parser can be configured
using the following options.

1. encoding: Can be UTF16 to parse utf16 arrays
2. term: A terminator - by default this is the null character but you
   can specify the empty string for no terminator or another sequence
   of characters.
3. length, max_length: The length of the string - if not specified we
   use the terminator to find the end of the string. This can also be
   a lambda to derive the length from another field.


### References

When dereferencing a struct member we receive the basic type contained
in the struct.

Consider the following struct:

```json
[
  ["TestStruct", 0, [
     ["Field1", 2, "uint8"],
     ["Field2", 0, Value, {
         value: "x=>x.Field1 + 1"
     }]
  ]]
]
```

In VQL, accessing the field will produce a uint8 type, so `Field2`
will be calculated by adding 1 to the uint8 content of Field1.

While this makes it easy and intuitive to use, sometimes we need to
calculate some metadata over the struct field instead of it's literal
value. For example metadata such as its starting offset, ending offset
etc.

Suppose we wanted to add another field, immediately following Field1:

```json
[
  ["TestStruct", 0, [
     ["Field1", 2, "uint8"],
     ["Next", "x=>x.Field1.EndOf", "uint16"],
  ]]
]
```

This is not going to work, because `x.Field1` is a `uint8` integer
type and it does not have an `EndOf` method.

This is where references come in. We can obtain a struct field
reference by using the @ character when accessing the field. A
reference is a wrapper around the field which provides metadata about
it.

```json
[
  ["TestStruct", 0, [
     ["Field1", 2, "uint8"],
     ["Next", "x=>x.`@Field1`.RelEndOf", "uint16"],
  ]]
]
```

Note that the field name must be enclosed in backticks for VQL to
identify the @ as part of the field name.

Accessing a struct field with a name begining with @ will return a
reference to the field instead of the field itself.

The reference has many useful methods:

- *SizeOf*, *Size*: These return the size of the field in bytes.
- *StartOf*, *Start*, *OffsetOf*: These return the byte offset of the
  beginning of the field.

  NOTE that this offset is absolute - i.e. this offset will be
  relative to the beginning of the file.

- *RelOffset*: The relative offset of the field within the struct.

  In the following, the Next and Next2 fields are equivalent:

```json
[
  ["TestStruct", 0, [
     ["Field1", 2, "uint8"],
     ["Next", "x=>x.`@Field1`.StartOf + 4 - x.OffsetOf", "uint16"],
     ["Next2", "x=>x.`@Field1`.RelOffset + 4", "uint16"],
  ]]
]
```

- *EndOf*, *End*: These return the end of the field in absolute bytes,
  relative to the file.

  In the following, the Next and Next2 fields are equivalent and are
  both located directly after `Field1`:

```json
[
  ["TestStruct", 0, [
     ["Field1", 2, "uint8"],
     ["Next", "x=>x.`@Field1`.EndOf - x.OffsetOf", "uint16"],
     ["Next2", "x=>x.`@Field1`.RelOffset + x.`@Field1`.SizeOf", "uint16"],
  ]]
]
```

- *RelEndOf*: Like *EndOf* but relative to the start of the struct.

  In particular notice that the profile struct syntax requires
  specifying the offset of a field relative to the struct. Therefore
  this field is especially useful to specify the field should be
  located immediately after the previous field.

```json
[
  ["TestStruct", 0, [
     ["Field1", 2, "uint8"],
     ["Next", "x=>x.`@Field1`.RelEndOf", "uint16"],
  ]]
]
```

  This also works for fields with variable size. For example consider
  a null terminated string followed immediately with an integer.

```json
[
  ["TestStruct", 0, [
     ["Field1", 0, String],
     ["Next", "x=>x.`@Field1`.RelEndOf", "uint32"],
  ]]
]
```

- *Value*: It is possible to dereference the reference to obtain the
  real value.

```json
[
  ["TestStruct", 0, [
     ["Field1", 2, "uint8"],
     ["Field2", 0, Value, {
         value: "x=>x.`@Field1`.Value + 1"
     }]
  ]]
]
```
