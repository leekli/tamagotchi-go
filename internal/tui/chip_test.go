package tui_test

import (
	"io"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"

	"github.com/leekli/tamagotchi-go/internal/tui"
)

const testFramesPerPulse = 15

func TestPulseLevelSpansDimToBrightAcrossOnePulse(t *testing.T) {
	t.Parallel()

	seen := map[int]bool{}
	for f := 0; f <= testFramesPerPulse; f++ {
		lvl := tui.PulseLevel(f, testFramesPerPulse)
		assert.GreaterOrEqual(t, lvl, 0)
		assert.LessOrEqual(t, lvl, 2)
		seen[lvl] = true
	}
	assert.True(t, seen[0], "the pulse dips to its dim floor")
	assert.True(t, seen[2], "and rises to its bright peak")
}

func TestPulseLevelIsPeriodic(t *testing.T) {
	t.Parallel()

	for f := 0; f < 3*testFramesPerPulse; f++ {
		assert.Equal(t, tui.PulseLevel(f, testFramesPerPulse), tui.PulseLevel(f+testFramesPerPulse, testFramesPerPulse),
			"the pulse repeats every framesPerPulse frames")
	}
}

func TestPulseLevelPeaksMidPulse(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 0, tui.PulseLevel(0, testFramesPerPulse), "a fresh pulse starts at the dim floor")
	assert.Equal(t, 2, tui.PulseLevel(testFramesPerPulse/2, testFramesPerPulse), "and is brightest halfway through")
}

// chipStyles builds a throwaway dim/mid/bright style triple sharing one
// constant background, the shape every real chip style set follows: only the
// foreground differs between pulse levels.
func chipStyles() (dim, mid, bright lipgloss.Style) {
	r := lipgloss.NewRenderer(io.Discard)
	r.SetColorProfile(termenv.ANSI256)
	bg := lipgloss.Color("205")
	return r.NewStyle().Background(bg).Foreground(lipgloss.Color("240")),
		r.NewStyle().Background(bg).Foreground(lipgloss.Color("250")),
		r.NewStyle().Background(bg).Foreground(lipgloss.Color("231")).Bold(true)
}

func TestRenderChipChangesStyleWithThePulse(t *testing.T) {
	t.Parallel()

	dim, mid, bright := chipStyles()
	low := tui.RenderChip("Press Enter", 0, testFramesPerPulse, dim, mid, bright, "")
	high := tui.RenderChip("Press Enter", testFramesPerPulse/2, testFramesPerPulse, dim, mid, bright, "")

	assert.NotEqual(t, low, high, "the chip is styled differently at the pulse extremes")
	assert.Contains(t, low, "Press Enter")
	assert.Contains(t, high, "Press Enter")
}

func TestRenderChipCarriesItsTextRegardlessOfZoneID(t *testing.T) {
	t.Parallel()

	dim, mid, bright := chipStyles()
	rendered := tui.RenderChip("Press Enter", 0, testFramesPerPulse, dim, mid, bright, "some.zone")
	assert.Contains(t, rendered, "Press Enter")
}
