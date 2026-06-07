// Package config locates Elden Ring save files on Linux, including saves stored
// in non-default Steam library folders (parsed from libraryfolders.vdf) and
// Flatpak Steam installs.
package config

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// eldenRingAppID is Elden Ring's Steam app id; its Proton prefix lives under
// steamapps/compatdata/<appid>/.
const eldenRingAppID = "1245620"

// SaveFile is a discovered save file.
type SaveFile struct {
	Path    string
	SteamID string // the numeric folder name containing ER0000.*
	CoOp    bool   // true for .co2 (Seamless Co-op)
}

// steamRoots returns existing candidate Steam installation roots on Linux.
func steamRoots() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	candidates := []string{
		filepath.Join(home, ".local/share/Steam"),
		filepath.Join(home, ".steam/steam"),
		filepath.Join(home, ".steam/root"),
		filepath.Join(home, ".var/app/com.valvesoftware.Steam/.local/share/Steam"),
	}
	var out []string
	seen := map[string]bool{}
	for _, c := range candidates {
		c = filepath.Clean(c)
		if seen[c] {
			continue
		}
		if fi, err := os.Stat(c); err == nil && fi.IsDir() {
			seen[c] = true
			out = append(out, c)
		}
	}
	return out
}

// libraryPaths returns every Steam library root reachable from a Steam install,
// parsed from its libraryfolders.vdf, always including the install root itself.
func libraryPaths(steamRoot string) []string {
	out := []string{steamRoot}
	for _, vdf := range []string{
		filepath.Join(steamRoot, "config/libraryfolders.vdf"),
		filepath.Join(steamRoot, "steamapps/libraryfolders.vdf"),
	} {
		data, err := os.ReadFile(vdf)
		if err != nil {
			continue
		}
		out = append(out, parseVDFPaths(string(data))...)
	}
	return out
}

// parseVDFPaths extracts the "path" values from a libraryfolders.vdf body.
func parseVDFPaths(s string) []string {
	var out []string
	for line := range strings.SplitSeq(s, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(strings.ToLower(line), `"path"`) {
			continue
		}
		// Format: "path"        "/games/SteamLibrary"
		_, after, ok := strings.Cut(line[len(`"path"`):], `"`)
		if !ok {
			continue
		}
		val, _, ok := strings.Cut(after, `"`)
		if !ok {
			continue
		}
		if p := strings.ReplaceAll(val, `\\`, "/"); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// DetectSaves discovers all Elden Ring save files across every Steam library.
func DetectSaves() []SaveFile {
	var libs []string
	seenLib := map[string]bool{}
	for _, root := range steamRoots() {
		for _, lib := range libraryPaths(root) {
			lib = filepath.Clean(lib)
			if !seenLib[lib] {
				seenLib[lib] = true
				libs = append(libs, lib)
			}
		}
	}

	var saves []SaveFile
	seenSave := map[string]bool{}
	add := func(path string) {
		path = filepath.Clean(path)
		if seenSave[path] {
			return
		}
		seenSave[path] = true
		saves = append(saves, SaveFile{
			Path:    path,
			SteamID: filepath.Base(filepath.Dir(path)),
			CoOp:    strings.HasSuffix(path, ".co2"),
		})
	}

	const tail = "pfx/drive_c/users/steamuser/AppData/Roaming/EldenRing/*/ER0000."
	for _, lib := range libs {
		base := filepath.Join(lib, "steamapps/compatdata")
		for _, ext := range []string{"sl2", "co2"} {
			// Standard Elden Ring prefix first, then any prefix (covers Seamless
			// Co-op launchers that use a different app id).
			for _, prefix := range []string{eldenRingAppID, "*"} {
				matches, _ := filepath.Glob(filepath.Join(base, prefix, tail+ext))
				for _, m := range matches {
					add(m)
				}
			}
		}
	}
	sort.Slice(saves, func(i, j int) bool { return saves[i].Path < saves[j].Path })
	return saves
}
