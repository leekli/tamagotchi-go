package next

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/leekli/tamagotchi-go/internal/tui"
)

// styles holds the Lip Gloss styles the Next Screen paints with, derived
// from the shared palette rather than inventing a new colour scheme.
type styles struct {
	art          lipgloss.Style // the Pet's Egg/Baby art
	meter        lipgloss.Style // Hunger/Happiness pip meters
	info         lipgloss.Style // Age and Weight
	mess         lipgloss.Style // the Mess indicator glyph
	icon         lipgloss.Style // an unselected icon-bar entry
	iconSelected lipgloss.Style // the currently selected icon-bar entry
	flourish     lipgloss.Style // a Care action's text feedback
}

func newStyles(p tui.Palette) styles {
	return styles{
		art:          lipgloss.NewStyle().Foreground(p.Accent),
		meter:        lipgloss.NewStyle().Foreground(p.Screen),
		info:         lipgloss.NewStyle().Foreground(p.Dim),
		mess:         lipgloss.NewStyle().Foreground(p.Dim),
		icon:         lipgloss.NewStyle().Foreground(p.Dim),
		iconSelected: lipgloss.NewStyle().Foreground(p.Accent).Bold(true),
		flourish:     lipgloss.NewStyle().Foreground(p.Highlight),
	}
}
