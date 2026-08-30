package welcome_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/leekli/tamagotchi-go/internal/tui"
	"github.com/leekli/tamagotchi-go/internal/tui/welcome"
)

func newScreen(t *testing.T) tui.Screen {
	t.Helper()
	s := welcome.New()
	require.Equal(t, tui.WelcomeScreenID, s.ID())
	return s
}

func requireNavigates(t *testing.T, cmd tea.Cmd, to tui.ScreenID) {
	t.Helper()
	require.NotNil(t, cmd, "expected a navigation command")
	nav, ok := cmd().(tui.NavigateMsg)
	require.True(t, ok, "expected NavigateMsg, got %T", cmd())
	assert.Equal(t, to, nav.To)
}

func TestEnterBeginsTheGame(t *testing.T) {
	t.Parallel()

	_, cmd := newScreen(t).Update(tea.KeyMsg{Type: tea.KeyEnter})
	requireNavigates(t, cmd, tui.NextScreenID)
}

func TestLeftClickBeginsTheGame(t *testing.T) {
	t.Parallel()

	_, cmd := newScreen(t).Update(tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
	})
	requireNavigates(t, cmd, tui.NextScreenID)
}

func TestOtherKeysDoNothing(t *testing.T) {
	t.Parallel()

	s, cmd := newScreen(t).Update(tea.KeyMsg{Type: tea.KeySpace})
	assert.Nil(t, cmd)
	assert.Equal(t, tui.WelcomeScreenID, s.ID())
}

func TestMouseReleaseDoesNotBegin(t *testing.T) {
	t.Parallel()

	_, cmd := newScreen(t).Update(tea.MouseMsg{
		Action: tea.MouseActionRelease,
		Button: tea.MouseButtonLeft,
	})
	assert.Nil(t, cmd)
}

func TestViewShowsTitleAndPrompt(t *testing.T) {
	t.Parallel()

	s := newScreen(t)
	s, _ = s.Update(tea.WindowSizeMsg{Width: 80, Height: 23})

	view := s.View()
	assert.Contains(t, view, "TAMAGOTCHI")
	assert.Contains(t, view, "Press Enter or click to begin")
}

func TestScreenIsScrollable(t *testing.T) {
	t.Parallel()
	assert.True(t, newScreen(t).Scrollable())
}

func TestInitReturnsNoCommand(t *testing.T) {
	t.Parallel()
	assert.Nil(t, newScreen(t).Init())
}

func TestShortHelpAdvertisesBegin(t *testing.T) {
	t.Parallel()

	hp, ok := newScreen(t).(tui.HelpProvider)
	require.True(t, ok, "Welcome Screen should provide help hints")

	bindings := hp.ShortHelp()
	require.Len(t, bindings, 1)
	assert.Equal(t, "begin", bindings[0].Help().Desc)
}
