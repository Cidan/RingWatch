package main

import (
	"embed"
	"encoding/json"
	"fmt"

	"github.com/Cidan/RingWatch/internal/tracker"
)

// Per-kind curated location files (one name->{region,location} object each),
// agent-curated from the wikis against the exact in-game names. These are the
// primary location source and take priority over the Mjolniar baseline.
//
//go:embed curated/*.json
var curatedFS embed.FS

var curatedKindFile = map[tracker.ItemKind]string{
	tracker.KindWeapon:      "weapon",
	tracker.KindShield:      "shield",
	tracker.KindArmor:       "armor",
	tracker.KindSorcery:     "sorcery",
	tracker.KindIncantation: "incantation",
	tracker.KindTalisman:    "talisman",
	tracker.KindAsh:         "ash",
}

// applyCuratedLocations sets region/location from the vendored per-kind files,
// matched by kind + normalized name — so items that share a name across kinds (e.g.
// the Glintstone Pebble sorcery vs the Glintstone Pebble ash of war) get their own
// locations. Returns the number of items matched.
func applyCuratedLocations(items []tracker.Item) (int, error) {
	byKind := map[tracker.ItemKind]map[string]curatedLoc{}
	for kind, fname := range curatedKindFile {
		data, err := curatedFS.ReadFile("curated/" + fname + ".json")
		if err != nil {
			continue
		}
		var m map[string]curatedLoc
		if err := json.Unmarshal(data, &m); err != nil {
			return 0, fmt.Errorf("curated/%s.json: %w", fname, err)
		}
		norm := make(map[string]curatedLoc, len(m))
		for name, loc := range m {
			norm[normalizeName(name)] = loc
		}
		byKind[kind] = norm
	}
	n := 0
	for i := range items {
		m, ok := byKind[items[i].Kind]
		if !ok {
			continue
		}
		loc, ok := m[normalizeName(items[i].Name)]
		if !ok || (loc.Region == "" && loc.Location == "") {
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
