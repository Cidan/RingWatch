package tracker

import "testing"

func flagsSet(ids ...uint32) Defeater {
	set := map[uint32]bool{}
	for _, id := range ids {
		set[id] = true
	}
	return func(id uint32) (bool, bool) { return set[id], true }
}

func ownsIDs(ids ...uint32) Owner {
	set := map[uint32]bool{}
	for _, id := range ids {
		set[id] = true
	}
	return func(it Item) bool { return set[it.ID] }
}

func sampleQuest() Quest {
	return Quest{
		ID: "q", Giver: "Tester", Region: "Limgrave",
		Steps: []QuestStep{
			{Title: "s0 talk"},                                     // no anchor
			{Title: "s1 item", Flag: 1000},                         // flag anchor
			{Title: "s2 optional", Optional: true, Flag: 2000},     // optional, flag anchor
			{Title: "s3 talk"},                                     // no anchor
			{Title: "s4 reward", Item: 5000, ItemKind: KindWeapon}, // inventory anchor
		},
	}
}

func currentTitle(q QuestStatus) string {
	if s, ok := q.CurrentStep(); ok {
		return s.Title
	}
	return ""
}

func TestQuestRollupNothingDone(t *testing.T) {
	q := computeQuest(sampleQuest(), flagsSet(), ownsIDs())
	if q.Done() != 0 || q.Total() != 4 { // 5 steps, 1 optional excluded from totals
		t.Errorf("done/total = %d/%d, want 0/4", q.Done(), q.Total())
	}
	if got := currentTitle(q); got != "s0 talk" {
		t.Errorf("current step = %q, want first step", got)
	}
	if !q.Trackable {
		t.Error("quest with flag and item anchors should be Trackable")
	}
}

func TestQuestRollupFromLaterFlag(t *testing.T) {
	// Only the s1 flag is set; the unanchored s0 before it must roll up to done,
	// and the current objective must advance to the next incomplete required step.
	q := computeQuest(sampleQuest(), flagsSet(1000), ownsIDs())
	if !q.Steps[0].Done || !q.Steps[1].Done {
		t.Error("s0 (unanchored) and s1 (flagged) should be done")
	}
	if q.Steps[4].Done {
		t.Error("s4 should not be done — no later anchor is set")
	}
	if got := currentTitle(q); got != "s3 talk" {
		t.Errorf("current step = %q, want s3 talk (optional s2 is skipped)", got)
	}
	if q.Done() != 2 {
		t.Errorf("required done = %d, want 2", q.Done())
	}
}

func TestQuestRollupInventoryCompletes(t *testing.T) {
	// Owning the final reward item completes the whole quest by roll-up.
	q := computeQuest(sampleQuest(), flagsSet(), ownsIDs(5000))
	if !q.Complete() {
		t.Errorf("owning the final reward should complete the quest; done/total = %d/%d", q.Done(), q.Total())
	}
	if _, ok := q.CurrentStep(); ok {
		t.Error("a complete quest should have no current step")
	}
}

func TestQuestUntrackable(t *testing.T) {
	q := computeQuest(Quest{Giver: "Guide", Steps: []QuestStep{{Title: "a"}, {Title: "b"}}}, flagsSet(), ownsIDs())
	if q.Trackable {
		t.Error("a quest with no anchors should not be Trackable")
	}
}

func TestComputeQuestsRealDataset(t *testing.T) {
	p := ComputeQuests(flagsSet(), ownsIDs())
	if len(p.Quests) < 30 {
		t.Fatalf("expected the full questline roster, got %d", len(p.Quests))
	}
	// Progression order: a base-game quest must precede the first DLC quest.
	firstDLC := -1
	for i, q := range p.Quests {
		if q.DLC {
			firstDLC = i
			break
		}
	}
	if firstDLC <= 0 {
		t.Fatalf("expected base quests before DLC quests, firstDLC=%d", firstDLC)
	}
	for _, q := range p.Quests[:firstDLC] {
		if q.DLC {
			t.Errorf("DLC quest %q sorted among base quests", q.Giver)
		}
	}
}
