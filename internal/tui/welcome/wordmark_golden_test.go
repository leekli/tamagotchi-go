package welcome

import (
	"testing"

	"github.com/charmbracelet/x/exp/golden"
)

// TestWordmarkGoldenFrames locks the shine-sweep rendering at three points: the
// first frame (band at the left edge), a mid-sweep frame, and a static frame
// after the sweep has finished. Regenerate the fixtures after an intentional
// change with:
//
//	go test ./internal/tui/welcome -run TestWordmarkGoldenFrames -update
func TestWordmarkGoldenFrames(t *testing.T) {
	base, shine := fixedStyles()

	for name, frame := range map[string]int{
		"first": 0,
		"mid":   6,
		"final": 24,
	} {
		t.Run(name, func(t *testing.T) {
			golden.RequireEqual(t, []byte(renderWordmark(frame, base, shine)))
		})
	}
}
