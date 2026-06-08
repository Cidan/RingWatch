package tracker

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

// items.generated.json is produced by cmd/gen-items from the game's regulation.bin
// (param ids/types/stats) joined with curated names and acquisition locations.
// Schema is one flat array, mirroring bosses.generated.json.
//
//go:embed data/items.generated.json
var itemsJSON []byte

// flask-upgrades.json is the curated Golden Seed / Sacred Tear dataset. Unlike
// items.generated.json it is NOT regulation-derived: seeds/tears are consumable
// goods (one shared param id each) tracked by per-pickup event flag, sourced from
// thefifthmatt/SoulsRandomizers itemslots.txt and verified against real saves.
// It is kept separate so regenerating items.generated.json never clobbers it; the
// two are merged below into a single item list.
//
//go:embed data/flask-upgrades.json
var flaskJSON []byte

// ItemKind classifies a trackable collectible. Shields are split from weapons (so
// they can be filtered separately) even though the game stores them in the same
// param; ash = Ash of War (a Gem), spirit = Spirit Ash summon.
type ItemKind string

const (
	KindWeapon      ItemKind = "weapon"
	KindShield      ItemKind = "shield"
	KindArmor       ItemKind = "armor"
	KindSorcery     ItemKind = "sorcery"
	KindIncantation ItemKind = "incantation"
	KindTalisman    ItemKind = "talisman"
	KindAsh         ItemKind = "ash"    // Ash of War
	KindSpirit      ItemKind = "spirit" // Spirit Ash summon
	KindFlask       ItemKind = "flask"  // Golden Seed / Sacred Tear (flask upgrade)
)

// KindOrder is the canonical display order of kinds.
var KindOrder = []ItemKind{
	KindWeapon, KindShield, KindArmor, KindSorcery,
	KindIncantation, KindTalisman, KindAsh, KindSpirit, KindFlask,
}

// KindLabel maps a kind to its plural display heading.
var KindLabel = map[ItemKind]string{
	KindWeapon:      "Weapons",
	KindShield:      "Shields",
	KindArmor:       "Armor",
	KindSorcery:     "Sorceries",
	KindIncantation: "Incantations",
	KindTalisman:    "Talismans",
	KindAsh:         "Ashes of War",
	KindSpirit:      "Spirit Ashes",
	KindFlask:       "Flask Upgrades",
}

// Item is one trackable collectible. ID is the param id used to match the save's
// owned-item sets: for weapons/shields it is the base weapon id (affinity and
// upgrade level stripped); for everything else it is the plain param id (for
// spells, the goods id that appears in the inventory).
type Item struct {
	ID          uint32            `json:"id"`
	Name        string            `json:"name"`
	Kind        ItemKind          `json:"kind"`
	WeaponType  string            `json:"weapon_type,omitempty"` // weapons/shields, e.g. "Katana"
	ArmorSlot   string            `json:"armor_slot,omitempty"`  // armor: Head/Body/Arms/Legs
	Str         int               `json:"str,omitempty"`         // stat requirement
	Dex         int               `json:"dex,omitempty"`
	Int         int               `json:"int,omitempty"`
	Fai         int               `json:"fai,omitempty"`
	Arc         int               `json:"arc,omitempty"`
	PrimaryStat string            `json:"primary_stat,omitempty"` // Str/Dex/Int/Fai/Arc or "" if n/a
	Scaling     map[string]string `json:"scaling,omitempty"`      // stat -> grade (weapons)
	Weight      float64           `json:"weight,omitempty"`
	Region      string            `json:"region,omitempty"`   // acquisition region ("" = unknown)
	Location    string            `json:"location,omitempty"` // prose, optional
	DLC         bool              `json:"dlc"`

	// Flag is the event flag set when this item is acquired. It is populated only
	// for flask upgrades (Golden Seeds / Sacred Tears), which are consumed on use
	// and so cannot be tracked by inventory ownership — ownership is read from this
	// pickup flag instead (the same flag space as boss defeats). 0 for all other
	// kinds, which are tracked by owning their param ID.
	Flag uint32 `json:"flag,omitempty"`
}

var allItems = loadItems()

func loadItems() []Item {
	var its []Item
	if err := json.Unmarshal(itemsJSON, &its); err != nil {
		panic(fmt.Sprintf("tracker: invalid embedded item data: %v", err))
	}
	var flask []Item
	if err := json.Unmarshal(flaskJSON, &flask); err != nil {
		panic(fmt.Sprintf("tracker: invalid embedded flask data: %v", err))
	}
	return append(its, flask...)
}

// Items returns a copy of the full item dataset.
func Items() []Item {
	out := make([]Item, len(allItems))
	copy(out, allItems)
	return out
}
