package ui

import (
	"strings"
	"testing"

	"github.com/Cidan/RingWatch/internal/config"
	"github.com/Cidan/RingWatch/internal/save"
	"github.com/Cidan/RingWatch/internal/tracker"
)

// TestQuestsViewRenders renders the quests section headlessly to catch layout/panic
// regressions, on the giver list and after focusing a giver's steps.
func TestQuestsViewRenders(t *testing.T) {
	m := newTestModel(t, 110, 32)
	m.section = sectionQuests

	v := strip(m.View().Content)
	if lipglossHeight(v) != 32 {
		t.Errorf("view height = %d, want 32", lipglossHeight(v))
	}
	for _, want := range []string{"Quests", "EARLY GAME"} {
		if !strings.Contains(v, want) {
			t.Errorf("quests view missing %q", want)
		}
	}

	// Focus the steps pane; the selected giver's steps should render.
	m.focus = focusBosses
	v = strip(m.View().Content)
	if lipglossHeight(v) != 32 {
		t.Errorf("steps view height = %d, want 32", lipglossHeight(v))
	}
	t.Logf("\n%s", v)
}

// TestQuestsViewSearch filters the giver list by NPC name.
func TestQuestsViewSearch(t *testing.T) {
	m := newTestModel(t, 110, 32)
	m.section = sectionQuests
	m.filter = "ranni"
	qs := m.visibleQuests()
	if len(qs) == 0 {
		t.Fatal("search for 'ranni' returned no quest-givers")
	}
	for _, q := range qs {
		if !strings.Contains(strings.ToLower(q.Giver), "ranni") {
			t.Errorf("search returned non-matching giver %q", q.Giver)
		}
	}
}

// TestQuestProgressLiveSave is the end-to-end check that quest anchors (event flags
// + owned items) resolve against a real save, logging per-quest progress so a broken
// resolver is visible.
func TestQuestProgressLiveSave(t *testing.T) {
	saves := config.DetectSaves()
	if len(saves) == 0 {
		t.Skip("no live save detected")
	}
	s, err := save.Open(saves[0].Path)
	if err != nil {
		t.Fatalf("open live save: %v", err)
	}
	act := s.ActiveCharacters()
	if len(act) == 0 {
		t.Skip("no active characters")
	}
	slot := act[0].Slot
	flagReader := func(id uint32) (bool, bool) { return s.IsDefeated(slot, id) }
	owned, _ := s.OwnedItems(slot)
	flags, _ := s.EventFlags(slot)

	p := tracker.ComputeQuests(flagReader, ownerFor(owned, flags))
	for _, q := range p.Quests {
		if q.Complete {
			t.Logf("✓ %s", q.Giver)
		}
	}
	complete, trackable, total := p.Counts()
	t.Logf("TOTAL %d/%d quests complete (%d tracked, %d total incl. guides)", complete, trackable, trackable, total)
}
