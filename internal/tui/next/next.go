// Package next renders the Next Screen: the Screen reached from the Welcome
// Screen. It shows the Pet — the creature the player is raising — as it
// exists, ages, hatches from an Egg into a Baby, and decays over real
// elapsed time, whether or not the game is running. There is no player
// action yet (feed/play/clean); that is a later feature.
package next

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/leekli/tamagotchi-go/internal/anim"
	"github.com/leekli/tamagotchi-go/internal/pet"
	"github.com/leekli/tamagotchi-go/internal/tui"
)

// Screen is the Next Screen.
type Screen struct {
	pet   pet.Pet
	store pet.Store

	// now is the Screen's own view of wall-clock time, seeded from the
	// initial Pet's LastSeenAt and refreshed on every anim.TickMsg. It
	// drives Stage/Age rendering only — never Advance, which runs on the
	// much slower pet.Beat clock. Tracking it separately keeps the visible
	// Egg→Baby hatch responsive rather than lagging behind the Beat.
	now time.Time

	width, height int
	frame         int

	styles styles
}

// New builds a Next Screen holding initial, persisting through store.
func New(initial pet.Pet, store pet.Store) tui.Screen {
	return &Screen{
		pet:    initial,
		store:  store,
		now:    initial.LastSeenAt,
		styles: newStyles(tui.DefaultPalette()),
	}
}

// ID implements tui.Screen.
func (s *Screen) ID() tui.ScreenID { return tui.NextScreenID }

// Frame reports the current animation frame. Intended for diagnostics and
// tests; the number only ever increases and is unaffected by resizes.
func (s *Screen) Frame() int { return s.frame }

// Pet reports the Screen's current Pet. Intended for tests.
func (s *Screen) Pet() pet.Pet { return s.pet }

// Init implements tui.Screen. It starts both of the Screen's clocks: the
// fast animation clock and the much slower simulation Beat.
func (s *Screen) Init() tea.Cmd {
	return tea.Batch(anim.Tick(), pet.Beat())
}

// Update implements tui.Screen.
func (s *Screen) Update(msg tea.Msg) (tui.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height

	case anim.TickMsg:
		s.frame++
		s.now = msg.Time
		return s, anim.Tick()

	case pet.BeatMsg:
		s.pet = s.pet.Advance(msg.Time)
		return s, tea.Batch(pet.Beat(), pet.SaveCmd(s.store, s.pet))
	}
	return s, nil
}

// OnQuit implements tui.QuitHandler: it saves the current Pet before the App
// quits, so at most a periodic Beat's worth of play is ever lost on exit.
func (s *Screen) OnQuit() tea.Cmd {
	return pet.SaveCmd(s.store, s.pet)
}

// View implements tui.Screen.
func (s *Screen) View() string {
	frameArt, bob := eggArt, 0
	if s.pet.Stage(s.now) == pet.StageBaby {
		frameArt, bob = babyPose(s.frame), babyBob(s.frame)
	}

	stack := lipgloss.JoinVertical(
		lipgloss.Center,
		renderArtBox(frameArt, bob, s.styles.art),
		"",
		renderMeter("Hunger", s.pet.Hunger, s.styles.meter),
		renderMeter("Happiness", s.pet.Happiness, s.styles.meter),
		"",
		s.styles.info.Render(infoLine(s.pet, s.now)),
	)
	return lipgloss.Place(s.width, s.height, lipgloss.Center, lipgloss.Center, stack)
}

// Scrollable implements tui.Screen. The Next Screen is now authored to fit
// the minimum terminal and paints every cell, so — like the Welcome Screen —
// it is framed without a viewport.
func (s *Screen) Scrollable() bool { return false }
