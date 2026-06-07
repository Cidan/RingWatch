package ui

import (
	"strings"

	"github.com/Cidan/RingWatch/internal/locations"
	"github.com/Cidan/RingWatch/internal/save"
	"github.com/Cidan/RingWatch/internal/tracker"
)

// ownerFor builds a tracker.Owner from a character's parsed owned-item sets,
// mapping each item kind to the inventory category that backs it. Shields live in
// the weapon set; sorceries/incantations/spirit ashes are all goods; ashes of war
// are gems.
func ownerFor(owned *save.OwnedItems) tracker.Owner {
	return func(it tracker.Item) bool {
		if owned == nil {
			return false
		}
		switch it.Kind {
		case tracker.KindWeapon, tracker.KindShield:
			return owned.Weapons[it.ID]
		case tracker.KindArmor:
			return owned.Protectors[it.ID]
		case tracker.KindTalisman:
			return owned.Accessories[it.ID]
		case tracker.KindSorcery, tracker.KindIncantation:
			return owned.Goods[it.ID]
		case tracker.KindSpirit:
			// Spirit ashes upgrade to consecutive goods ids (base..base+10); the
			// dataset stores the base, so owning any level counts.
			for k := uint32(0); k <= 10; k++ {
				if owned.Goods[it.ID+k] {
					return true
				}
			}
			return false
		case tracker.KindAsh:
			return owned.Gems[it.ID]
		}
		return false
	}
}

// recomputeItems rebuilds the item progress for the current grouping + filter,
// without re-reading the save (called when only the grouping/filter changes).
func (m *Model) recomputeItems() {
	var owned *save.OwnedItems
	if m.saveFile != nil {
		owned, _ = m.saveFile.OwnedItems(m.slot)
	}
	m.itemProg = tracker.ComputeItems(ownerFor(owned), m.kindFilter, m.groupBy)
}

// kindFilterOrder is the cycle of category filters in the items view ("" = All).
var kindFilterOrder = append([]tracker.ItemKind{""}, tracker.KindOrder...)

func (m *Model) cycleGroupBy(delta int) {
	n := len(tracker.GroupByOrder)
	cur := 0
	for i, g := range tracker.GroupByOrder {
		if g == m.groupBy {
			cur = i
			break
		}
	}
	m.groupBy = tracker.GroupByOrder[((cur+delta)%n+n)%n]
	m.recomputeItems()
	m.groupIdx, m.itemIdx = 0, 0
}

func (m *Model) cycleFilter(delta int) {
	n := len(kindFilterOrder)
	cur := 0
	for i, f := range kindFilterOrder {
		if f == m.kindFilter {
			cur = i
			break
		}
	}
	m.kindFilter = kindFilterOrder[((cur+delta)%n+n)%n]
	m.recomputeItems()
	m.groupIdx, m.itemIdx = 0, 0
}

// filterLabel is the display name of the active category filter.
func (m Model) filterLabel() string {
	if m.kindFilter == "" {
		return "All"
	}
	return tracker.KindLabel[m.kindFilter]
}

// itemRow pairs an item status with its group (the group label matters in search
// results, where rows come from many groups).
type itemRow struct {
	St    tracker.ItemStatus
	Group string
}

// visibleGroups honors the DLC toggle by dropping DLC items (and any group left
// empty) when DLC is hidden.
func (m Model) visibleGroups() []tracker.ItemGroup {
	if m.showDLC {
		return m.itemProg.Groups
	}
	out := make([]tracker.ItemGroup, 0, len(m.itemProg.Groups))
	for _, g := range m.itemProg.Groups {
		var kept []tracker.ItemStatus
		for _, it := range g.Items {
			if !it.DLC {
				kept = append(kept, it)
			}
		}
		if len(kept) > 0 {
			out = append(out, tracker.ItemGroup{Name: g.Name, Items: kept})
		}
	}
	return out
}

// visibleItems returns the rows for the right pane: global search results when a
// filter is active, otherwise the selected group's items.
func (m Model) visibleItems() []itemRow {
	groups := m.visibleGroups()
	var rows []itemRow
	if m.filter != "" {
		q := strings.ToLower(m.filter)
		for _, g := range groups {
			for _, it := range g.Items {
				if strings.Contains(strings.ToLower(it.Name), q) {
					rows = append(rows, itemRow{St: it, Group: g.Name})
				}
			}
		}
		return rows
	}
	if len(groups) == 0 {
		return rows
	}
	idx := clamp(m.groupIdx, 0, len(groups)-1)
	for _, it := range groups[idx].Items {
		rows = append(rows, itemRow{St: it, Group: groups[idx].Name})
	}
	return rows
}

// onItems reports whether navigation currently targets the item (right) pane.
func (m Model) onItems() bool { return m.filter != "" || m.focus == focusBosses }

func (m *Model) itemMoveUp() {
	if m.onItems() {
		if m.itemIdx > 0 {
			m.itemIdx--
		}
		return
	}
	if m.groupIdx > 0 {
		m.groupIdx--
		m.itemIdx = 0
	}
}

func (m *Model) itemMoveDown() {
	if m.onItems() {
		if m.itemIdx < len(m.visibleItems())-1 {
			m.itemIdx++
		}
		return
	}
	if m.groupIdx < len(m.visibleGroups())-1 {
		m.groupIdx++
		m.itemIdx = 0
	}
}

func (m Model) openCurrentItem() {
	rows := m.visibleItems()
	if m.itemIdx >= 0 && m.itemIdx < len(rows) {
		openURL(locations.Fextralife(rows[m.itemIdx].St.Name))
	}
}
