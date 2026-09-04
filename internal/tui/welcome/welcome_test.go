package welcome_test

import (
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/leekli/tamagotchi-go/internal/anim"
	"github.com/leekli/tamagotchi-go/internal/tui"
	"github.com/leekli/tamagotchi-go/internal/tui/welcome"
)

// TestMain gives the package a live bubblezone manager and a deterministic
// colour profile, so click zones resolve and the shine sweep produces visible
// escape codes regardless of whether the test host is a terminal.
func TestMain(m *testing.M) {
	zone.NewGlobal()
	lipgloss.SetColorProfile(termenv.ANSI256)
	os.Exit(m.Run())
}

func newScreen(t *testing.T) tui.Screen {
	t.Helper()
	s := welcome.New()
	require.Equal(t, tui.WelcomeScreenID, s.ID())
	return s
}

func sizedScreen(t *testing.T, w, h int) tui.Screen {
	t.Helper()
	s := newScreen(t)
	s, _ = s.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return s
}

// advance feeds n animation ticks and returns the screen.
func advance(t *testing.T, s tui.Screen, n int) tui.Screen {
	t.Helper()
	for i := 0; i < n; i++ {
		var cmd tea.Cmd
		s, cmd = s.Update(anim.TickMsg{})
		require.NotNil(t, cmd, "every tick should reschedule the next")
	}
	return s
}

// requireNavigatesToNext asserts cmd is a navigation to the Next Screen — the
// only place the Welcome Screen can advance to.
func requireNavigatesToNext(t *testing.T, cmd tea.Cmd) {
	t.Helper()
	require.NotNil(t, cmd, "expected a navigation command")
	nav, ok := cmd().(tui.NavigateMsg)
	require.True(t, ok, "expected NavigateMsg, got %T", cmd())
	assert.Equal(t, tui.NextScreenID, nav.To)
}

// clickBeginPrompt renders and scans the screen until the begin zone resolves,
// then clicks its centre and returns the resulting command. It retries the whole
// scan-then-click: the process-wide bubblezone manager records positions on a
// worker goroutine, so the zone is not known the instant the first frame is
// scanned. This helper is the only place the welcome_test suite calls
// zone.Scan, which keeps the shared "welcome.begin" entry uncontended.
func clickBeginPrompt(t *testing.T, s tui.Screen) tea.Cmd {
	t.Helper()
	var out tea.Cmd
	require.Eventually(t, func() bool {
		zone.Scan(s.View())
		z := zone.Get(welcome.BeginZoneID)
		if z.IsZero() {
			return false
		}
		_, cmd := s.Update(tea.MouseMsg{
			Action: tea.MouseActionPress,
			Button: tea.MouseButtonLeft,
			X:      (z.StartX + z.EndX) / 2,
			Y:      z.StartY,
		})
		if cmd == nil {
			return false
		}
		out = cmd
		return true
	}, 2*time.Second, 10*time.Millisecond, "the begin prompt should become clickable")
	return out
}

func TestEnterBeginsTheGame(t *testing.T) {
	t.Parallel()

	_, cmd := newScreen(t).Update(tea.KeyMsg{Type: tea.KeyEnter})
	requireNavigatesToNext(t, cmd)
}

func TestOtherKeysDoNothing(t *testing.T) {
	t.Parallel()

	s, cmd := newScreen(t).Update(tea.KeyMsg{Type: tea.KeySpace})
	assert.Nil(t, cmd)
	assert.Equal(t, tui.WelcomeScreenID, s.ID())
}

func TestInitStartsTheAnimationClock(t *testing.T) {
	t.Parallel()

	cmd := newScreen(t).Init()
	require.NotNil(t, cmd)
	assert.IsType(t, anim.TickMsg{}, cmd())
}

func TestTickAdvancesTheFrameAndReschedules(t *testing.T) {
	t.Parallel()

	s := newScreen(t)
	ws, ok := s.(*welcome.Screen)
	require.True(t, ok)
	require.Equal(t, 0, ws.Frame())

	s = advance(t, s, 3)
	assert.Equal(t, 3, s.(*welcome.Screen).Frame())
}

func TestScreenIsNotScrollable(t *testing.T) {
	t.Parallel()
	assert.False(t, newScreen(t).Scrollable(),
		"the Welcome Screen paints every cell and is framed without a viewport")
}

func TestShortHelpAdvertisesBeginAndQuit(t *testing.T) {
	t.Parallel()

	hp, ok := newScreen(t).(tui.HelpProvider)
	require.True(t, ok, "Welcome Screen should provide help hints")

	bindings := hp.ShortHelp()
	require.Len(t, bindings, 2)
	assert.Equal(t, "begin", bindings[0].Help().Desc)
	assert.Equal(t, "esc", bindings[1].Help().Key)
	assert.Equal(t, "quit", bindings[1].Help().Desc)
}

func TestEscQuits(t *testing.T) {
	t.Parallel()

	_, cmd := newScreen(t).Update(tea.KeyMsg{Type: tea.KeyEsc})
	require.NotNil(t, cmd)
	_, isQuit := cmd().(tea.QuitMsg)
	assert.True(t, isQuit, "expected tea.QuitMsg")
}

func TestViewShowsWordmarkAndPrompt(t *testing.T) {
	t.Parallel()

	view := stripANSI(sizedScreen(t, 90, 24).View())
	assert.Contains(t, view, "Press Enter or click to begin")
	// A distinctive slice of the "Tamagotchi" wordmark art.
	assert.Contains(t, view, "(_  _)")
}

func TestViewFillsTheBodyArea(t *testing.T) {
	t.Parallel()

	const w, h = 100, 28
	view := sizedScreen(t, w, h).View()
	assert.Equal(t, h, strings.Count(view, "\n")+1, "view should be exactly the body height")
	for _, line := range strings.Split(view, "\n") {
		assert.LessOrEqual(t, lipgloss.Width(line), w)
	}
}

func TestResizeDoesNotReplayTheSweep(t *testing.T) {
	t.Parallel()

	// Advance well past the 800ms sweep so the wordmark is static.
	s := advance(t, sizedScreen(t, 90, 24), 40)
	before := s.View()

	s, _ = s.Update(tea.WindowSizeMsg{Width: 90, Height: 24})
	after := s.View()

	assert.Equal(t, before, after, "a resize must not restart the shine sweep")
}

func TestWordmarkAnimatesEarlyThenGoesStatic(t *testing.T) {
	t.Parallel()

	s := sizedScreen(t, 90, 24)
	early := wordmarkRegion(s.View())

	s = advance(t, s, 4) // ~270ms in: sweep still crossing
	mid := wordmarkRegion(s.View())
	assert.NotEqual(t, early, mid, "the shine sweep should change the wordmark while it runs")

	s = advance(t, s, 20) // well past 800ms
	late := wordmarkRegion(s.View())
	s = advance(t, s, 5)
	later := wordmarkRegion(s.View())
	assert.Equal(t, late, later, "once the sweep is done the wordmark should stop changing")
}

func TestCharacterMovesBetweenFrames(t *testing.T) {
	t.Parallel()

	s := sizedScreen(t, 90, 24)
	first := s.View()
	s = advance(t, s, 12)
	second := s.View()
	assert.NotEqual(t, first, second, "the Character should wander over time")
}

// beginZone scans the screen until the shared "welcome.begin" zone resolves
// (bubblezone records positions on a worker goroutine) and returns its bounds.
func beginZone(t *testing.T, s tui.Screen) *zone.ZoneInfo {
	t.Helper()
	var z *zone.ZoneInfo
	require.Eventually(t, func() bool {
		zone.Scan(s.View())
		z = zone.Get(welcome.BeginZoneID)
		return !z.IsZero()
	}, 2*time.Second, 10*time.Millisecond, "begin zone should resolve after a render")
	return z
}

// TestPromptClickZone is deliberately not parallel: it owns the shared
// "welcome.begin" zone during the serial phase of the package's test run.
func TestPromptClickZone(t *testing.T) {
	t.Run("a click on the prompt begins the game", func(t *testing.T) {
		requireNavigatesToNext(t, clickBeginPrompt(t, sizedScreen(t, 100, 30)))
	})

	t.Run("a click just outside the prompt does nothing", func(t *testing.T) {
		s := sizedScreen(t, 100, 30)
		z := beginZone(t, s)
		_, cmd := s.Update(tea.MouseMsg{
			Action: tea.MouseActionPress,
			Button: tea.MouseButtonLeft,
			X:      z.StartX - 3,
			Y:      z.StartY,
		})
		assert.Nil(t, cmd, "just left of the prompt is outside its zone")
	})

	t.Run("a top-corner click does nothing", func(t *testing.T) {
		s := sizedScreen(t, 100, 30)
		beginZone(t, s) // ensure the zone is known, so this is a real miss
		_, cmd := s.Update(tea.MouseMsg{
			Action: tea.MouseActionPress,
			Button: tea.MouseButtonLeft,
			X:      0,
			Y:      0,
		})
		assert.Nil(t, cmd)
	})

	t.Run("a mouse release on the prompt does nothing", func(t *testing.T) {
		s := sizedScreen(t, 100, 30)
		z := beginZone(t, s)
		_, cmd := s.Update(tea.MouseMsg{
			Action: tea.MouseActionRelease,
			Button: tea.MouseButtonLeft,
			X:      (z.StartX + z.EndX) / 2,
			Y:      z.StartY,
		})
		assert.Nil(t, cmd, "only a press advances, not a release")
	})
}

var ansiSeq = regexp.MustCompile("\x1b\\[[0-9;]*m")

// stripANSI removes SGR colour sequences so plain-text art can be matched.
func stripANSI(s string) string { return ansiSeq.ReplaceAllString(s, "") }

// wordmarkRegion keeps only the rendered wordmark rows of a centred view: they
// are the ones carrying the '|' strokes of the block letters, which neither the
// Character box nor the prompt nor the padding contain.
func wordmarkRegion(view string) string {
	kept := make([]string, 0, 8)
	for _, l := range strings.Split(view, "\n") {
		if strings.Contains(l, "|") {
			kept = append(kept, l)
		}
	}
	return strings.Join(kept, "\n")
}
