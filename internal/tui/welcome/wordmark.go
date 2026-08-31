package welcome

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/leekli/tamagotchi-go/internal/anim"
	"github.com/leekli/tamagotchi-go/internal/art"
)

// wordmarkRows is the embedded "Tamagotchi" Wordmark, loaded once at startup.
var wordmarkRows = art.MustLoad("wordmark.txt")

const (
	// sweepDuration is how long the shine sweep takes to cross the Wordmark. It
	// runs once, on Screen entry, then never again.
	sweepDuration = 800 * time.Millisecond
	// sweepBand is the width in columns of the brighter shine band.
	sweepBand = 3
)

// sweepStart returns the leftmost column of the shine band at the given elapsed
// time, and whether the sweep is still running. The band travels linearly from
// just off the left edge to just past the right edge over sweepDuration; after
// that the Wordmark renders in a single flat colour.
func sweepStart(elapsed time.Duration, width int) (col int, active bool) {
	if elapsed >= sweepDuration {
		return 0, false
	}
	t := float64(elapsed) / float64(sweepDuration)
	return int(anim.Lerp(-sweepBand, float64(width), t)), true
}

// renderWordmark draws the Wordmark for the given animation frame. Every glyph
// cell is painted in base; while the sweep is running, cells inside the shine
// band are painted in shine instead. Spaces are left untouched so the sweep
// only lights up the letters.
func renderWordmark(frame int, base, shine lipgloss.Style) string {
	start, active := sweepStart(anim.Elapsed(frame), art.Width(wordmarkRows))

	var b strings.Builder
	for i, row := range wordmarkRows {
		if i > 0 {
			b.WriteByte('\n')
		}
		for col, r := range []rune(row) {
			switch {
			case r == ' ':
				b.WriteByte(' ')
			case active && col >= start && col < start+sweepBand:
				b.WriteString(shine.Render(string(r)))
			default:
				b.WriteString(base.Render(string(r)))
			}
		}
	}
	return b.String()
}
