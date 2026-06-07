package tracker

import (
	"sort"
	"strings"
)

// Owner reports whether the player owns a given item. The UI builds this from a
// save's parsed owned-item sets; the tracker stays save-format agnostic.
type Owner func(it Item) bool

// ItemStatus is an item plus its resolved ownership.
type ItemStatus struct {
	Item
	Owned bool
}

// ItemGroup is a set of items sharing a grouping key (a region, a kind, a weapon
// type, or a primary stat).
type ItemGroup struct {
	Name  string
	Items []ItemStatus
}

// Owned returns the number of owned items in the group.
func (g ItemGroup) Owned() int {
	n := 0
	for _, it := range g.Items {
		if it.Owned {
			n++
		}
	}
	return n
}

// Total returns the number of items in the group.
func (g ItemGroup) Total() int { return len(g.Items) }

// Complete reports whether every item in the group is owned.
func (g ItemGroup) Complete() bool { return g.Total() > 0 && g.Owned() == g.Total() }

// ItemProgress is the computed collection state for one grouping + filter.
type ItemProgress struct {
	Groups []ItemGroup
}

// Totals returns overall owned and total counts.
func (p ItemProgress) Totals() (owned, total int) {
	for _, g := range p.Groups {
		owned += g.Owned()
		total += g.Total()
	}
	return
}

// CategoryTotals splits totals into base-game and DLC.
func (p ItemProgress) CategoryTotals() (baseOwned, baseTotal, dlcOwned, dlcTotal int) {
	for _, g := range p.Groups {
		for _, it := range g.Items {
			if it.DLC {
				dlcTotal++
				if it.Owned {
					dlcOwned++
				}
			} else {
				baseTotal++
				if it.Owned {
					baseOwned++
				}
			}
		}
	}
	return
}

// GroupBy selects how items are partitioned in the left pane.
type GroupBy int

const (
	GroupRegion GroupBy = iota
	GroupKind
	GroupWeaponType
	GroupPrimaryStat
)

// GroupByOrder is the cycle order of grouping strategies.
var GroupByOrder = []GroupBy{GroupRegion, GroupKind, GroupWeaponType, GroupPrimaryStat}

// Label is the display name of a grouping strategy.
func (g GroupBy) Label() string {
	switch g {
	case GroupRegion:
		return "Location"
	case GroupKind:
		return "Category"
	case GroupWeaponType:
		return "Weapon Type"
	case GroupPrimaryStat:
		return "Primary Stat"
	}
	return "?"
}

// ComputeItems builds item progress from the full embedded dataset.
func ComputeItems(owner Owner, filter ItemKind, by GroupBy) ItemProgress {
	return computeItemsFrom(allItems, owner, filter, by)
}

// computeItemsFrom is the testable core: filter by kind (empty = all), partition by
// the grouping strategy, order the groups and the items within them.
func computeItemsFrom(items []Item, owner Owner, filter ItemKind, by GroupBy) ItemProgress {
	keyFn := groupKeyFunc(by)
	byKey := map[string][]ItemStatus{}
	for _, it := range items {
		if filter != "" && it.Kind != filter {
			continue
		}
		owned := owner != nil && owner(it)
		k := keyFn(it)
		byKey[k] = append(byKey[k], ItemStatus{Item: it, Owned: owned})
	}

	keys := make([]string, 0, len(byKey))
	for k := range byKey {
		keys = append(keys, k)
	}
	orderGroupKeys(by, keys)

	groups := make([]ItemGroup, 0, len(keys))
	for _, k := range keys {
		its := byKey[k]
		sort.SliceStable(its, func(i, j int) bool { return itemLess(its[i].Item, its[j].Item) })
		groups = append(groups, ItemGroup{Name: k, Items: its})
	}
	return ItemProgress{Groups: groups}
}

func groupKeyFunc(by GroupBy) func(Item) string {
	switch by {
	case GroupKind:
		return func(it Item) string { return KindLabel[it.Kind] }
	case GroupWeaponType:
		return func(it Item) string {
			if it.WeaponType != "" {
				return it.WeaponType
			}
			return KindLabel[it.Kind]
		}
	case GroupPrimaryStat:
		return func(it Item) string {
			if it.PrimaryStat == "" {
				return "—"
			}
			return it.PrimaryStat
		}
	default: // GroupRegion
		return func(it Item) string {
			if it.Region == "" {
				return "Unknown"
			}
			return it.Region
		}
	}
}

func orderGroupKeys(by GroupBy, keys []string) {
	switch by {
	case GroupKind:
		sort.SliceStable(keys, func(i, j int) bool { return kindLabelRank(keys[i]) < kindLabelRank(keys[j]) })
	case GroupWeaponType:
		sort.SliceStable(keys, func(i, j int) bool { return wepTypeRank(keys[i]) < wepTypeRank(keys[j]) })
	case GroupPrimaryStat:
		sort.SliceStable(keys, func(i, j int) bool { return statRank(keys[i]) < statRank(keys[j]) })
	default: // GroupRegion
		sort.SliceStable(keys, func(i, j int) bool {
			ri, rj := regionOrderRank(keys[i]), regionOrderRank(keys[j])
			if ri != rj {
				return ri < rj
			}
			return keys[i] < keys[j]
		})
	}
}

// regionOrderRank ranks a region name into base progression order, then DLC order,
// then known-but-unlisted (merchants etc.), with Unknown dead last.
func regionOrderRank(name string) int {
	if name == "" || name == "Unknown" {
		return 100000
	}
	for i, r := range baseRegionOrder {
		if r == name {
			return i
		}
	}
	for i, r := range dlcRegionOrder {
		if r == name {
			return 1000 + i
		}
	}
	return 2000
}

func kindLabelRank(label string) int {
	for i, k := range KindOrder {
		if KindLabel[k] == label {
			return i
		}
	}
	return 999
}

// weaponTypeOrder is the canonical display order of weapon categories (mirrors the
// in-game WEP_TYPE grouping).
var weaponTypeOrder = []string{
	"Dagger", "Straight Sword", "Greatsword", "Colossal Sword",
	"Thrusting Sword", "Heavy Thrusting Sword", "Curved Sword", "Curved Greatsword",
	"Katana", "Twinblade", "Axe", "Greataxe", "Hammer", "Flail", "Great Hammer",
	"Colossal Weapon", "Spear", "Great Spear", "Halberd", "Reaper", "Whip", "Fist", "Claw",
	"Hand-to-Hand", "Throwing Blade", "Backhand Blade", "Perfume Bottle", "Beast Claw", "Light Greatsword",
	"Great Katana", "Light Bow", "Bow", "Greatbow", "Crossbow", "Ballista",
	"Glintstone Staff", "Sacred Seal", "Torch",
	"Small Shield", "Medium Shield", "Greatshield",
}

func wepTypeRank(key string) int {
	for i, t := range weaponTypeOrder {
		if t == key {
			return i
		}
	}
	// Non-weapon kinds (grouped under their kind label) sort after all weapon types.
	if r := kindLabelRank(key); r < 999 {
		return 100 + r
	}
	return 999
}

var primaryStatOrder = []string{"Str", "Dex", "Int", "Fai", "Arc", "—"}

func statRank(key string) int {
	for i, s := range primaryStatOrder {
		if s == key {
			return i
		}
	}
	return len(primaryStatOrder)
}

func itemLess(a, b Item) bool {
	an, bn := strings.ToLower(a.Name), strings.ToLower(b.Name)
	if an != bn {
		return an < bn
	}
	return a.ID < b.ID
}
