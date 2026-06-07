package tracker

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
)

// quests.generated.json is a hand-authored dataset of NPC questlines: each quest is
// a quest-giver with an ordered list of walkthrough steps (text sourced from the
// Fextralife wiki) plus an optional `complete` signal — a single high-confidence
// save condition (a "received the end reward" event flag, or ownership of the unique
// end-reward gear) that marks the quest complete. Schema is one flat array, mirroring
// bosses.generated.json.
//
//go:embed data/quests.generated.json
var questsJSON []byte

// Quest is one NPC questline.
//
// Progress is intentionally COARSE: a quest is reported complete only via Complete,
// a single high-confidence save signal (a "received the quest's end reward" event
// flag, or ownership of its unique end-reward gear). We deliberately do NOT claim
// per-step position — Elden Ring's intermediate "talk to X" beats are tracked (if at
// all) by flags that are generic, location-shared, or cleared as you progress, which
// no amount of validation against a handful of saves can reliably disambiguate.
// Claiming "you're on step 4 of 22" was wrong far more often than right, so the steps
// are presented as a walkthrough guide and only completion is asserted. Quests with
// no reliable completion signal (Complete == nil) are pure guides.
type Quest struct {
	ID           string       `json:"id"`                 // kebab-case slug, also the link key
	Giver        string       `json:"giver"`              // display name, e.g. "Ranni the Witch"
	Wiki         string       `json:"wiki,omitempty"`     // Fextralife page title; defaults to Giver
	Region       string       `json:"region"`             // region the quest begins in (drives ordering)
	DLC          bool         `json:"dlc"`                // Shadow of the Erdtree questline
	Summary      string       `json:"summary,omitempty"`  // one-line what + payoff
	Reward       string       `json:"reward,omitempty"`   // headline reward(s)
	CompleteWhen *QuestSignal `json:"complete,omitempty"` // completion detector; nil = untracked guide
	Steps        []QuestStep  `json:"steps"`
}

// QuestSignal is a save-derived condition: an event flag set, or ownership of a
// specific item. Used to detect quest completion.
type QuestSignal struct {
	Flag     uint32   `json:"flag,omitempty"`
	Item     uint32   `json:"item,omitempty"`
	ItemKind ItemKind `json:"item_kind,omitempty"`
}

// QuestStep is one ordered beat of a questline — walkthrough text (what to do /
// where to go) shown as a guide. Steps carry no per-step save state by design (see
// Quest); only quest-level completion is asserted.
type QuestStep struct {
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
