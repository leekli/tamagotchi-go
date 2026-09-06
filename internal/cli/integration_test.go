package cli

import (
	"bytes"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	zone "github.com/lrstanley/bubblezone"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/leekli/tamagotchi-go/internal/anim"
	"github.com/leekli/tamagotchi-go/internal/pet"
	"github.com/leekli/tamagotchi-go/internal/tui"
	"github.com/leekli/tamagotchi-go/internal/tui/next"
)

// newTestApp builds a fully wired App with a fresh Pet, persisted to a temp
// save file isolated to this test.
func newTestApp(t *testing.T) *tui.App {
	t.Helper()
	store := pet.NewFileStore(filepath.Join(t.TempDir(), "save.json"))
	return tui.NewApp(ScreenFactories(pet.New(time.Now()), store), tui.WelcomeScreenID)
}

// TestWelcomeToNextToQuitFlow drives the fully wired App through its Phase 1
// journey: land on the Welcome Screen, press Enter to reach the Next Screen,
// then quit with Ctrl+C.
func TestWelcomeToNextToQuitFlow(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(100, 30))

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("Press Enter or click to begin"))
	}, teatest.WithDuration(3*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("Hunger"))
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

	app := newTestApp(t)
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

	app := newTestApp(t)
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

	app := newTestApp(t)
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

	app := newTestApp(t)
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
		return bytes.Contains(b, []byte("Hunger"))
	}, teatest.WithDuration(3*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	final, ok := tm.FinalModel(t).(*tui.App)
	require.True(t, ok)
	assert.Equal(t, tui.NextScreenID, final.Current().ID())
}

// waitForZone scans this process's shared bubblezone manager until id
// resolves, the same pattern the Welcome Screen's tests use — it works here
// because tui.NewApp enables that same process-wide manager, and teatest
// drives the real App in this process.
func waitForZone(t *testing.T, id string) *zone.ZoneInfo {
	t.Helper()
	var z *zone.ZoneInfo
	require.Eventually(t, func() bool {
		z = zone.Get(id)
		return !z.IsZero()
	}, 3*time.Second, 20*time.Millisecond, "zone %q should resolve", id)
	return z
}

// runPlayThenClean drives the Welcome Screen to the Next Screen, fast-forwards
// past EggDuration by injecting a future anim.TickMsg directly (the same
// technique TestAcceptance_HatchesAfterEggDuration uses against a bare
// Screen — no real sleeping), then activates Play and Clean either by
// hotkey or by clicking their icon-bar zones, and returns the resulting Pet.
func runPlayThenClean(t *testing.T, useMouse bool) pet.Pet {
	t.Helper()

	app := newTestApp(t)
	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(100, 30))

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("Press Enter or click to begin"))
	}, teatest.WithDuration(3*time.Second))
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("Hunger"))
	}, teatest.WithDuration(3*time.Second))

	tm.Send(anim.TickMsg{Time: time.Now().Add(pet.EggDuration + time.Second)})
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("Play"))
	}, teatest.WithDuration(3*time.Second))

	activate := func(hotkey rune, zoneID string) {
		if useMouse {
			z := waitForZone(t, zoneID)
			tm.Send(tea.MouseMsg{
				Action: tea.MouseActionPress, Button: tea.MouseButtonLeft,
				X: (z.StartX + z.EndX) / 2, Y: z.StartY,
			})
			return
		}
		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{hotkey}})
	}

	activate('p', next.PlayZoneID)
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("plays happily"))
	}, teatest.WithDuration(3*time.Second))

	activate('c', next.CleanZoneID)
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("tidied up"))
	}, teatest.WithDuration(3*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	final, ok := tm.FinalModel(t).(*tui.App)
	require.True(t, ok)
	screen, ok := final.Current().(*next.Screen)
	require.True(t, ok)
	return screen.Pet()
}

// TestPlayAndCleanKeyboardAndMouseReachTheSameState proves the keyboard and
// mouse paths through the icon bar are equivalent, matching the project's
// "keyboard and mouse throughout" feature claim.
//
// Deliberately not t.Parallel(): its mouse-driven half calls waitForZone,
// which reads bubblezone's process-wide DefaultManager — the same manager
// every other running App in this process shares. Running concurrently with
// another test that registers the same zone id (e.g. "next.play") risks
// reading that other test's coordinates and clicking the wrong window,
// exactly the hazard welcome_test.go's TestPromptClickZone comment already
// documents for the same manager.
func TestPlayAndCleanKeyboardAndMouseReachTheSameState(t *testing.T) {
	byKeyboard := runPlayThenClean(t, false)
	byMouse := runPlayThenClean(t, true)

	assert.Equal(t, pet.MaxStat, byKeyboard.Happiness)
	assert.Equal(t, pet.MaxStat, byMouse.Happiness)
	assert.False(t, byKeyboard.HasMess(time.Now()))
	assert.False(t, byMouse.HasMess(time.Now()))
}

// runFeedSnack drives the Welcome Screen to the Next Screen, fast-forwards
// past EggDuration the same way runPlayThenClean does, then selects Feed and
// chooses Snack either by hotkey or by clicking the icon-bar and chooser
// zones, and returns the resulting Pet.
func runFeedSnack(t *testing.T, useMouse bool) pet.Pet {
	t.Helper()

	app := newTestApp(t)
	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(100, 30))

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("Press Enter or click to begin"))
	}, teatest.WithDuration(3*time.Second))
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("Hunger"))
	}, teatest.WithDuration(3*time.Second))

	tm.Send(anim.TickMsg{Time: time.Now().Add(pet.EggDuration + time.Second)})
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("Feed"))
	}, teatest.WithDuration(3*time.Second))

	if useMouse {
		z := waitForZone(t, next.FeedZoneID)
		tm.Send(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: (z.StartX + z.EndX) / 2, Y: z.StartY})
	} else {
		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	}
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("Snack"))
	}, teatest.WithDuration(3*time.Second))

	if useMouse {
		z := waitForZone(t, next.SnackZoneID)
		tm.Send(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: (z.StartX + z.EndX) / 2, Y: z.StartY})
	} else {
		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	}
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("nom nom"))
	}, teatest.WithDuration(3*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	final, ok := tm.FinalModel(t).(*tui.App)
	require.True(t, ok)
	screen, ok := final.Current().(*next.Screen)
	require.True(t, ok)
	return screen.Pet()
}

// TestFeedSnackKeyboardAndMouseReachTheSameState mirrors
// TestPlayAndCleanKeyboardAndMouseReachTheSameState for the Feed sub-menu.
//
// Deliberately not t.Parallel() — see that test's comment on waitForZone and
// bubblezone's shared DefaultManager.
func TestFeedSnackKeyboardAndMouseReachTheSameState(t *testing.T) {
	byKeyboard := runFeedSnack(t, false)
	byMouse := runFeedSnack(t, true)

	assert.Equal(t, pet.MaxStat, byKeyboard.Happiness)
	assert.Equal(t, pet.MaxStat, byMouse.Happiness)
	assert.Equal(t, pet.BaseWeight+1, byKeyboard.Weight)
	assert.Equal(t, pet.BaseWeight+1, byMouse.Weight)
}

// runFullCareLoop drives the Welcome Screen to the Next Screen, fast-forwards
// past EggDuration, then exercises all three Care actions in one continuous
// run — Feed (Snack), then Play, then Clean — either entirely by hotkey or
// entirely by clicking icon-bar and chooser zones, and returns the resulting
// Pet. This is the "keyboard and mouse throughout" claim made good across
// the complete loop, not just per action.
func runFullCareLoop(t *testing.T, useMouse bool) pet.Pet {
	t.Helper()

	app := newTestApp(t)
	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(100, 30))

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("Press Enter or click to begin"))
	}, teatest.WithDuration(3*time.Second))
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("Hunger"))
	}, teatest.WithDuration(3*time.Second))

	tm.Send(anim.TickMsg{Time: time.Now().Add(pet.EggDuration + time.Second)})
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("Feed"))
	}, teatest.WithDuration(3*time.Second))

	press := func(hotkey rune, zoneID string) {
		if useMouse {
			z := waitForZone(t, zoneID)
			tm.Send(tea.MouseMsg{
				Action: tea.MouseActionPress, Button: tea.MouseButtonLeft,
				X: (z.StartX + z.EndX) / 2, Y: z.StartY,
			})
			return
		}
		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{hotkey}})
	}

	press('f', next.FeedZoneID)
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("Snack"))
	}, teatest.WithDuration(3*time.Second))

	press('s', next.SnackZoneID)
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("nom nom"))
	}, teatest.WithDuration(3*time.Second))

	press('p', next.PlayZoneID)
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("plays happily"))
	}, teatest.WithDuration(3*time.Second))

	press('c', next.CleanZoneID)
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("tidied up"))
	}, teatest.WithDuration(3*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	final, ok := tm.FinalModel(t).(*tui.App)
	require.True(t, ok)
	screen, ok := final.Current().(*next.Screen)
	require.True(t, ok)
	return screen.Pet()
}

// TestFullCareLoopKeyboardAndMouseReachTheSameState proves a mouse-only run
// and a keyboard-only run of the complete Feed -> Play -> Clean loop reach
// identical end states.
//
// Deliberately not t.Parallel() — see
// TestPlayAndCleanKeyboardAndMouseReachTheSameState's comment on waitForZone
// and bubblezone's shared DefaultManager.
func TestFullCareLoopKeyboardAndMouseReachTheSameState(t *testing.T) {
	byKeyboard := runFullCareLoop(t, false)
	byMouse := runFullCareLoop(t, true)

	assert.Equal(t, pet.MaxStat, byKeyboard.Happiness)
	assert.Equal(t, pet.MaxStat, byMouse.Happiness)
	assert.Equal(t, pet.BaseWeight+1, byKeyboard.Weight)
	assert.Equal(t, pet.BaseWeight+1, byMouse.Weight)
	assert.False(t, byKeyboard.HasMess(time.Now()))
	assert.False(t, byMouse.HasMess(time.Now()))
}
