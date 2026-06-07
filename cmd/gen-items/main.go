// Command gen-items generates internal/tracker/data/items.generated.json from the
// game's regulation.bin. It decrypts and decompresses the regulation, parses the
// weapon/protector/accessory/goods/magic/gem params (field offsets driven by the
// vendored Paramdex defs), joins English names from the vendored Paramdex name
// tables, and optionally merges a curated locations file. Pure Go; run from the
// repo root:
//
//	go run ./cmd/gen-items                              # uses the default game path
//	go run ./cmd/gen-items -regulation /path/to/regulation.bin -locations loc.json
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/Cidan/RingWatch/internal/tracker"
)

const defaultRegulation = "/home/antonio/Data/SteamLibrary/steamapps/common/ELDEN RING/Game/regulation.bin"

func main() {
	reg := flag.String("regulation", defaultRegulation, "path to the game's regulation.bin")
	out := flag.String("out", "internal/tracker/data/items.generated.json", "output JSON path")
	locPath := flag.String("locations", "", "optional curated locations JSON (name -> {region,location}) to merge")
	flag.Parse()

	if err := run(*reg, *out, *locPath); err != nil {
		fmt.Fprintln(os.Stderr, "gen-items:", err)
		os.Exit(1)
	}
}

func run(reg, out, locPath string) error {
	raw, err := os.ReadFile(reg)
	if err != nil {
		return err
	}
	params, err := readRegulation(raw)
	if err != nil {
		return err
	}
	names, err := loadNames()
	if err != nil {
		return err
	}
	items, err := buildItems(params, names)
	if err != nil {
		return err
	}

	cur, err := applyCuratedLocations(items)
	if err != nil {
		return fmt.Errorf("curated locations: %w", err)
	}
	base, err := applyMjolniarBaseline(items)
	if err != nil {
		return fmt.Errorf("mjolniar baseline: %w", err)
	}
	if locPath != "" {
		n, err := mergeLocations(items, locPath)
		if err != nil {
			return fmt.Errorf("locations override: %w", err)
		}
		fmt.Printf("applied %d location overrides from %s\n", n, locPath)
	}
	fmt.Printf("locations: %d curated + %d Mjolniar baseline\n", cur, base)

	sortItems(items)
	if err := writeItems(out, items); err != nil {
		return err
	}
	printSummary(items, out)
	return nil
}

func sortItems(items []tracker.Item) {
	rank := map[tracker.ItemKind]int{}
	for i, k := range tracker.KindOrder {
		rank[k] = i
	}
	sort.SliceStable(items, func(i, j int) bool {
		if rank[items[i].Kind] != rank[items[j].Kind] {
			return rank[items[i].Kind] < rank[items[j].Kind]
		}
		if items[i].Name != items[j].Name {
			return items[i].Name < items[j].Name
		}
		return items[i].ID < items[j].ID
	})
}

func writeItems(path string, items []tracker.Item) error {
	b, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}

func printSummary(items []tracker.Item, out string) {
	byKind := map[tracker.ItemKind]int{}
	withRegion := 0
	for _, it := range items {
		byKind[it.Kind]++
		if it.Region != "" {
			withRegion++
		}
	}
	fmt.Printf("wrote %d items -> %s\n", len(items), out)
	for _, k := range tracker.KindOrder {
		fmt.Printf("  %-13s %d\n", tracker.KindLabel[k], byKind[k])
	}
	fmt.Printf("  with location: %d/%d\n", withRegion, len(items))
}
