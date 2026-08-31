package welcome

import (
	"io"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"
)

func TestPromptLevelSpansDimToBrightAcrossOnePulse(t *testing.T) {
	t.Parallel()

	seen := map[int]bool{}
	for f := 0; f <= promptFramesPerPulse; f++ {
		lvl := promptLevel(f)
		assert.GreaterOrEqual(t, lvl, 0)
		assert.LessOrEqual(t, lvl, 2)
		seen[lvl] = true
	}
	assert.True(t, seen[0], "the pulse dips to its dim floor")
	assert.True(t, seen[2], "and rises to its bright peak")
}

func TestPromptLevelIsPeriodic(t *testing.T) {
	t.Parallel()

	for f := 0; f < 3*promptFramesPerPulse; f++ {
		assert.Equal(t, promptLevel(f), promptLevel(f+promptFramesPerPulse),
			"the pulse repeats every promptFramesPerPulse frames")
	}
}

func TestPromptLevelPeaksMidPulse(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 0, promptLevel(0), "a fresh Screen starts at the dim floor")
	assert.Equal(t, 2, promptLevel(promptFramesPerPulse/2), "and is brightest halfway through")
}

func TestRenderPromptChangesStyleWithThePulse(t *testing.T) {
	t.Parallel()

	r := lipgloss.NewRenderer(io.Discard)
	r.SetColorProfile(termenv.ANSI256)
	dim := r.NewStyle().Foreground(lipgloss.Color("240"))
	mid := r.NewStyle().Foreground(lipgloss.Color("250"))
	bright := r.NewStyle().Foreground(lipgloss.Color("231")).Bold(true)

	low := renderPrompt(0, dim, mid, bright)
	high := renderPrompt(promptFramesPerPulse/2, dim, mid, bright)
	assert.NotEqual(t, low, high, "the prompt is styled differently at the pulse extremes")
	assert.Contains(t, low, promptText)
	assert.Contains(t, high, promptText)
}
