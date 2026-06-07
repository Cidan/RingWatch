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

func TestQuestSignalFlag(t *testing.T) {
	sig := &QuestSignal{Flag: 1000}
	if !sig.satisfied(flagsSet(1000), nil) {
		t.Error("set flag should satisfy the signal")
	}
	if sig.satisfied(flagsSet(), nil) {
		t.Error("unset flag should not satisfy the signal")
	}
}

func TestQuestSignalItem(t *testing.T) {
	sig := &QuestSignal{Item: 5000, ItemKind: KindWeapon}
	if !sig.satisfied(nil, ownsIDs(5000)) {
		t.Error("owned item should satisfy the signal")
	}
	if sig.satisfied(nil, ownsIDs(1)) {
		t.Error("unowned item should not satisfy the signal")
	}
}

func TestComputeQuestsCoarse(t *testing.T) {
	p := ComputeQuests(flagsSet(), ownsIDs())
	if len(p.Quests) < 30 {
		t.Fatalf("expected the full questline roster, got %d", len(p.Quests))
	}

	complete, trackable, total := p.Counts()
	if complete != 0 {
		t.Errorf("with no flags/items set, no quest should be complete, got %d", complete)
	}
	// Auto-detection is intentionally OFF for now: every quest is a reference guide
	// (no completion signal). If signals are re-added, update this expectation.
	if trackable != 0 {
		t.Errorf("expected all quests to be guides (0 trackable), got %d/%d", trackable, total)
	}
	for _, q := range p.Quests {
		if q.Trackable || q.CompleteWhen != nil {
			t.Errorf("quest %q should be an untracked guide", q.Giver)
		}
	}

	// Progression order: base quests precede the first DLC quest.
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
