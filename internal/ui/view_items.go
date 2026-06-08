package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/Cidan/RingWatch/internal/locations"
	"github.com/Cidan/RingWatch/internal/tracker"
)

// itemPanesView renders the items section: groups on the left, items on the right.
func (m Model) itemPanesView(w, h int) string {
	leftOuter := max(min(38, w/2), 22)
	rightOuter := w - leftOuter
	leftInner := leftOuter - 2
	rightInner := rightOuter - 2
	innerH := max(h-2, 1)

	leftFocused := m.focus == focusRegions && m.filter == ""
	rightFocused := m.focus == focusBosses || m.filter != ""

	left := boxStyle(leftFocused).Render(m.groupsPane(leftInner, innerH))
	right := boxStyle(rightFocused).Render(m.itemsPane(rightInner, innerH))
	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (m Model) groupsPane(w, h int) string {
	groups := m.visibleGroups()
	var lines []string
	selLine := 0
	for i, g := range groups {
		if i == m.groupIdx {
			selLine = len(lines)
		}
		lines = append(lines, itemGroupRowStyled(g, i == m.groupIdx, w))
	}
	if len(lines) == 0 {
		lines = append(lines, subtleStyle.Render("  (no items)"))
	}
	return fitBlock(windowLines(lines, selLine, h), w, h)
}

func itemGroupRowStyled(g tracker.ItemGroup, selected bool, w int) string {
	icon := "•"
	if g.Complete() {
		icon = "✓"
	}
	counts := fmt.Sprintf("%d/%d", g.Owned(), g.Total())
	name := truncate(g.Name, w-len([]rune(counts))-4)
	plain := lineLR(fmt.Sprintf(" %s %s", icon, name), counts+" ", w)

	st := lipgloss.NewStyle().Width(w)
	switch {
	case selected:
		st = st.Background(colSelBg).Foreground(colGoldBright).Bold(true)
	case g.Complete():
		st = st.Foreground(colDone)
	default:
		st = st.Foreground(colText)
	}
	return st.Render(plain)
}

func (m Model) itemsPane(w, h int) string {
	rows := m.visibleItems()

	var head string
	if m.filter != "" {
		q := m.filter
		if m.mode == modeTyping {
			q += "▏"
		}
		head = lineLR(
			goldStyle.Render(" Search ")+subtleStyle.Render(q),
			subtleStyle.Render(fmt.Sprintf("%d found ", len(rows))), w)
	} else {
		groups := m.visibleGroups()
		name, own, tot := "—", 0, 0
		if len(groups) > 0 {
			idx := clamp(m.groupIdx, 0, len(groups)-1)
			name = groups[idx].Name
			own, tot = groups[idx].Owned(), groups[idx].Total()
		}
		head = lineLR(
			goldStyle.Bold(true).Render(" "+truncate(name, w-12)),
			subtleStyle.Render(fmt.Sprintf("%d/%d ", own, tot)), w)
	}
	rule := ruleStyle.Render(strings.Repeat("─", w))

	listH := max(h-2, 1)
	var bodyLines []string
	for i, r := range rows {
		bodyLines = append(bodyLines, m.itemRowStyled(r, i == m.itemIdx, w))
	}
	if len(bodyLines) == 0 {
		bodyLines = append(bodyLines, subtleStyle.Render("  (no items here)"))
	}
	bodyLines = windowLines(bodyLines, clamp(m.itemIdx, 0, max0(len(bodyLines)-1)), listH)
	return fitBlock(append([]string{head, rule}, bodyLines...), w, h)
}

func (m Model) itemRowStyled(r itemRow, selected bool, w int) string {
	it := r.St
	icon := "○"
	if it.Owned {
		icon = "✓"
	}
	right := ""
	if m.filter != "" {
		right = truncate(r.Group, 18)
	} else {
		right = truncate(m.itemAnnotation(it.Item), 18)
	}
	nameW := max(w-4-lipgloss.Width(right), 4)
	name := truncate(it.Name, nameW)
	link := locations.Hyperlink(locations.Fextralife(it.Name), name)
	plain := lineLR(fmt.Sprintf(" %s %s", icon, link), right+" ", w)

	style := lipgloss.NewStyle().Width(w)
	switch {
	case selected && it.Owned:
		style = style.Background(colSelBg).Foreground(colDone).Bold(true)
	case selected:
		style = style.Background(colSelBg).Foreground(colGoldBright).Bold(true)
	case it.Owned:
		style = style.Foreground(colDone)
	default:
		style = style.Foreground(colText)
	}
	return style.Render(plain)
}

// itemAnnotation picks a useful right-hand descriptor that is NOT the current
// grouping dimension (so it adds information rather than repeating the group).
func (m Model) itemAnnotation(it tracker.Item) string {
	if it.Kind == tracker.KindFlask {
		// Flask upgrades share one name ("Golden Seed"/"Sacred Tear"); the locating
		// info is what distinguishes rows. Show the region everywhere except the
		// Location grouping, where the region is already the group heading.
		if m.groupBy == tracker.GroupRegion {
			return flaskLocator(it)
		}
		return it.Region
	}
	switch m.groupBy {
	case tracker.GroupWeaponType, tracker.GroupKind:
		if it.Region != "" {
			return it.Region
		}
		return itemKindDetail(it)
	case tracker.GroupPrimaryStat:
		return itemKindDetail(it)
	default: // GroupRegion
		return itemKindDetail(it)
	}
}

// itemKindDetail is the item's most identifying secondary attribute.
func itemKindDetail(it tracker.Item) string {
	switch it.Kind {
	case tracker.KindWeapon, tracker.KindShield:
		return it.WeaponType
	case tracker.KindArmor:
		return it.ArmorSlot
	case tracker.KindSorcery:
		return "Sorcery"
	case tracker.KindIncantation:
		return "Incantation"
	case tracker.KindTalisman:
		return "Talisman"
	case tracker.KindAsh:
		return "Ash of War"
	case tracker.KindSpirit:
		return "Spirit Ash"
	case tracker.KindFlask:
		return "Flask Upgrade"
	}
	return ""
}

// flaskLocator condenses a flask upgrade's location prose for the right-hand
// annotation, dropping the repetitive "Under a/the Golden Seed tree" lead-in so the
// distinguishing part of the spot leads (rows otherwise share name + prefix).
func flaskLocator(it tracker.Item) string {
	if _, after, ok := strings.Cut(it.Location, "Seed tree "); ok {
		return strings.TrimSpace(after)
	}
	return it.Location
}
