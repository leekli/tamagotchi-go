package next

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"

	"github.com/leekli/tamagotchi-go/internal/pet"
	"github.com/leekli/tamagotchi-go/internal/tui"
)

// Zone ids for the icon bar's and Feed chooser's click targets, following
// the same zone.Mark/zone.Get pattern the Welcome Screen's begin prompt uses.
const (
	FeedZoneID  = "next.feed"
	PlayZoneID  = "next.play"
	CleanZoneID = "next.clean"

	MealZoneID  = "next.meal"
	SnackZoneID = "next.snack"
)

// icon indexes the icon bar, in on-screen order.
type icon int

const (
	iconFeed icon = iota
	iconPlay
	iconClean
	numIcons
)

var iconLabel = [numIcons]string{
	iconFeed:  "Feed",
	iconPlay:  "Play",
	iconClean: "Clean",
}

var iconZoneID = [numIcons]string{
	iconFeed:  FeedZoneID,
	iconPlay:  PlayZoneID,
	iconClean: CleanZoneID,
}

// feedOption indexes Feed's Meal/Snack chooser, kept as its own index space
// from icon so a bug in one selection can't corrupt the other.
type feedOption int

const (
	optionMeal feedOption = iota
	optionSnack
	numFeedOptions
)

var feedOptionLabel = [numFeedOptions]string{
	optionMeal:  "Meal",
	optionSnack: "Snack",
}

var feedZoneID = [numFeedOptions]string{
	optionMeal:  MealZoneID,
	optionSnack: SnackZoneID,
}

// menu names which selection the icon bar is currently showing.
type menu int

const (
	// menuIcons is the top-level Feed/Play/Clean bar.
	menuIcons menu = iota
	// menuFeedChoice is Feed's inline Meal/Snack chooser.
	menuFeedChoice
)

// keyMap is the Next Screen's own bindings, distinct from tui.KeyMap's
// App-wide ones.
type keyMap struct {
	Left  key.Binding
	Right key.Binding
	Enter key.Binding
	Esc   key.Binding
	Feed  key.Binding
	Play  key.Binding
	Clean key.Binding
	Meal  key.Binding
	Snack key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		Left:  key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/→", "select")),
		Right: key.NewBinding(key.WithKeys("right", "l")),
		Enter: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "activate")),
		Esc:   key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
		Feed:  key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "feed")),
		Play:  key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "play")),
		Clean: key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "clean")),
		Meal:  key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "meal")),
		Snack: key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "snack")),
	}
}

// Selected reports the currently selected icon's index. Intended for tests.
func (s *Screen) Selected() int { return s.selected }

// updateIconBarKey handles a key press against whichever menu is currently
// open. It is a no-op before the Pet has hatched into a Baby — there is
// nothing to act on yet.
func (s *Screen) updateIconBarKey(msg tea.KeyMsg) (tui.Screen, tea.Cmd) {
	if s.pet.Stage(s.now) != pet.StageBaby {
		return s, nil
	}
	if s.menu == menuFeedChoice {
		return s.updateFeedChoiceKey(msg)
	}
	return s.updateIconsKey(msg)
}

// updateIconsKey handles the top-level icon bar. Reaching this method at all
// already implies the Feed chooser is closed, so Play/Clean's hotkeys and
// re-pressing Feed are exactly the inputs live here — the chooser's own
// keys are handled only by updateFeedChoiceKey.
func (s *Screen) updateIconsKey(msg tea.KeyMsg) (tui.Screen, tea.Cmd) {
	switch {
	case key.Matches(msg, s.keys.Left):
		s.selected = wrap(s.selected, -1, int(numIcons))
	case key.Matches(msg, s.keys.Right):
		s.selected = wrap(s.selected, 1, int(numIcons))
	case key.Matches(msg, s.keys.Enter):
		return s.activateIcon(icon(s.selected))
	case key.Matches(msg, s.keys.Feed):
		return s.activateIcon(iconFeed)
	case key.Matches(msg, s.keys.Play):
		return s.activateIcon(iconPlay)
	case key.Matches(msg, s.keys.Clean):
		return s.activateIcon(iconClean)
	}
	return s, nil
}

// updateFeedChoiceKey handles the Meal/Snack chooser. Only its own keys are
// recognised here — the outer bar's Feed/Play/Clean hotkeys are simply not
// matched by any case below, so they (and a re-press of Feed) are no-ops
// while the chooser is open, exactly as updateIconsKey's hotkeys are no-ops
// once the chooser has taken over.
func (s *Screen) updateFeedChoiceKey(msg tea.KeyMsg) (tui.Screen, tea.Cmd) {
	switch {
	case key.Matches(msg, s.keys.Left):
		s.feedSelected = wrap(s.feedSelected, -1, int(numFeedOptions))
	case key.Matches(msg, s.keys.Right):
		s.feedSelected = wrap(s.feedSelected, 1, int(numFeedOptions))
	case key.Matches(msg, s.keys.Enter):
		return s.chooseFeed(feedOption(s.feedSelected))
	case key.Matches(msg, s.keys.Meal):
		return s.chooseFeed(optionMeal)
	case key.Matches(msg, s.keys.Snack):
		return s.chooseFeed(optionSnack)
	case key.Matches(msg, s.keys.Esc):
		s.menu = menuIcons
	}
	return s, nil
}

// updateIconBarMouse handles a mouse message against whichever menu's click
// zones are currently live. Like updateIconBarKey, it is a no-op before the
// Pet has hatched.
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

	if s.menu == menuFeedChoice {
		for i := feedOption(0); i < numFeedOptions; i++ {
			if zone.Get(feedZoneID[i]).InBounds(msg) {
				return s.chooseFeed(i)
			}
		}
		return s, nil
	}
	for i := icon(0); i < numIcons; i++ {
		if zone.Get(iconZoneID[i]).InBounds(msg) {
			return s.activateIcon(i)
		}
	}
	return s, nil
}

// activateIcon selects i. Feed opens the Meal/Snack chooser with no stat
// change; Play and Clean run their Care action immediately and show its
// flourish.
func (s *Screen) activateIcon(i icon) (tui.Screen, tea.Cmd) {
	s.selected = int(i)
	switch i {
	case iconFeed:
		s.menu = menuFeedChoice
		s.feedSelected = int(optionMeal)
	case iconPlay:
		s.pet = s.pet.Play()
		s.setFlourish("*plays happily*")
	case iconClean:
		s.pet = s.pet.Clean(s.now)
		s.setFlourish("*tidied up*")
	}
	return s, nil
}

// chooseFeed applies opt's Feed effect, shows its flourish, and returns to
// the main icon bar.
func (s *Screen) chooseFeed(opt feedOption) (tui.Screen, tea.Cmd) {
	s.feedSelected = int(opt)
	switch opt {
	case optionSnack:
		s.pet = s.pet.Feed(pet.Snack)
		s.setFlourish("*nom nom*")
	default: // optionMeal
		s.pet = s.pet.Feed(pet.Meal)
		s.setFlourish("*munch munch*")
	}
	s.menu = menuIcons
	return s, nil
}

// wrap cycles i by delta, wrapping at both ends of a menu of size n.
func wrap(i, delta, n int) int {
	return ((i+delta)%n + n) % n
}

// shortHelp returns the current menu's key hints, or nil before the Pet has
// hatched — there's nothing to hint at yet.
func (s *Screen) shortHelp() []key.Binding {
	if s.pet.Stage(s.now) != pet.StageBaby {
		return nil
	}
	if s.menu == menuFeedChoice {
		return []key.Binding{s.keys.Left, s.keys.Right, s.keys.Enter, s.keys.Meal, s.keys.Snack, s.keys.Esc}
	}
	return []key.Binding{s.keys.Left, s.keys.Right, s.keys.Enter, s.keys.Feed, s.keys.Play, s.keys.Clean}
}
