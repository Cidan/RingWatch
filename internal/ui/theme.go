package ui

import (
	"math"
	"strings"

	"charm.land/lipgloss/v2"
)

// Elden Ring-flavoured palette: tarnished gold on near-black, sage green for
// felled bosses, ember for accents.
var (
	colGold       = lipgloss.Color("#d8b25a")
	colGoldBright = lipgloss.Color("#f0d68c")
	colText       = lipgloss.Color("#e6dfca")
	colMuted      = lipgloss.Color("#8f8979")
	colDone       = lipgloss.Color("#8fb86a")
	colEmber      = lipgloss.Color("#cf7b34")
	colBorderDim  = lipgloss.Color("#4a4334")
	colSelBg      = lipgloss.Color("#332c1d")
	colBarEmpty   = lipgloss.Color("#3a3528")
)

var (
	titleStyle  = lipgloss.NewStyle().Foreground(colGoldBright).Bold(true)
	subtleStyle = lipgloss.NewStyle().Foreground(colMuted)
	ruleStyle   = lipgloss.NewStyle().Foreground(colBorderDim)
	doneStyle   = lipgloss.NewStyle().Foreground(colDone)
	goldStyle   = lipgloss.NewStyle().Foreground(colGold)
)

// truncate shortens s to at most max runes, adding an ellipsis if cut.
func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max == 1 {
		return "…"
	}
	return string(r[:max-1]) + "…"
}

// padRight pads s with spaces to a visible width of w (ANSI/OSC-8 aware).
func padRight(s string, w int) string {
	vw := lipgloss.Width(s)
	if vw >= w {
		return s
	}
	return s + strings.Repeat(" ", w-vw)
}

// lineLR places left and right on a single line of visible width w.
func lineLR(left, right string, w int) string {
	gap := max(w-lipgloss.Width(left)-lipgloss.Width(right), 1)
	return left + strings.Repeat(" ", gap) + right
}

// wrapText word-wraps s to lines of at most w columns (by rune count), preserving
// existing newlines. Used for multi-line quest-step instructions.
func wrapText(s string, w int) []string {
	if w <= 0 {
		return []string{s}
	}
	var lines []string
	for para := range strings.SplitSeq(s, "\n") {
		words := strings.Fields(para)
		if len(words) == 0 {
			lines = append(lines, "")
			continue
		}
		cur := words[0]
		for _, word := range words[1:] {
			if len([]rune(cur))+1+len([]rune(word)) <= w {
				cur += " " + word
			} else {
				lines = append(lines, cur)
				cur = word
			}
		}
		lines = append(lines, cur)
	}
	return lines
}

// progressBar renders a fixed-width bar; it turns green once complete.
func progressBar(done, total, width int) string {
	if width <= 0 {
		return ""
	}
	frac := 0.0
	if total > 0 {
		frac = float64(done) / float64(total)
	}
	filled := clamp(int(math.Round(frac*float64(width))), 0, width)
	col := colGold
	if total > 0 && done >= total {
		col = colDone
	}
	full := lipgloss.NewStyle().Foreground(col).Render(strings.Repeat("█", filled))
	empty := lipgloss.NewStyle().Foreground(colBarEmpty).Render(strings.Repeat("░", width-filled))
	return full + empty
}

func pct(done, total int) int {
	if total <= 0 {
		return 0
	}
	return int(math.Round(float64(done) / float64(total) * 100))
}
