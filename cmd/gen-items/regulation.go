package main

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"fmt"
	"strings"
	"unicode/utf16"

	"github.com/klauspost/compress/zstd"
)

// regulationKey is the AES-256 key for Elden Ring's regulation.bin (well-known;
// from ClayAmore/ER-Save-Editor src/util/regulation.rs).
var regulationKey = []byte{
	0x99, 0xBF, 0xFC, 0x36, 0x6A, 0x6B, 0xC8, 0xC6, 0xF5, 0x82, 0x7D, 0x09, 0x36, 0x02, 0xD6, 0x76,
	0xC4, 0x28, 0x92, 0xA0, 0x1C, 0x20, 0x7F, 0xB0, 0x24, 0xD3, 0xAF, 0x4E, 0x49, 0x3F, 0xEF, 0x99,
}

// readRegulation decrypts and decompresses regulation.bin and splits the BND4 into
// a map of param-name -> raw param bytes.
func readRegulation(raw []byte) (map[string][]byte, error) {
	plain, err := decryptRegulation(raw)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	bnd4, err := decompressDCX(plain)
	if err != nil {
		return nil, fmt.Errorf("dcx: %w", err)
	}
	return parseBND4(bnd4)
}

// decryptRegulation runs AES-256-CBC (no padding); IV is the first 16 bytes,
// ciphertext is the remainder.
func decryptRegulation(data []byte) ([]byte, error) {
	if len(data) < 32 {
		return nil, fmt.Errorf("regulation too small (%d bytes)", len(data))
	}
	iv := data[:16]
	ct := make([]byte, len(data)-16)
	copy(ct, data[16:])
	if len(ct)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("ciphertext not block-aligned (%d)", len(ct))
	}
	block, err := aes.NewCipher(regulationKey)
	if err != nil {
		return nil, err
	}
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(ct, ct)
	return ct, nil
}

// decompressDCX inflates a DCX container (Elden Ring DLC uses ZSTD).
func decompressDCX(p []byte) ([]byte, error) {
	if len(p) < 0x4C || string(p[0:4]) != "DCX\x00" {
		return nil, fmt.Errorf("not a DCX container")
	}
	method := string(p[0x28:0x2C])
	if method != "ZSTD" {
		return nil, fmt.Errorf("unsupported DCX compression %q (expected ZSTD)", method)
	}
	be := binary.BigEndian
	uncompressed := be.Uint32(p[0x1C:])
	compressed := be.Uint32(p[0x20:])
	const start = 0x4C
	if start+int(compressed) > len(p) {
		return nil, fmt.Errorf("DCX compressed size %d out of range", compressed)
	}
	dec, err := zstd.NewReader(nil)
	if err != nil {
		return nil, err
	}
	defer dec.Close()
	out, err := dec.DecodeAll(p[start:start+int(compressed)], nil)
	if err != nil {
		return nil, err
	}
	if uint32(len(out)) != uncompressed {
		return nil, fmt.Errorf("DCX size mismatch: got %d, header says %d", len(out), uncompressed)
	}
	return out, nil
}

// parseBND4 splits an Elden Ring BND4 (little-endian, long offsets, names, inner
// files stored raw) into a basename -> bytes map.
func parseBND4(b []byte) (map[string][]byte, error) {
	if len(b) < 0x40 || string(b[0:4]) != "BND4" {
		return nil, fmt.Errorf("not a BND4")
	}
	if b[0x09] != 0 {
		return nil, fmt.Errorf("unexpected big-endian BND4")
	}
	le := binary.LittleEndian
	bitBigEndian := b[0x0A] == 0 // SoulsFormats stores this negated (bitBigEndian = !byte)
	fileCount := int(int32(le.Uint32(b[0x0C:])))
	unicode := b[0x30] != 0
	format := normalizeFormat(b[0x31], bitBigEndian)

	hasLongOffsets := format&0x10 != 0
	hasCompression := format&0x20 != 0
	hasIDs := format&0x02 != 0
	hasNames := format&(0x04|0x08) != 0

	out := make(map[string][]byte, fileCount)
	pos := 0x40
	for i := 0; i < fileCount; i++ {
		pos += 1 + 3 + 4 // flags(1) + zeros(3) + int32(-1)
		if pos+8 > len(b) {
			return nil, fmt.Errorf("bnd4 entry %d header out of bounds", i)
		}
		compressedSize := le.Uint64(b[pos:])
		pos += 8 // compressedSize (i64)
		var uncompressedSize uint64
		if hasCompression {
			uncompressedSize = le.Uint64(b[pos:])
			pos += 8
		} else {
			uncompressedSize = uint64(le.Uint32(b[pos:]))
			pos += 4
		}
		var dataOffset uint64
		if hasLongOffsets {
			dataOffset = le.Uint64(b[pos:])
			pos += 8
		} else {
			dataOffset = uint64(le.Uint32(b[pos:]))
			pos += 4
		}
		if hasIDs {
			pos += 4
		}
		nameOffset := -1
		if hasNames {
			nameOffset = int(le.Uint32(b[pos:]))
			pos += 4
		}
		name := ""
		if nameOffset >= 0 {
			if unicode {
				name = readUTF16(b, nameOffset)
			} else {
				name = readCString(b, nameOffset)
			}
		}
		end := dataOffset + uncompressedSize
		if end > uint64(len(b)) {
			return nil, fmt.Errorf("bnd4 file %q (entry %d/%d): comp=%d uncomp=%d dataOff=%d end=%d > len=%d",
				name, i, fileCount, compressedSize, uncompressedSize, dataOffset, end, len(b))
		}
		out[baseName(name)] = b[dataOffset:end]
	}
	return out, nil
}

// normalizeFormat reproduces SoulsFormats' BND4 format-flag bit-order handling.
func normalizeFormat(raw byte, bitBigEndian bool) byte {
	reverse := bitBigEndian || ((raw&0x01) != 0 && (raw&0x80) == 0)
	if reverse {
		return raw
	}
	var r byte
	for i := 0; i < 8; i++ {
		r |= ((raw >> uint(i)) & 1) << uint(7-i)
	}
	return r
}

func readUTF16(b []byte, off int) string {
	var u []uint16
	for i := off; i+1 < len(b); i += 2 {
		c := uint16(b[i]) | uint16(b[i+1])<<8
		if c == 0 {
			break
		}
		u = append(u, c)
	}
	return string(utf16.Decode(u))
}

// baseName turns "N:\\GR\\data\\...\\EquipParamWeapon.param" into "EquipParamWeapon".
func baseName(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	if i := strings.LastIndexByte(p, '/'); i >= 0 {
		p = p[i+1:]
	}
	if i := strings.LastIndexByte(p, '.'); i >= 0 {
		p = p[:i]
	}
	return p
}
