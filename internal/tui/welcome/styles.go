package welcome

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/leekli/tamagotchi-go/internal/tui"
)

// styles holds the Lip Gloss styles the Welcome Screen paints with, derived
// from the shared palette.
type styles struct {
	wordmark  lipgloss.Style // flat teal Wordmark
	shine     lipgloss.Style // brighter shine-sweep band
	character lipgloss.Style // the wandering Character
	promptDim lipgloss.Style // begin prompt, pulse low
	promptMid lipgloss.Style // begin prompt, pulse mid
	promptHi  lipgloss.Style // begin prompt, pulse high
}

func newStyles(p tui.Palette) styles {
	return styles{
		wordmark:  lipgloss.NewStyle().Foreground(p.Shell),
		shine:     lipgloss.NewStyle().Foreground(p.Highlight).Bold(true),
		character: lipgloss.NewStyle().Foreground(p.Accent),
		promptDim: lipgloss.NewStyle().Foreground(p.Dim),
		promptMid: lipgloss.NewStyle().Foreground(p.Shell),
		promptHi:  lipgloss.NewStyle().Foreground(p.Highlight).Bold(true),
	}
}
