package ui

import (
	"strings"

	"github.com/Cidan/RingWatch/internal/locations"
	"github.com/Cidan/RingWatch/internal/save"
	"github.com/Cidan/RingWatch/internal/tracker"
)

// recomputeQuests rebuilds quest completion from the current character: an event-flag
// reader (as the boss view uses) and an item-ownership reader (as the items view uses).
func (m *Model) recomputeQuests() {
	slot := m.slot
	flagReader := func(id uint32) (bool, bool) {
		if m.saveFile == nil {
			return false, false
		}
		return m.saveFile.IsDefeated(slot, id)
	}
	var owned *save.OwnedItems
	var flags []byte
	if m.saveFile != nil {
		owned, _ = m.saveFile.OwnedItems(slot)
		flags, _ = m.saveFile.EventFlags(slot)
	}
	m.questProg = tracker.ComputeQuests(flagReader, ownerFor(owned, flags))
}

// onSteps reports whether navigation currently targets the steps (right) pane. The
// quest search narrows the giver list (left) rather than the steps, so this is a
// pure focus check — unlike the bosses/items views where a filter forces the right.
func (m Model) onSteps() bool { return m.focus == focusBosses }

// visibleQuests returns the quest-givers for the left pane, honoring the DLC toggle
// and the name search filter.
func (m Model) visibleQuests() []tracker.QuestStatus {
	q := strings.ToLower(m.filter)
	out := make([]tracker.QuestStatus, 0, len(m.questProg.Quests))
	for _, qs := range m.questProg.Quests {
		if !m.showDLC && qs.DLC {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(qs.Giver), q) {
			continue
		}
		out = append(out, qs)
	}
	return out
}

// currentQuest returns the selected quest-giver, if any.
func (m Model) currentQuest() (tracker.QuestStatus, bool) {
	qs := m.visibleQuests()
	if len(qs) == 0 {
		return tracker.QuestStatus{}, false
	}
	return qs[clamp(m.questIdx, 0, len(qs)-1)], true
}

// visibleSteps returns the walkthrough steps of the selected quest-giver (the right
// pane). Steps are guide text only — no per-step save state.
func (m Model) visibleSteps() []tracker.QuestStep {
	if q, ok := m.currentQuest(); ok {
		return q.Steps
	}
	return nil
}

func (m *Model) questMoveUp() {
	if m.onSteps() {
		if m.stepIdx > 0 {
			m.stepIdx--
		}
		return
	}
	if m.questIdx > 0 {
		m.questIdx--
		m.stepIdx = 0
	}
}

func (m *Model) questMoveDown() {
	if m.onSteps() {
		if m.stepIdx < len(m.visibleSteps())-1 {
			m.stepIdx++
		}
		return
	}
	if m.questIdx < len(m.visibleQuests())-1 {
		m.questIdx++
		m.stepIdx = 0
	}
}

// openCurrentQuest opens a wiki page: the selected step's location when focused on
// the steps pane (the "where to go"), otherwise the quest-giver's questline page.
func (m Model) openCurrentQuest() {
	q, ok := m.currentQuest()
	if !ok {
		return
	}
	if m.onSteps() {
		steps := m.visibleSteps()
		if m.stepIdx >= 0 && m.stepIdx < len(steps) {
			if loc := stepLocationName(steps[m.stepIdx].Location); loc != "" {
				openURL(locations.Fextralife(loc))
				return
			}
		}
	}
	openURL(locations.Fextralife(q.PageTitle()))
}

// stepLocationName trims the region suffix off a step location ("Church of the
// Plague, Caelid" -> "Church of the Plague") so it maps to a wiki page slug.
func stepLocationName(loc string) string {
	if i := strings.IndexByte(loc, ','); i >= 0 {
		loc = loc[:i]
	}
	return strings.TrimSpace(loc)
}
