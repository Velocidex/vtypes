//
package vtypes

import (
	"encoding/binary"
)

func AddModel(profile *Profile) {
	profile.types["uint8"] = NewIntParser(
		"uint8", func(buf []byte) int64 {
			return int64(uint8(buf[0]))
		})
	profile.types["uint16"] = NewIntParser(
		"uint16", func(buf []byte) int64 {
			return int64(binary.LittleEndian.Uint16(buf))
		})
	profile.types["uint32"] = NewIntParser(
		"uint32", func(buf []byte) int64 {
			return int64(binary.LittleEndian.Uint32(buf))
		})
	profile.types["uint64"] = NewIntParser(
		"uint64", func(buf []byte) int64 {
			return int64(binary.LittleEndian.Uint64(buf))
		})
	profile.types["int8"] = NewIntParser(
		"int8", func(buf []byte) int64 {
			return int64(int8(buf[0]))
		})
	profile.types["int16"] = NewIntParser(
		"int16", func(buf []byte) int64 {
			return int64(int16(binary.LittleEndian.Uint16(buf)))
		})
	profile.types["int32"] = NewIntParser(
		"int32", func(buf []byte) int64 {
			return int64(int32(binary.LittleEndian.Uint32(buf)))
		})
	profile.types["int64"] = NewIntParser(
		"int64", func(buf []byte) int64 {
			return int64(binary.LittleEndian.Uint64(buf))
		})

	profile.types["String"] = NewStringParser("string")
	profile.types["Enumeration"] = NewEnumeration("Enumeration", profile)
	profile.types["Flags"] = NewFlagsParser("Flags", profile)
	profile.types["Array"] = NewArrayParser("Array", "", profile, nil)
	profile.types["BitField"] = NewBitField("BitField", profile)

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
