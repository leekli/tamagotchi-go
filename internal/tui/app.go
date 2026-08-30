package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	// minWidth and minHeight are the smallest terminal the game renders in.
	// Below either, the App shows a resize notice instead of the active Screen.
	minWidth  = 80
	minHeight = 24

	// helpBarHeight is the number of rows the App reserves at the bottom for the
	// short help bar.
	helpBarHeight = 1
)

// ScreenFactory builds a fresh Screen. The App holds factories rather than
// Screen values so every navigation produces a clean Screen with its Init run.
type ScreenFactory func() Screen

// App is the root model. It owns the single active Screen, routes messages to it,
// applies navigation, and frames the Screen with shared chrome: a resize notice
// when the terminal is too small, an optional scrolling viewport, and the help
// bar on the last row.
type App struct {
	screens map[ScreenID]ScreenFactory
	current Screen

	keys   KeyMap
	help   help.Model
	styles Styles

	viewport viewport.Model
	width    int
	height   int
	hasSize  bool
}

// NewApp builds an App from a set of Screen factories and the ID of the Screen to
// start on. It panics if start has no registered factory: that is a wiring bug,
// caught by tests, not a runtime condition.
func NewApp(screens map[ScreenID]ScreenFactory, start ScreenID) *App {
	factory, ok := screens[start]
	if !ok {
		panic(fmt.Sprintf("tui: no factory registered for start screen %q", start))
	}
	vp := viewport.New(0, 0)
	vp.MouseWheelEnabled = true
	return &App{
		screens:  screens,
		current:  factory(),
		keys:     DefaultKeyMap(),
		help:     help.New(),
		styles:   NewStyles(DefaultPalette()),
		viewport: vp,
	}
}

// Current returns the active Screen. Intended for tests.
func (a *App) Current() Screen { return a.current }

// Init implements tea.Model.
func (a *App) Init() tea.Cmd {
	return a.current.Init()
}

// Update implements tea.Model.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, a.keys.Quit) {
			return a, tea.Quit
		}

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.hasSize = true
		a.resizeViewport()
		updated, cmd := a.current.Update(a.bodySizeMsg())
		a.current = updated
		a.syncViewport()
		return a, cmd

	case NavigateMsg:
		return a, a.navigate(msg.To)
	}

	updated, cmd := a.current.Update(msg)
	a.current = updated
	cmds := []tea.Cmd{cmd}

	if a.viewportActive() {
		var vpCmd tea.Cmd
		a.viewport, vpCmd = a.viewport.Update(msg)
		cmds = append(cmds, vpCmd)
		a.syncViewport()
	}
	return a, tea.Batch(cmds...)
}

// View implements tea.Model.
func (a *App) View() string {
	if !a.hasSize {
		// Bubble Tea calls View once before the first WindowSizeMsg.
		return ""
	}
	if a.width < minWidth || a.height < minHeight {
		return a.resizeNotice()
	}

	body := a.current.View()
	if a.viewportActive() {
		a.viewport.SetContent(body)
		body = a.viewport.View()
	}

	helpBar := a.styles.HelpBar.Render(a.help.ShortHelpView(a.shortHelp()))
	return lipgloss.JoinVertical(lipgloss.Left, body, helpBar)
}

// navigate replaces the active Screen with a fresh instance of the one named by
// to. An unknown ID is ignored: Screens name destinations only from the ScreenID
// constants, so an unknown ID means a registration gap, which the wiring test
// catches rather than crashing a live session.
func (a *App) navigate(to ScreenID) tea.Cmd {
	factory, ok := a.screens[to]
	if !ok {
		return nil
	}
	a.current = factory()
	cmds := []tea.Cmd{a.current.Init()}
	if a.hasSize {
		updated, cmd := a.current.Update(a.bodySizeMsg())
		a.current = updated
		cmds = append(cmds, cmd)
	}
	a.syncViewport()
	return tea.Batch(cmds...)
}

// bodySizeMsg is the WindowSizeMsg the App forwards to Screens: the full width
// but the height minus the reserved help-bar row.
func (a *App) bodySizeMsg() tea.WindowSizeMsg {
	return tea.WindowSizeMsg{Width: a.width, Height: a.bodyHeight()}
}

func (a *App) bodyHeight() int {
	if h := a.height - helpBarHeight; h > 0 {
		return h
	}
	return 0
}

func (a *App) resizeViewport() {
	a.viewport.Width = a.width
	a.viewport.Height = a.bodyHeight()
}

// syncViewport keeps the viewport's content in step with the active Screen so
// scroll position and bounds stay correct between renders.
func (a *App) syncViewport() {
	if a.viewportActive() {
		a.viewport.SetContent(a.current.View())
	}
}

// viewportActive reports whether the App is framing the current Screen in the
// scrolling viewport right now.
func (a *App) viewportActive() bool {
	return a.hasSize && a.current.Scrollable()
}

func (a *App) shortHelp() []key.Binding {
	if hp, ok := a.current.(HelpProvider); ok {
		return append(hp.ShortHelp(), a.keys.Quit)
	}
	return []key.Binding{a.keys.Quit}
}

func (a *App) resizeNotice() string {
	msg := fmt.Sprintf("Please resize your terminal to at least %d×%d.", minWidth, minHeight)
	return lipgloss.Place(
		max(a.width, 1), max(a.height, 1),
		lipgloss.Center, lipgloss.Center,
		a.styles.Notice.Render(msg),
	)
}
