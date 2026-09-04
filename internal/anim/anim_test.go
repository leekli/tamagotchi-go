package anim_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/leekli/tamagotchi-go/internal/anim"
)

func TestFrameIntervalMatchesFPS(t *testing.T) {
	t.Parallel()
	assert.Equal(t, time.Second/anim.FPS, anim.FrameInterval)
}

func TestTickEmitsATickMsgAfterOneInterval(t *testing.T) {
	t.Parallel()

	cmd := anim.Tick()
	require.NotNil(t, cmd)

	start := time.Now()
	msg := cmd()
	elapsed := time.Since(start)

	tick, ok := msg.(anim.TickMsg)
	require.True(t, ok, "expected anim.TickMsg, got %T", msg)
	assert.False(t, tick.Time.IsZero(), "tick should carry a timestamp")
	assert.GreaterOrEqual(t, elapsed, anim.FrameInterval/2, "tick should wait roughly one frame")
}

func TestElapsedScalesFramesByInterval(t *testing.T) {
	t.Parallel()

	assert.Equal(t, time.Duration(0), anim.Elapsed(0))
	assert.Equal(t, anim.FrameInterval, anim.Elapsed(1))
	assert.Equal(t, 15*anim.FrameInterval, anim.Elapsed(15))
}

func TestClamp01(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 0.0, anim.Clamp01(-0.5))
	assert.Equal(t, 0.25, anim.Clamp01(0.25))
	assert.Equal(t, 1.0, anim.Clamp01(1.5))
}

func TestLerp(t *testing.T) {
	t.Parallel()

	assert.InDelta(t, 10.0, anim.Lerp(10, 20, 0), 1e-9)
	assert.InDelta(t, 15.0, anim.Lerp(10, 20, 0.5), 1e-9)
	assert.InDelta(t, 20.0, anim.Lerp(10, 20, 1), 1e-9)
	// t is clamped at both ends.
	assert.InDelta(t, 10.0, anim.Lerp(10, 20, -1), 1e-9)
	assert.InDelta(t, 20.0, anim.Lerp(10, 20, 2), 1e-9)
}

func TestTriangleRisesThenFalls(t *testing.T) {
	t.Parallel()

	assert.InDelta(t, 0.0, anim.Triangle(0), 1e-9)
	assert.InDelta(t, 0.5, anim.Triangle(0.25), 1e-9)
	assert.InDelta(t, 1.0, anim.Triangle(0.5), 1e-9)
	assert.InDelta(t, 0.5, anim.Triangle(0.75), 1e-9)
	assert.InDelta(t, 0.0, anim.Triangle(1), 1e-9)
	// Periodic: t and t+1 agree.
	assert.InDelta(t, anim.Triangle(0.3), anim.Triangle(1.3), 1e-9)
	// Never leaves [0, 1].
	for x := -3.0; x <= 3.0; x += 0.017 {
		v := anim.Triangle(x)
		assert.GreaterOrEqual(t, v, 0.0)
		assert.LessOrEqual(t, v, 1.0)
	}
}

func TestBobCyclesThroughFlatUpFlatDown(t *testing.T) {
	t.Parallel()

	const framesPerStep = 4
	got := make([]int, 8)
	for f := 0; f < 8; f++ {
		got[f] = anim.Bob(f*framesPerStep, framesPerStep)
	}
	assert.Equal(t, []int{0, -1, 0, 1, 0, -1, 0, 1}, got)
}

func TestPulseIsASmoothRaisedCosine(t *testing.T) {
	t.Parallel()

	assert.InDelta(t, 0.0, anim.Pulse(0), 1e-9)
	assert.InDelta(t, 1.0, anim.Pulse(0.5), 1e-9)
	assert.InDelta(t, 0.0, anim.Pulse(1), 1e-9)
	// Symmetric about the midpoint.
	assert.InDelta(t, anim.Pulse(0.25), anim.Pulse(0.75), 1e-9)
	// Flatter near the extremes than a triangle wave.
	assert.Less(t, anim.Pulse(0.05), anim.Triangle(0.05))
	for x := -2.0; x <= 2.0; x += 0.013 {
		v := anim.Pulse(x)
		assert.GreaterOrEqual(t, v, -1e-9)
		assert.LessOrEqual(t, v, 1.0+1e-9)
	}
}
