// Package save reads Elden Ring PC save files (ER0000.sl2 / ER0000.co2).
//
// These files are plaintext BND4 containers (NOT encrypted — that is a Dark Souls
// trait that Elden Ring dropped); the only integrity mechanism is a per-section
// MD5 checksum. The package extracts the character list and each character's
// event-flags blob, from which boss-defeat state is derived.
package save

import (
	"fmt"
	"os"
)

const (
	magic      = "BND4"
	slotStride = 0x280010 // distance between consecutive character slots
	slot0Body  = 0x310    // file offset of slot 0's body (after magic+header+checksum)
	slotBody   = 0x280000 // body length of a character slot
	numSlots   = 10
)

// Character is one save slot's summary plus its parsed event flags.
type Character struct {
	Slot    int
	Name    string
	Level   int
	Seconds int // seconds played
	Active  bool

	eventFlags []byte      // populated for active slots; nil otherwise
	ownedItems *OwnedItems // populated for active slots; nil otherwise
}

// Playtime renders seconds played as a compact "Xh Ym" string.
func (c Character) Playtime() string {
	h := c.Seconds / 3600
	m := (c.Seconds % 3600) / 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}

// Save is a parsed save file.
type Save struct {
	Path       string
	Characters []Character
}

// Open reads and parses a save file from disk.
func Open(path string) (*Save, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(path, data)
}

// Parse parses save bytes already in memory.
func Parse(path string, data []byte) (*Save, error) {
	if len(data) < 4 || string(data[0:4]) != magic {
		return nil, fmt.Errorf("save: not a PC BND4 save (bad magic in %s)", path)
	}
	chars, err := parseCharacters(data)
	if err != nil {
		return nil, fmt.Errorf("save: reading profiles: %w", err)
	}
	for i := range chars {
		if !chars[i].Active {
			continue
		}
		bodyStart := slot0Body + chars[i].Slot*slotStride
		if bodyStart+slotBody > len(data) {
			return nil, fmt.Errorf("save: slot %d body out of range", chars[i].Slot)
		}
		sd, err := parseSlot(data[bodyStart : bodyStart+slotBody])
		if err != nil {
			return nil, fmt.Errorf("slot %d (%q): %w", chars[i].Slot, chars[i].Name, err)
		}
		chars[i].eventFlags = sd.eventFlags
		chars[i].ownedItems = sd.owned
	}
	return &Save{Path: path, Characters: chars}, nil
}

// Character returns the character in the given slot, or nil.
func (s *Save) Character(slot int) *Character {
	for i := range s.Characters {
		if s.Characters[i].Slot == slot {
			return &s.Characters[i]
		}
	}
	return nil
}

// ActiveCharacters returns only the occupied slots.
func (s *Save) ActiveCharacters() []Character {
	var out []Character
	for _, c := range s.Characters {
		if c.Active {
			out = append(out, c)
		}
	}
	return out
}

// IsDefeated reports whether the boss with the given event flag is defeated for
// the character in slot. ok is false if the slot is inactive/unparsed or the
// flag is not addressable.
func (s *Save) IsDefeated(slot int, eventID uint32) (defeated, ok bool) {
	c := s.Character(slot)
	if c == nil || c.eventFlags == nil {
		return false, false
	}
	return flagSet(c.eventFlags, eventID)
}

// EventFlags returns a slot's raw event-flags blob for batch queries.
func (s *Save) EventFlags(slot int) ([]byte, bool) {
	c := s.Character(slot)
	if c == nil || c.eventFlags == nil {
		return nil, false
	}
	return c.eventFlags, true
}

// OwnedItems returns a slot's parsed owned-item sets, or ok=false if the slot is
// inactive or was not parsed.
func (s *Save) OwnedItems(slot int) (*OwnedItems, bool) {
	c := s.Character(slot)
	if c == nil || c.ownedItems == nil {
		return nil, false
	}
	return c.ownedItems, true
}

// FlagSet reports whether an event flag is set in a raw event-flags blob.
// Exposed so callers holding an EventFlags slice can query without a *Save.
func FlagSet(eventFlags []byte, eventID uint32) (set, ok bool) {
	return flagSet(eventFlags, eventID)
}
