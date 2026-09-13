package next

import (
	"io"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/leekli/tamagotchi-go/internal/pet"
	"github.com/leekli/tamagotchi-go/internal/tui"
)

func TestMeterBarRendersExactlyMaxStatSegments(t *testing.T) {
	t.Parallel()

	tests := map[int]string{
		0: "░░░░",
		1: "▓░░░",
		2: "▓▓░░",
		3: "▓▓▓░",
		4: "▓▓▓▓",
	}
	for value, want := range tests {
		assert.Equal(t, want, meterBar(value), "value %d", value)
	}
}

func TestMeterStyleGradesByValueAgainstMaxStat(t *testing.T) {
	t.Parallel()

	r := lipgloss.NewRenderer(io.Discard)
	r.SetColorProfile(termenv.ANSI256)
	good := r.NewStyle().Foreground(lipgloss.Color("2"))
	fair := r.NewStyle().Foreground(lipgloss.Color("3"))
	low := r.NewStyle().Foreground(lipgloss.Color("1"))

	// 3-4 of pet.MaxStat (4) grade good, 2 grades fair, 0-1 grade low.
	assert.Equal(t, good, meterStyle(pet.MaxStat, good, fair, low))
	assert.Equal(t, good, meterStyle(pet.MaxStat-1, good, fair, low))
	assert.Equal(t, fair, meterStyle(2, good, fair, low))
	assert.Equal(t, low, meterStyle(1, good, fair, low))
	assert.Equal(t, low, meterStyle(0, good, fair, low))
}

func TestRenderMeterFormatsLabelBarAndFraction(t *testing.T) {
	t.Parallel()

	good := lipgloss.NewStyle()
	fair := lipgloss.NewStyle()
	low := lipgloss.NewStyle()

	assert.Equal(t, "Hunger     ▓▓▓░ 3/4", renderMeter("Hunger", 3, good, fair, low))
	assert.Equal(t, "Happiness  ▓▓▓▓ 4/4", renderMeter("Happiness", 4, good, fair, low))
}

func TestRenderTabHasFixedDimensionsRegardlessOfLabelLength(t *testing.T) {
	t.Parallel()

	normal := lipgloss.NewStyle()
	selected := lipgloss.NewStyle()
	border := tui.DefaultPalette().Dim

	for _, label := range []string{"Feed", "Play", "Clean", "Meal", "Snack"} {
		tab := renderTab(label, false, normal, selected, border)
		lines := strings.Split(tab, "\n")
		require.Len(t, lines, tabHeight, "label %q", label)
		for i, line := range lines {
			assert.Equal(t, tabWidth, lipgloss.Width(line), "label %q line %d", label, i)
		}
		assert.Contains(t, lines[1], label)
	}
}

func TestRenderTabSelectedUsesTheFilledStyle(t *testing.T) {
	t.Parallel()

	r := lipgloss.NewRenderer(io.Discard)
	r.SetColorProfile(termenv.ANSI256)
	normal := r.NewStyle().Foreground(lipgloss.Color("240"))
	selected := r.NewStyle().Background(lipgloss.Color("205")).Foreground(lipgloss.Color("0")).Bold(true)
	border := tui.DefaultPalette().Dim

	unselectedTab := renderTab("Feed", false, normal, selected, border)
	selectedTab := renderTab("Feed", true, normal, selected, border)
	assert.NotEqual(t, unselectedTab, selectedTab)
}

func TestRenderIconBarRowHasItsFixedWidthForBothMenus(t *testing.T) {
	t.Parallel()

	normal := lipgloss.NewStyle()
	selected := lipgloss.NewStyle()
	border := tui.DefaultPalette().Dim

	threeTabRow := renderIconBar(0, normal, selected, border)
	twoTabRow := renderFeedChoice(0, normal, selected, border)

	for _, row := range []string{threeTabRow, twoTabRow} {
		for _, line := range strings.Split(row, "\n") {
			assert.Equal(t, iconRowWidth, lipgloss.Width(line))
		}
	}
}
