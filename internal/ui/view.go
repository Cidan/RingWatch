package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/Cidan/RingWatch/internal/locations"
	"github.com/Cidan/RingWatch/internal/tracker"
)

// View implements tea.Model. In Bubble Tea v2 it returns a tea.View whose
// AltScreen field puts the program in the alternate screen buffer.
func (m Model) View() tea.View {
	if m.width < 40 || m.height < 12 {
		v := tea.NewView("Terminal too small — enlarge the window.")
		v.AltScreen = true
		return v
	}
	midH := max(m.height-4, 3) // header (3) + footer (1)

	header := m.headerView()
	footer := m.footerView()

	var mid string
	switch {
	case m.mode == modePickChar:
		mid = m.charPickerView(m.width, midH)
	case m.section == sectionItems:
		mid = m.itemPanesView(m.width, midH)
	case m.section == sectionQuests:
		mid = m.questPanesView(m.width, midH)
	default:
		mid = m.panesView(m.width, midH)
	}
	v := tea.NewView(strings.Join([]string{header, mid, footer}, "\n"))
	v.AltScreen = true
	return v
}

func (m Model) headerView() string {
	w := m.width

	title := titleStyle.Render("✦  RINGWATCH  ✦")
	watch := subtleStyle.Render("○ static")
	if m.watching {
		watch = doneStyle.Render("● watching")
	}
	line1 := lineLR("  "+title+"   "+m.sectionTabs(), watch+"  ", w)

	var who string
	if c, ok := m.currentChar(); ok {
		who = fmt.Sprintf("%s  %s  Lv %d  %s  %s",
			goldStyle.Render(c.Name), subtleStyle.Render("·"),
			c.Level, subtleStyle.Render("·"), subtleStyle.Render(c.Playtime()))
	} else {
		who = subtleStyle.Render("no character")
	}

	def, tot := m.prog.Totals()
	switch m.section {
	case sectionItems:
		def, tot = m.itemProg.Totals()
	case sectionQuests:
		def, tot = m.questProg.Totals()
	}
	right := fmt.Sprintf("%s  %s  %s",
		progressBar(def, tot, 22),
		goldStyle.Render(fmt.Sprintf("%d/%d", def, tot)),
		subtleStyle.Render(fmt.Sprintf("%d%%", pct(def, tot))))
	switch m.section {
	case sectionItems:
		right = subtleStyle.Render(fmt.Sprintf("%s · %s   ", m.groupBy.Label(), m.filterLabel())) + right
	case sectionQuests:
		_, _, total := m.questProg.Counts()
		right = subtleStyle.Render(fmt.Sprintf("%d quests · %d tracked   ", total, tot)) + right
	}
	line2 := lineLR("  "+who, right+"  ", w)

	line3 := ruleStyle.Render(strings.Repeat("─", w))
	return line1 + "\n" + line2 + "\n" + line3
}

// sectionTabs renders the Bosses/Items top-level tab chips.
func (m Model) sectionTabs() string {
	chip := func(label string, active bool) string {
		s := lipgloss.NewStyle().Padding(0, 1)
		if active {
			return s.Background(colSelBg).Foreground(colGoldBright).Bold(true).Render(label)
		}
		return s.Foreground(colText).Render(label)
	}
	return chip("Bosses", m.section == sectionBosses) +
		chip("Items", m.section == sectionItems) +
		chip("Quests", m.section == sectionQuests)
}

func (m Model) panesView(w, h int) string {
	leftOuter := max(min(38, w/2), 22)
	rightOuter := w - leftOuter
	leftInner := leftOuter - 2
	rightInner := rightOuter - 2
	innerH := max(h-2, 1)

	leftFocused := m.focus == focusRegions && m.filter == ""
	rightFocused := m.focus == focusBosses || m.filter != ""

	// regionsPane/bossesPane return content blocks of exactly innerW×innerH, so
	// the border (added outside) yields predictable outer dimensions — avoiding
	// lipgloss's "Width includes border" wrapping surprise.
	left := boxStyle(leftFocused).Render(m.regionsPane(leftInner, innerH))
	right := boxStyle(rightFocused).Render(m.bossesPane(rightInner, innerH))
	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func boxStyle(focused bool) lipgloss.Style {
	c := colBorderDim
	if focused {
		c = colGold
	}
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(c)
}

func (m Model) regionsPane(w, h int) string {
	regions := m.visibleRegions()
	var lines []string
	selLine := 0
	lastPhase := ""
	for i, r := range regions {
		if r.Phase != lastPhase {
			lastPhase = r.Phase
			lines = append(lines, phaseHeaderLine(r.Phase, w))
		}
		if i == m.regionIdx {
			selLine = len(lines)
		}
		lines = append(lines, regionRowStyled(r, i == m.regionIdx, w))
	}
	return fitBlock(windowLines(lines, selLine, h), w, h)
}

func phaseHeaderLine(phase string, w int) string {
	label := strings.ToUpper(tracker.PhaseLabel[phase])
	head := lipgloss.NewStyle().Foreground(colEmber).Bold(true).Render(label + " ")
	rest := max(w-lipgloss.Width(head), 0)
	return head + ruleStyle.Render(strings.Repeat("─", rest))
}

func regionRowStyled(r tracker.Region, selected bool, w int) string {
	icon := "•"
	if r.Complete() {
		icon = "✓"
	}
	counts := fmt.Sprintf("%d/%d", r.Defeated(), r.Total())
	name := truncate(r.Name, w-len([]rune(counts))-4)
	plain := lineLR(fmt.Sprintf(" %s %s", icon, name), counts+" ", w)

	st := lipgloss.NewStyle().Width(w)
	switch {
	case selected:
		st = st.Background(colSelBg).Foreground(colGoldBright).Bold(true)
	case r.Complete():
		st = st.Foreground(colDone)
	default:
		st = st.Foreground(colText)
	}
	return st.Render(plain)
}

func (m Model) bossesPane(w, h int) string {
	rows := m.visibleBosses()

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
		regions := m.visibleRegions()
		name, def, tot := "—", 0, 0
		if len(regions) > 0 {
			idx := clamp(m.regionIdx, 0, len(regions)-1)
			name = regions[idx].Name
			def, tot = regions[idx].Defeated(), regions[idx].Total()
		}
		head = lineLR(
			goldStyle.Bold(true).Render(" "+truncate(name, w-12)),
			subtleStyle.Render(fmt.Sprintf("%d/%d ", def, tot)), w)
	}
	rule := ruleStyle.Render(strings.Repeat("─", w))

	listH := max(h-2, 1)
	var bodyLines []string
	for i, br := range rows {
		bodyLines = append(bodyLines, m.bossRowStyled(br, i == m.bossIdx, w))
	}
	if len(bodyLines) == 0 {
		bodyLines = append(bodyLines, subtleStyle.Render("  (no bosses here)"))
	}
	bodyLines = windowLines(bodyLines, clamp(m.bossIdx, 0, max0(len(bodyLines)-1)), listH)
	return fitBlock(append([]string{head, rule}, bodyLines...), w, h)
}

func (m Model) bossRowStyled(br bossRow, selected bool, w int) string {
	st := br.St
	icon := "○"
	if st.Defeated {
		icon = "✓"
	}
	right := ""
	if m.filter != "" {
		right = truncate(br.Region, 18)
	}
	nameW := max(w-4-lipgloss.Width(right), 4)
	name := truncate(st.Name, nameW)
	link := locations.Hyperlink(locations.Fextralife(st.Name), name)
	plain := lineLR(fmt.Sprintf(" %s %s", icon, link), right+" ", w)

	style := lipgloss.NewStyle().Width(w)
	switch {
	case selected && st.Defeated:
		style = style.Background(colSelBg).Foreground(colDone).Bold(true)
	case selected:
		style = style.Background(colSelBg).Foreground(colGoldBright).Bold(true)
	case st.Defeated:
		style = style.Foreground(colDone)
	default:
		style = style.Foreground(colText)
	}
	return style.Render(plain)
}

func (m Model) footerView() string {
	if m.toast != "" {
		return lipgloss.NewStyle().Foreground(colEmber).Bold(true).Width(m.width).Render(truncate("  "+m.toast, m.width))
	}
	var keys string
	switch m.mode {
	case modeTyping:
		keys = "type to filter  ·  enter apply  ·  esc clear"
	case modePickChar:
		keys = "↑/↓ choose  ·  enter select  ·  esc cancel"
	default:
		switch m.section {
		case sectionItems:
			keys = "↑↓ move · tab pane · g group · f filter · / search · enter wiki · c char · d dlc · 1/2/3 tabs · q quit"
		case sectionQuests:
			keys = "↑↓ move · tab pane · / search NPC · enter wiki · c char · d dlc · 1/2/3 tabs · q quit"
		default:
			keys = "↑↓ move · tab pane · / search · enter wiki · c char · d dlc · 1/2/3 tabs · q quit"
		}
	}
	return subtleStyle.Width(m.width).Render(truncate("  "+keys, m.width))
}

func (m Model) charPickerView(w, h int) string {
	rows := []string{titleStyle.Render("Select Character"), ""}
	for i, c := range m.chars {
		line := fmt.Sprintf(" %-18s  Lv %-4d  %-10s ", truncate(c.Name, 18), c.Level, c.Playtime())
		st := lipgloss.NewStyle().Foreground(colText)
		if i == m.pickIdx {
			st = st.Background(colSelBg).Foreground(colGoldBright).Bold(true)
		}
		rows = append(rows, st.Render(line))
	}
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colGold).
		Padding(1, 2).
		Render(strings.Join(rows, "\n"))
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, box)
}

func windowLines(lines []string, sel, h int) []string {
	if len(lines) <= h {
		return lines
	}
	top := clamp(sel-h/2, 0, len(lines)-h)
	return lines[top : top+h]
}

// fitBlock normalizes lines into exactly h rows, each padded to visible width w,
// so a surrounding border renders with predictable dimensions.
func fitBlock(lines []string, w, h int) string {
	out := make([]string, 0, h)
	for _, ln := range lines {
		if len(out) == h {
			break
		}
		out = append(out, padRight(ln, w))
	}
	for len(out) < h {
		out = append(out, strings.Repeat(" ", w))
	}
	return strings.Join(out, "\n")
}
