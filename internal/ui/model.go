package ui

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"

	"github.com/Cidan/RingWatch/internal/locations"
	"github.com/Cidan/RingWatch/internal/save"
	"github.com/Cidan/RingWatch/internal/tracker"
)

// toastDuration is how long a "boss felled" toast stays on screen.
const toastDuration = 4 * time.Second

type mode int

const (
	modeBrowse mode = iota
	modeTyping
	modePickChar
)

const (
	focusRegions = 0
	focusBosses  = 1
)

// Top-level sections, switched with 1/2. focusRegions/focusBosses double as
// left/right pane focus within either section.
const (
	sectionBosses = 0
	sectionItems  = 1
)

// bossRow pairs a boss status with its region (region matters in search results).
type bossRow struct {
	St     tracker.BossStatus
	Region string
}

// Messages.
type (
	saveChangedMsg struct{}
	toastExpireMsg struct{ seq int }
)

// Model is the root Bubble Tea model.
type Model struct {
	savePath string
	width    int
	height   int

	saveFile *save.Save
	chars    []save.Character
	slot     int
	prog     tracker.Progress
	loadErr  error

	section int

	mode      mode
	focus     int
	regionIdx int
	bossIdx   int
	pickIdx   int
	filter    string
	showDLC   bool

	// items section state
	groupBy    tracker.GroupBy
	kindFilter tracker.ItemKind
	groupIdx   int
	itemIdx    int
	itemProg   tracker.ItemProgress

	changes  <-chan struct{}
	watching bool

	toast    string
	toastSeq int
}

func newModel(opts Options, s *save.Save) Model {
	m := Model{
		savePath: opts.SavePath,
		saveFile: s,
		chars:    s.ActiveCharacters(),
		showDLC:  true,
		focus:    focusRegions,
		watching: opts.Watch,
	}
	m.slot = pickSlot(m.chars, opts.Slot)
	m.recompute()
	return m
}

func pickSlot(chars []save.Character, want int) int {
	if want >= 0 {
		for _, c := range chars {
			if c.Slot == want {
				return want
			}
		}
	}
	if len(chars) > 0 {
		return chars[0].Slot
	}
	return 0
}

func (m *Model) recompute() {
	if m.saveFile == nil {
		return
	}
	slot := m.slot
	m.prog = tracker.Compute(func(id uint32) (bool, bool) {
		return m.saveFile.IsDefeated(slot, id)
	})
	m.recomputeItems()
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	if m.watching {
		return waitForChange(m.changes)
	}
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case saveChangedMsg:
		old := m.prog
		if s, err := save.Open(m.savePath); err == nil {
			m.saveFile = s
			m.chars = s.ActiveCharacters()
			m.slot = pickSlot(m.chars, m.slot)
			m.recompute()
			if names := newlyDefeated(old, m.prog); len(names) > 0 {
				m.toastSeq++
				if len(names) == 1 {
					m.toast = "⚔  Felled: " + names[0]
				} else {
					m.toast = fmt.Sprintf("⚔  Felled %d bosses", len(names))
				}
				return m, tea.Batch(waitForChange(m.changes), toastTickCmd(m.toastSeq))
			}
		} else {
			m.loadErr = err
		}
		return m, waitForChange(m.changes)

	case toastExpireMsg:
		if msg.seq == m.toastSeq {
			m.toast = ""
		}
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	k := msg.String()
	if k == "ctrl+c" {
		return m, tea.Quit
	}

	switch m.mode {
	case modePickChar:
		switch k {
		case "esc", "c", "q":
			m.mode = modeBrowse
		case "up", "k":
			if m.pickIdx > 0 {
				m.pickIdx--
			}
		case "down", "j":
			if m.pickIdx < len(m.chars)-1 {
				m.pickIdx++
			}
		case "enter":
			if m.pickIdx >= 0 && m.pickIdx < len(m.chars) {
				m.slot = m.chars[m.pickIdx].Slot
				m.recompute()
				m.regionIdx, m.bossIdx = 0, 0
				m.groupIdx, m.itemIdx = 0, 0
			}
			m.mode = modeBrowse
		}
		return m, nil

	case modeTyping:
		switch k {
		case "esc":
			m.filter = ""
			m.mode = modeBrowse
			m.bossIdx, m.itemIdx = 0, 0
		case "enter":
			m.mode = modeBrowse
			m.focus = focusBosses
			m.bossIdx, m.itemIdx = 0, 0
		case "backspace":
			if r := []rune(m.filter); len(r) > 0 {
				m.filter = string(r[:len(r)-1])
				m.bossIdx, m.itemIdx = 0, 0
			}
		case "ctrl+u":
			m.filter = ""
			m.bossIdx, m.itemIdx = 0, 0
		default:
			if utf8.RuneCountInString(k) == 1 {
				m.filter += k
				m.bossIdx, m.itemIdx = 0, 0
			}
		}
		return m, nil
	}

	// modeBrowse: shared keys first, then section-specific navigation.
	switch k {
	case "q":
		return m, tea.Quit
	case "1":
		m.section = sectionBosses
		return m, nil
	case "2":
		m.section = sectionItems
		return m, nil
	case "tab":
		m.focus = 1 - m.focus
		return m, nil
	case "left", "h":
		m.focus = focusRegions
		return m, nil
	case "right", "l":
		m.focus = focusBosses
		return m, nil
	case "/":
		m.mode = modeTyping
		m.focus = focusBosses
		return m, nil
	case "esc":
		if m.filter != "" {
			m.filter = ""
			m.bossIdx, m.itemIdx = 0, 0
		}
		return m, nil
	case "d":
		m.showDLC = !m.showDLC
		m.regionIdx, m.bossIdx = 0, 0
		m.groupIdx, m.itemIdx = 0, 0
		return m, nil
	case "c":
		m.mode = modePickChar
		m.pickIdx = m.charIndex(m.slot)
		return m, nil
	}

	if m.section == sectionItems {
		return m.handleItemsKey(k)
	}
	return m.handleBossesKey(k)
}

// handleBossesKey handles browse-mode navigation in the bosses section.
func (m Model) handleBossesKey(k string) (tea.Model, tea.Cmd) {
	switch k {
	case "up", "k":
		m.moveUp()
	case "down", "j":
		m.moveDown()
	case "g", "home":
		if m.onBosses() {
			m.bossIdx = 0
		} else {
			m.regionIdx, m.bossIdx = 0, 0
		}
	case "G", "end":
		if m.onBosses() {
			m.bossIdx = max0(len(m.visibleBosses()) - 1)
		} else {
			m.regionIdx = max0(len(m.visibleRegions()) - 1)
			m.bossIdx = 0
		}
	case "enter", "o":
		m.openCurrentBoss()
	}
	return m, nil
}

// handleItemsKey handles browse-mode navigation in the items section. g/f cycle
// the grouping and category filter (shift-G / shift-F reverse).
func (m Model) handleItemsKey(k string) (tea.Model, tea.Cmd) {
	switch k {
	case "g":
		m.cycleGroupBy(1)
	case "G":
		m.cycleGroupBy(-1)
	case "f":
		m.cycleFilter(1)
	case "F":
		m.cycleFilter(-1)
	case "up", "k":
		m.itemMoveUp()
	case "down", "j":
		m.itemMoveDown()
	case "home":
		if m.onItems() {
			m.itemIdx = 0
		} else {
			m.groupIdx, m.itemIdx = 0, 0
		}
	case "end":
		if m.onItems() {
			m.itemIdx = max0(len(m.visibleItems()) - 1)
		} else {
			m.groupIdx = max0(len(m.visibleGroups()) - 1)
			m.itemIdx = 0
		}
	case "enter", "o":
		m.openCurrentItem()
	}
	return m, nil
}

// onBosses reports whether navigation currently targets the boss list.
func (m Model) onBosses() bool { return m.filter != "" || m.focus == focusBosses }

func (m *Model) moveUp() {
	if m.onBosses() {
		if m.bossIdx > 0 {
			m.bossIdx--
		}
		return
	}
	if m.regionIdx > 0 {
		m.regionIdx--
		m.bossIdx = 0
	}
}

func (m *Model) moveDown() {
	if m.onBosses() {
		if m.bossIdx < len(m.visibleBosses())-1 {
			m.bossIdx++
		}
		return
	}
	if m.regionIdx < len(m.visibleRegions())-1 {
		m.regionIdx++
		m.bossIdx = 0
	}
}

func (m Model) openCurrentBoss() {
	rows := m.visibleBosses()
	if m.bossIdx >= 0 && m.bossIdx < len(rows) {
		openURL(locations.Fextralife(rows[m.bossIdx].St.Name))
	}
}

// visibleRegions returns regions honoring the DLC toggle.
func (m Model) visibleRegions() []tracker.Region {
	if m.showDLC {
		return m.prog.Regions
	}
	out := make([]tracker.Region, 0, len(m.prog.Regions))
	for _, r := range m.prog.Regions {
		if !r.DLC {
			out = append(out, r)
		}
	}
	return out
}

// visibleBosses returns the boss rows for the right pane: global search results
// when a filter is active, otherwise the selected region's bosses.
func (m Model) visibleBosses() []bossRow {
	regions := m.visibleRegions()
	var rows []bossRow
	if m.filter != "" {
		q := strings.ToLower(m.filter)
		for _, r := range regions {
			for _, b := range r.Bosses {
				if strings.Contains(strings.ToLower(b.Name), q) {
					rows = append(rows, bossRow{St: b, Region: r.Name})
				}
			}
		}
		return rows
	}
	if len(regions) == 0 {
		return rows
	}
	idx := clamp(m.regionIdx, 0, len(regions)-1)
	for _, b := range regions[idx].Bosses {
		rows = append(rows, bossRow{St: b, Region: regions[idx].Name})
	}
	return rows
}

func (m Model) currentChar() (save.Character, bool) {
	for _, c := range m.chars {
		if c.Slot == m.slot {
			return c, true
		}
	}
	return save.Character{}, false
}

func (m Model) charIndex(slot int) int {
	for i, c := range m.chars {
		if c.Slot == slot {
			return i
		}
	}
	return 0
}

func newlyDefeated(old, neo tracker.Progress) []string {
	was := map[uint32]bool{}
	for _, r := range old.Regions {
		for _, b := range r.Bosses {
			if b.Defeated {
				was[b.EventID] = true
			}
		}
	}
	var names []string
	for _, r := range neo.Regions {
		for _, b := range r.Bosses {
			if b.Defeated && !was[b.EventID] {
				names = append(names, b.Name)
			}
		}
	}
	return names
}

func waitForChange(ch <-chan struct{}) tea.Cmd {
	return func() tea.Msg {
		if ch == nil {
			return nil
		}
		if _, ok := <-ch; !ok {
			return nil
		}
		return saveChangedMsg{}
	}
}

func toastTickCmd(seq int) tea.Cmd {
	return tea.Tick(toastDuration, func(time.Time) tea.Msg { return toastExpireMsg{seq: seq} })
}

func openURL(u string) {
	_ = exec.Command("xdg-open", u).Start()
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func max0(v int) int {
	if v < 0 {
		return 0
	}
	return v
}
