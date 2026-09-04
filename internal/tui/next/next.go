// Package next renders the Next Screen: the Screen reached from the Welcome
// Screen. For now it holds placeholder text and nothing else; its real content
// is a later feature. It exists so the router has a real destination to navigate
// to and so navigation is exercised end to end from Phase 1.
package next

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/leekli/tamagotchi-go/internal/tui"
)

// placeholderText is the entire content of the Next Screen for now.
const placeholderText = "Nothing here yet. Ctrl+C to quit."

// Screen is the Next Screen.
type Screen struct {
	width  int
	height int
}

// New builds a Next Screen.
func New() tui.Screen { return &Screen{} }

// ID implements tui.Screen.
func (s *Screen) ID() tui.ScreenID { return tui.NextScreenID }

// Init implements tui.Screen.
func (s *Screen) Init() tea.Cmd { return nil }

// Update implements tui.Screen.
func (s *Screen) Update(msg tea.Msg) (tui.Screen, tea.Cmd) {
	if m, ok := msg.(tea.WindowSizeMsg); ok {
		s.width = m.Width
		s.height = m.Height
	}
	return s, nil
}

// View implements tui.Screen.
func (s *Screen) View() string {
	return lipgloss.Place(s.width, s.height, lipgloss.Center, lipgloss.Center, placeholderText)
}

// Scrollable implements tui.Screen.
func (s *Screen) Scrollable() bool { return true }
