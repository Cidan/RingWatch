package main

import (
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/Cidan/RingWatch/internal/tracker"
)

// nameMaps holds id->English name per param category (spells use the goods name).
type nameMaps struct {
	weapon    map[uint32]string
	protector map[uint32]string
	accessory map[uint32]string
	goods     map[uint32]string
	gem       map[uint32]string
}

// buildItems reads the six params from a decrypted regulation and classifies every
// in-scope row into a tracker.Item (without locations, which are merged later).
// Rows without a known English name are skipped (cut/unused content).
func buildItems(params map[string][]byte, names nameMaps) ([]tracker.Item, error) {
	wepType, err := loadEnum("WEP_TYPE")
	if err != nil {
		return nil, err
	}
	protCat, err := loadEnum("PROTECTOR_CATEGORY")
	if err != nil {
		return nil, err
	}

	var items []tracker.Item

	// --- Weapons & shields (EquipParamWeapon; base rows only: id % 10000 == 0) ---
	def, pt, err := loadParam(params, "EquipParamWeapon")
	if err != nil {
		return nil, err
	}
	wInt := set("wepType", "properStrength", "properAgility", "properMagic", "properFaith", "properLuck")
	wFloat := set("correctStrength", "correctAgility", "correctMagic", "correctFaith", "correctLuck", "weight")
	for _, id := range pt.ids {
		if id <= 0 || id%10000 != 0 {
			continue
		}
		name := names.weapon[uint32(id)]
		if !named(name) {
			continue
		}
		ints, floats, err := def.readRow(pt.bytes, pt.rows[id], wInt, wFloat)
		if err != nil {
			return nil, err
		}
		wt := int(ints["wepType"])
		if wt == 0 || isAmmoType(wt) {
			continue
		}
		kind := tracker.KindWeapon
		if isShieldType(wt) {
			kind = tracker.KindShield
		}
		sc := map[string]float64{
			"Str": floats["correctStrength"], "Dex": floats["correctAgility"],
			"Int": floats["correctMagic"], "Fai": floats["correctFaith"], "Arc": floats["correctLuck"],
		}
		items = append(items, tracker.Item{
			ID: uint32(id), Name: name, Kind: kind, WeaponType: wepType[wt],
			Str: int(ints["properStrength"]), Dex: int(ints["properAgility"]),
			Int: int(ints["properMagic"]), Fai: int(ints["properFaith"]), Arc: int(ints["properLuck"]),
			PrimaryStat: primaryStat([]string{"Str", "Dex", "Int", "Fai", "Arc"}, sc),
			Scaling:     scalingGrades(sc),
			Weight:      round1(floats["weight"]),
		})
	}

	// --- Protectors / armor (EquipParamProtector) ---
	def, pt, err = loadParam(params, "EquipParamProtector")
	if err != nil {
		return nil, err
	}
	for _, id := range pt.ids {
		if id <= 0 {
			continue
		}
		name := names.protector[uint32(id)]
		if !named(name) {
			continue
		}
		ints, floats, err := def.readRow(pt.bytes, pt.rows[id], set("protectorCategory"), set("weight"))
		if err != nil {
			return nil, err
		}
		cat := int(ints["protectorCategory"])
		if cat > 3 { // 0..3 = Head/Body/Arms/Legs; 4 = Hair (cosmetic) — skip
			continue
		}
		items = append(items, tracker.Item{
			ID: uint32(id), Name: name, Kind: tracker.KindArmor,
			ArmorSlot: protCat[cat], Weight: round1(floats["weight"]),
		})
	}

	// --- Accessories / talismans (EquipParamAccessory) ---
	def, pt, err = loadParam(params, "EquipParamAccessory")
	if err != nil {
		return nil, err
	}
	for _, id := range pt.ids {
		if id <= 0 {
			continue
		}
		name := names.accessory[uint32(id)]
		if !named(name) {
			continue
		}
		_, floats, err := def.readRow(pt.bytes, pt.rows[id], nil, set("weight"))
		if err != nil {
			return nil, err
		}
		items = append(items, tracker.Item{
			ID: uint32(id), Name: name, Kind: tracker.KindTalisman, Weight: round1(floats["weight"]),
		})
	}

	// --- Goods: sorceries, incantations, spirit ashes (EquipParamGoods + MagicParam) ---
	def, pt, err = loadParam(params, "EquipParamGoods")
	if err != nil {
		return nil, err
	}
	magicDef, magicPt, err := loadParam(params, "MagicParam")
	if err != nil {
		return nil, err
	}
	mInt := set("requirementIntellect", "requirementFaith", "requirementLuck", "ezStateBehaviorType")
	for _, id := range pt.ids {
		if id <= 0 {
			continue
		}
		ints, _, err := def.readRow(pt.bytes, pt.rows[id], set("goodsType", "refId_default"), nil)
		if err != nil {
			return nil, err
		}
		kind, isSpell := goodsKind(int(ints["goodsType"]))
		if kind == "" {
			continue
		}
		name := names.goods[uint32(id)]
		if !named(name) {
			continue
		}
		if !isSpell && levelSuffixRx.MatchString(name) {
			continue // spirit-ash upgrade level (+1..+10); keep only the base summon
		}
		it := tracker.Item{ID: uint32(id), Name: cleanGoodsName(name), Kind: kind}
		if isSpell {
			mid := int32(ints["refId_default"])
			if off, ok := magicPt.rows[mid]; ok {
				m, _, err := magicDef.readRow(magicPt.bytes, off, mInt, nil)
				if err != nil {
					return nil, err
				}
				it.Int, it.Fai, it.Arc = int(m["requirementIntellect"]), int(m["requirementFaith"]), int(m["requirementLuck"])
				switch int(m["ezStateBehaviorType"]) {
				case 0:
					it.Kind = tracker.KindSorcery
				case 1:
					it.Kind = tracker.KindIncantation
				}
				it.PrimaryStat = primaryStatInt(map[string]int{"Int": it.Int, "Fai": it.Fai, "Arc": it.Arc})
			}
		}
		items = append(items, it)
	}

	// --- Gems / ashes of war (EquipParamGem). Many AoW have a low-id template row
	// plus the real high-id row the inventory uses; dedupe by name keeping the
	// highest id, and strip the "Ash of War:" prefix. ---
	_, pt, err = loadParam(params, "EquipParamGem")
	if err != nil {
		return nil, err
	}
	gemByName := map[string]tracker.Item{}
	for _, id := range pt.ids {
		if id <= 0 {
			continue
		}
		name := names.gem[uint32(id)]
		if !named(name) {
			continue
		}
		name = strings.TrimSpace(strings.TrimPrefix(name, "Ash of War:"))
		if name == "" {
			continue
		}
		if cur, ok := gemByName[name]; !ok || uint32(id) > cur.ID {
			gemByName[name] = tracker.Item{ID: uint32(id), Name: name, Kind: tracker.KindAsh}
		}
	}
	for _, it := range gemByName {
		items = append(items, it)
	}

	return items, nil
}

func loadParam(params map[string][]byte, name string) (*paramdef, *paramTable, error) {
	def, err := loadDef(name)
	if err != nil {
		return nil, nil, fmt.Errorf("%s def: %w", name, err)
	}
	file := name
	if name == "MagicParam" { // the magic param's file is named "Magic" inside regulation
		file = "Magic"
	}
	raw, ok := params[file]
	if !ok {
		return nil, nil, fmt.Errorf("param %s not present in regulation", file)
	}
	pt, err := parseParam(raw)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", name, err)
	}
	if err := checkDef(def, pt); err != nil {
		return nil, nil, fmt.Errorf("%s: %w", name, err)
	}
	return def, pt, nil
}

func set(keys ...string) map[string]bool {
	if len(keys) == 0 {
		return nil
	}
	m := make(map[string]bool, len(keys))
	for _, k := range keys {
		m[k] = true
	}
	return m
}

func isShieldType(wt int) bool { return wt == 65 || wt == 67 || wt == 69 || wt == 90 }
func isAmmoType(wt int) bool   { return wt == 81 || wt == 83 || wt == 85 || wt == 86 }

// placeholderRx matches the "Type N" filler names Paramdex uses for unnamed/default
// param rows (e.g. the default body slots in EquipParamProtector).
var placeholderRx = regexp.MustCompile(`^Type \d+$`)

// named reports whether a param name is a real, player-obtainable item name: not
// empty, not a "Type N" placeholder, and not an "[NPC] …" row (NPCs wield separate
// param copies of many weapons/shields that the player can never own).
func named(name string) bool {
	return name != "" && !placeholderRx.MatchString(name) && !strings.HasPrefix(name, "[NPC]")
}

// goodsPrefixRx strips the leading "[Sorcery]"/"[Incantation]" tag Paramdex puts on
// spell goods names; levelSuffixRx matches the trailing " +N" of spirit-ash upgrade
// levels (which are separate goods rows we collapse to the base summon).
var (
	goodsPrefixRx = regexp.MustCompile(`^\[[^\]]*\]\s*`)
	levelSuffixRx = regexp.MustCompile(`\s*\+\d+$`)
)

func cleanGoodsName(s string) string { return strings.TrimSpace(goodsPrefixRx.ReplaceAllString(s, "")) }

func round1(f float64) float64 { return math.Round(f*10) / 10 }

// grade maps a scaling coefficient (% number from EquipParamWeapon.correct*) to its
// letter grade (approximate, matching the in-game S/A/B/C/D/E breakpoints).
func grade(c float64) string {
	switch {
	case c >= 175:
		return "S"
	case c >= 140:
		return "A"
	case c >= 90:
		return "B"
	case c >= 60:
		return "C"
	case c >= 25:
		return "D"
	case c > 0:
		return "E"
	}
	return ""
}

func scalingGrades(sc map[string]float64) map[string]string {
	out := map[string]string{}
	for stat, c := range sc {
		if g := grade(c); g != "" {
			out[stat] = g
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// primaryStat returns the stat with the highest value (>0), breaking ties by the
// given order. Used for weapons (scaling coefficients).
func primaryStat(order []string, vals map[string]float64) string {
	best, bestV := "", 0.0
	for _, s := range order {
		if vals[s] > bestV {
			bestV = vals[s]
			best = s
		}
	}
	return best
}

// primaryStatInt is the integer-requirement variant (used for spells: Int/Fai/Arc).
func primaryStatInt(vals map[string]int) string {
	best, bestV := "", 0
	for _, s := range []string{"Str", "Dex", "Int", "Fai", "Arc"} {
		if vals[s] > bestV {
			bestV = vals[s]
			best = s
		}
	}
	return best
}

// goodsKind classifies an EquipParamGoods.goodsType into a tracked kind. The bool
// reports whether it's a spell (needs MagicParam resolution); spirit ashes are not.
func goodsKind(gt int) (tracker.ItemKind, bool) {
	switch gt {
	case 5, 17: // Sorcery, Self Buff - Sorcery
		return tracker.KindSorcery, true
	case 16, 18: // Incantation, Self Buff - Incantation
		return tracker.KindIncantation, true
	case 7, 8: // Spirit Summon - Lesser / Greater
		return tracker.KindSpirit, false
	}
	return "", false
}
