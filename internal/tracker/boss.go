// Package tracker holds the boss dataset and computes completion progress from a
// character's event flags. It has no save-format or UI dependencies, so the same
// shape can back an item tracker later (a sibling dataset + progress type).
package tracker

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
)

// bosses.generated.json was assembled from The Tarnished Chronicle's own dataset
// (recovered from its compiled resources) cross-checked against several
// independent machine-readable game-data sources. Schema is one flat array.
//
//go:embed data/bosses.generated.json
var bossesJSON []byte

// Boss is one trackable boss (anything with an on-screen health bar).
type Boss struct {
	EventID  uint32 `json:"event_id"` // defeat event flag
	Name     string `json:"name"`
	Region   string `json:"region"`
	DLC      bool   `json:"dlc"`
	Optional bool   `json:"optional"` // false = required for main progression
	Notes    string `json:"notes,omitempty"`
}

var allBosses = loadBosses()

func loadBosses() []Boss {
	var bs []Boss
	if err := json.Unmarshal(bossesJSON, &bs); err != nil {
		panic(fmt.Sprintf("tracker: invalid embedded boss data: %v", err))
	}
	return bs
}

// Bosses returns a copy of the full boss dataset.
func Bosses() []Boss {
	out := make([]Boss, len(allBosses))
	copy(out, allBosses)
	return out
}

// Game-phase identifiers, in display order.
const (
	PhaseEarly = "early"
	PhaseMid   = "mid"
	PhaseLate  = "late"
	PhaseDLC   = "dlc"
)

// PhaseOrder is the order phases are presented in.
var PhaseOrder = []string{PhaseEarly, PhaseMid, PhaseLate, PhaseDLC}

// PhaseLabel maps a phase id to its display heading.
var PhaseLabel = map[string]string{
	PhaseEarly: "Early Game · Levels 1–40",
	PhaseMid:   "Mid Game · Levels 40–90",
	PhaseLate:  "Late Game · Levels 100+",
	PhaseDLC:   "Shadow of the Erdtree",
}

// baseRegionOrder is the base-game progression order (from the upstream
// LOCATION_PROGRESSION_ORDER). phaseAnchors below switch the heading as the list
// crosses into a new game phase.
var baseRegionOrder = []string{
	"Limgrave",
	"Weeping Peninsula",
	"Stormveil Castle",
	"Liurnia of the Lakes",
	"Academy of Raya Lucaria",
	"Siofra River",
	"Ainsel River",
	"Caelid",
	"Altus Plateau",
	"Dragonbarrow",
	"Mt. Gelmir",
	"Capital Outskirts",
	"Nokron, Eternal City",
	"Deeproot Depths",
	"Lake of Rot",
	"Leyndell, Royal Capital",
	"Forbidden Lands",
	"Mountaintops of the Giants",
	"Consecrated Snowfield",
	"Moonlight Altar",
	"Mohgwyn Dynasty Mausoleum",
	"Crumbling Farum Azula",
	"Miquella's Haligtree",
	"Leyndell, Ashen Capital",
	"Elden Throne",
}

// dlcRegionOrder is the Shadow of the Erdtree progression order.
var dlcRegionOrder = []string{
	"Gravesite Plain",
	"Scadu Altus",
	"Cerulean Coast",
	"Charo's Hidden Grave",
	"Scaduview",
	"Ancient Ruins of Rauh",
	"Rauh Base",
	"Abyssal Woods",
	"Jagged Peak",
	"Enir-Ilim",
}

// phaseAnchors switches the running phase when the ordered region list reaches
// the named region.
var phaseAnchors = map[string]string{
	"Limgrave":             PhaseEarly,
	"Liurnia of the Lakes": PhaseMid,
	"Forbidden Lands":      PhaseLate,
}

// regionRank returns a sort key placing base regions first (in progression
// order), then DLC regions (in progression order), then any unlisted region
// alphabetically at the end.
func regionRank(region string, dlc bool) int {
	for i, r := range baseRegionOrder {
		if r == region {
			return i
		}
	}
	for i, r := range dlcRegionOrder {
		if r == region {
			return 1000 + i
		}
	}
	if dlc {
		return 3000
	}
	return 2000
}

// orderedRegionNames returns the regions present in the dataset, sorted into
// progression order, each tagged with its phase and DLC flag.
type regionMeta struct {
	Name  string
	DLC   bool
	Phase string
}

func orderRegions(bosses []Boss) []regionMeta {
	seen := map[string]bool{}
	var metas []regionMeta
	for _, b := range bosses {
		if seen[b.Region] {
			continue
		}
		seen[b.Region] = true
		metas = append(metas, regionMeta{Name: b.Region, DLC: b.DLC})
	}
	sort.SliceStable(metas, func(i, j int) bool {
		ri, rj := regionRank(metas[i].Name, metas[i].DLC), regionRank(metas[j].Name, metas[j].DLC)
		if ri != rj {
			return ri < rj
		}
		return metas[i].Name < metas[j].Name
	})
	// Assign phases by carrying the running phase forward across the ordered list.
	phase := PhaseEarly
	for i := range metas {
		switch {
		case metas[i].DLC:
			phase = PhaseDLC
		default:
			if p, ok := phaseAnchors[metas[i].Name]; ok {
				phase = p
			}
		}
		metas[i].Phase = phase
	}
	return metas
}
