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
)

// KindOrder is the canonical display order of kinds.
var KindOrder = []ItemKind{
	KindWeapon, KindShield, KindArmor, KindSorcery,
	KindIncantation, KindTalisman, KindAsh, KindSpirit,
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
}

var allItems = loadItems()

func loadItems() []Item {
	var its []Item
	if err := json.Unmarshal(itemsJSON, &its); err != nil {
		panic(fmt.Sprintf("tracker: invalid embedded item data: %v", err))
	}
	return its
}

// Items returns a copy of the full item dataset.
func Items() []Item {
	out := make([]Item, len(allItems))
	copy(out, allItems)
	return out
}
