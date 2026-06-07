// Package locations builds external reference links for bosses and renders them
// as clickable OSC 8 terminal hyperlinks.
package locations

import "strings"

// Hyperlink wraps label in an OSC 8 terminal hyperlink to url. Terminals without
// OSC 8 support render just the label, and the escape sequences are zero-width so
// they don't disturb layout measurement.
func Hyperlink(url, label string) string {
	if url == "" {
		return label
	}
	return "\x1b]8;;" + url + "\x1b\\" + label + "\x1b]8;;\x1b\\"
}

const fextralifeBase = "https://eldenring.wiki.fextralife.com/"

// fextralifeOverrides maps boss names whose wiki slug differs from the default
// rule. It exists to be grown as exceptions surface.
var fextralifeOverrides = map[string]string{}

// Fextralife returns the Elden Ring Fextralife wiki URL for a boss name.
func Fextralife(bossName string) string {
	if s, ok := fextralifeOverrides[bossName]; ok {
		return fextralifeBase + s
	}
	return fextralifeBase + slug(bossName)
}

// slug converts a boss name to a Fextralife page slug: keep letters, digits,
// hyphens and apostrophes; collapse spaces to '+'; drop other punctuation.
func slug(name string) string {
	var b strings.Builder
	pendingPlus := false
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '\'':
			if pendingPlus {
				b.WriteByte('+')
				pendingPlus = false
			}
			b.WriteRune(r)
		case r == ' ':
			if b.Len() > 0 {
				pendingPlus = true
			}
		default:
			// drop commas, periods, colons, etc.
		}
	}
	return b.String()
}
