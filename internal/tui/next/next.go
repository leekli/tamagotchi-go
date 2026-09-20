// Package next renders the Next Screen: the Screen reached from the Welcome
// Screen. It shows the Pet — the creature the player is raising — as it
// exists, ages, hatches from an Egg into a Baby, and decays over real
// elapsed time, whether or not the game is running. There is no player
// action yet (feed/play/clean); that is a later feature.
package next

import (
	"fmt"
	"strings"
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

	// now is the Screen's own view of wall-clock time, seeded from the latest
	// instant the initial Pet's record reaches (see pet.Pet.RecordedUntil),
	// read from the real clock straight away by Init's anim.Now, and refreshed on
	// every anim.TickMsg. For a Pet that died long ago the seed is the moment of
	// death, so even the first frame is the Death panel. It
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
		now:    initial.RecordedUntil(),
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

// advanceToNow brings the Pet up to the Screen's own clock and reports whether it
// is still alive. Pet.Advance only runs on the slow Beat, so between Beats the
// Pet lags s.now by up to a Beat; a Care action applied to that stale Pet would
// land on old state and, for Clean and Snack/Play, would change Happiness's Decay
// rate retroactively over the time since the last Beat. Every state-changing Care
// action therefore calls this first, as Pet.Advance's contract requires. It
// neither saves nor reschedules the Beat: the Beat still owns both.
//
// Advancing can be what finds the Pet dead (it may have starved since the last
// Beat), in which case there is nothing left to act on: the caller must do
// nothing, and the Screen has already cleared its interaction state.
func (s *Screen) advanceToNow() (alive bool) {
	s.pet = s.pet.Advance(s.now)
	if s.dead() {
		s.settleDeath()
		return false
	}
	return true
}

// dead reports whether the Pet is dead as of the Screen's clock.
func (s *Screen) dead() bool { return s.pet.Stage(s.now) == pet.StageDeath }

// settleDeath clears the state of any Care action in progress once the Pet has
// died: the selection, an open Feed chooser and a showing flourish. A dead Pet
// has no Icon bar to hold them, and the Restart that follows starts clean.
func (s *Screen) settleDeath() {
	s.selected = 0
	s.menu = menuIcons
	s.feedSelected = 0
	s.flourish = ""
}

// ID implements tui.Screen.
func (s *Screen) ID() tui.ScreenID { return tui.NextScreenID }

// Frame reports the current animation frame. Intended for diagnostics and
// tests; the number only ever increases and is unaffected by resizes.
func (s *Screen) Frame() int { return s.frame }

// Pet reports the Screen's current Pet. Intended for tests.
func (s *Screen) Pet() pet.Pet { return s.pet }

// Init implements tui.Screen. It starts both of the Screen's clocks, the fast
// animation clock and the much slower simulation Beat, and reads the real clock
// at once, so that Restart (which stamps the new Egg's birth from the Screen's
// clock) never runs on the seed.
func (s *Screen) Init() tea.Cmd {
	return tea.Batch(anim.Now(), anim.Tick(), pet.Beat())
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
		if s.dead() {
			s.settleDeath()
		}
		return s, anim.Tick()

	case pet.BeatMsg:
		s.pet = s.pet.Advance(msg.Time)
		if s.dead() {
			s.settleDeath()
		}
		return s, tea.Batch(pet.Beat(), pet.SaveCmd(s.store, s.pet))

	case tea.KeyMsg:
		// Death is handled entirely separately from the icon bar: once dead
		// there is no icon bar to route through, only the Restart binding.
		if s.pet.Stage(s.now) == pet.StageDeath {
			return s.updateDeathKey(msg)
		}
		return s.updateIconBarKey(msg)

	case tea.MouseMsg:
		if s.pet.Stage(s.now) == pet.StageDeath {
			return s.updateDeathMouse(msg)
		}
		return s.updateIconBarMouse(msg)
	}
	return s, nil
}

// ShortHelp implements tui.HelpProvider: the icon bar's key hints while Care
// actions are available, the restart hint once the Pet has reached Death, or
// none before it has hatched.
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
	if stage == pet.StageDeath {
		return s.viewDeath()
	}

	frameArt, bob := eggArt, 0
	switch stage {
	case pet.StageBaby:
		frameArt, bob = alternatingPose(babyFrames, babyPoseFramesPerStep, s.frame), hatchedBob(s.frame)
	case pet.StageChild:
		frameArt, bob = childArt, hatchedBob(s.frame)
	case pet.StageTeen:
		frameArt, bob = alternatingPose(teenFrames, teenPoseFramesPerStep, s.frame), hatchedBob(s.frame)
	case pet.StageAdult:
		frameArt, bob = alternatingPose(adultFrames, adultPoseFramesPerStep, s.frame), hatchedBob(s.frame)
	}

	petContent := lipgloss.JoinVertical(lipgloss.Center,
		renderArtBox(frameArt, bob, s.styles.art),
		renderStageLabel(stage, s.styles.stageLabel),
	)

	// The Mess line and the flourish line each always occupy a row, blank
	// when absent, the same fixed-height discipline renderArtBox uses for
	// the art (via artBoxHeight) — so a Mess appearing/clearing or a
	// flourish showing/clearing never shifts the rest of the centred stack.
	// The STATS panel's own fixed height (statsPanelHeight) backs this up
	// regardless.
	messLine := ""
	if s.pet.HasMess(s.now) {
		messLine = renderMessLine(s.styles.mess)
	}
	sickLine := ""
	if s.pet.Sick(s.now) {
		sickLine = renderSickLine(s.styles.sick)
	}
	attentionLine := renderAttentionLine(
		s.pet.HungerAttention(s.now), s.pet.HappinessAttention(s.now),
		s.styles.attention, s.styles.attentionLapsed)
	statsContent := lipgloss.JoinVertical(lipgloss.Center,
		messLine,
		sickLine,
		attentionLine,
		"",
		renderMeter("Hunger", s.pet.Hunger, s.styles.meterGood, s.styles.meterFair, s.styles.meterLow),
		renderMeter("Happiness", s.pet.Happiness, s.styles.meterGood, s.styles.meterFair, s.styles.meterLow),
		"",
		s.styles.info.Render(infoLine(s.pet, s.now)),
	)

	topRow := lipgloss.JoinHorizontal(lipgloss.Top,
		s.styles.petPanel.Render(petContent),
		panelGap,
		s.styles.statsPanel.Render(statsContent),
	)

	rows := []string{topRow}
	if stage.CareAvailable() {
		menuRow := renderIconBar(s.selected, s.styles.tabNormal, s.styles.tabSelected, s.styles.tabBorder)
		if s.menu == menuFeedChoice {
			menuRow = renderFeedChoice(s.feedSelected, s.styles.tabNormal, s.styles.tabSelected, s.styles.tabBorder)
		}
		flourishLine := ""
		if s.flourish != "" {
			flourishLine = s.styles.flourish.Render(s.flourish)
		}
		rows = append(rows, "", menuRow, flourishLine)
	} else {
		// Egg: no Icon bar yet, but the same rows (gap, the tab row's full
		// height, flourish) are still reserved blank, so Hatch never
		// visibly grows the screen.
		blankMenuRow := strings.Repeat("\n", tabHeight-1)
		rows = append(rows, "", blankMenuRow, "")
	}

	stack := lipgloss.JoinVertical(lipgloss.Center, rows...)
	return lipgloss.Place(s.width, s.height, lipgloss.Center, lipgloss.Center, stack)
}

// viewDeath renders the Death Stage on its own: the gravestone art, the
// "Death" label, the Pet's frozen Age/Weight, and the restart prompt. There
// are deliberately no Hunger/Happiness meters, no Mess indicator, and no
// Care-action icon bar here — none of them mean anything once the Pet has
// died, so they are left out of this render path entirely rather than
// hidden by a condition inside the shared one above.
func (s *Screen) viewDeath() string {
	restartChip := tui.RenderChip(restartPromptText, s.frame, restartFramesPerPulse,
		s.styles.restartDim, s.styles.restartMid, s.styles.restartHi, RestartZoneID)

	content := lipgloss.JoinVertical(lipgloss.Center,
		renderArtBox(deathArt, 0, s.styles.art),
		renderStageLabel(pet.StageDeath, s.styles.stageLabel),
		"",
		s.styles.info.Render(infoLine(s.pet, s.now)),
		s.styles.info.Render("Cause: "+s.pet.CauseAt(s.now).String()),
		s.styles.info.Render(fmt.Sprintf("Care mistakes: %d", s.pet.CareMistakes)),
		"",
		// One extra blank row of breathing room before Restart, so the
		// Death panel's total height matches the hatched composite's
		// (docs/adr/0007) rather than differing by one row.
		"",
		restartChip,
	)

	panel := s.styles.deathPanel.Render(content)
	return lipgloss.Place(s.width, s.height, lipgloss.Center, lipgloss.Center, panel)
}

// Scrollable implements tui.Screen. The Next Screen is now authored to fit
// the minimum terminal and paints every cell, so — like the Welcome Screen —
// it is framed without a viewport.
func (s *Screen) Scrollable() bool { return false }
