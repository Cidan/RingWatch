package config

import "testing"

func TestParseVDFPaths(t *testing.T) {
	sample := `"libraryfolders"
{
	"0"
	{
		"path"		"/home/user/.local/share/Steam"
	}
	"1"
	{
		"path"		"/games/SteamLibrary"
	}
}`
	got := parseVDFPaths(sample)
	want := []string{"/home/user/.local/share/Steam", "/games/SteamLibrary"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("path %d = %q, want %q", i, got[i], want[i])
		}
	}
}

// TestDetectSaves is informational — it reports what is found on this machine.
func TestDetectSaves(t *testing.T) {
	saves := DetectSaves()
	t.Logf("found %d save file(s)", len(saves))
	for _, s := range saves {
		t.Logf("  %s  (steamID=%s, coop=%v)", s.Path, s.SteamID, s.CoOp)
	}
}
