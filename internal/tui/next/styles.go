package next

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/leekli/tamagotchi-go/internal/tui"
)

// petPanelWidth/Height and statsPanelWidth/Height are the Next Screen's PET
// and STATS panels' fixed outer dimensions (border and padding included),
// per docs/adr/0007: petPanel is sized to Adult's art (the widest of any
// Stage, 14 columns); statsPanel to the Hunger/Happiness Meter row (19
// characters, e.g. "Happiness  ▓▓▓▓ 4/4"). panelGap is the fixed spacing
// between them when shown side by side.
const (
	petPanelWidth  = 18
	petPanelHeight = 10

	statsPanelWidth  = 23
	statsPanelHeight = 10

	panelGap = "  "
)

// styles holds the Lip Gloss styles the Next Screen paints with, derived
// from the shared palette rather than inventing a new colour scheme.
type styles struct {
	petPanel   tui.Panel // frames the Pet's art and Stage label
	statsPanel tui.Panel // frames Hunger/Happiness/Mess/Age/Weight

	art        lipgloss.Style // the Pet's Egg/Baby/Child/Teen art
	stageLabel lipgloss.Style // the Stage label ("Egg"/"Baby"/"Child"/"Teen")
	meterGood  lipgloss.Style // Meter grading: 3-4 of pet.MaxStat
	meterFair  lipgloss.Style // Meter grading: 2
	meterLow   lipgloss.Style // Meter grading: 0-1
	info       lipgloss.Style // Age and Weight
	mess       lipgloss.Style // the Mess indicator glyph

	icon         lipgloss.Style // an unselected icon-bar entry
	iconSelected lipgloss.Style // the currently selected icon-bar entry
	flourish     lipgloss.Style // a Care action's text feedback

	// restartDim/Mid/Hi are the restart prompt's three pulse levels, shown
	// once the Pet has reached Death — the same dim -> mid -> bright
	// progression the Welcome Screen's begin prompt already uses.
	restartDim lipgloss.Style
	restartMid lipgloss.Style
	restartHi  lipgloss.Style
}

func newStyles(p tui.Palette) styles {
	return styles{
		petPanel:   tui.Panel{Width: petPanelWidth, Height: petPanelHeight, Caption: "PET", Border: p.Dim},
		statsPanel: tui.Panel{Width: statsPanelWidth, Height: statsPanelHeight, Caption: "STATS", Border: p.Dim},

		art:        lipgloss.NewStyle().Foreground(p.Accent),
		stageLabel: lipgloss.NewStyle().Foreground(p.Dim),
		meterGood:  lipgloss.NewStyle().Foreground(p.Screen),
		meterFair:  lipgloss.NewStyle().Foreground(p.Amber),
		meterLow:   lipgloss.NewStyle().Foreground(p.Danger),
		info:       lipgloss.NewStyle().Foreground(p.Dim),
		mess:       lipgloss.NewStyle().Foreground(p.Danger),

		icon:         lipgloss.NewStyle().Foreground(p.Dim),
		iconSelected: lipgloss.NewStyle().Foreground(p.Accent).Bold(true),
		flourish:     lipgloss.NewStyle().Foreground(p.Highlight),

		restartDim: lipgloss.NewStyle().Foreground(p.Dim),
		restartMid: lipgloss.NewStyle().Foreground(p.Screen),
		restartHi:  lipgloss.NewStyle().Foreground(p.Highlight).Bold(true),
	}
}
