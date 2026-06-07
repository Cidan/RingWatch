package main

import (
	"encoding/binary"
	"fmt"
	"strings"
)

// paramTable is a parsed .param file: its rows' data offsets keyed by id, in file
// order, plus the raw bytes (rows are read lazily by the paramdef walker).
type paramTable struct {
	paramType    string
	dataVersion  int
	ids          []int32
	rows         map[int32]int64 // id -> row data offset
	bytes        []byte
	detectedSize int64
}

// parseParam parses a FromSoftware PARAM (the format used by Elden Ring regulation
// params: little-endian, 64-bit data offsets, param type stored at an offset).
func parseParam(b []byte) (*paramTable, error) {
	if len(b) < 0x40 {
		return nil, fmt.Errorf("param too small (%d bytes)", len(b))
	}
	if b[0x2C] == 0xFF {
		return nil, fmt.Errorf("unexpected big-endian param")
	}
	le := binary.LittleEndian
	flags1 := b[0x2D]
	flag01 := flags1&0x01 != 0
	intDataOffset := flags1&0x02 != 0
	longDataOffset := flags1&0x04 != 0
	offsetParamType := flags1&0x80 != 0

	pos := 4 // stringsOffset (u32) — unreliable, skipped
	if (flag01 && intDataOffset) || longDataOffset {
		pos += 2 // int16 == 0
	} else {
		pos += 2 // uint16 dataStart
	}
	pos += 2 // Unk06 (int16)
	dataVersion := int(int16(le.Uint16(b[pos:])))
	pos += 2
	rowCount := int(le.Uint16(b[pos:]))
	pos += 2

	var paramType string
	if offsetParamType {
		pos += 4 // int32 == 0
		typeOff := int(le.Uint64(b[pos:]))
		pos += 8
		pos += 0x14 // reserved zeros
		paramType = readCString(b, typeOff)
	} else {
		paramType = strings.TrimRight(string(b[pos:pos+0x20]), "\x00")
		pos += 0x20
	}

	pos += 4 // the format word at 0x2C..0x2F (already read individually)
	if flag01 && intDataOffset {
		pos += 4 + 12 // int32 dataStart + 3x int32 0
	} else if longDataOffset {
		pos += 8 + 8 // int64 dataStart + int64 0
	}

	pt := &paramTable{
		paramType:   paramType,
		dataVersion: dataVersion,
		rows:        make(map[int32]int64, rowCount),
		ids:         make([]int32, 0, rowCount),
		bytes:       b,
	}
	for i := 0; i < rowCount; i++ {
		if pos+8 > len(b) {
			return nil, fmt.Errorf("row index %d out of bounds", i)
		}
		id := int32(le.Uint32(b[pos:]))
		pos += 4
		var dataOff int64
		if longDataOffset {
			pos += 4 // padding
			dataOff = int64(le.Uint64(b[pos:]))
			pos += 8
			pos += 8 // nameOffset
		} else {
			dataOff = int64(le.Uint32(b[pos:]))
			pos += 4
			pos += 4 // nameOffset
		}
		pt.rows[id] = dataOff
		pt.ids = append(pt.ids, id)
	}

	pt.detectedSize = -1
	if len(pt.ids) > 1 {
		pt.detectedSize = pt.rows[pt.ids[1]] - pt.rows[pt.ids[0]]
	}
	return pt, nil
}

// checkDef validates that a paramdef matches a parsed param (data version + row
// size). A mismatch means the def and regulation are out of sync, which would make
// every field offset wrong — so we fail loudly rather than emit garbage.
func checkDef(def *paramdef, pt *paramTable) error {
	if def.DataVersion != pt.dataVersion {
		return fmt.Errorf("paramdef DataVersion %d != param %d", def.DataVersion, pt.dataVersion)
	}
	if pt.detectedSize > 0 && int64(def.rowSize()) != pt.detectedSize {
		return fmt.Errorf("row size mismatch: def computes %d, param row is %d", def.rowSize(), pt.detectedSize)
	}
	return nil
}

func readCString(b []byte, off int) string {
	if off < 0 || off >= len(b) {
		return ""
	}
	end := off
	for end < len(b) && b[end] != 0 {
		end++
	}
	return string(b[off:end])
}
