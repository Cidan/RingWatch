package watcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatcherFiresOnChange(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ER0000.sl2")
	if err := os.WriteFile(path, []byte("init"), 0o644); err != nil {
		t.Fatal(err)
	}
	w, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	time.Sleep(50 * time.Millisecond) // let the watcher start
	if err := os.WriteFile(path, []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}

	select {
	case <-w.Events():
	case <-time.After(3 * time.Second):
		t.Fatal("expected a debounced change event within 3s")
	}
}
