package welcome

import (
	"io"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"

	"github.com/leekli/tamagotchi-go/internal/anim"
	"github.com/leekli/tamagotchi-go/internal/art"
)

// fixedStyles returns base/shine styles on a renderer pinned to a colour
// profile, so sweep frames differ by escape codes no matter the test host.
func fixedStyles() (base, shine lipgloss.Style) {
	r := lipgloss.NewRenderer(io.Discard)
	r.SetColorProfile(termenv.ANSI256)
	return r.NewStyle().Foreground(lipgloss.Color("30")),
		r.NewStyle().Foreground(lipgloss.Color("159")).Bold(true)
}

func TestSweepStartCrossesLeftToRightThenStops(t *testing.T) {
	t.Parallel()

	const width = 60

	col0, active0 := sweepStart(0, width)
	assert.True(t, active0)
	assert.Equal(t, -sweepBand, col0, "the band starts just off the left edge")

	// Monotonic non-decreasing across the run.
	prev := col0 - 1
	for e := time.Duration(0); e < sweepDuration; e += 20 * time.Millisecond {
		col, active := sweepStart(e, width)
		assert.True(t, active)
		assert.GreaterOrEqual(t, col, prev)
		prev = col
	}

	colEnd, activeEnd := sweepStart(sweepDuration, width)
	assert.False(t, activeEnd, "the sweep is done once its duration elapses")
	assert.Equal(t, 0, colEnd)

	_, activePast := sweepStart(2*sweepDuration, width)
	assert.False(t, activePast, "the sweep never runs a second time")
}

func TestSweepReachesPastTheRightEdge(t *testing.T) {
	t.Parallel()

	const width = 60
	col, active := sweepStart(sweepDuration-time.Millisecond, width)
	assert.True(t, active)
	assert.GreaterOrEqual(t, col, width-sweepBand, "the band clears the last column before it stops")
}

func TestRenderWordmarkKeepsEveryGlyphRow(t *testing.T) {
	t.Parallel()

	base, shine := fixedStyles()
	out := renderWordmark(0, base, shine)
	assert.Equal(t, len(wordmarkRows), strings.Count(out, "\n")+1)
}

func TestRenderWordmarkEmitsASolidRectangle(t *testing.T) {
	t.Parallel()

	// Every rendered row must be exactly the Wordmark width. A ragged block is
	// re-centred row-by-row by lipgloss.JoinVertical and the vertical strokes
	// drift, making the word unreadable.
	base, shine := fixedStyles()
	want := art.Width(wordmarkRows)
	for _, frame := range []int{0, 6, 40} {
		for i, line := range strings.Split(renderWordmark(frame, base, shine), "\n") {
			assert.Equal(t, want, lipgloss.Width(line), "frame %d row %d width", frame, i)
		}
	}
}

func TestWordmarkSourceRowsAreEqualWidth(t *testing.T) {
	t.Parallel()

	want := len([]rune(wordmarkRows[0]))
	for i, row := range wordmarkRows {
		assert.Equal(t, want, len([]rune(row)), "wordmark.txt row %d", i)
	}
}

func TestRenderWordmarkIsAPureFunctionOfFrame(t *testing.T) {
	t.Parallel()

	base, shine := fixedStyles()
	assert.Equal(t, renderWordmark(3, base, shine), renderWordmark(3, base, shine))
}

func TestRenderWordmarkPaintsTheShineBandWhileSweeping(t *testing.T) {
	t.Parallel()

	base, shine := fixedStyles()
	// Frame 3 is ~200ms in, mid-sweep; frame 40 is ~2.6s in, long static.
	assert.NotEqual(t, renderWordmark(3, base, shine), renderWordmark(40, base, shine))

	// Mid-sweep, swapping the shine style in changes the output: some cells
	// are painted with it.
	assert.NotEqual(t, renderWordmark(3, base, base), renderWordmark(3, base, shine),
		"a mid-sweep frame carries shine-styled cells")

	// Once static, the shine style is never used, so swapping it makes no
	// difference.
	assert.Equal(t, renderWordmark(40, base, base), renderWordmark(40, base, shine),
		"a static frame is a single flat colour")
}

func TestSweepAlignsWithFrameElapsed(t *testing.T) {
	t.Parallel()

	// The sweep is finished by the frame whose elapsed time first reaches
	// sweepDuration.
	done := int(sweepDuration/anim.FrameInterval) + 1
	_, active := sweepStart(anim.Elapsed(done), art.Width(wordmarkRows))
	assert.False(t, active)

	_, active = sweepStart(anim.Elapsed(done-2), art.Width(wordmarkRows))
	assert.True(t, active)
}
