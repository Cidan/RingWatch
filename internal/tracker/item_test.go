package tracker

import "testing"

func sampleItems() []Item {
	return []Item{
		{ID: 1010000, Name: "Longsword", Kind: KindWeapon, WeaponType: "Straight Sword", PrimaryStat: "Str", Region: "Limgrave"},
		{ID: 16170000, Name: "Rivers of Blood", Kind: KindWeapon, WeaponType: "Katana", PrimaryStat: "Arc", Region: "Mountaintops of the Giants"},
		{ID: 50000, Name: "Knight Helm", Kind: KindArmor, ArmorSlot: "Head", Region: "Limgrave"},
		{ID: 4000, Name: "Glintstone Pebble", Kind: KindSorcery, PrimaryStat: "Int", Region: "Academy of Raya Lucaria"},
		{ID: 1070, Name: "Radagon's Soreseal", Kind: KindTalisman, Region: ""},
		{ID: 18120000, Name: "Backhand Blade", Kind: KindWeapon, WeaponType: "Backhand Blade", PrimaryStat: "Dex", Region: "Gravesite Plain", DLC: true},
	}
}

func ownerOf(ids ...uint32) Owner {
	set := map[uint32]bool{}
	for _, id := range ids {
		set[id] = true
	}
	return func(it Item) bool { return set[it.ID] }
}

func groupNames(p ItemProgress) []string {
	ns := make([]string, 0, len(p.Groups))
	for _, g := range p.Groups {
		ns = append(ns, g.Name)
	}
	return ns
}

func indexOf(ss []string, want string) int {
	for i, s := range ss {
		if s == want {
			return i
		}
	}
	return -1
}

func TestComputeItemsByRegion(t *testing.T) {
	p := computeItemsFrom(sampleItems(), ownerOf(1010000, 4000), "", GroupRegion)
	names := groupNames(p)

	if names[0] != "Limgrave" {
		t.Errorf("expected Limgrave first, got %v", names)
	}
	if names[len(names)-1] != "Unknown" {
		t.Errorf("expected Unknown last, got %v", names)
	}
	if a, g := indexOf(names, "Academy of Raya Lucaria"), indexOf(names, "Gravesite Plain"); a < 0 || g < 0 || a > g {
		t.Errorf("expected base region before DLC region, got %v", names)
	}
	if owned, total := p.Totals(); owned != 2 || total != 6 {
		t.Errorf("totals owned/total = %d/%d, want 2/6", owned, total)
	}
	// Limgrave holds Longsword (owned) + Knight Helm (not) = 1/2.
	lim := p.Groups[0]
	if lim.Owned() != 1 || lim.Total() != 2 {
		t.Errorf("Limgrave owned/total = %d/%d, want 1/2", lim.Owned(), lim.Total())
	}
}

func TestComputeItemsByKind(t *testing.T) {
	p := computeItemsFrom(sampleItems(), nil, "", GroupKind)
	got := groupNames(p)
	want := []string{"Weapons", "Armor", "Sorceries", "Talismans"}
	if len(got) != len(want) {
		t.Fatalf("groups = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("group[%d] = %q, want %q (full %v)", i, got[i], want[i], got)
		}
	}
	if p.Groups[0].Total() != 3 {
		t.Errorf("Weapons total = %d, want 3", p.Groups[0].Total())
	}
}

func TestComputeItemsByWeaponType(t *testing.T) {
	p := computeItemsFrom(sampleItems(), nil, "", GroupWeaponType)
	names := groupNames(p)
	// Weapon types come first in canonical order, then non-weapon kinds by label.
	want := []string{"Straight Sword", "Katana", "Backhand Blade"}
	for i, w := range want {
		if names[i] != w {
			t.Errorf("group[%d] = %q, want %q (full %v)", i, names[i], w, names)
		}
	}
	if armor := indexOf(names, "Armor"); armor < len(want) {
		t.Errorf("expected non-weapon kinds after weapon types, got %v", names)
	}
}

func TestComputeItemsByPrimaryStat(t *testing.T) {
	p := computeItemsFrom(sampleItems(), nil, "", GroupPrimaryStat)
	got := groupNames(p)
	want := []string{"Str", "Dex", "Int", "Arc", "—"} // Fai absent
	if len(got) != len(want) {
		t.Fatalf("groups = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("group[%d] = %q, want %q (full %v)", i, got[i], want[i], got)
		}
	}
}

func TestComputeItemsFilter(t *testing.T) {
	p := computeItemsFrom(sampleItems(), nil, KindWeapon, GroupKind)
	if _, total := p.Totals(); total != 3 {
		t.Errorf("weapon-filtered total = %d, want 3", total)
	}
	if len(p.Groups) != 1 || p.Groups[0].Name != "Weapons" {
		t.Errorf("expected a single Weapons group, got %v", groupNames(p))
	}
}
