package welcome

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/leekli/tamagotchi-go/internal/tui"
)

// TestChipStylesShareAConstantBackground proves the begin chip's pulse lives
// entirely in the foreground: all three levels fill with the same Accent
// background, matching the selected-tab convention elsewhere in the app, and
// only the foreground (and, at its brightest, weight) changes.
func TestChipStylesShareAConstantBackground(t *testing.T) {
	p := tui.DefaultPalette()
	s := newStyles(p)

	assert.Equal(t, p.Accent, s.chipDim.GetBackground())
	assert.Equal(t, p.Accent, s.chipMid.GetBackground())
	assert.Equal(t, p.Accent, s.chipBright.GetBackground())

	assert.Equal(t, p.Dim, s.chipDim.GetForeground())
	assert.Equal(t, p.Shell, s.chipMid.GetForeground())
	assert.Equal(t, p.OnAccent, s.chipBright.GetForeground())
	assert.True(t, s.chipBright.GetBold(), "the brightest pulse level should carry the weight, not just colour")
}

// TestCardUsesTheFixedDimensionsFromADR0007 locks the Welcome card's outer
// size to the values recorded in docs/adr/0007, so a future change to either
// is a deliberate edit here, not an accidental drift.
func TestCardUsesTheFixedDimensionsFromADR0007(t *testing.T) {
	s := newStyles(tui.DefaultPalette())
	assert.Equal(t, 75, s.card.Width)
	assert.Equal(t, 18, s.card.Height)
	assert.Empty(t, s.card.Caption, "the Welcome Screen's sole panel stays uncaptioned")
}
