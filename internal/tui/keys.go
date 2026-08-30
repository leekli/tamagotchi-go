package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap holds the bindings the App handles itself, whatever Screen is active.
type KeyMap struct {
	Quit key.Binding
}

// DefaultKeyMap returns the standard App bindings: Ctrl+Q (and the Unix-habitual
// Ctrl+C) always quit.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("ctrl+q", "ctrl+c"),
			key.WithHelp("ctrl+q", "quit"),
		),
	}
}

// HelpProvider is an optional interface a Screen implements to contribute its own
// key hints to the App's help bar.
type HelpProvider interface {
	ShortHelp() []key.Binding
}
