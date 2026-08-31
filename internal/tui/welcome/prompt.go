package welcome

import (
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"

	"github.com/leekli/tamagotchi-go/internal/anim"
)

// promptText is the begin prompt.
const promptText = "Press Enter or click to begin"

// promptFramesPerPulse is one full dim -> bright -> dim cycle of the prompt, in
// animation frames (~1s at anim.FPS).
const promptFramesPerPulse = anim.FPS

// promptLevel maps the animation frame to a brightness step: 0 dim, 1 mid, 2
// bright. It follows a smooth raised-cosine pulse, quantised into three bands
// so the change is legible without relying on colour blending.
func promptLevel(frame int) int {
	phase := float64(frame%promptFramesPerPulse) / float64(promptFramesPerPulse)
	switch v := anim.Pulse(phase); {
	case v < 1.0/3.0:
		return 0
	case v < 2.0/3.0:
		return 1
	default:
		return 2
	}
}

// renderPrompt styles the begin prompt for the given frame and wraps it in its
// bubblezone marker, so a click can be tested against exactly the prompt's
// cells.
func renderPrompt(frame int, dim, mid, bright lipgloss.Style) string {
	style := dim
	switch promptLevel(frame) {
	case 1:
		style = mid
	case 2:
		style = bright
	}

	rendered := style.Render(promptText)
	if zone.DefaultManager == nil {
		return rendered
	}
	return zone.Mark(BeginZoneID, rendered)
}
