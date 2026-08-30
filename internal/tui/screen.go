// Package tui holds the terminal UI: the App router, the Screen abstraction, and
// the shared chrome (help bar, resize notice, scrolling viewport) that every
// Screen is framed with.
package tui

import tea "github.com/charmbracelet/bubbletea"

// ScreenID names a Screen so one Screen can request navigation to another
// without importing it.
type ScreenID string

const (
	// WelcomeScreenID is the Screen shown on launch.
	WelcomeScreenID ScreenID = "welcome"
	// NextScreenID is the Screen reached from the Welcome Screen.
	NextScreenID ScreenID = "next"
)

// AllScreenIDs lists every registered ScreenID. Wiring tests iterate it to prove
// every destination resolves to a factory.
func AllScreenIDs() []ScreenID {
	return []ScreenID{WelcomeScreenID, NextScreenID}
}

// Screen is one full-terminal state the player occupies. Exactly one Screen is
// active at a time; new features are added as new Screens.
type Screen interface {
	// ID reports which Screen this is.
	ID() ScreenID
	// Init returns any command to run when the Screen becomes active.
	Init() tea.Cmd
	// Update handles a message and returns the (possibly replaced) Screen plus
	// any command to run. Implementations must not perform I/O directly.
	Update(msg tea.Msg) (Screen, tea.Cmd)
	// View renders the Screen's body only. The App adds the help bar and, for
	// scrollable Screens, the viewport frame. The body is expected to fill the
	// area described by the most recent tea.WindowSizeMsg the Screen received.
	View() string
	// Scrollable reports whether the App should wrap this Screen's View in a
	// scrolling viewport. Screens whose content can outgrow the body area
	// return true; full-bleed Screens that paint every cell return false.
	Scrollable() bool
}

// NavigateMsg asks the App to replace the active Screen with the one named by To.
type NavigateMsg struct {
	To ScreenID
}

// Navigate returns a command that emits a NavigateMsg for the given destination.
func Navigate(to ScreenID) tea.Cmd {
	return func() tea.Msg {
		return NavigateMsg{To: to}
	}
}
