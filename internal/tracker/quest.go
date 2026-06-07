package tracker

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
)

// quests.generated.json is a hand-authored dataset of NPC questlines: each quest
// is a quest-giver with an ordered list of steps. Step text (what to do / where to
// go) is sourced from the Fextralife wiki; each step's completion is keyed to an
// Elden Ring event flag in the same id space as boss-defeat flags — quest-item
// receipts (from the item-randomizer's reverse-engineered EMEVD), boss defeats, and
// known progression flags. Steps without a clean dedicated flag carry Flag == 0 and
// resolve by roll-up (see questprogress.go). Schema is one flat array, mirroring
// bosses.generated.json.
//
//go:embed data/quests.generated.json
var questsJSON []byte

// Quest is one NPC questline.
type Quest struct {
	ID      string      `json:"id"`                // kebab-case slug, also the link key
	Giver   string      `json:"giver"`             // display name, e.g. "Ranni the Witch"
	Wiki    string      `json:"wiki,omitempty"`    // Fextralife page title; defaults to Giver
	Region  string      `json:"region"`            // region the quest begins in (drives ordering)
	DLC     bool        `json:"dlc"`               // Shadow of the Erdtree questline
	Summary string      `json:"summary,omitempty"` // one-line what + payoff
	Reward  string      `json:"reward,omitempty"`  // headline reward(s)
	Steps   []QuestStep `json:"steps"`
}

// QuestStep is one ordered beat of a questline.
//
// Completion is detected from the save by Flag: an NPC quest-progression event flag
// that the game sets when the step is reached/completed, in the same id space as
// boss-defeat flags. These are sourced from empirical save-diffs and validated so
// that they are durable (stay set) and monotonic across known progress — see
// questprogress.go. Flag == 0 means the step has no dedicated reliable signal and is
// resolved by roll-up from a later anchored step.
//
// (Earlier versions also anchored on inventory ownership of a reward item and on
// boss defeats; both were dropped because owning an item or killing a boss is not
// gated by the quest — e.g. a world-pickup reward would falsely complete the quest.)
type QuestStep struct {
	Flag     uint32 `json:"flag,omitempty"`
	Title    string `json:"title"`
	Detail   string `json:"detail,omitempty"`   // what to do / where to go
	Location string `json:"location,omitempty"` // place name (annotation + link)
	Optional bool   `json:"optional,omitempty"`
	Warning  string `json:"warning,omitempty"` // missable / point-of-no-return note
}

// PageTitle is the Fextralife page title for the quest-giver (Wiki override, else
// the giver name).
func (q Quest) PageTitle() string {
	if q.Wiki != "" {
		return q.Wiki
	}
	return q.Giver
}

var allQuests = loadQuests()

func loadQuests() []Quest {
	var qs []Quest
	if err := json.Unmarshal(questsJSON, &qs); err != nil {
		panic(fmt.Sprintf("tracker: invalid embedded quest data: %v", err))
	}
	return qs
}

// Quests returns a copy of the full quest dataset.
func Quests() []Quest {
	out := make([]Quest, len(allQuests))
	copy(out, allQuests)
	return out
}

// baseRegionPhase maps each base-game region to its game phase, computed once by
// carrying the running phase forward across baseRegionOrder (same rule orderRegions
// uses for the bosses view). Lets a quest resolve its phase from its single region.
var baseRegionPhase = func() map[string]string {
	m := make(map[string]string, len(baseRegionOrder))
	phase := PhaseEarly
	for _, r := range baseRegionOrder {
		if p, ok := phaseAnchors[r]; ok {
			phase = p
		}
		m[r] = phase
	}
	return m
}()

// Quest-only hubs that aren't boss regions, with the progression slot and phase
// they sort into. The Roundtable Hold is reached around the Liurnia/mid stretch and
// is where several quest-givers first appear.
var (
	questHubRank  = map[string]int{"Roundtable Hold": 3} // alongside Liurnia of the Lakes
	questHubPhase = map[string]string{"Roundtable Hold": PhaseMid}
)

// regionPhase returns the display phase for a quest's starting region.
func regionPhase(region string, dlc bool) string {
	if dlc {
		return PhaseDLC
	}
	if p, ok := questHubPhase[region]; ok {
		return p
	}
	if p, ok := baseRegionPhase[region]; ok {
		return p
	}
	return PhaseEarly
}

// questRank ranks a quest's starting region into progression order, honoring the
// quest-only hubs before falling back to the boss-region ranking.
func questRank(region string, dlc bool) int {
	if r, ok := questHubRank[region]; ok && !dlc {
		return r
	}
	return regionRank(region, dlc)
}

// questMeta is a quest tagged with its resolved phase, in progression order.
type questMeta struct {
	Quest
	Phase string
}

// orderQuests sorts quests into progression order (by starting-region rank, then
// giver name) and tags each with its game phase.
func orderQuests(quests []Quest) []questMeta {
	metas := make([]questMeta, 0, len(quests))
	for _, q := range quests {
		metas = append(metas, questMeta{Quest: q, Phase: regionPhase(q.Region, q.DLC)})
	}
	sort.SliceStable(metas, func(i, j int) bool {
		ri, rj := questRank(metas[i].Region, metas[i].DLC), questRank(metas[j].Region, metas[j].DLC)
		if ri != rj {
			return ri < rj
		}
		return metas[i].Giver < metas[j].Giver
	})
	return metas
}
