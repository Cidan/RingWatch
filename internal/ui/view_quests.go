package ui

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/Cidan/RingWatch/internal/locations"
	"github.com/Cidan/RingWatch/internal/tracker"
)

// questPanesView renders the quests section: quest-givers on the left (grouped by
// game phase), the selected giver's walkthrough on the right.
func (m Model) questPanesView(w, h int) string {
	leftOuter := max(min(38, w/2), 22)
	rightOuter := w - leftOuter
	leftInner := leftOuter - 2
	rightInner := rightOuter - 2
	innerH := max(h-2, 1)

	// For quests the search narrows the giver list (left), so focus is not forced
	// to the right pane the way it is in the bosses/items views.
	leftFocused := m.focus == focusRegions
	rightFocused := m.focus == focusBosses

	left := boxStyle(leftFocused).Render(m.giversPane(leftInner, innerH))
	right := boxStyle(rightFocused).Render(m.stepsPane(rightInner, innerH))
	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (m Model) giversPane(w, h int) string {
	quests := m.visibleQuests()
	selIdx := clamp(m.questIdx, 0, max0(len(quests)-1))
	var lines []string
	selLine := 0
	lastPhase := ""
	for i, q := range quests {
		if q.Phase != lastPhase {
			lastPhase = q.Phase
			lines = append(lines, phaseHeaderLine(q.Phase, w))
		}
		if i == selIdx {
			selLine = len(lines)
		}
		lines = append(lines, m.questGiverRow(q, i == selIdx, w))
	}
	if len(lines) == 0 {
		lines = append(lines, subtleStyle.Render("  (no quests)"))
	}
	return fitBlock(windowLines(lines, selLine, h), w, h)
}

// questGiverRow shows the coarse state: ✓ complete, ○ not complete, ◇ untracked guide.
func (m Model) questGiverRow(q tracker.QuestStatus, selected bool, w int) string {
	icon, right := "○", ""
	switch {
	case !q.Trackable:
		icon, right = "◇", "guide"
	case q.Complete:
		icon, right = "✓", "done"
	}
	name := truncate(q.Giver, w-len([]rune(right))-4)
	plain := lineLR(fmt.Sprintf(" %s %s", icon, name), right+" ", w)

	st := lipgloss.NewStyle().Width(w)
	switch {
	case selected:
		st = st.Background(colSelBg).Foreground(colGoldBright).Bold(true)
	case q.Complete:
		st = st.Foreground(colDone)
	case !q.Trackable:
		st = st.Foreground(colMuted)
	default:
		st = st.Foreground(colText)
	}
	return st.Render(plain)
}

func (m Model) stepsPane(w, h int) string {
	q, ok := m.currentQuest()
	if !ok {
		return fitBlock([]string{subtleStyle.Render("  (no quests here)")}, w, h)
	}

	// Header: giver (links to the questline wiki page) + coarse state badge.
	var badge string
	switch {
	case !q.Trackable:
		badge = subtleStyle.Render("guide ")
	case q.Complete:
		badge = doneStyle.Bold(true).Render("✓ complete ")
	default:
		badge = subtleStyle.Render("not complete ")
	}
	giver := locations.Hyperlink(locations.Fextralife(q.PageTitle()), truncate(q.Giver, max(w-16, 6)))
	head := lineLR(goldStyle.Bold(true).Render(" "+giver), badge, w)
	rule := ruleStyle.Render(strings.Repeat("─", w))

	var body []string
	if q.Summary != "" {
		for _, ln := range wrapText(q.Summary, w-2) {
			body = append(body, lipgloss.NewStyle().Foreground(colText).Width(w).Render(" "+ln))
		}
	}
	if q.Reward != "" {
		for _, ln := range wrapText("Reward: "+q.Reward, w-2) {
			body = append(body, goldStyle.Width(w).Render(" "+ln))
		}
	}
	if !q.Trackable {
		body = append(body, subtleStyle.Width(w).Render(" Progress isn't save-tracked for this quest — steps are a guide."))
	}
	if len(body) > 0 {
		body = append(body, "")
	}

	steps := q.Steps
	selStep := clamp(m.stepIdx, 0, max0(len(steps)-1))
	starts := make([]int, len(steps))
	for i, s := range steps {
		starts[i] = len(body)
		body = append(body, m.stepBlock(i, s, i == selStep, w)...)
		body = append(body, "")
	}
	if len(steps) == 0 {
		body = append(body, subtleStyle.Render("  (no steps)"))
	}

	listH := max(h-2, 1)
	sel := 0
	if len(starts) > 0 {
		sel = starts[selStep]
	}
	body = windowLines(body, sel, listH)
	return fitBlock(append([]string{head, rule}, body...), w, h)
}

// stepBlock renders one walkthrough step: a numbered title line (location right-
// aligned and clickable) plus the wrapped instructions and any missable warning.
// The selected step is highlighted for reading; there is no per-step completion mark.
func (m Model) stepBlock(idx int, s tracker.QuestStep, selected bool, w int) []string {
	right := ""
	if loc := stepLocationName(s.Location); loc != "" {
		right = locations.Hyperlink(locations.Fextralife(loc), truncate(loc, 22))
	}
	title := fmt.Sprintf(" %2d. %s", idx+1, s.Title)
	if s.Optional {
		title += " (optional)"
	}
	title = truncate(title, max(w-lipgloss.Width(right)-1, 4))

	titleFg, detailFg := colText, colMuted
	if selected {
		titleFg, detailFg = colGoldBright, colText
	}

	var lines []string
	lines = append(lines, renderQuestLine(lineLR(title, right+" ", w), w, titleFg, selected, selected))
	for _, dl := range wrapText(s.Detail, w-6) {
		lines = append(lines, renderQuestLine("     "+dl, w, detailFg, selected, false))
	}
	if s.Warning != "" {
		for _, wl := range wrapText("⚠ "+s.Warning, w-6) {
			lines = append(lines, renderQuestLine("     "+wl, w, colEmber, selected, false))
		}
	}
	return lines
}

// renderQuestLine pads text to width w, applying the foreground (and a selection
// background when selected). text must already fit within w columns.
func renderQuestLine(text string, w int, fg color.Color, selected, bold bool) string {
	st := lipgloss.NewStyle().Width(w).Foreground(fg)
	if selected {
		st = st.Background(colSelBg)
	}
	if bold {
		st = st.Bold(true)
	}
	return st.Render(text)
}
