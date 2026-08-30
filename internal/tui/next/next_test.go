package next_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/leekli/tamagotchi-go/internal/tui"
	"github.com/leekli/tamagotchi-go/internal/tui/next"
)

func TestID(t *testing.T) {
	t.Parallel()
	assert.Equal(t, tui.NextScreenID, next.New().ID())
}

func TestInitReturnsNoCommand(t *testing.T) {
	t.Parallel()
	assert.Nil(t, next.New().Init())
}

func TestUpdateIgnoresInputAndStaysPut(t *testing.T) {
	t.Parallel()

	s, cmd := next.New().Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd)
	assert.Equal(t, tui.NextScreenID, s.ID())
}

func TestViewShowsPlaceholderTextSizedToArea(t *testing.T) {
	t.Parallel()

	s := next.New()
	s, _ = s.Update(tea.WindowSizeMsg{Width: 80, Height: 23})

	view := s.View()
	assert.Contains(t, view, "Nothing here yet. Ctrl+Q to quit.")

	lines := 1
	for _, r := range view {
		if r == '\n' {
			lines++
		}
	}
	require.Equal(t, 23, lines, "placeholder should fill the body area it was given")
}

func TestScreenIsScrollable(t *testing.T) {
	t.Parallel()
	assert.True(t, next.New().Scrollable())
}
