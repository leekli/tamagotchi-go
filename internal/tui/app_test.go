package tui_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/leekli/tamagotchi-go/internal/tui"
)

// fakeScreen is a configurable Screen stand-in for router tests.
type fakeScreen struct {
	id          tui.ScreenID
	body        string
	scrollable  bool
	initCmd     tea.Cmd
	lastMsg     tea.Msg
	sizeW       int
	sizeH       int
	updateCalls int
}

func (f *fakeScreen) ID() tui.ScreenID { return f.id }
func (f *fakeScreen) Init() tea.Cmd    { return f.initCmd }

func (f *fakeScreen) Update(msg tea.Msg) (tui.Screen, tea.Cmd) {
	f.updateCalls++
	f.lastMsg = msg
	if m, ok := msg.(tea.WindowSizeMsg); ok {
		f.sizeW, f.sizeH = m.Width, m.Height
	}
	return f, nil
}

func (f *fakeScreen) View() string     { return f.body }
func (f *fakeScreen) Scrollable() bool { return f.scrollable }

// helpFakeScreen also contributes a key hint to the help bar.
type helpFakeScreen struct {
	*fakeScreen
	binding key.Binding
}

func (h *helpFakeScreen) Update(msg tea.Msg) (tui.Screen, tea.Cmd) {
	h.fakeScreen.Update(msg)
	return h, nil
}

func (h *helpFakeScreen) ShortHelp() []key.Binding { return []key.Binding{h.binding} }

func staticFactories(screens ...tui.Screen) map[tui.ScreenID]tui.ScreenFactory {
	m := make(map[tui.ScreenID]tui.ScreenFactory, len(screens))
	for _, s := range screens {
		s := s
		m[s.ID()] = func() tui.Screen { return s }
	}
	return m
}

func sendKey(t *testing.T, app *tui.App, k tea.KeyType) (*tui.App, tea.Cmd) {
	t.Helper()
	model, cmd := app.Update(tea.KeyMsg{Type: k})
	next, ok := model.(*tui.App)
	require.True(t, ok)
	return next, cmd
}

func TestNewAppPanicsOnUnknownStartScreen(t *testing.T) {
	t.Parallel()

	assert.PanicsWithValue(t,
		`tui: no factory registered for start screen "missing"`,
		func() { tui.NewApp(map[tui.ScreenID]tui.ScreenFactory{}, tui.ScreenID("missing")) },
	)
}

func TestNewAppStartsOnRequestedScreen(t *testing.T) {
	t.Parallel()

	welcome := &fakeScreen{id: tui.WelcomeScreenID}
	app := tui.NewApp(staticFactories(welcome), tui.WelcomeScreenID)

	assert.Equal(t, tui.WelcomeScreenID, app.Current().ID())
}

func TestInitRunsTheStartScreensInit(t *testing.T) {
	t.Parallel()

	type initMsg struct{}
	screen := &fakeScreen{
		id:      tui.WelcomeScreenID,
		initCmd: func() tea.Msg { return initMsg{} },
	}
	app := tui.NewApp(staticFactories(screen), tui.WelcomeScreenID)

	cmd := app.Init()
	require.NotNil(t, cmd)
	assert.IsType(t, initMsg{}, cmd())
}

func TestQuitKeysReturnQuitCommand(t *testing.T) {
	t.Parallel()

	for name, kt := range map[string]tea.KeyType{
		"ctrl+q": tea.KeyCtrlQ,
		"ctrl+c": tea.KeyCtrlC,
	} {
		kt := kt
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			app := tui.NewApp(staticFactories(&fakeScreen{id: tui.WelcomeScreenID}), tui.WelcomeScreenID)
			_, cmd := sendKey(t, app, kt)

			require.NotNil(t, cmd)
			_, isQuit := cmd().(tea.QuitMsg)
			assert.True(t, isQuit, "expected tea.QuitMsg")
		})
	}
}

func TestWindowSizeForwardsBodyAreaToScreen(t *testing.T) {
	t.Parallel()

	screen := &fakeScreen{id: tui.WelcomeScreenID}
	app := tui.NewApp(staticFactories(screen), tui.WelcomeScreenID)

	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	assert.Equal(t, 80, screen.sizeW)
	assert.Equal(t, 23, screen.sizeH, "screen height should exclude the help-bar row")
}

func TestNavigateReplacesScreenWithFreshInstance(t *testing.T) {
	t.Parallel()

	welcome := &fakeScreen{id: tui.WelcomeScreenID}
	built := 0
	factories := map[tui.ScreenID]tui.ScreenFactory{
		tui.WelcomeScreenID: func() tui.Screen { return welcome },
		tui.NextScreenID: func() tui.Screen {
			built++
			return &fakeScreen{id: tui.NextScreenID}
		},
	}
	app := tui.NewApp(factories, tui.WelcomeScreenID)
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	model, _ := app.Update(tui.NavigateMsg{To: tui.NextScreenID})
	app = model.(*tui.App)

	assert.Equal(t, tui.NextScreenID, app.Current().ID())
	assert.Equal(t, 1, built, "navigation should build a fresh Screen from its factory")

	next := app.Current().(*fakeScreen)
	assert.Equal(t, 100, next.sizeW)
	assert.Equal(t, 29, next.sizeH, "a freshly navigated Screen should receive the current body size")
}

func TestNavigateToUnknownScreenIsIgnored(t *testing.T) {
	t.Parallel()

	welcome := &fakeScreen{id: tui.WelcomeScreenID}
	app := tui.NewApp(staticFactories(welcome), tui.WelcomeScreenID)

	model, cmd := app.Update(tui.NavigateMsg{To: tui.ScreenID("nope")})
	app = model.(*tui.App)

	assert.Equal(t, tui.WelcomeScreenID, app.Current().ID())
	assert.Nil(t, cmd)
}

func TestViewIsEmptyBeforeFirstWindowSize(t *testing.T) {
	t.Parallel()

	app := tui.NewApp(staticFactories(&fakeScreen{id: tui.WelcomeScreenID, body: "hello"}), tui.WelcomeScreenID)
	assert.Empty(t, app.View())
}

func TestViewShowsResizeNoticeBelowMinimumSize(t *testing.T) {
	t.Parallel()

	for name, size := range map[string]struct{ w, h int }{
		"too narrow":        {79, 24},
		"too short":         {80, 23},
		"degenerate height": {100, 1},
	} {
		size := size
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			app := tui.NewApp(staticFactories(&fakeScreen{id: tui.WelcomeScreenID, body: "SCREEN BODY"}), tui.WelcomeScreenID)
			app.Update(tea.WindowSizeMsg{Width: size.w, Height: size.h})

			view := app.View()
			assert.Contains(t, view, "resize your terminal to at least 80×24")
			assert.NotContains(t, view, "SCREEN BODY")
		})
	}
}

func TestViewRendersScreenBodyAndHelpBar(t *testing.T) {
	t.Parallel()

	app := tui.NewApp(staticFactories(&fakeScreen{id: tui.WelcomeScreenID, body: "SCREEN BODY"}), tui.WelcomeScreenID)
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	view := app.View()
	assert.Contains(t, view, "SCREEN BODY")
	assert.Contains(t, view, "quit", "help bar should always advertise the quit key")
}

func TestHelpBarIncludesActiveScreenBindings(t *testing.T) {
	t.Parallel()

	screen := &helpFakeScreen{
		fakeScreen: &fakeScreen{id: tui.WelcomeScreenID, body: "body"},
		binding: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "begin"),
		),
	}
	app := tui.NewApp(staticFactories(screen), tui.WelcomeScreenID)
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	view := app.View()
	assert.Contains(t, view, "begin")
	assert.Contains(t, view, "quit")
}

func TestScrollableScreenScrollsWithinViewport(t *testing.T) {
	t.Parallel()

	var lines []string
	for i := 0; i < 100; i++ {
		lines = append(lines, fmt.Sprintf("line-%03d", i))
	}
	screen := &fakeScreen{id: tui.WelcomeScreenID, body: strings.Join(lines, "\n"), scrollable: true}
	app := tui.NewApp(staticFactories(screen), tui.WelcomeScreenID)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	before := app.View()
	assert.Contains(t, before, "line-000")
	assert.NotContains(t, before, "line-090")

	app.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	after := app.View()

	assert.NotEqual(t, before, after, "page-down should move the viewport")
	assert.NotContains(t, after, "line-000")
}

func TestNonScrollableScreenIsNotWrappedInViewport(t *testing.T) {
	t.Parallel()

	var lines []string
	for i := 0; i < 100; i++ {
		lines = append(lines, fmt.Sprintf("row-%03d", i))
	}
	screen := &fakeScreen{id: tui.WelcomeScreenID, body: strings.Join(lines, "\n"), scrollable: false}
	app := tui.NewApp(staticFactories(screen), tui.WelcomeScreenID)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// A non-scrollable Screen owns every cell, so even off-area content is
	// passed straight through rather than clipped by a viewport.
	assert.Contains(t, app.View(), "row-099")
}

func TestUnhandledMessagesReachTheActiveScreen(t *testing.T) {
	t.Parallel()

	screen := &fakeScreen{id: tui.WelcomeScreenID}
	app := tui.NewApp(staticFactories(screen), tui.WelcomeScreenID)

	type customMsg struct{}
	app.Update(customMsg{})

	assert.Equal(t, 1, screen.updateCalls)
	assert.IsType(t, customMsg{}, screen.lastMsg)
}
