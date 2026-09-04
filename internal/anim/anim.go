// Package anim is the timing spine for the game's animations: a fixed-rate
// frame clock plus a few easing helpers.
//
// A Screen animates by advancing an integer frame counter one step per
// [TickMsg] and re-issuing [Tick] from its Update. All on-screen motion is then
// a pure function of that counter, so tests drive animation by feeding TickMsgs
// rather than sleeping.
package anim

import (
	"math"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// FPS is the animation frame rate. It is deliberately low: the art is coarse
// and a slower cadence keeps CPU use negligible.
const FPS = 15

// FrameInterval is the wall-clock gap between animation frames.
const FrameInterval = time.Second / FPS

// TickMsg is delivered once per animation frame. It carries the wall-clock time
// the frame fired; Screens that only count frames can ignore the payload.
type TickMsg struct {
	Time time.Time
}

// Tick returns a command that emits one [TickMsg] after a single
// [FrameInterval]. A Screen re-issues it from Update on every TickMsg to keep
// the animation running.
func Tick() tea.Cmd {
	return tea.Tick(FrameInterval, func(t time.Time) tea.Msg {
		return TickMsg{Time: t}
	})
}

// Elapsed converts a frame count into the wall-clock duration it represents at
// [FPS]. It bridges a Screen's frame counter and the duration-based helpers
// below.
func Elapsed(frames int) time.Duration {
	return time.Duration(frames) * FrameInterval
}

// Clamp01 constrains t to the [0, 1] interval.
func Clamp01(t float64) float64 {
	switch {
	case t < 0:
		return 0
	case t > 1:
		return 1
	default:
		return t
	}
}

// Lerp linearly interpolates from a to b as t runs 0..1. t is clamped, so
// values outside [0, 1] return the nearer endpoint.
func Lerp(a, b, t float64) float64 {
	return a + (b-a)*Clamp01(t)
}

// Triangle maps t (any real) onto a 0->1->0 ramp of period 1: it rises linearly
// over the first half of each period and falls over the second. Used for
// back-and-forth motion.
func Triangle(t float64) float64 {
	t -= math.Floor(t)
	if t < 0.5 {
		return t * 2
	}
	return 2 - t*2
}

// Pulse maps t (any real) onto a smooth 0->1->0 wave of period 1 using a raised
// cosine. Gentler at the extremes than [Triangle], so it reads as a breathing
// pulse rather than a bounce.
func Pulse(t float64) float64 {
	return 0.5 - 0.5*math.Cos(2*math.Pi*t)
}

// bobPattern is the ±1-row vertical wobble sequence shared by every
// character bob in the game: flat, up, flat, down.
var bobPattern = [4]int{0, -1, 0, 1}

// Bob returns the vertical bob offset for frame, advancing one step through
// bobPattern every framesPerStep frames. Used for the Welcome Screen's
// walking Character and the Next Screen's idling Baby.
func Bob(frame, framesPerStep int) int {
	return bobPattern[(frame/framesPerStep)%len(bobPattern)]
}
