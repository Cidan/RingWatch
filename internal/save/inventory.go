package save

// Inventory-list capacities (number of entries) within a character slot. The held
// inventory and the storage box are fixed-size arrays on disk; empty entries have a
// zero handle. The fixed sizes are what let the slot walk skip past them by a
// constant amount (held 0x9010, storage 0x6010).
const (
	heldCommonCap  = 0xA80 // 2688
	heldKeyCap     = 0x180 // 384
	storeCommonCap = 0x780 // 1920
	storeKeyCap    = 0x80  // 128
)

// GaItem handle category nibbles — the top 4 bits of an inventory entry's handle
// identify what kind of item it is. (Mirrors ClayAmore/ER-Save-Editor's
// InventoryGaitemType.)
const (
	catWeapon    = 0x80000000
	catProtector = 0x90000000
	catAccessory = 0xA0000000
	catGoods     = 0xB0000000
	catGem       = 0xC0000000
	catMask      = 0xF0000000
	idMask       = 0x0FFFFFFF
)

// OwnedItems is the set of item param ids a character possesses, partitioned by
// inventory category. Weapon ids are normalized to their base (affinity and upgrade
// level stripped) so any variant counts as owning that weapon; the rest are plain
// param ids. Goods covers sorceries, incantations and spirit ashes; Gems covers
// ashes of war.
type OwnedItems struct {
	Weapons     map[uint32]bool
	Protectors  map[uint32]bool
	Accessories map[uint32]bool
	Goods       map[uint32]bool
	Gems        map[uint32]bool
}

func newOwnedItems() *OwnedItems {
	return &OwnedItems{
		Weapons:     map[uint32]bool{},
		Protectors:  map[uint32]bool{},
		Accessories: map[uint32]bool{},
		Goods:       map[uint32]bool{},
		Gems:        map[uint32]bool{},
	}
}

// Total reports the number of distinct owned items across all categories.
func (o *OwnedItems) Total() int {
	return len(o.Weapons) + len(o.Protectors) + len(o.Accessories) + len(o.Goods) + len(o.Gems)
}

// add resolves one inventory entry into the owned sets. Weapons, armor and gems
// store only a GaItem handle whose real param id lives in the gaitem table;
// accessories and goods encode their param id directly in the handle's low 28 bits.
// (Mirrors ClayAmore/ER-Save-Editor's inventory resolution.)
func (o *OwnedItems) add(gaitems map[uint32]uint32, handle, quantity uint32) {
	if handle == 0 {
		return
	}
	switch handle & catMask {
	case catWeapon:
		if id, ok := gaitems[handle]; ok && id != 0xFFFFFFFF {
			o.Weapons[(id/10000)*10000] = true
		}
	case catProtector:
		if id, ok := gaitems[handle]; ok && id != 0xFFFFFFFF {
			o.Protectors[id&idMask] = true
		}
	case catAccessory:
		o.Accessories[handle&idMask] = true
	case catGoods:
		if quantity > 0 {
			o.Goods[handle&idMask] = true
		}
	case catGem:
		if id, ok := gaitems[handle]; ok && id != 0xFFFFFFFF {
			o.Gems[id&idMask] = true
		}
	}
}

// readInventory parses one Inventory structure — a fixed-capacity common list and
// key list, each preceded by a populated-count u32, followed by two counter u32s —
// resolving every populated entry into owned. The fixed capacities make the whole
// structure a constant byte size, keeping the surrounding slot walk aligned.
func readInventory(r *reader, owned *OwnedItems, gaitems map[uint32]uint32, commonCap, keyCap int) {
	r.u32() // common populated count (we iterate the full array and skip empty handles)
	for i := 0; i < commonCap; i++ {
		handle := r.u32()
		quantity := r.u32()
		r.u32() // acquisition index
		owned.add(gaitems, handle, quantity)
	}
	r.u32() // key populated count
	for i := 0; i < keyCap; i++ {
		handle := r.u32()
		quantity := r.u32()
		r.u32() // acquisition index
		owned.add(gaitems, handle, quantity)
	}
	r.u32() // equip_index_counter
	r.u32() // acquisition_index_counter
}
