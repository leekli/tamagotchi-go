// Package welcome renders the Welcome Screen, the first Screen the player sees.
//
// In this phase it is a functional stub: a plain title line plus the begin
// prompt, wired for navigation. The Wordmark, shine sweep, and Character arrive
// in Phase 2 and replace the body of View without touching the router.
package welcome

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/leekli/tamagotchi-go/internal/tui"
)

// promptText is the begin prompt shown under the title.
const promptText = "Press Enter or click to begin"

// Screen is the Welcome Screen.
type Screen struct {
	width  int
	height int
	keys   keyMap
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

// New builds a Welcome Screen.
func New() tui.Screen {
	return &Screen{keys: defaultKeyMap()}
}

// ID implements tui.Screen.
func (s *Screen) ID() tui.ScreenID { return tui.WelcomeScreenID }

// Init implements tui.Screen.
func (s *Screen) Init() tea.Cmd { return nil }

// Update implements tui.Screen.
func (s *Screen) Update(msg tea.Msg) (tui.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height

	case tea.KeyMsg:
		if key.Matches(msg, s.keys.Begin) {
			return s, tui.Navigate(tui.NextScreenID)
		}

	case tea.MouseMsg:
		// Phase 2: restrict this to the prompt's own zone via bubblezone.
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			return s, tui.Navigate(tui.NextScreenID)
		}
	}
	return s, nil
}

// ShortHelp implements tui.HelpProvider.
func (s *Screen) ShortHelp() []key.Binding {
	return []key.Binding{s.keys.Begin}
}

// View implements tui.Screen.
func (s *Screen) View() string {
	title := lipgloss.NewStyle().Bold(true).Render("TAMAGOTCHI")
	body := lipgloss.JoinVertical(lipgloss.Center, title, "", promptText)
	return lipgloss.Place(s.width, s.height, lipgloss.Center, lipgloss.Center, body)
}

// Scrollable implements tui.Screen.
func (s *Screen) Scrollable() bool { return true }
