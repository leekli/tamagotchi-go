package welcome

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/leekli/tamagotchi-go/internal/tui"
)

// cardWidth and cardHeight are the Welcome Screen's card's fixed outer
// dimensions (border and padding included), per docs/adr/0007. They hold
// regardless of what the wordmark, Character, or chip actually render.
const (
	cardWidth  = 75
	cardHeight = 18
)

// styles holds the Lip Gloss styles the Welcome Screen paints with, derived
// from the shared palette.
type styles struct {
	wordmark   lipgloss.Style // flat teal Wordmark
	shine      lipgloss.Style // brighter shine-sweep band
	character  lipgloss.Style // the wandering Character
	chipDim    lipgloss.Style // begin chip, pulse low
	chipMid    lipgloss.Style // begin chip, pulse mid
	chipBright lipgloss.Style // begin chip, pulse high
	card       tui.Panel      // the uncaptioned card framing the whole Screen
}

func newStyles(p tui.Palette) styles {
	return styles{
		wordmark:  lipgloss.NewStyle().Foreground(p.Shell),
		shine:     lipgloss.NewStyle().Foreground(p.Highlight).Bold(true),
		character: lipgloss.NewStyle().Foreground(p.Accent),
		// The chip's three pulse levels share a constant Accent background —
		// the same "this is the pressable thing" fill a selected Icon bar
		// tab uses — and differ only in foreground, continuing the exact
		// dim/mid/bright tokens the old bare-text prompt already used.
		chipDim:    lipgloss.NewStyle().Background(p.Accent).Foreground(p.Dim),
		chipMid:    lipgloss.NewStyle().Background(p.Accent).Foreground(p.Shell),
		chipBright: lipgloss.NewStyle().Background(p.Accent).Foreground(p.OnAccent).Bold(true),
		card:       tui.Panel{Width: cardWidth, Height: cardHeight, Border: p.Dim},
	}
}
