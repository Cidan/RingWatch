package save

import (
	"bufio"
	_ "embed"
	"strconv"
	"strings"
)

// eventflag_bst.txt comes from ClayAmore/ER-Save-Lib (src/res/eventflag_bst.txt).
// Each line is "block,group": it maps an event-flag block (eventID/1000) to the
// group multiplier used to locate that block within the save's event-flags blob.
//
//go:embed eventflag_bst.txt
var bstData string

var bstMap = parseBST(bstData)

func parseBST(data string) map[uint32]uint32 {
	m := make(map[uint32]uint32, 12000)
	sc := bufio.NewScanner(strings.NewReader(data))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		bs, gs, ok := strings.Cut(line, ",")
		if !ok {
			continue
		}
		block, err1 := strconv.ParseUint(strings.TrimSpace(bs), 10, 32)
		group, err2 := strconv.ParseUint(strings.TrimSpace(gs), 10, 32)
		if err1 != nil || err2 != nil {
			continue
		}
		m[uint32(block)] = uint32(group)
	}
	return m
}

const (
	flagDivisor = 1000
	blockSize   = 125
)

// flagSet reports whether the event flag eventID is set within the event-flags
// blob. ok is false when the flag's block is not present in the BST map (i.e. the
// flag is not addressable in this save format).
//
// The mapping mirrors ER-Save-Lib's get_event_flag:
//
//	block = id / 1000;  index = id % 1000
//	group = bst[block]
//	byte  = group*125 + index/8
//	bit   = 7 - (index % 8)        // MSB-first within the byte
func flagSet(eventFlags []byte, eventID uint32) (set bool, ok bool) {
	block := eventID / flagDivisor
	index := eventID % flagDivisor
	group, found := bstMap[block]
	if !found {
		return false, false
	}
	byteOff := int(group)*blockSize + int(index)/8
	if byteOff < 0 || byteOff >= len(eventFlags) {
		return false, false
	}
	bit := 7 - (index % 8)
	return (eventFlags[byteOff]>>bit)&1 == 1, true
}

// BSTSize reports the number of loaded block→group mappings (used in tests).
func BSTSize() int { return len(bstMap) }

// FlagAddressable reports whether an event flag can be located in the save's
// event-flags blob (its block is mapped and the computed byte is in range).
// A boss whose flag is not addressable can never be reported as defeated.
func FlagAddressable(eventID uint32) bool {
	block := eventID / flagDivisor
	index := eventID % flagDivisor
	group, found := bstMap[block]
	if !found {
		return false
	}
	byteOff := int(group)*blockSize + int(index)/8
	return byteOff >= 0 && byteOff < eventFlagsLen
}
