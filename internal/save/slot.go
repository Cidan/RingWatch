package save

import "fmt"

// eventFlagsLen is the fixed size of the event-flags blob in a PC character slot.
const eventFlagsLen = 0x1BF99F

// extractEventFlags walks a character-slot body (the bytes after the slot's
// 0x10-byte MD5 checksum, i.e. starting at the version u32) and returns a
// zero-copy slice of its event-flags blob.
//
// The blob's offset is NOT constant: five fields before it are data-dependent
// (gaitem_map, acquired_projectiles, unlocked_regions, menu_profile_save_load,
// tutorial_data), so we parse sequentially. Field order and sizes mirror
// ER-Save-Lib's UserDataX (PC). A zero terminator byte immediately follows the
// blob (deku assert_eq=0); we verify it as a self-check that the walk stayed
// aligned.
func extractEventFlags(body []byte) ([]byte, error) {
	off, err := walkToEventFlags(body)
	if err != nil {
		return nil, err
	}
	if off+eventFlagsLen >= len(body) {
		return nil, fmt.Errorf("save: event-flags blob out of range (offset %d)", off)
	}
	if term := body[off+eventFlagsLen]; term != 0 {
		return nil, fmt.Errorf("save: event-flags terminator not zero (got 0x%02x at offset %d); slot layout misaligned", term, off+eventFlagsLen)
	}
	return body[off : off+eventFlagsLen], nil
}

// walkToEventFlags parses a slot body up to (but not including) the event-flags
// blob and returns the blob's byte offset within the body.
func walkToEventFlags(body []byte) (int, error) {
	r := &reader{b: body}

	version := r.u32() // version
	r.skip(4)          // map_id [4]
	r.skip(8)          // unk0x8 [8]
	r.skip(16)         // unk0x10 [0x10]

	// gaitem_map: fixed count of variable-size Gaitem entries. Each entry is
	// 8 bytes (handle + item_id) plus optional trailing fields keyed on the
	// handle's high nibble (mirrors ER-Save-Lib's Gaitem skip conditions).
	gcount := 0x1400
	if version <= 81 {
		gcount = 0x13FE
	}
	for i := 0; i < gcount; i++ {
		handle := r.u32()
		r.u32() // item_id
		switch hi := handle & 0xF0000000; {
		case handle == 0 || hi == 0xC0000000:
			// no trailing fields
		case hi == 0x80000000:
			r.skip(13) // unk0x10 + unk0x14 + gem_gaitem_handle + unk0x1c
		default: // hi == 0 (handle != 0) or hi == 0x90000000
			r.skip(8) // unk0x10 + unk0x14
		}
	}

	r.skip(0x1B0)    // player_game_data
	r.skip(0xD * 16) // sp_effects (13 * 16)
	r.skip(0x58)     // equipped_items_equip_index (22 u32)
	r.skip(0x1C)     // active_weapon_slots_and_arm_style (7 u32)
	r.skip(0x58)     // equipped_items_item_id (22 u32)
	r.skip(0x58)     // equipped_items_gaitem_handle (22 u32)
	r.skip(0x9010)   // inventory_held (cap 0xa80 / 0x180)
	r.skip(0x74)     // equipped_spells (14*8 + u32)
	r.skip(0x8C)     // equipped_items
	r.skip(0x18)     // equipped_gestures (6 u32)

	projCount := r.u32() // acquired_projectiles: count then count*8
	r.skip(int(projCount) * 8)

	r.skip(0x9C)   // equipped_armaments_and_items (39 u32)
	r.skip(0xC)    // equipped_physics (3 u32)
	r.skip(0x12F)  // face_data (in-slot variant; 303 bytes)
	r.skip(0x6010) // inventory_storage_box (cap 0x780 / 0x80)
	r.skip(0x100)  // gestures (0x40 u32)

	regionCount := r.u32() // unlocked_regions: count then count*4
	r.skip(int(regionCount) * 4)

	r.skip(0x28) // horse (RideGameData)
	r.skip(1)    // control_byte_maybe
	r.skip(0x44) // blood_stain
	r.skip(8)    // unk gamedataman x2 (2 u32)

	r.skip(4) // menu_profile_save_load: two u16
	menuSize := r.u32()
	r.skip(int(menuSize))

	r.skip(0x34)        // trophy_equip_data
	r.skip(8 + 7000*16) // gaitem_game_data (count i64 + 7000 * 16-byte entries = 112008)

	r.skip(4) // tutorial_data: two u16
	tutSize := r.u32()
	tutCount := r.u32() // TutorialDataChunk.count
	if tutCount != 0 {
		r.skip(int(tutSize) - 4)
	}

	r.skip(3) // gameman 0x8c / 0x8d / 0x8e
	r.skip(4) // total_deaths_count
	// character_type(4) in_online_session_flag(1) character_type_online(4)
	// last_rested_grace(4) not_alone_flag(1) in_game_countdown_timer(4)
	// unk_gamedataman_0x124_or_0x134(4) = 22 bytes
	r.skip(22)

	if r.err != nil {
		return 0, r.err
	}
	return r.pos, nil
}
