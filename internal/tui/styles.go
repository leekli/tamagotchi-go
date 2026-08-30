package tui

import "github.com/charmbracelet/lipgloss"

// Palette is the project's adaptive colour set, tuned to the 1990s toy. Each
// colour resolves differently on light and dark terminals; Lip Gloss degrades
// truecolor -> 256 -> 16 -> mono by terminal capability.
type Palette struct {
	Shell     lipgloss.AdaptiveColor // teal shell
	Highlight lipgloss.AdaptiveColor // shine-sweep band
	Accent    lipgloss.AdaptiveColor // pink-magenta
	Screen    lipgloss.AdaptiveColor // LCD grey-green
	Dim       lipgloss.AdaptiveColor // muted text
}

// DefaultPalette returns the standard palette.
func DefaultPalette() Palette {
	return Palette{
		Shell:     lipgloss.AdaptiveColor{Light: "#0f766e", Dark: "#2dd4bf"},
		Highlight: lipgloss.AdaptiveColor{Light: "#155e75", Dark: "#e0f2fe"},
		Accent:    lipgloss.AdaptiveColor{Light: "#be185d", Dark: "#f472b6"},
		Screen:    lipgloss.AdaptiveColor{Light: "#3f6212", Dark: "#a3e635"},
		Dim:       lipgloss.AdaptiveColor{Light: "#57534e", Dark: "#a8a29e"},
	}
}

// Styles bundles the reusable Lip Gloss styles derived from a Palette.
type Styles struct {
	HelpBar lipgloss.Style
	Notice  lipgloss.Style
}

// NewStyles builds the style set for a Palette.
func NewStyles(p Palette) Styles {
	return Styles{
		HelpBar: lipgloss.NewStyle().Foreground(p.Dim),
		Notice:  lipgloss.NewStyle().Foreground(p.Accent).Bold(true),
	}
}
