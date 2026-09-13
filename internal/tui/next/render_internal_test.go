package next

import (
	"io"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"

	"github.com/leekli/tamagotchi-go/internal/pet"
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
