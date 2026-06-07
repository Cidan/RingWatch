package tracker

// Defeater reports whether the boss with the given event flag is defeated for
// the active character. ok is false when the flag is not addressable in the save
// (which the UI can surface as "unknown" rather than "not done").
type Defeater func(eventID uint32) (defeated, ok bool)

// BossStatus is a boss plus its resolved completion state.
type BossStatus struct {
	Boss
	Defeated bool
	Known    bool // false if the flag could not be read
}

// Region is a group of bosses sharing an in-game location.
type Region struct {
	Name   string
	DLC    bool
	Phase  string
	Bosses []BossStatus
}

// Defeated returns the number of defeated bosses in the region.
func (r Region) Defeated() int {
	n := 0
	for _, b := range r.Bosses {
		if b.Defeated {
			n++
		}
	}
	return n
}

// Total returns the number of bosses in the region.
func (r Region) Total() int { return len(r.Bosses) }

// Complete reports whether every boss in the region is defeated.
func (r Region) Complete() bool { return r.Total() > 0 && r.Defeated() == r.Total() }

// Progress is the full computed completion state, regions in progression order.
type Progress struct {
	Regions []Region
}

// Compute builds progress for every region from the supplied Defeater.
func Compute(def Defeater) Progress {
	byRegion := map[string][]BossStatus{}
	for _, b := range allBosses {
		defeated, ok := def(b.EventID)
		byRegion[b.Region] = append(byRegion[b.Region], BossStatus{
			Boss:     b,
			Defeated: defeated && ok,
			Known:    ok,
		})
	}
	var regions []Region
	for _, m := range orderRegions(allBosses) {
		regions = append(regions, Region{
			Name:   m.Name,
			DLC:    m.DLC,
			Phase:  m.Phase,
			Bosses: byRegion[m.Name],
		})
	}
	return Progress{Regions: regions}
}

// Totals returns overall defeated and total counts.
func (p Progress) Totals() (defeated, total int) {
	for _, r := range p.Regions {
		defeated += r.Defeated()
		total += r.Total()
	}
	return
}

// CategoryTotals splits totals into base-game and DLC.
func (p Progress) CategoryTotals() (baseDefeated, baseTotal, dlcDefeated, dlcTotal int) {
	for _, r := range p.Regions {
		if r.DLC {
			dlcDefeated += r.Defeated()
			dlcTotal += r.Total()
		} else {
			baseDefeated += r.Defeated()
			baseTotal += r.Total()
		}
	}
	return
}
