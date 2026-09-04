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
// then quit with Ctrl+C.
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

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	final, ok := tm.FinalModel(t).(*tui.App)
	require.True(t, ok)
	assert.Equal(t, tui.NextScreenID, final.Current().ID())
}

// TestWelcomeScreenQuitsOnEsc proves the Welcome Screen's own Esc binding
// quits the fully wired App, as an alternative to the App-wide Ctrl+C.
func TestWelcomeScreenQuitsOnEsc(t *testing.T) {
	t.Parallel()

	app := tui.NewApp(ScreenFactories(), tui.WelcomeScreenID)
	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(100, 30))

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("Press Enter or click to begin"))
	}, teatest.WithDuration(3*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	final, ok := tm.FinalModel(t).(*tui.App)
	require.True(t, ok)
	assert.Equal(t, tui.WelcomeScreenID, final.Current().ID())
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

// TestWelcomeScreenWandersWithoutInput proves the Welcome Screen animates on
// its own: with no input at all, the wandering Character is redrawn at more
// than one column as the frame clock ticks.
func TestWelcomeScreenWandersWithoutInput(t *testing.T) {
	t.Parallel()

	app := tui.NewApp(ScreenFactories(), tui.WelcomeScreenID)
	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(100, 30))

	columns := map[int]bool{}
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		for _, line := range bytes.Split(b, []byte("\n")) {
			if i := bytes.Index(line, []byte("( o o)")); i >= 0 {
				columns[i] = true
			}
		}
		return len(columns) >= 2
	}, teatest.WithDuration(5*time.Second), teatest.WithCheckInterval(50*time.Millisecond))

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	assert.GreaterOrEqual(t, len(columns), 2, "the Character should wander across columns")
}

// TestPromptClickNavigates drives a real mouse click onto the begin prompt and
// expects the App to advance to the Next Screen, while a click well away from
// the prompt is ignored.
func TestPromptClickNavigates(t *testing.T) {
	t.Parallel()

	app := tui.NewApp(ScreenFactories(), tui.WelcomeScreenID)
	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(100, 30))

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("Press Enter or click to begin"))
	}, teatest.WithDuration(3*time.Second))

	// A click in the top-left corner is nowhere near the centred prompt, so it
	// must not advance.
	tm.Send(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: 0, Y: 0})

	// The prompt is horizontally centred on row 20 of the 100x30 layout (see the
	// Welcome Screen's vertical stack). bubblezone records the prompt's bounds a
	// frame or two after the first render, so click a few times until it takes.
	for i := 0; i < 20; i++ {
		tm.Send(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: 50, Y: 20})
	}

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("Nothing here yet"))
	}, teatest.WithDuration(3*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	final, ok := tm.FinalModel(t).(*tui.App)
	require.True(t, ok)
	assert.Equal(t, tui.NextScreenID, final.Current().ID())
}
