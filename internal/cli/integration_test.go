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
	return newTestAppWithPet(t, pet.New(time.Now()))
}

// newTestAppWithPet is newTestApp around a given initial Pet.
func newTestAppWithPet(t *testing.T, initial pet.Pet) *tui.App {
	t.Helper()
	store := pet.NewFileStore(filepath.Join(t.TempDir(), "save.json"))
	return tui.NewApp(ScreenFactories(initial, store), tui.WelcomeScreenID)
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

// runCure drives the Welcome Screen to the Next Screen with a hatched Pet that
// is Sick from the first frame, then Cures it either by hotkey or by clicking the
// Cure tab's zone, and returns the resulting Pet. The Pet is one kept up to date
// as of the moment it is Sick, so the Screen starts there (see
// runFullCareLoop for why a Pet merely fast-forwarded by a tick would lag).
func runCure(t *testing.T, useMouse bool) (cured pet.Pet, sickAt time.Time) {
	t.Helper()

	born := time.Now()
	sickAt = born.Add(pet.EggDuration + time.Second)
	initial := pet.New(born)
	initial.LastSeenAt, initial.HappinessLastSeenAt = sickAt, sickAt
	initial.LastCleanedAt = sickAt.Add(-(pet.MessInterval + pet.SickAfterMess)) // Sick exactly from sickAt

	app := newTestAppWithPet(t, initial)
	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(100, 30))

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("Press Enter or click to begin"))
	}, teatest.WithDuration(3*time.Second))
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// One wait for the Sick indicator and the Cure tab together: they are on the
	// first frame, and teatest.WaitFor consumes the output it reads.
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("Sick")) && bytes.Contains(b, []byte("Cure"))
	}, teatest.WithDuration(3*time.Second))

	if useMouse {
		z := waitForZone(t, next.CureZoneID)
		tm.Send(tea.MouseMsg{
			Action: tea.MouseActionPress, Button: tea.MouseButtonLeft,
			X: (z.StartX + z.EndX) / 2, Y: z.StartY,
		})
	} else {
		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	}
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("feels better"))
	}, teatest.WithDuration(3*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	final, ok := tm.FinalModel(t).(*tui.App)
	require.True(t, ok)
	screen, ok := final.Current().(*next.Screen)
	require.True(t, ok)
	return screen.Pet(), sickAt
}

// TestCureKeyboardAndMouseReachTheSameState proves the keyboard and mouse paths
// to Cure are equivalent, as for every other Care action.
//
// Deliberately not t.Parallel(): its mouse-driven half calls waitForZone, which
// reads bubblezone's process-wide DefaultManager — see
// TestPlayAndCleanKeyboardAndMouseReachTheSameState.
func TestCureKeyboardAndMouseReachTheSameState(t *testing.T) {
	byKeyboard, sickAt := runCure(t, false)
	byMouse, _ := runCure(t, true)

	assert.False(t, byKeyboard.Sick(sickAt), "the hotkey should Cure a Sick Pet")
	assert.False(t, byMouse.Sick(sickAt), "so should a click on the Cure tab")
	assert.False(t, byKeyboard.LastCuredAt.IsZero())
	assert.False(t, byMouse.LastCuredAt.IsZero())
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
// by growBy (past EggDuration to reach Baby, or past
// EggDuration+BabyDuration to reach Child), then exercises all three Care
// actions in one continuous run — Feed (Snack), then Play, then Clean —
// either entirely by hotkey or entirely by clicking icon-bar and chooser
// zones, and returns the resulting Pet. This is the "keyboard and mouse
// throughout" claim made good across the complete loop, not just per
// action, at whichever Stage growBy reaches. waitFor is the text to wait for
// once the fast-forward tick lands, confirming the Screen has rendered that
// Stage before any Care action is attempted.
func runFullCareLoop(t *testing.T, useMouse bool, growBy time.Duration, waitFor string) pet.Pet {
	t.Helper()

	// The Pet is one kept up to date and looked after until the moment the
	// clock is fast-forwarded to, as the periodic Beat and an attentive player
	// would leave it. A Pet born at the start and merely fast-forwarded by a
	// tick would lag that clock by many minutes, which real play cannot reach
	// and which a Care action (advancing the Pet to the Screen's clock first)
	// would rightly decay before applying.
	born := time.Now()
	fastForwardedTo := born.Add(growBy)
	initial := pet.New(born)
	initial.LastSeenAt, initial.HappinessLastSeenAt, initial.LastCleanedAt = fastForwardedTo, fastForwardedTo, fastForwardedTo

	app := newTestAppWithPet(t, initial)
	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(100, 30))

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("Press Enter or click to begin"))
	}, teatest.WithDuration(3*time.Second))
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Not real time: the Pet above already stands growBy after its birth, so the
	// Next Screen's first frame is at the Stage under test — no sleeping. Wait
	// for that Stage's marker (its label, or "Feed" for the Icon bar) together
	// with the Meters, in one wait: teatest.WaitFor consumes the output it reads
	// and Bubble Tea only re-emits changed lines, so a marker on the first frame
	// would be gone by a second wait.
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("Hunger")) && bytes.Contains(b, []byte(waitFor))
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
	byKeyboard := runFullCareLoop(t, false, pet.EggDuration+time.Second, "Feed")
	byMouse := runFullCareLoop(t, true, pet.EggDuration+time.Second, "Feed")

	assert.Equal(t, pet.MaxStat, byKeyboard.Happiness)
	assert.Equal(t, pet.MaxStat, byMouse.Happiness)
	// Snack adds 1 Weight and the subsequent Play removes 1, netting back to
	// BaseWeight.
	assert.Equal(t, pet.BaseWeight, byKeyboard.Weight)
	assert.Equal(t, pet.BaseWeight, byMouse.Weight)
	assert.False(t, byKeyboard.HasMess(time.Now()))
	assert.False(t, byMouse.HasMess(time.Now()))
}

// TestFullCareLoopKeyboardAndMouseReachTheSameStateAsChild mirrors
// TestFullCareLoopKeyboardAndMouseReachTheSameState, proving the same
// keyboard/mouse equivalence once the Pet has grown into a Child, not just
// as a Baby — waiting for the "Child" Stage label specifically, rather than
// just "Feed", confirms the fast-forward actually reached that Stage.
//
// Deliberately not t.Parallel() — see
// TestPlayAndCleanKeyboardAndMouseReachTheSameState's comment on waitForZone
// and bubblezone's shared DefaultManager.
func TestFullCareLoopKeyboardAndMouseReachTheSameStateAsChild(t *testing.T) {
	growBy := pet.EggDuration + pet.BabyDuration + time.Second
	byKeyboard := runFullCareLoop(t, false, growBy, "Child")
	byMouse := runFullCareLoop(t, true, growBy, "Child")

	assert.Equal(t, pet.MaxStat, byKeyboard.Happiness)
	assert.Equal(t, pet.MaxStat, byMouse.Happiness)
	// Snack adds 1 Weight and the subsequent Play removes 1, netting back to
	// BaseWeight.
	assert.Equal(t, pet.BaseWeight, byKeyboard.Weight)
	assert.Equal(t, pet.BaseWeight, byMouse.Weight)
	assert.False(t, byKeyboard.HasMess(time.Now()))
	assert.False(t, byMouse.HasMess(time.Now()))
}

// TestFullCareLoopKeyboardAndMouseReachTheSameStateAsTeen mirrors
// TestFullCareLoopKeyboardAndMouseReachTheSameStateAsChild, proving the same
// keyboard/mouse equivalence once the Pet has grown into a Teen — waiting
// for the "Teen" Stage label specifically confirms the fast-forward actually
// reached that Stage.
//
// Deliberately not t.Parallel() — see
// TestPlayAndCleanKeyboardAndMouseReachTheSameState's comment on waitForZone
// and bubblezone's shared DefaultManager.
func TestFullCareLoopKeyboardAndMouseReachTheSameStateAsTeen(t *testing.T) {
	growBy := pet.EggDuration + pet.BabyDuration + pet.ChildDuration + time.Second
	byKeyboard := runFullCareLoop(t, false, growBy, "Teen")
	byMouse := runFullCareLoop(t, true, growBy, "Teen")

	assert.Equal(t, pet.MaxStat, byKeyboard.Happiness)
	assert.Equal(t, pet.MaxStat, byMouse.Happiness)
	// Snack adds 1 Weight and the subsequent Play removes 1, netting back to
	// BaseWeight.
	assert.Equal(t, pet.BaseWeight, byKeyboard.Weight)
	assert.Equal(t, pet.BaseWeight, byMouse.Weight)
	assert.False(t, byKeyboard.HasMess(time.Now()))
	assert.False(t, byMouse.HasMess(time.Now()))
}

// TestFullCareLoopKeyboardAndMouseReachTheSameStateAsAdult mirrors
// TestFullCareLoopKeyboardAndMouseReachTheSameStateAsTeen, proving the same
// keyboard/mouse equivalence once the Pet has grown into an Adult — waiting
// for the "Adult" Stage label specifically confirms the fast-forward actually
// reached that Stage.
//
// Deliberately not t.Parallel() — see
// TestPlayAndCleanKeyboardAndMouseReachTheSameState's comment on waitForZone
// and bubblezone's shared DefaultManager.
func TestFullCareLoopKeyboardAndMouseReachTheSameStateAsAdult(t *testing.T) {
	growBy := pet.EggDuration + pet.BabyDuration + pet.ChildDuration + pet.TeenDuration + time.Second
	byKeyboard := runFullCareLoop(t, false, growBy, "Adult")
	byMouse := runFullCareLoop(t, true, growBy, "Adult")

	assert.Equal(t, pet.MaxStat, byKeyboard.Happiness)
	assert.Equal(t, pet.MaxStat, byMouse.Happiness)
	// Snack adds 1 Weight and the subsequent Play removes 1, netting back to
	// BaseWeight.
	assert.Equal(t, pet.BaseWeight, byKeyboard.Weight)
	assert.Equal(t, pet.BaseWeight, byMouse.Weight)
	assert.False(t, byKeyboard.HasMess(time.Now()))
	assert.False(t, byMouse.HasMess(time.Now()))
}

// runLaunchAfterDeath seeds a save file with seed (back-dated, so the Pet died
// while the game was closed), loads it through the real loadPet catch-up, and
// drives the Welcome Screen to the Next Screen. It waits for the Death panel's
// cause text, restarts by hotkey or by clicking the prompt's zone, and quits.
// It returns the Pet the Screen ended with and the Pet re-read from the save
// file, so both the on-screen result and what persisted can be checked.
func runLaunchAfterDeath(t *testing.T, seed pet.Pet, wantCause string, useMouse bool) (screen, saved pet.Pet) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "save.json")
	store := pet.NewFileStore(path)
	require.NoError(t, store.Save(seed))

	var errOut bytes.Buffer
	initial, _ := loadPet(&errOut, path)
	require.NotEqual(t, pet.NotDead, initial.CauseAt(time.Now()), "sanity check: the Pet died while the game was closed")

	app := tui.NewApp(ScreenFactories(initial, store), tui.WelcomeScreenID)
	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(100, 30))

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("Press Enter or click to begin"))
	}, teatest.WithDuration(3*time.Second))
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte(wantCause))
	}, teatest.WithDuration(3*time.Second))

	if useMouse {
		z := waitForZone(t, next.RestartZoneID)
		tm.Send(tea.MouseMsg{
			Action: tea.MouseActionPress, Button: tea.MouseButtonLeft,
			X: (z.StartX + z.EndX) / 2, Y: z.StartY,
		})
	} else {
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	}

	// The Meters only draw for a living Pet, and this one was dead on arrival.
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("Hunger"))
	}, teatest.WithDuration(3*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	final, ok := tm.FinalModel(t).(*tui.App)
	require.True(t, ok)
	current, ok := final.Current().(*next.Screen)
	require.True(t, ok)

	saved, found, err := store.Load()
	require.NoError(t, err)
	require.True(t, found)
	return current.Pet(), saved
}

// TestAPetThatDiedWhileTheGameWasClosedShowsItsCauseAndRestarts: launching with
// a save whose Pet died while away shows the Death panel naming the cause, and
// Restart, by key or by mouse, hatches a fresh Egg that is saved, for each cause.
//
// Deliberately not t.Parallel(): the mouse variants call waitForZone, which
// reads bubblezone's process-wide DefaultManager — see
// TestPlayAndCleanKeyboardAndMouseReachTheSameState.
func TestAPetThatDiedWhileTheGameWasClosedShowsItsCauseAndRestarts(t *testing.T) {
	causes := map[string]struct {
		seed func(now time.Time) pet.Pet
		want string
	}{
		// Cleaned a minute ago but never fed: no Mess yet, so nothing makes it Sick
		// before Hunger, Empty from 12:00, starves it at 22:00.
		"Starvation": {
			seed: func(now time.Time) pet.Pet {
				p := pet.New(now.Add(-40 * time.Minute))
				p.LastCleanedAt = now.Add(-time.Minute)
				return p
			},
			want: "Cause: Starvation",
		},
		// Untended for 40 minutes: Sick at 10:00 and dead of it at 18:00, ahead of the
		// 22:00 at which it would have starved.
		"Sickness": {
			seed: func(now time.Time) pet.Pet { return pet.New(now.Add(-40 * time.Minute)) },
			want: "Cause: Sickness",
		},
		// Three Care mistakes already on its saved tally, last seen at 50 minutes and
		// left for two hours: its Adult lasts 14 minutes rather than 20, so it dies of
		// Neglect at 54:30, a full 6 minutes before old age would have come.
		"Neglect": {
			seed: func(now time.Time) pet.Pet {
				born := now.Add(-2 * time.Hour)
				p := pet.New(born)
				p.CareMistakes = 3
				last := born.Add(50 * time.Minute)
				p.LastSeenAt, p.HappinessLastSeenAt, p.LastCleanedAt = last, last, last
				return p
			},
			want: "Cause: Neglect",
		},
		"Old age": {
			seed: func(now time.Time) pet.Pet {
				born := now.Add(-2 * time.Hour)
				p := pet.New(born)
				last := born.Add(59 * time.Minute)
				p.LastSeenAt, p.HappinessLastSeenAt, p.LastCleanedAt = last, last, last
				return p
			},
			want: "Cause: Old age",
		},
	}

	for name, tt := range causes {
		for _, input := range []string{"keyboard", "mouse"} {
			t.Run(name+" by "+input, func(t *testing.T) {
				started := time.Now()

				screen, saved := runLaunchAfterDeath(t, tt.seed(started), tt.want, input == "mouse")

				for where, p := range map[string]pet.Pet{"on screen": screen, "in the save file": saved} {
					assert.True(t, p.DiedAt.IsZero(), "the Pet %s is the new one, with no death", where)
					assert.Zero(t, p.CareMistakes, "%s", where)
					assert.Equal(t, pet.MaxStat, p.Hunger, "%s", where)
					assert.Equal(t, pet.StageEgg, p.Stage(time.Now()), "%s", where)
					assert.True(t, p.CreatedAt.After(started), "born after the launch, not the seeded Pet: %s", where)
				}
			})
		}
	}
}

// runDeathThenRestart drives the Welcome Screen to the Next Screen,
// fast-forwards past every Stage duration up to and including AdultDuration
// (the same injected-anim.TickMsg technique runFullCareLoop already uses —
// no real sleeping) so the Pet has died, then restarts it either by hotkey
// or by clicking the restart prompt's zone, and returns the resulting Pet.
// Unlike runFullCareLoop, there is no Care action to prove parity on: the
// thing being proven here is that both input methods reach the reset state,
// not that they reach the same care-loop outcome.
func runDeathThenRestart(t *testing.T, useMouse bool) pet.Pet {
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

	growBy := pet.EggDuration + pet.BabyDuration + pet.ChildDuration + pet.TeenDuration + pet.AdultDuration + time.Second
	tm.Send(anim.TickMsg{Time: time.Now().Add(growBy)})
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("Death"))
	}, teatest.WithDuration(3*time.Second))

	if useMouse {
		z := waitForZone(t, next.RestartZoneID)
		tm.Send(tea.MouseMsg{
			Action: tea.MouseActionPress, Button: tea.MouseButtonLeft,
			X: (z.StartX + z.EndX) / 2, Y: z.StartY,
		})
	} else {
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	}

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("Egg"))
	}, teatest.WithDuration(3*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	final, ok := tm.FinalModel(t).(*tui.App)
	require.True(t, ok)
	screen, ok := final.Current().(*next.Screen)
	require.True(t, ok)
	return screen.Pet()
}

// TestDeathThenRestartByKeyboardProducesAFreshEgg and
// TestDeathThenRestartByMouseProducesAFreshEgg prove restart reaches the
// same reset state by keyboard and by mouse — this project's existing
// keyboard/mouse parity philosophy, extended to a reset rather than only
// forward growth.
//
// Deliberately not t.Parallel() — see
// TestPlayAndCleanKeyboardAndMouseReachTheSameState's comment on waitForZone
// and bubblezone's shared DefaultManager.
func TestDeathThenRestartByKeyboardProducesAFreshEgg(t *testing.T) {
	restarted := runDeathThenRestart(t, false)

	assert.Equal(t, pet.MaxStat, restarted.Hunger)
	assert.Equal(t, pet.MaxStat, restarted.Happiness)
	assert.Equal(t, pet.BaseWeight, restarted.Weight)
	assert.False(t, restarted.HasMess(time.Now()))
}

func TestDeathThenRestartByMouseProducesAFreshEgg(t *testing.T) {
	restarted := runDeathThenRestart(t, true)

	assert.Equal(t, pet.MaxStat, restarted.Hunger)
	assert.Equal(t, pet.MaxStat, restarted.Happiness)
	assert.Equal(t, pet.BaseWeight, restarted.Weight)
	assert.False(t, restarted.HasMess(time.Now()))
}
