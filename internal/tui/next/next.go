// Package next renders the Next Screen: the Screen reached from the Welcome
// Screen. It shows the Pet — the creature the player is raising — as it
// exists, ages, hatches from an Egg into a Baby, and decays over real
// elapsed time, whether or not the game is running. There is no player
// action yet (feed/play/clean); that is a later feature.
package next

import (
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/leekli/tamagotchi-go/internal/anim"
	"github.com/leekli/tamagotchi-go/internal/pet"
	"github.com/leekli/tamagotchi-go/internal/tui"
)

// flourishDuration is how long a Care action's text flourish (e.g. "*tidied
// up*") stays on screen before clearing itself, timed on the same animation
// clock the Welcome Screen's begin-prompt pulse uses.
const flourishDuration = 2 * time.Second

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

	// selected is the currently highlighted icon in the icon bar (see
	// iconbar.go). It cycles across numIcons, wrapping at both ends.
	selected int
	// menu is which selection is currently showing: the top-level icon bar,
	// or Feed's Meal/Snack chooser.
	menu menu
	// feedSelected is the chooser's own selection index, kept separate from
	// selected so a bug in one selection can't corrupt the other.
	feedSelected int
	keys         keyMap

	// flourish is the current Care-action feedback text (e.g. "*tidied
	// up*"), or "" when none is showing. flourishUntil is the frame at which
	// it clears itself.
	flourish      string
	flourishUntil int

	styles styles
}

// New builds a Next Screen holding initial, persisting through store.
func New(initial pet.Pet, store pet.Store) tui.Screen {
	return &Screen{
		pet:    initial,
		store:  store,
		now:    initial.LastSeenAt,
		keys:   defaultKeyMap(),
		styles: newStyles(tui.DefaultPalette()),
	}
}

// setFlourish shows text as the current Care-action feedback until
// flourishDuration's worth of animation frames have passed.
func (s *Screen) setFlourish(text string) {
	s.flourish = text
	s.flourishUntil = s.frame + int(flourishDuration/anim.FrameInterval)
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
		if msg.Time.After(s.now) {
			// Only ever move forward: a corrected system clock ticking
			// backwards must not un-hatch the Pet or show a negative Age, the
			// same discipline pet.Advance already applies to Decay.
			s.now = msg.Time
		}
		if s.flourish != "" && s.frame >= s.flourishUntil {
			s.flourish = ""
		}
		return s, anim.Tick()

	case pet.BeatMsg:
		s.pet = s.pet.Advance(msg.Time)
		return s, tea.Batch(pet.Beat(), pet.SaveCmd(s.store, s.pet))

	case tea.KeyMsg:
		return s.updateIconBarKey(msg)

	case tea.MouseMsg:
		return s.updateIconBarMouse(msg)
	}
	return s, nil
}

// ShortHelp implements tui.HelpProvider: the icon bar's key hints once the
// Pet has hatched, or none before then.
func (s *Screen) ShortHelp() []key.Binding {
	return s.shortHelp()
}

// OnQuit implements tui.QuitHandler: it saves the current Pet before the App
// quits, so at most a periodic Beat's worth of play is ever lost on exit.
func (s *Screen) OnQuit() tea.Cmd {
	return pet.SaveCmd(s.store, s.pet)
}

// View implements tui.Screen.
func (s *Screen) View() string {
	stage := s.pet.Stage(s.now)

	frameArt, bob := eggArt, 0
	switch stage {
	case pet.StageBaby:
		frameArt, bob = babyPose(s.frame), hatchedBob(s.frame)
	case pet.StageChild:
		frameArt, bob = childArt, hatchedBob(s.frame)
	}

	// The Mess line and the flourish line each always occupy a row, blank
	// when absent, the same fixed-height discipline renderArtBox uses for
	// the art (via artBoxHeight) — so a Mess appearing/clearing or a
	// flourish showing/clearing never shifts the rest of the centred stack.
	messLine := ""
	if s.pet.HasMess(s.now) {
		messLine = renderMessLine(s.styles.mess)
	}

	rows := []string{
		renderArtBox(frameArt, bob, s.styles.art),
		messLine,
		"",
		renderMeter("Hunger", s.pet.Hunger, s.styles.meter),
		renderMeter("Happiness", s.pet.Happiness, s.styles.meter),
		"",
		s.styles.info.Render(infoLine(s.pet, s.now)),
	}
	if stage.Hatched() {
		menuRow := renderIconBar(s.selected, s.styles.icon, s.styles.iconSelected)
		if s.menu == menuFeedChoice {
			menuRow = renderFeedChoice(s.feedSelected, s.styles.icon, s.styles.iconSelected)
		}
		flourishLine := ""
		if s.flourish != "" {
			flourishLine = s.styles.flourish.Render(s.flourish)
		}
		rows = append(rows, "", menuRow, flourishLine)
	}

	stack := lipgloss.JoinVertical(lipgloss.Center, rows...)
	return lipgloss.Place(s.width, s.height, lipgloss.Center, lipgloss.Center, stack)
}

// Scrollable implements tui.Screen. The Next Screen is now authored to fit
// the minimum terminal and paints every cell, so — like the Welcome Screen —
// it is framed without a viewport.
func (s *Screen) Scrollable() bool { return false }
