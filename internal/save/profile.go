package save

import (
	"fmt"
	"strings"
	"unicode/utf16"
)

// user_data_10 holds the profile summaries used for the character list.
const (
	ud10Body      = 0x300 + numSlots*slotStride + 0x10 // body after the 0x10 checksum
	profileStride = 0x24C                              // bytes per Profile entry
	profileCount  = 10
)

// decodeWString decodes a fixed-size UTF-16LE buffer, stopping at the first NUL.
func decodeWString(b []byte) string {
	u16s := make([]uint16, 0, len(b)/2)
	for i := 0; i+1 < len(b); i += 2 {
		c := uint16(b[i]) | uint16(b[i+1])<<8
		if c == 0 {
			break
		}
		u16s = append(u16s, c)
	}
	return strings.TrimRight(string(utf16.Decode(u16s)), " ")
}

// parseCharacters reads the user_data_10 ProfileSummary: the active-slot bitmap
// followed by ten Profile records (name / level / seconds_played).
func parseCharacters(file []byte) ([]Character, error) {
	if len(file) < ud10Body {
		return nil, fmt.Errorf("save: file too small for user_data_10 (len %d)", len(file))
	}
	r := &reader{b: file, pos: ud10Body}
	r.skip(4)     // version
	r.skip(8)     // steam_id (u64)
	r.skip(0x140) // Settings
	r.skip(4)     // MenuSystemSaveLoad: two u16
	menuSize := r.u32()
	r.skip(int(menuSize))

	active := r.slice(profileCount) // active_profiles [bool;10]
	if r.err != nil {
		return nil, r.err
	}

	chars := make([]Character, 0, profileCount)
	for i := 0; i < profileCount; i++ {
		start := r.pos
		name := decodeWString(r.slice(32)) // character_name (32-byte wstring)
		r.skip(2)                          // name terminator
		level := r.u32()                   // +0x22
		seconds := r.u32()                 // +0x26
		r.pos = start + profileStride      // advance to next Profile
		if r.err != nil {
			return nil, r.err
		}
		chars = append(chars, Character{
			Slot:    i,
			Name:    name,
			Level:   int(level),
			Seconds: int(seconds),
			Active:  active[i] != 0,
		})
	}
	return chars, nil
}
