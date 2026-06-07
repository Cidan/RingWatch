// Package watcher watches a save file for changes and emits debounced change
// signals. It watches the containing directory (not the file) so it survives the
// atomic write/replace and .bak churn the game performs on save.
package watcher

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// debounce coalesces the burst of writes a single in-game save produces.
const debounce = 400 * time.Millisecond

// Watcher emits a signal on Events() shortly after the watched save changes.
type Watcher struct {
	fw   *fsnotify.Watcher
	out  chan struct{}
	done chan struct{}
}

// New starts watching the directory of savePath for changes to that save.
func New(savePath string) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	dir := filepath.Dir(savePath)
	if err := fw.Add(dir); err != nil {
		fw.Close()
		return nil, err
	}
	w := &Watcher{
		fw:   fw,
		out:  make(chan struct{}, 1),
		done: make(chan struct{}),
	}
	go w.loop(filepath.Base(savePath))
	return w, nil
}

// Events returns a channel that receives a value shortly after each change.
func (w *Watcher) Events() <-chan struct{} { return w.out }

// Close stops watching and releases resources.
func (w *Watcher) Close() error {
	close(w.done)
	return w.fw.Close()
}

func (w *Watcher) loop(base string) {
	timer := time.NewTimer(debounce)
	timer.Stop()
	pending := false
	for {
		select {
		case <-w.done:
			timer.Stop()
			return
		case ev, ok := <-w.fw.Events:
			if !ok {
				return
			}
			// Match the save itself and its sidecar/temp files (e.g. ER0000.sl2.bak).
			if name := filepath.Base(ev.Name); name == base || strings.HasPrefix(name, base) {
				if !pending {
					pending = true
				}
				timer.Reset(debounce)
			}
		case <-timer.C:
			pending = false
			select {
			case w.out <- struct{}{}:
			default: // a signal is already queued; coalesce
			}
		case <-w.fw.Errors:
			// transient watcher errors are non-fatal
		}
	}
}
