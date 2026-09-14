package next

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"

	"github.com/leekli/tamagotchi-go/internal/anim"
	"github.com/leekli/tamagotchi-go/internal/pet"
	"github.com/leekli/tamagotchi-go/internal/tui"
)

// RestartZoneID is the bubblezone id of the restart prompt shown once the
// Pet has reached Death. It is the Next Screen's only click target once
// dead, the same way welcome.BeginZoneID is the Welcome Screen's only click
// target.
const RestartZoneID = "next.restart"

// restartPromptText is deliberately worded differently from the Welcome
// Screen's "Press Enter or click to begin", so a player can never mistake
// reaching Death for simply being back at the start.
const restartPromptText = "Press Enter or click to hatch a new Egg"

// restartFramesPerPulse mirrors the Welcome Screen begin prompt's pulse
// timing exactly — one full dim -> bright -> dim cycle per second — but is
// kept local to this package since Screens never import one another
// (ADR-0003). Only the timing constant is local; the pulse logic and
// rendering are the shared tui.PulseLevel/tui.RenderChip.
const restartFramesPerPulse = anim.FPS

// isRestartClick reports whether msg is a left press inside the restart
// prompt's zone, the same nil-safe pattern welcome.isBeginClick uses.
func isRestartClick(msg tea.MouseMsg) bool {
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return false
	}
	if zone.DefaultManager == nil {
		return false
	}
	return zone.Get(RestartZoneID).InBounds(msg)
}

// updateDeathKey handles a key press once the Pet has reached Death: only
// the Restart binding does anything.
func (s *Screen) updateDeathKey(msg tea.KeyMsg) (tui.Screen, tea.Cmd) {
	if key.Matches(msg, s.keys.Restart) {
		return s.restart()
	}
	return s, nil
}

// updateDeathMouse handles a mouse message once the Pet has reached Death:
// only a click on the restart prompt's zone does anything.
func (s *Screen) updateDeathMouse(msg tea.MouseMsg) (tui.Screen, tea.Cmd) {
	if isRestartClick(msg) {
		return s.restart()
	}
	return s, nil
}

// restart discards the current (dead) Pet and replaces it with a brand new
// one — the same starting state a first launch already produces — and
// saves immediately, so a quit right after restarting still persists the
// fresh Egg rather than the Pet that just died. It also resets the icon
// bar's own selection state, so a restarted Pet never inherits a stray
// selection, an open Feed chooser, or a lingering flourish from its
// previous life.
func (s *Screen) restart() (tui.Screen, tea.Cmd) {
	s.pet = pet.New(s.now)
	s.selected = 0
	s.menu = menuIcons
	s.feedSelected = 0
	s.flourish = ""
	return s, pet.SaveCmd(s.store, s.pet)
}
