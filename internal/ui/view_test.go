package ui

import (
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/Cidan/RingWatch/internal/save"
)

const fixture = "../../repo/SavesTest/ER0000.sl2"

// ansiRe strips SGR color codes and OSC 8 hyperlink wrappers for readable logs.
var ansiRe = regexp.MustCompile("\x1b\\][0-9];;[^\x1b]*\x1b\\\\|\x1b\\[[0-9;]*m")

func strip(s string) string { return ansiRe.ReplaceAllString(s, "") }

func newTestModel(t *testing.T, w, h int) Model {
	t.Helper()
	s, err := save.Open(fixture)
	if err != nil {
		t.Skipf("fixture unavailable: %v", err)
	}
	m := newModel(Options{SavePath: fixture, Slot: -1, Watch: false}, s)
	nm, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return nm.(Model)
}

func TestViewRendersAtSize(t *testing.T) {
	m := newTestModel(t, 100, 30)
	v := m.View().Content
	if got := lipgloss.Height(v); got != 30 {
		t.Errorf("view height = %d, want 30", got)
	}
	plain := strip(v)
	for _, want := range []string{"RINGWATCH", "Limgrave", "EARLY GAME"} {
		if !strings.Contains(plain, want) {
			t.Errorf("rendered view missing %q", want)
		}
	}
	t.Logf("\n%s", plain)
}

func TestViewFilter(t *testing.T) {
	m := newTestModel(t, 100, 30)
	m.filter = "malenia"
	plain := strip(m.View().Content)
	if !strings.Contains(strings.ToLower(plain), "malenia") {
		t.Errorf("filtered view should list Malenia:\n%s", plain)
	}
	t.Logf("\n%s", plain)
}

func TestCharPickerView(t *testing.T) {
	m := newTestModel(t, 100, 30)
	m.mode = modePickChar
	plain := strip(m.View().Content)
	if !strings.Contains(plain, "Select Character") || !strings.Contains(plain, "Davosso") {
		t.Errorf("char picker missing content:\n%s", plain)
	}
	t.Logf("\n%s", plain)
}
