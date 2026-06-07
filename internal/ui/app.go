package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/Cidan/RingWatch/internal/save"
	"github.com/Cidan/RingWatch/internal/watcher"
)

// Options configures a TUI run.
type Options struct {
	SavePath string
	Slot     int // -1 for auto (first active character)
	Watch    bool
}

// Run opens the save, optionally starts the file watcher, and runs the TUI.
func Run(opts Options) error {
	s, err := save.Open(opts.SavePath)
	if err != nil {
		return err
	}
	m := newModel(opts, s)

	if opts.Watch {
		w, werr := watcher.New(opts.SavePath)
		if werr == nil {
			m.changes = w.Events()
			defer w.Close()
		} else {
			m.watching = false
		}
	}

	// Alt-screen is enabled per-frame via the View's AltScreen field (see View).
	p := tea.NewProgram(m)
	_, err = p.Run()
	return err
}
