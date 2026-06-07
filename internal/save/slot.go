package save

import "fmt"

// eventFlagsLen is the fixed size of the event-flags blob in a PC character slot.
const eventFlagsLen = 0x1BF99F

// slotData is everything we extract from a single character slot: the event-flags
// blob (boss state) and the owned-item sets (item collection state).
type slotData struct {
	eventFlags []byte
	owned      *OwnedItems
}

// parseSlot walks a character-slot body (the bytes after the slot's 0x10-byte MD5
// checksum, i.e. starting at the version u32) and extracts both the event-flags blob
// and the character's owned items.
//
// Neither sits at a constant offset: the gaitem table and several lists before them
// are data-dependent, so we parse sequentially. Field order and sizes mirror
// ER-Save-Lib's UserDataX (PC). The two inventory lists are fixed-capacity, so we
// resolve their entries in place; the event-flags blob follows, terminated by a zero
// byte we verify as a self-check that the walk stayed aligned.
func parseSlot(body []byte) (*slotData, error) {
	r := &reader{b: body}
	owned := newOwnedItems()
	gaitems := make(map[uint32]uint32, 0x1400)

	version := r.u32() // version
	r.skip(4)          // map_id [4]
	r.skip(8)          // unk0x8 [8]
	r.skip(16)         // unk0x10 [0x10]

	// gaitem_map: fixed count of variable-size Gaitem entries. Each is 8 bytes
	// (handle + item_id) plus optional trailing fields keyed on the handle's high
	// nibble (mirrors ER-Save-Lib's Gaitem skip conditions). We record
	// handle->item_id so weapon/armor/gem inventory entries — which store only a
	// handle — can be resolved to their real param ids.
	gcount := 0x1400
	if version <= 81 {
		gcount = 0x13FE
	}
	for i := 0; i < gcount; i++ {
		handle := r.u32()
		itemID := r.u32()
		if handle != 0 && itemID != 0xFFFFFFFF {
			gaitems[handle] = itemID
		}
		switch hi := handle & catMask; {
		case handle == 0 || hi == catGem:
			// no trailing fields
		case hi == catWeapon:
			r.skip(13) // unk0x10 + unk0x14 + gem_gaitem_handle + unk0x1c
		default: // hi == 0 (handle != 0) or hi == catProtector
			r.skip(8) // unk0x10 + unk0x14
		}
	}

	r.skip(0x1B0)    // player_game_data
	r.skip(0xD * 16) // sp_effects (13 * 16)
	r.skip(0x58)     // equipped_items_equip_index (22 u32)
	r.skip(0x1C)     // active_weapon_slots_and_arm_style (7 u32)
	r.skip(0x58)     // equipped_items_item_id (22 u32)
	r.skip(0x58)     // equipped_items_gaitem_handle (22 u32)

	readInventory(r, owned, gaitems, heldCommonCap, heldKeyCap) // inventory_held (0x9010)

	r.skip(0x74) // equipped_spells (14*8 + u32)
	r.skip(0x8C) // equipped_items
	r.skip(0x18) // equipped_gestures (6 u32)

	projCount := r.u32() // acquired_projectiles: count then count*8
	r.skip(int(projCount) * 8)

	r.skip(0x9C)  // equipped_armaments_and_items (39 u32)
	r.skip(0xC)   // equipped_physics (3 u32)
	r.skip(0x12F) // face_data (in-slot variant; 303 bytes)

	readInventory(r, owned, gaitems, storeCommonCap, storeKeyCap) // inventory_storage_box (0x6010)

	r.skip(0x100) // gestures (0x40 u32)

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

	r.skip(3)  // gameman 0x8c / 0x8d / 0x8e
	r.skip(4)  // total_deaths_count
	r.skip(22) // character_type..unk_gamedataman_0x124_or_0x134 (22 bytes)

	if r.err != nil {
		return nil, r.err
	}
	off := r.pos
	if off+eventFlagsLen >= len(body) {
		return nil, fmt.Errorf("save: event-flags blob out of range (offset %d)", off)
	}
	if term := body[off+eventFlagsLen]; term != 0 {
		return nil, fmt.Errorf("save: event-flags terminator not zero (got 0x%02x at offset %d); slot layout misaligned", term, off+eventFlagsLen)
	}
	return &slotData{eventFlags: body[off : off+eventFlagsLen], owned: owned}, nil
}
