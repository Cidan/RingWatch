package ui

import (
	"strings"
	"testing"

	"github.com/Cidan/RingWatch/internal/config"
	"github.com/Cidan/RingWatch/internal/save"
	"github.com/Cidan/RingWatch/internal/tracker"
)

// TestItemsViewRenders renders the items section headlessly across each grouping to
// catch layout/panic regressions (no TTY needed).
func TestItemsViewRenders(t *testing.T) {
	m := newTestModel(t, 100, 30)
	m.section = sectionItems
	for _, g := range tracker.GroupByOrder {
		m.groupBy = g
		m.recomputeItems()
		v := strip(m.View().Content)
		if lipglossHeight(v) != 30 {
			t.Errorf("grouping %s: view height = %d, want 30", g.Label(), lipglossHeight(v))
		}
		if !strings.Contains(v, "Items") {
			t.Errorf("grouping %s: missing Items tab", g.Label())
		}
		if g == tracker.GroupRegion {
			t.Logf("items view (by %s):\n%s", g.Label(), v)
		}
	}
}

func lipglossHeight(s string) int { return strings.Count(s, "\n") + 1 }

// TestItemOwnershipLiveSave is the end-to-end check that owned item ids parsed from
// the real save actually resolve against the generated dataset, per category. It
// logs owned/total for each kind so a broken category (e.g. 0 owned despite the
// save holding some) is visible.
func TestItemOwnershipLiveSave(t *testing.T) {
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

	// Pick the character owning the most items (most representative for matching).
	best := act[0].Slot
	bestN := -1
	for _, c := range act {
		if o, ok := s.OwnedItems(c.Slot); ok && o.Total() > bestN {
			bestN, best = o.Total(), c.Slot
		}
	}
	owned, _ := s.OwnedItems(best)
	flags, _ := s.EventFlags(best)

	prog := tracker.ComputeItems(ownerFor(owned, flags), "", tracker.GroupKind)
	for _, g := range prog.Groups {
		t.Logf("%-14s %3d/%d owned", g.Name, g.Owned(), g.Total())
	}
	o, total := prog.Totals()
	t.Logf("TOTAL %d/%d owned", o, total)
	if o == 0 {
		t.Fatal("no items resolved as owned — matching is broken")
	}
	// Flask upgrades are tracked by pickup event flag, not inventory; a played save
	// should resolve at least one Golden Seed / Sacred Tear through that path.
	for _, g := range prog.Groups {
		if g.Name == "Flask Upgrades" && g.Owned() == 0 {
			t.Error("no flask upgrades resolved as collected — flag-based ownership is broken")
		}
	}
}
