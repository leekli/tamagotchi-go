// Package welcome renders the Welcome Screen, the first Screen the player sees:
// the "Tamagotchi" Wordmark with a one-pass shine sweep, the wandering
// Character (Marutchi), and a pulsing begin prompt, stacked and centred.
//
// All motion is a pure function of an integer frame counter that advances one
// step per anim.TickMsg, so the animation is deterministic under test. The
// Screen fills the body area it is given and paints every cell, so the App
// frames it without a scrolling viewport.
package welcome

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"

	"github.com/leekli/tamagotchi-go/internal/anim"
	"github.com/leekli/tamagotchi-go/internal/tui"
)

// BeginZoneID is the bubblezone id of the begin prompt. It is the only click
// target on the Welcome Screen: a left click advances only when it lands on
// these cells.
const BeginZoneID = "welcome.begin"

// Screen is the Welcome Screen.
type Screen struct {
	width  int
	height int
	frame  int
	keys   keyMap
	styles styles
}

type keyMap struct {
	Begin key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		Begin: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "begin"),
		),
	}
}

// New builds a Welcome Screen with its animation clock ready to start.
func New() tui.Screen {
	return &Screen{
		keys:   defaultKeyMap(),
		styles: newStyles(tui.DefaultPalette()),
	}
}

// ID implements tui.Screen.
func (s *Screen) ID() tui.ScreenID { return tui.WelcomeScreenID }

// Init implements tui.Screen. It starts the animation clock.
func (s *Screen) Init() tea.Cmd { return anim.Tick() }

// Frame reports the current animation frame. Intended for diagnostics and
// tests; the number only ever increases and is unaffected by resizes.
func (s *Screen) Frame() int { return s.frame }

// Update implements tui.Screen.
func (s *Screen) Update(msg tea.Msg) (tui.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height

	case anim.TickMsg:
		s.frame++
		return s, anim.Tick()

	case tea.KeyMsg:
		if key.Matches(msg, s.keys.Begin) {
			return s, tui.Navigate(tui.NextScreenID)
		}

	case tea.MouseMsg:
		if isBeginClick(msg) {
			return s, tui.Navigate(tui.NextScreenID)
		}
	}
	return s, nil
}

// isBeginClick reports whether msg is a left press inside the begin prompt's
// zone. The zone is unknown until App.View has scanned at least one frame, and
// nil-safe lookups keep a stray early click harmless.
func isBeginClick(msg tea.MouseMsg) bool {
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return false
	}
	if zone.DefaultManager == nil {
		return false
	}
	return zone.Get(BeginZoneID).InBounds(msg)
}

// ShortHelp implements tui.HelpProvider.
func (s *Screen) ShortHelp() []key.Binding {
	return []key.Binding{s.keys.Begin}
}

// View implements tui.Screen.
func (s *Screen) View() string {
	stack := lipgloss.JoinVertical(
		lipgloss.Center,
		renderWordmark(s.frame, s.styles.wordmark, s.styles.shine),
		"",
		renderCharBox(s.frame, s.styles.character),
		"",
		renderPrompt(s.frame, s.styles.promptDim, s.styles.promptMid, s.styles.promptHi),
	)
	return lipgloss.Place(s.width, s.height, lipgloss.Center, lipgloss.Center, stack)
}

// Scrollable implements tui.Screen. The Welcome Screen is authored to fit the
// minimum terminal and paints every cell, so it is framed without a viewport.
func (s *Screen) Scrollable() bool { return false }
