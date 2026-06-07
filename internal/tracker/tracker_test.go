package tracker

import (
	"testing"

	"github.com/Cidan/RingWatch/internal/save"
)

const fixturePath = "../../repo/SavesTest/ER0000.sl2"

func TestDatasetLoads(t *testing.T) {
	bs := Bosses()
	if len(bs) < 200 {
		t.Fatalf("expected ~208 bosses, got %d", len(bs))
	}
	var base, dlc int
	for _, b := range bs {
		if b.DLC {
			dlc++
		} else {
			base++
		}
	}
	t.Logf("loaded %d bosses (%d base, %d DLC)", len(bs), base, dlc)
}

// TestAllFlagsAddressable is the critical guard: every boss flag must be
// reachable in the event-flags blob, or that boss can never show as defeated.
func TestAllFlagsAddressable(t *testing.T) {
	var bad []Boss
	for _, b := range Bosses() {
		if !save.FlagAddressable(b.EventID) {
			bad = append(bad, b)
		}
	}
	if len(bad) > 0 {
		for _, b := range bad {
			t.Errorf("unaddressable flag: %d %q (%s)", b.EventID, b.Name, b.Region)
		}
		t.Fatalf("%d/%d boss flags are not addressable", len(bad), len(Bosses()))
	}
}

func TestComputeShape(t *testing.T) {
	// All-undefeated baseline.
	p := Compute(func(uint32) (bool, bool) { return false, true })
	def, tot := p.Totals()
	if def != 0 {
		t.Errorf("expected 0 defeated, got %d", def)
	}
	if tot != len(Bosses()) {
		t.Errorf("total %d != dataset %d", tot, len(Bosses()))
	}
	if len(p.Regions) < 30 {
		t.Errorf("expected ~35 regions, got %d", len(p.Regions))
	}
	// Regions must be ordered: first region should be Limgrave (early game).
	if p.Regions[0].Name != "Limgrave" {
		t.Errorf("expected first region Limgrave, got %q", p.Regions[0].Name)
	}
}

// TestComputeWithFixture runs the real endgame character through Compute.
func TestComputeWithFixture(t *testing.T) {
	s, err := save.Open(fixturePath)
	if err != nil {
		t.Skipf("fixture unavailable: %v", err)
	}
	act := s.ActiveCharacters()
	if len(act) == 0 {
		t.Skip("no active characters")
	}
	slot := act[0].Slot // Davosso, endgame
	p := Compute(func(id uint32) (bool, bool) { return s.IsDefeated(slot, id) })
	def, tot := p.Totals()
	bd, bt, dd, dt := p.CategoryTotals()
	t.Logf("%s: %d/%d total  (base %d/%d, dlc %d/%d)", act[0].Name, def, tot, bd, bt, dd, dt)

	// Early main bosses must be defeated for any endgame character.
	for _, b := range []struct {
		id   uint32
		name string
	}{{10000850, "Margit"}, {10000800, "Godrick"}} {
		if d, ok := s.IsDefeated(slot, b.id); !ok || !d {
			t.Errorf("%s expected defeated (defeated=%v known=%v)", b.name, d, ok)
		}
	}
	// Sanity range: progressed well into the game but not necessarily 100%.
	if def < 30 || def > tot {
		t.Errorf("defeated count %d outside plausible range (total %d)", def, tot)
	}
}
