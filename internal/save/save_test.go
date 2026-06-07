package save

import (
	"os"
	"testing"

	"github.com/Cidan/RingWatch/internal/config"
)

// The Tarnished Chronicle ships a real (unencrypted) save fixture we can verify
// against. It lives in the cloned upstream repo next to this project.
const testSavePath = "../../repo/SavesTest/ER0000.sl2"

func TestBSTLoaded(t *testing.T) {
	if n := BSTSize(); n < 11000 {
		t.Fatalf("BST map too small: %d", n)
	}
}

func TestParseFixture(t *testing.T) {
	if _, err := os.Stat(testSavePath); err != nil {
		t.Skipf("fixture not present: %v", err)
	}
	s, err := Open(testSavePath)
	if err != nil {
		// Open validates the event-flags terminator, so this fails loudly if the
		// slot walk is misaligned.
		t.Fatalf("Open: %v", err)
	}

	active := s.ActiveCharacters()
	if len(active) == 0 {
		t.Fatal("expected at least one active character")
	}
	for _, c := range active {
		t.Logf("slot %d: %q  level %d  playtime %s", c.Slot, c.Name, c.Level, c.Playtime())
	}

	// Spot-check well-known boss flags on the first active (endgame) character.
	slot := active[0].Slot
	for _, b := range []struct {
		name string
		id   uint32
	}{
		{"Margit, the Fell Omen", 10000850},
		{"Godrick the Grafted", 10000800},
	} {
		def, ok := s.IsDefeated(slot, b.id)
		if !ok {
			t.Errorf("flag for %s (%d) not addressable", b.name, b.id)
			continue
		}
		t.Logf("slot %d  %-24s defeated=%v", slot, b.name, def)
		if !def {
			t.Errorf("expected %s to be defeated for endgame character in slot %d", b.name, slot)
		}
	}
}

// TestParseLiveSave parses whatever real save is auto-detected on this machine —
// typically a different save-format version than the bundled fixture. Skips when
// no live save is present.
func TestParseLiveSave(t *testing.T) {
	saves := config.DetectSaves()
	if len(saves) == 0 {
		t.Skip("no live save detected")
	}
	s, err := Open(saves[0].Path)
	if err != nil {
		t.Fatalf("Open live save: %v", err)
	}
	act := s.ActiveCharacters()
	if len(act) == 0 {
		t.Fatal("no active characters in live save")
	}
	for _, c := range act {
		t.Logf("slot %d: %q  level %d  playtime %s", c.Slot, c.Name, c.Level, c.Playtime())
	}
}
