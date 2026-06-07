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
	if trackable < 15 || trackable >= total {
		t.Errorf("expected a substantial trackable subset (< total), got %d/%d", trackable, total)
	}

	// Guides (no completion signal) must never be Trackable, and there must be some.
	guides := 0
	for _, q := range p.Quests {
		if q.CompleteWhen == nil && q.Trackable {
			t.Errorf("quest %q has no signal but is marked Trackable", q.Giver)
		}
		if !q.Trackable {
			guides++
		}
	}
	if guides == 0 {
		t.Error("expected some reference-guide quests")
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
