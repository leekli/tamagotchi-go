package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap holds the bindings the App handles itself, whatever Screen is active.
type KeyMap struct {
	Quit key.Binding
}

// DefaultKeyMap returns the standard App bindings: the Unix-habitual Ctrl+C
// always quits, whatever Screen is active.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "quit"),
		),
	}
}

// HelpProvider is an optional interface a Screen implements to contribute its own
// key hints to the App's help bar.
type HelpProvider interface {
	ShortHelp() []key.Binding
}
