package main

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/Cidan/RingWatch/internal/tracker"
)

// curatedLoc is one entry in the curated locations file (keyed by item name).
type curatedLoc struct {
	Region   string `json:"region"`
	Location string `json:"location"`
}

// dlcRegions are the Shadow of the Erdtree regions; an item found in one is DLC.
var dlcRegions = map[string]bool{
	"Gravesite Plain": true, "Scadu Altus": true, "Cerulean Coast": true,
	"Charo's Hidden Grave": true, "Scaduview": true, "Ancient Ruins of Rauh": true,
	"Rauh Base": true, "Abyssal Woods": true, "Jagged Peak": true, "Enir-Ilim": true,
	"Shadow Keep": true, "Belurat, Tower Settlement": true, "Land of Shadow": true,
}

// mergeLocations applies a curated name->{region,location} JSON onto the items,
// marking DLC items by their region. Returns the number of items matched.
func mergeLocations(items []tracker.Item, path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	var raw map[string]curatedLoc
	if err := json.Unmarshal(data, &raw); err != nil {
		return 0, err
	}
	byNorm := make(map[string]curatedLoc, len(raw))
	for name, loc := range raw {
		byNorm[normalizeName(name)] = loc
	}
	n := 0
	for i := range items {
		loc, ok := byNorm[normalizeName(items[i].Name)]
		if !ok {
			continue
		}
		items[i].Region = loc.Region
		items[i].Location = loc.Location
		if dlcRegions[loc.Region] {
			items[i].DLC = true
		}
		n++
	}
	return n, nil
}

// normalizeName lowercases, drops punctuation/apostrophes and a leading "the ", and
// collapses whitespace — so curated names join to in-game names despite small
// formatting differences.
func normalizeName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.TrimPrefix(s, "the ")
	var b strings.Builder
	lastSpace := false
	for _, r := range s {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			lastSpace = false
		case r == ' ' || r == '-' || r == '_':
			if !lastSpace {
				b.WriteByte(' ')
				lastSpace = true
			}
		}
	}
	return strings.TrimSpace(b.String())
}
