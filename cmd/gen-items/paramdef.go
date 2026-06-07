package main

import (
	"embed"
	"encoding/xml"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

//go:embed defs/*.xml
var defFS embed.FS

// paramdef is a parsed Paramdex PARAMDEF XML: the ordered field layout of a param.
type paramdef struct {
	ParamType   string
	DataVersion int
	Fields      []field
}

// field is one parsed field definition (from the "Def" attribute, e.g.
// "u8 properStrength", "dummy8 pad[3]", "u8 disableParam_NT:1").
type field struct {
	Type string // s8/u8/s16/u16/s32/u32/b32/f32/f64/angle32/dummy8/fixstr/fixstrW
	Name string
	Bits int // -1 if not a packed bitfield
	Arr  int // array length (1 if not an array)
}

var defGrammar = regexp.MustCompile(`^([A-Za-z0-9_]+)\s+([A-Za-z0-9_]+)(?::(\d+))?(?:\[(\d+)\])?$`)

func loadDef(name string) (*paramdef, error) {
	data, err := defFS.ReadFile("defs/" + name + ".xml")
	if err != nil {
		return nil, err
	}
	return parseParamdef(data)
}

func parseParamdef(data []byte) (*paramdef, error) {
	var x struct {
		ParamType   string `xml:"ParamType"`
		DataVersion int    `xml:"DataVersion"`
		Fields      []struct {
			Def string `xml:"Def,attr"`
		} `xml:"Fields>Field"`
	}
	if err := xml.Unmarshal(data, &x); err != nil {
		return nil, err
	}
	pd := &paramdef{ParamType: x.ParamType, DataVersion: x.DataVersion}
	for _, rf := range x.Fields {
		def := strings.TrimSpace(rf.Def)
		if i := strings.IndexByte(def, '='); i >= 0 { // strip " = default"
			def = strings.TrimSpace(def[:i])
		}
		m := defGrammar.FindStringSubmatch(def)
		if m == nil {
			return nil, fmt.Errorf("unparseable field def %q", rf.Def)
		}
		f := field{Type: m[1], Name: m[2], Bits: -1, Arr: 1}
		if m[3] != "" {
			f.Bits, _ = strconv.Atoi(m[3])
		}
		if m[4] != "" {
			f.Arr, _ = strconv.Atoi(m[4])
		}
		pd.Fields = append(pd.Fields, f)
	}
	return pd, nil
}

// ---- field sizing / bit-packing primitives (mirror SoulsFormats ParamUtil) ----

func valueSize(t string) int {
	switch t {
	case "s8", "u8", "dummy8", "fixstr":
		return 1
	case "s16", "u16", "fixstrW":
		return 2
	case "s32", "u32", "b32", "f32", "angle32":
		return 4
	case "f64":
		return 8
	}
	return 0
}

func isBitType(t string) bool {
	switch t {
	case "s8", "u8", "s16", "u16", "s32", "u32", "dummy8":
		return true
	}
	return false
}

func isArrayType(t string) bool { return t == "dummy8" || t == "fixstr" || t == "fixstrW" }
func isSignedBit(t string) bool { return t == "s8" || t == "s16" || t == "s32" }

func bitLimit(t string) int {
	switch t {
	case "s8", "u8", "dummy8":
		return 8
	case "s16", "u16":
		return 16
	case "s32", "u32":
		return 32
	}
	return 0
}

// rowSize computes the fixed byte size of a row (SoulsFormats GetFieldsSize over
// all fields), honoring bit-field packing. Used to validate against the param's
// detected row size.
func (pd *paramdef) rowSize() int {
	size := 0
	f := pd.Fields
	for i := 0; i < len(f); i++ {
		if isArrayType(f[i].Type) {
			size += valueSize(f[i].Type) * f[i].Arr
		} else {
			size += valueSize(f[i].Type)
		}
		if isBitType(f[i].Type) && f[i].Bits != -1 {
			bitOff := f[i].Bits
			lim := bitLimit(f[i].Type)
			for i < len(f)-1 {
				n := f[i+1]
				if !isBitType(n.Type) || n.Bits == -1 || bitLimit(n.Type) != lim || bitOff+n.Bits > lim {
					break
				}
				bitOff += n.Bits
				i++
			}
		}
	}
	return size
}

// readRow walks every field of one row (mirroring SoulsFormats Row.ReadCells) so
// the cursor stays correct through bit-packed runs, capturing the requested int and
// float fields. b is the whole param's bytes; dataOff is the row's data offset.
func (pd *paramdef) readRow(b []byte, dataOff int64, wantInt, wantFloat map[string]bool) (map[string]int64, map[string]float64, error) {
	ints := map[string]int64{}
	floats := map[string]float64{}
	pos := int(dataOff)
	bitOffset, bitLim := -1, -1
	var bitValue uint64

	rd := func(n int) (uint64, error) {
		if pos < 0 || pos+n > len(b) {
			return 0, fmt.Errorf("row read out of bounds at %d (+%d, len %d)", pos, n, len(b))
		}
		var v uint64
		for i := 0; i < n; i++ {
			v |= uint64(b[pos+i]) << (8 * uint(i)) // little-endian
		}
		pos += n
		return v, nil
	}

	for _, f := range pd.Fields {
		t := f.Type
		switch {
		case t == "f32" || t == "angle32":
			u, err := rd(4)
			if err != nil {
				return nil, nil, err
			}
			bitOffset = -1
			if wantFloat[f.Name] {
				floats[f.Name] = float64(math.Float32frombits(uint32(u)))
			}
		case t == "f64":
			u, err := rd(8)
			if err != nil {
				return nil, nil, err
			}
			bitOffset = -1
			if wantFloat[f.Name] {
				floats[f.Name] = math.Float64frombits(u)
			}
		case t == "b32":
			u, err := rd(4)
			if err != nil {
				return nil, nil, err
			}
			bitOffset = -1
			if wantInt[f.Name] {
				ints[f.Name] = int64(int32(u))
			}
		case t == "fixstr":
			pos += f.Arr
			bitOffset = -1
		case t == "fixstrW":
			pos += f.Arr * 2
			bitOffset = -1
		case isBitType(t) && f.Bits == -1:
			if t == "dummy8" { // padding array
				pos += f.Arr
				bitOffset = -1
				continue
			}
			sz := valueSize(t)
			u, err := rd(sz)
			if err != nil {
				return nil, nil, err
			}
			bitOffset = -1
			if wantInt[f.Name] {
				ints[f.Name] = signExtend(t, u)
			}
		default: // packed bitfield
			if !isBitType(t) || f.Bits == -1 {
				return nil, nil, fmt.Errorf("unhandled field type %q (%s)", t, f.Name)
			}
			if bitOffset == -1 || bitLim != bitLimit(t) || bitOffset+f.Bits > bitLim {
				bitOffset = 0
				bitLim = bitLimit(t)
				u, err := rd(bitLim / 8)
				if err != nil {
					return nil, nil, err
				}
				bitValue = u
			}
			if wantInt[f.Name] {
				left := uint(64 - f.Bits - bitOffset)
				right := uint(64 - f.Bits)
				if isSignedBit(t) {
					ints[f.Name] = int64(bitValue<<left) >> right
				} else {
					ints[f.Name] = int64(bitValue << left >> right)
				}
			}
			bitOffset += f.Bits
		}
	}
	return ints, floats, nil
}

func signExtend(t string, u uint64) int64 {
	switch t {
	case "s8":
		return int64(int8(u))
	case "s16":
		return int64(int16(u))
	case "s32":
		return int64(int32(u))
	default: // u8/u16/u32
		return int64(u)
	}
}
