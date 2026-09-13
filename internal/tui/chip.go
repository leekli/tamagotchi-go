package tui

import (
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"

	"github.com/leekli/tamagotchi-go/internal/anim"
)

// PulseLevel maps frame to a 3-step brightness level (0 dim, 1 mid, 2
// bright) using a raised-cosine pulse over framesPerPulse frames, quantised
// into three bands so the change is legible without relying on colour
// blending. This is the shared timing behind every pulsing prompt in the
// app — previously duplicated between the Welcome Screen's begin prompt and
// the Next Screen's restart prompt.
func PulseLevel(frame, framesPerPulse int) int {
	phase := float64(frame%framesPerPulse) / float64(framesPerPulse)
	switch v := anim.Pulse(phase); {
	case v < 1.0/3.0:
		return 0
	case v < 2.0/3.0:
		return 1
	default:
		return 2
	}
}

// RenderChip renders text as a single-row, filled affordance — the shared
// "press this" component behind the Begin and Restart prompts. dim, mid, and
// bright are already-built styles sharing one constant background, differing
// only in foreground; PulseLevel picks which applies for frame. If zoneID is
// non-empty and a bubblezone manager is active, the result is wrapped in a
// zone mark so a click can be tested against exactly its cells.
func RenderChip(text string, frame, framesPerPulse int, dim, mid, bright lipgloss.Style, zoneID string) string {
	style := dim
	switch PulseLevel(frame, framesPerPulse) {
	case 1:
		style = mid
	case 2:
		style = bright
	}

	rendered := style.Render(text)
	if zoneID == "" || zone.DefaultManager == nil {
		return rendered
	}
	return zone.Mark(zoneID, rendered)
}
