package cli

import (
	"bytes"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/leekli/tamagotchi-go/internal/tui"
)

// TestWelcomeToNextToQuitFlow drives the fully wired App through its Phase 1
// journey: land on the Welcome Screen, press Enter to reach the Next Screen,
// then quit with Ctrl+Q.
func TestWelcomeToNextToQuitFlow(t *testing.T) {
	t.Parallel()

	app := tui.NewApp(ScreenFactories(), tui.WelcomeScreenID)
	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(100, 30))

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("Press Enter or click to begin"))
	}, teatest.WithDuration(3*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("Nothing here yet"))
	}, teatest.WithDuration(3*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlQ})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	final, ok := tm.FinalModel(t).(*tui.App)
	require.True(t, ok)
	assert.Equal(t, tui.NextScreenID, final.Current().ID())
}

// TestResizeNoticeShownThenClearedOnGrow proves the small-terminal guard is live
// end to end and recovers when the terminal grows.
func TestResizeNoticeShownThenClearedOnGrow(t *testing.T) {
	t.Parallel()

	app := tui.NewApp(ScreenFactories(), tui.WelcomeScreenID)
	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(40, 10))

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("resize your terminal"))
	}, teatest.WithDuration(3*time.Second))

	tm.Send(tea.WindowSizeMsg{Width: 100, Height: 30})

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("Press Enter or click to begin"))
	}, teatest.WithDuration(3*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}
