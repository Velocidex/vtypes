//  Every profile contains some basic built in types that make it
//  easier to parse common structs. The model is a mapping between the
//  generic names of types and the corresponding parsers.

package vtypes

import (
	"encoding/binary"
	"math"
)

func AddModel(profile *Profile) {
	profile.types["uint8"] = NewIntParser(
		"uint8", 1, func(buf []byte) interface{} {
			return uint64(uint8(buf[0]))
		})
	profile.types["uint16"] = NewIntParser(
		"uint16", 2, func(buf []byte) interface{} {
			return uint64(binary.LittleEndian.Uint16(buf))
		})
	profile.types["uint32"] = NewIntParser(
		"uint32", 4, func(buf []byte) interface{} {
			return uint64(binary.LittleEndian.Uint32(buf))
		})
	profile.types["uint64"] = NewIntParser(
		"uint64", 8, func(buf []byte) interface{} {
			return uint64(binary.LittleEndian.Uint64(buf))
		})

	profile.types["uint16be"] = NewIntParser(
		"uint16be", 2, func(buf []byte) interface{} {
			return uint64(binary.BigEndian.Uint16(buf))
		})
	profile.types["uint32be"] = NewIntParser(
		"uint32be", 4, func(buf []byte) interface{} {
			return uint64(binary.BigEndian.Uint32(buf))
		})
	profile.types["uint64be"] = NewIntParser(
		"uint64be", 8, func(buf []byte) interface{} {
			return uint64(binary.BigEndian.Uint64(buf))
		})

	profile.types["float64"] = NewIntParser(
		"float64", 8, func(buf []byte) interface{} {
			bits := uint64(binary.LittleEndian.Uint64(buf))
			return math.Float64frombits(bits)
		})

	profile.types["float64be"] = NewIntParser(
		"float64be", 8, func(buf []byte) interface{} {
			bits := uint64(binary.BigEndian.Uint64(buf))
			return math.Float64frombits(bits)
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

	// adding BigEndian option
	profile.types["uint8b"] = NewIntParser(
		"uint8b", 1, func(buf []byte) interface{} {
			return uint64(uint8(buf[0]))
		})
	profile.types["uint16b"] = NewIntParser(
		"uint16b", 2, func(buf []byte) interface{} {
			return uint64(binary.BigEndian.Uint16(buf))
		})
	profile.types["uint32b"] = NewIntParser(
		"uint32b", 4, func(buf []byte) interface{} {
			return uint64(binary.BigEndian.Uint32(buf))
		})
	profile.types["uint64b"] = NewIntParser(
		"uint64b", 8, func(buf []byte) interface{} {
			return uint64(binary.BigEndian.Uint64(buf))
		})
	profile.types["int8b"] = NewIntParser(
		"int8b", 1, func(buf []byte) interface{} {
			return int64(int8(buf[0]))
		})
	profile.types["int16b"] = NewIntParser(
		"int16b", 2, func(buf []byte) interface{} {
			return int64(int16(binary.BigEndian.Uint16(buf)))
		})
	profile.types["int32b"] = NewIntParser(
		"int32b", 4, func(buf []byte) interface{} {
			return int64(int32(binary.BigEndian.Uint32(buf)))
		})
	profile.types["int64b"] = NewIntParser(
		"int64b", 8, func(buf []byte) interface{} {
			return int64(binary.BigEndian.Uint64(buf))
		})

	profile.types["Array"] = &ArrayParser{}
	profile.types["String"] = &StringParser{}
	profile.types["Value"] = &ValueParser{}
	profile.types["Enumeration"] = &EnumerationParser{}
	profile.types["BitField"] = &BitField{}
	profile.types["Flags"] = &Flags{}
	profile.types["WinFileTime"] = &WinFileTime{
		parser: NewIntParser(
			"int64", 8, func(buf []byte) interface{} {
				return int64(binary.LittleEndian.Uint64(buf))
			}),
		factor: 1,
	}
	profile.types["Timestamp"] = &EpochTimestamp{
		parser: NewIntParser(
			"int32", 4, func(buf []byte) interface{} {
				return int64(int32(binary.LittleEndian.Uint32(buf)))
			}),
		factor: 1,
	}
	profile.types["Union"] = &Union{}
	profile.types["FatTimestamp"] = &FatTimestamp{
		profile: profile,
	}
	profile.types["Pointer"] = &PointerParser{}
	profile.types["Profile"] = &ProfileParser{}

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
