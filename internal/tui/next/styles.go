package next

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/leekli/tamagotchi-go/internal/tui"
)

// petPanelWidth/Height and statsPanelWidth/Height are the Next Screen's PET
// and STATS panels' fixed outer dimensions (border and padding included),
// per docs/adr/0007: petPanel is sized to Adult's art (the widest of any
// Stage, 14 columns); statsPanel to the Hunger/Happiness Meter row (19
// characters, e.g. "Happiness  ▓▓▓▓ 4/4"). Both are 12 rows tall: STATS has 8
// content rows, of which today's content uses 6 and the rest are reserved
// blank for the Sick indicator and the Attention call. panelGap is the fixed
// spacing between them when shown side by side.
const (
	petPanelWidth  = 18
	petPanelHeight = 12

	statsPanelWidth  = 23
	statsPanelHeight = 12

	panelGap = "  "

	// deathPanelWidth/Height is the Death panel's fixed outer size, per
	// docs/adr/0007: driven by the Restart prompt's 39-character text, and
	// as tall as the hatched composite (petPanelHeight + gap + Icon bar area),
	// rather than leaving a discrepancy between mutually exclusive states.
	// Two of its 13 content rows are reserved blank for the cause of Death and
	// the Care mistake tally.
	deathPanelWidth  = 43
	deathPanelHeight = 17

	// tabFieldWidth is a tab's own label field width, sized to the longest
	// label across both the Feed/Play/Clean and Meal/Snack menus ("Clean"/
	// "Snack", 5 characters). tabWidth/tabHeight are the resulting fixed
	// outer size of every tab (border and 1-column horizontal padding, no
	// vertical padding), and iconRowWidth is the Icon bar's fixed total row
	// width, shared by both menus so switching between them never shifts
	// anything else on screen — all per docs/adr/0007. It is wide enough for
	// four tabs and their three gaps (4*9 + 3*2 = 42), the fourth being the
	// Cure tab; the three-tab bar and the two-tab chooser centre within it.
	tabFieldWidth = 5
	tabWidth      = 9
	tabHeight     = 3
	iconRowWidth  = 42

	// tabGap is the fixed spacing between adjacent tabs in the Icon bar.
	tabGap = "  "
)

// styles holds the Lip Gloss styles the Next Screen paints with, derived
// from the shared palette rather than inventing a new colour scheme.
type styles struct {
	petPanel   tui.Panel // frames the Pet's art and Stage label
	statsPanel tui.Panel // frames Hunger/Happiness/Mess/Age/Weight
	deathPanel tui.Panel // the single, uncaptioned panel shown once the Pet has died

	art        lipgloss.Style // the Pet's Egg/Baby/Child/Teen art
	stageLabel lipgloss.Style // the Stage label ("Egg"/"Baby"/"Child"/"Teen")
	meterGood  lipgloss.Style // Meter grading: 3-4 of pet.MaxStat
	meterFair  lipgloss.Style // Meter grading: 2
	meterLow   lipgloss.Style // Meter grading: 0-1
	info       lipgloss.Style // Age and Weight
	mess       lipgloss.Style // the Mess indicator glyph

	tabBorder   lipgloss.AdaptiveColor // every tab's border, selected or not
	tabNormal   lipgloss.Style         // an unselected tab's label
	tabSelected lipgloss.Style         // the currently selected tab's label
	flourish    lipgloss.Style         // a Care action's text feedback

	// restartDim/Mid/Hi are the restart chip's three pulse levels — the same
	// shared chip the Welcome Screen's begin prompt uses, sharing a constant
	// Accent background and differing only in foreground.
	restartDim lipgloss.Style
	restartMid lipgloss.Style
	restartHi  lipgloss.Style
}

func newStyles(p tui.Palette) styles {
	return styles{
		petPanel:   tui.Panel{Width: petPanelWidth, Height: petPanelHeight, Caption: "PET", PaddingX: 1, PaddingY: 1, Border: p.Dim},
		statsPanel: tui.Panel{Width: statsPanelWidth, Height: statsPanelHeight, Caption: "STATS", PaddingX: 1, PaddingY: 1, Border: p.Dim},
		deathPanel: tui.Panel{Width: deathPanelWidth, Height: deathPanelHeight, PaddingX: 1, PaddingY: 1, Border: p.Dim},

		art:        lipgloss.NewStyle().Foreground(p.Accent),
		stageLabel: lipgloss.NewStyle().Foreground(p.Dim),
		meterGood:  lipgloss.NewStyle().Foreground(p.Screen),
		meterFair:  lipgloss.NewStyle().Foreground(p.Amber),
		meterLow:   lipgloss.NewStyle().Foreground(p.Danger),
		info:       lipgloss.NewStyle().Foreground(p.Dim),
		mess:       lipgloss.NewStyle().Foreground(p.Danger),

		// A selected tab shares the exact "this is the pressable thing"
		// treatment the chip uses: a constant Accent fill and OnAccent text.
		tabBorder:   p.Dim,
		tabNormal:   lipgloss.NewStyle().Foreground(p.Dim),
		tabSelected: lipgloss.NewStyle().Background(p.Accent).Foreground(p.OnAccent).Bold(true),
		flourish:    lipgloss.NewStyle().Foreground(p.Highlight),

		restartDim: lipgloss.NewStyle().Background(p.Accent).Foreground(p.Dim),
		restartMid: lipgloss.NewStyle().Background(p.Accent).Foreground(p.Screen),
		restartHi:  lipgloss.NewStyle().Background(p.Accent).Foreground(p.OnAccent).Bold(true),
	}
}
