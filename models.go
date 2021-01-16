//  Every profile contains some basic built in types that make it
//  easier to parse common structs. The model is a mapping between the
//  generic names of types and the corresponding parsers.

package vtypes

import (
	"encoding/binary"
)

func AddModel(profile *Profile) {
	profile.types["uint8"] = NewIntParser(
		"uint8", 1, func(buf []byte) interface{} {
			return int64(uint8(buf[0]))
		})
	profile.types["uint16"] = NewIntParser(
		"uint16", 2, func(buf []byte) interface{} {
			return int64(binary.LittleEndian.Uint16(buf))
		})
	profile.types["uint32"] = NewIntParser(
		"uint32", 4, func(buf []byte) interface{} {
			return int64(binary.LittleEndian.Uint32(buf))
		})
	profile.types["uint64"] = NewIntParser(
		"uint64", 8, func(buf []byte) interface{} {
			return int64(binary.LittleEndian.Uint64(buf))
		})
	profile.types["int8"] = NewIntParser(
		"int8", 1, func(buf []byte) interface{} {
			return int64(int8(buf[0]))
		})
	profile.types["int16"] = NewIntParser(
		"int16", 2, func(buf []byte) interface{} {
			return int64(int16(binary.LittleEndian.Uint16(buf)))
		})
	profile.types["int32"] = NewIntParser(
		"int32", 4, func(buf []byte) interface{} {
			return int64(int32(binary.LittleEndian.Uint32(buf)))
		})
	profile.types["int64"] = NewIntParser(
		"int64", 8, func(buf []byte) interface{} {
			return int64(binary.LittleEndian.Uint64(buf))
		})

	profile.types["Array"] = &ArrayParser{}
	profile.types["String"] = &StringParser{}
	profile.types["Value"] = &ValueParser{}
	profile.types["Enumeration"] = &EnumerationParser{}
	profile.types["BitField"] = &BitField{}
	profile.types["Flags"] = &Flags{}
	profile.types["WinFileTime"] = &WinFileTime{}
	profile.types["Timestamp"] = &EpochTimestamp{}
	profile.types["Union"] = &Union{}
	profile.types["FatTimestamp"] = &FatTimestamp{}

	// Aliases
	profile.types["int"] = profile.types["int32"]
	profile.types["char"] = profile.types["int8"]
	profile.types["byte"] = profile.types["uint8"]
	profile.types["short int"] = profile.types["int16"]
	profile.types["unsigned char"] = profile.types["uint8"]
	profile.types["unsigned int"] = profile.types["uint32"]
	profile.types["unsigned long"] = profile.types["uint32"]
	profile.types["unsigned long long"] = profile.types["uint64"]
	profile.types["unsigned short"] = profile.types["uint16"]
}
