package locations

import (
	"strings"
	"testing"
)

func TestFextralifeSlug(t *testing.T) {
	cases := map[string]string{
		"Margit, the Fell Omen":      fextralifeBase + "Margit+the+Fell+Omen",
		"Godrick the Grafted":        fextralifeBase + "Godrick+the+Grafted",
		"Malenia, Blade of Miquella": fextralifeBase + "Malenia+Blade+of+Miquella",
		"Commander O'Neil":           fextralifeBase + "Commander+O'Neil",
	}
	for name, want := range cases {
		if got := Fextralife(name); got != want {
			t.Errorf("Fextralife(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestHyperlink(t *testing.T) {
	out := Hyperlink("https://example.com", "click")
	if !strings.Contains(out, "https://example.com") || !strings.Contains(out, "click") {
		t.Errorf("hyperlink missing url/label: %q", out)
	}
	if !strings.HasPrefix(out, "\x1b]8;;") {
		t.Errorf("hyperlink missing OSC 8 prefix: %q", out)
	}
	if Hyperlink("", "plain") != "plain" {
		t.Error("empty url should return plain label")
	}
}
