package next

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"

	"github.com/leekli/tamagotchi-go/internal/pet"
	"github.com/leekli/tamagotchi-go/internal/tui"
)

// Zone ids for the icon bar's click targets, following the same
// zone.Mark/zone.Get pattern the Welcome Screen's begin prompt uses.
const (
	PlayZoneID  = "next.play"
	CleanZoneID = "next.clean"
)

// icon indexes the icon bar, in on-screen order.
type icon int

const (
	iconPlay icon = iota
	iconClean
	numIcons
)

var iconLabel = [numIcons]string{
	iconPlay:  "Play",
	iconClean: "Clean",
}

var iconZoneID = [numIcons]string{
	iconPlay:  PlayZoneID,
	iconClean: CleanZoneID,
}

// keyMap is the Next Screen's own bindings, distinct from tui.KeyMap's
// App-wide ones.
type keyMap struct {
	Left  key.Binding
	Right key.Binding
	Enter key.Binding
	Play  key.Binding
	Clean key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		Left:  key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/→", "select icon")),
		Right: key.NewBinding(key.WithKeys("right", "l")),
		Enter: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "activate")),
		Play:  key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "play")),
		Clean: key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "clean")),
	}
}

// Selected reports the currently selected icon's index. Intended for tests.
func (s *Screen) Selected() int { return s.selected }

// updateIconBarKey handles a key press against the icon bar. It is a no-op
// before the Pet has hatched into a Baby — there is nothing to act on yet.
func (s *Screen) updateIconBarKey(msg tea.KeyMsg) (tui.Screen, tea.Cmd) {
	if s.pet.Stage(s.now) != pet.StageBaby {
		return s, nil
	}
	switch {
	case key.Matches(msg, s.keys.Left):
		s.selected = wrapIcon(s.selected, -1)
	case key.Matches(msg, s.keys.Right):
		s.selected = wrapIcon(s.selected, 1)
	case key.Matches(msg, s.keys.Enter):
		return s.activateIcon(icon(s.selected))
	case key.Matches(msg, s.keys.Play):
		return s.activateIcon(iconPlay)
	case key.Matches(msg, s.keys.Clean):
		return s.activateIcon(iconClean)
	}
	return s, nil
}

// updateIconBarMouse handles a mouse message against the icon bar's click
// zones. Like updateIconBarKey, it is a no-op before the Pet has hatched.
func (s *Screen) updateIconBarMouse(msg tea.MouseMsg) (tui.Screen, tea.Cmd) {
	if s.pet.Stage(s.now) != pet.StageBaby {
		return s, nil
	}
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return s, nil
	}
	if zone.DefaultManager == nil {
		return s, nil
	}
	for i := icon(0); i < numIcons; i++ {
		if zone.Get(iconZoneID[i]).InBounds(msg) {
			return s.activateIcon(i)
		}
	}
	return s, nil
}

// activateIcon selects and runs i's Care action, then shows its flourish.
func (s *Screen) activateIcon(i icon) (tui.Screen, tea.Cmd) {
	s.selected = int(i)
	switch i {
	case iconPlay:
		s.pet = s.pet.Play()
		s.setFlourish("*plays happily*")
	case iconClean:
		s.pet = s.pet.Clean(s.now)
		s.setFlourish("*tidied up*")
	}
	return s, nil
}

// wrapIcon cycles i by delta, wrapping at both ends of the icon bar.
func wrapIcon(i, delta int) int {
	n := int(numIcons)
	return ((i+delta)%n + n) % n
}

// shortHelp returns the icon bar's key hints, or nil before the Pet has
// hatched — there's nothing to hint at yet.
func (s *Screen) shortHelp() []key.Binding {
	if s.pet.Stage(s.now) != pet.StageBaby {
		return nil
	}
	return []key.Binding{s.keys.Left, s.keys.Right, s.keys.Enter, s.keys.Play, s.keys.Clean}
}
