package welcome

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/leekli/tamagotchi-go/internal/art"
)

func charWidth() int { return art.Width(marutchiRight[0]) }

func TestCharStartsCentredFacingRight(t *testing.T) {
	t.Parallel()

	x, dir, bob, step := charState(0)
	span := charBoxWidth - charWidth()
	assert.Equal(t, span/2, x, "the Character starts in the middle of the box")
	assert.Equal(t, facingRight, dir, "and heads right first")
	assert.Equal(t, 0, bob)
	assert.Equal(t, 0, step)
}

func TestCharWalksRightThenTurnsAround(t *testing.T) {
	t.Parallel()

	x0, _, _, _ := charState(0)
	x1, dir1, _, _ := charState(6)
	assert.Greater(t, x1, x0, "position increases while heading right")
	assert.Equal(t, facingRight, dir1)

	turnedLeft := false
	for f := 0; f < 400 && !turnedLeft; f++ {
		if _, dir, _, _ := charState(f); dir == facingLeft {
			turnedLeft = true
		}
	}
	assert.True(t, turnedLeft, "the Character eventually faces left")
}

func TestCharNeverLeavesTheBox(t *testing.T) {
	t.Parallel()

	w := charWidth()
	for f := 0; f < 4000; f++ {
		x, _, _, _ := charState(f)
		assert.GreaterOrEqual(t, x, 0)
		assert.LessOrEqual(t, x+w, charBoxWidth)
	}
}

func TestCharBobIsAtMostOneRow(t *testing.T) {
	t.Parallel()

	seen := map[int]bool{}
	for f := 0; f < 120; f++ {
		_, _, bob, _ := charState(f)
		assert.GreaterOrEqual(t, bob, -1)
		assert.LessOrEqual(t, bob, 1)
		seen[bob] = true
	}
	assert.True(t, seen[-1] && seen[0] && seen[1], "the bob uses its full one-row range")
}

func TestCharWalkFrameAlternates(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 0, mustStep(t, 0))
	assert.Equal(t, 1, mustStep(t, charFramesPerCell))
	assert.Equal(t, 0, mustStep(t, 2*charFramesPerCell))
}

func mustStep(t *testing.T, frame int) int {
	t.Helper()
	_, _, _, step := charState(frame)
	return step
}

func TestCharStateIsAPureFunctionOfFrame(t *testing.T) {
	t.Parallel()

	for _, f := range []int{0, 1, 7, 33, 128, 999} {
		a1, a2, a3, a4 := charState(f)
		b1, b2, b3, b4 := charState(f)
		assert.Equal(t, [4]int{a1, int(a2), a3, a4}, [4]int{b1, int(b2), b3, b4})
	}
}

func TestRenderCharBoxHasFixedDimensions(t *testing.T) {
	t.Parallel()

	style := lipgloss.NewStyle()
	for _, f := range []int{0, 5, 13, 27, 61, 140} {
		box := renderCharBox(f, style)
		assert.Equal(t, charBoxHeight, strings.Count(box, "\n")+1, "frame %d: height is fixed", f)
		for _, line := range strings.Split(box, "\n") {
			assert.LessOrEqual(t, lipgloss.Width(line), charBoxWidth, "frame %d: width is fixed", f)
		}
	}
}

func TestRenderCharBoxMovesTheCharacterOverTime(t *testing.T) {
	t.Parallel()

	style := lipgloss.NewStyle()
	assert.NotEqual(t, renderCharBox(0, style), renderCharBox(8, style))
}

func TestRenderCharBoxFacesLeftWhenHeadingLeft(t *testing.T) {
	t.Parallel()

	// Find a frame where the Character faces left, then check the box shows
	// the mirrored art.
	frame := -1
	for f := 0; f < 400; f++ {
		if _, dir, _, _ := charState(f); dir == facingLeft {
			frame = f
			break
		}
	}
	require.GreaterOrEqual(t, frame, 0)

	box := renderCharBox(frame, lipgloss.NewStyle())
	// The distinctive mirrored foot glyph appears only in the left-facing art.
	assert.Contains(t, box, "'-`<")
}
