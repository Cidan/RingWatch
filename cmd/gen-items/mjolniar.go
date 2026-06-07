package main

import (
	"embed"
	"encoding/json"
	"strings"

	"github.com/Cidan/RingWatch/internal/tracker"
)

// fextra-overrides.json is the MIT-licensed, Fextralife-derived item→location prose
// dataset from Mjolniar/elden-ring-index-build-planner (Copyright (c) 2025, MIT).
// Vendored as the baseline location source; we infer a canonical region from the prose.
//
//go:embed locations/fextra-overrides.json
var mjolniarFS embed.FS

type mjEntry struct {
	ItemName     string `json:"itemName"`
	LocationName string `json:"locationName"`
}

// applyMjolniarBaseline fills location/region on items (by normalized name) from the
// vendored Mjolniar prose, without overwriting anything a higher-priority curated
// source already set. Returns the number of items matched.
func applyMjolniarBaseline(items []tracker.Item) (int, error) {
	data, err := mjolniarFS.ReadFile("locations/fextra-overrides.json")
	if err != nil {
		return 0, err
	}
	var entries []mjEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return 0, err
	}
	byNorm := make(map[string]mjEntry, len(entries))
	for _, e := range entries {
		if e.ItemName != "" {
			byNorm[normalizeName(e.ItemName)] = e
		}
	}
	n := 0
	for i := range items {
		if items[i].Region != "" || items[i].Location != "" {
			continue
		}
		e, ok := byNorm[normalizeName(items[i].Name)]
		if !ok {
			continue
		}
		items[i].Location = e.LocationName
		items[i].Region = regionFromProse(e.LocationName)
		if dlcRegions[items[i].Region] {
			items[i].DLC = true
		}
		n++
	}
	return n, nil
}

// regionProse maps a lowercased prose substring to a canonical region. Checked in
// order — most specific / DLC first — so e.g. "Scadu Altus" wins over a bare "scadu".
var regionProse = []struct{ sub, region string }{
	{"enir-ilim", "Enir-Ilim"}, {"enir ilim", "Enir-Ilim"},
	{"jagged peak", "Jagged Peak"},
	{"abyssal wood", "Abyssal Woods"},
	{"ancient ruins of rauh", "Ancient Ruins of Rauh"}, {"rauh base", "Rauh Base"}, {"rauh", "Ancient Ruins of Rauh"},
	{"scaduview", "Scaduview"},
	{"charo", "Charo's Hidden Grave"},
	{"cerulean coast", "Cerulean Coast"},
	{"shadow keep", "Shadow Keep"}, {"specimen storehouse", "Shadow Keep"},
	{"belurat", "Belurat, Tower Settlement"},
	{"scadu altus", "Scadu Altus"}, {"scadu", "Scadu Altus"},
	{"gravesite plain", "Gravesite Plain"}, {"gravesite", "Gravesite Plain"},
	{"stormveil", "Stormveil Castle"},
	{"raya lucaria", "Academy of Raya Lucaria"}, {"academy", "Academy of Raya Lucaria"},
	{"weeping peninsula", "Weeping Peninsula"}, {"weeping", "Weeping Peninsula"},
	{"dragonbarrow", "Dragonbarrow"},
	{"caelid", "Caelid"},
	{"ruin-strewn precipice", "Liurnia of the Lakes"}, {"liurnia", "Liurnia of the Lakes"},
	{"volcano manor", "Mt. Gelmir"}, {"gelmir", "Mt. Gelmir"},
	{"capital outskirts", "Capital Outskirts"},
	{"subterranean shunning", "Leyndell, Royal Capital"}, {"shunning-grounds", "Leyndell, Royal Capital"},
	{"ashen capital", "Leyndell, Ashen Capital"},
	{"leyndell", "Leyndell, Royal Capital"}, {"royal capital", "Leyndell, Royal Capital"},
	{"altus plateau", "Altus Plateau"}, {"altus", "Altus Plateau"},
	{"nokron", "Nokron, Eternal City"},
	{"siofra", "Siofra River"},
	{"ainsel", "Ainsel River"},
	{"deeproot", "Deeproot Depths"},
	{"lake of rot", "Lake of Rot"},
	{"mohgwyn", "Mohgwyn Dynasty Mausoleum"},
	{"moonlight altar", "Moonlight Altar"},
	{"consecrated snowfield", "Consecrated Snowfield"},
	{"mountaintops", "Mountaintops of the Giants"}, {"forge of the giants", "Mountaintops of the Giants"},
	{"forbidden lands", "Forbidden Lands"},
	{"farum azula", "Crumbling Farum Azula"},
	{"haligtree", "Miquella's Haligtree"}, {"elphael", "Miquella's Haligtree"},
	{"limgrave", "Limgrave"},
	{"roundtable", "Roundtable Hold"}, {"twin maiden", "Roundtable Hold"},
}

func regionFromProse(prose string) string {
	p := strings.ToLower(prose)
	for _, m := range regionProse {
		if strings.Contains(p, m.sub) {
			return m.region
		}
	}
	return ""
}
