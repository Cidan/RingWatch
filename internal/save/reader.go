package save

import (
	"encoding/binary"
	"fmt"
)

// reader is a little-endian cursor over a byte slice. Once an out-of-bounds read
// occurs, err is set and all subsequent reads are no-ops returning zero values,
// so callers can check err once at the end of a parse walk.
type reader struct {
	b   []byte
	pos int
	err error
}

func (r *reader) need(n int) bool {
	if r.err != nil {
		return false
	}
	if n < 0 || r.pos+n > len(r.b) {
		r.err = fmt.Errorf("save: read out of bounds at offset %d (+%d, len %d)", r.pos, n, len(r.b))
		return false
	}
	return true
}

func (r *reader) skip(n int) {
	if r.need(n) {
		r.pos += n
	}
}

func (r *reader) u8() uint8 {
	if !r.need(1) {
		return 0
	}
	v := r.b[r.pos]
	r.pos++
	return v
}

func (r *reader) u16() uint16 {
	if !r.need(2) {
		return 0
	}
	v := binary.LittleEndian.Uint16(r.b[r.pos:])
	r.pos += 2
	return v
}

func (r *reader) u32() uint32 {
	if !r.need(4) {
		return 0
	}
	v := binary.LittleEndian.Uint32(r.b[r.pos:])
	r.pos += 4
	return v
}

func (r *reader) u64() uint64 {
	if !r.need(8) {
		return 0
	}
	v := binary.LittleEndian.Uint64(r.b[r.pos:])
	r.pos += 8
	return v
}

// slice returns a zero-copy subslice of n bytes and advances the cursor.
func (r *reader) slice(n int) []byte {
	if !r.need(n) {
		return nil
	}
	s := r.b[r.pos : r.pos+n]
	r.pos += n
	return s
}
