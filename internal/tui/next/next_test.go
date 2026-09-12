package next_test

import (
	"errors"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/leekli/tamagotchi-go/internal/anim"
	"github.com/leekli/tamagotchi-go/internal/pet"
	"github.com/leekli/tamagotchi-go/internal/tui"
	"github.com/leekli/tamagotchi-go/internal/tui/next"
)

// TestMain gives the package a live bubblezone manager (so the icon bar's
// click zones resolve) and forces a deterministic colour profile so the
// Screen's styling produces visible escape codes regardless of whether the
// test host is a terminal, matching the Welcome Screen's test setup.
func TestMain(m *testing.M) {
	zone.NewGlobal()
	lipgloss.SetColorProfile(termenv.ANSI256)
	os.Exit(m.Run())
}

// babyScreen builds a sized Screen already hatched into a Baby as of born,
// so icon-bar tests don't each need to advance past EggDuration themselves.
func babyScreen(t *testing.T, initial pet.Pet, store pet.Store) tui.Screen {
	t.Helper()
	s := sizedScreen(t, initial, store)
	return advanceAnim(t, s, born.Add(pet.EggDuration), 1)
}

// childScreen mirrors babyScreen, but grown all the way into a Child.
func childScreen(t *testing.T, initial pet.Pet, store pet.Store) tui.Screen {
	t.Helper()
	s := sizedScreen(t, initial, store)
	return advanceAnim(t, s, born.Add(pet.EggDuration+pet.BabyDuration), 1)
}

// teenScreen mirrors childScreen, but grown all the way into a Teen.
func teenScreen(t *testing.T, initial pet.Pet, store pet.Store) tui.Screen {
	t.Helper()
	s := sizedScreen(t, initial, store)
	return advanceAnim(t, s, born.Add(pet.EggDuration+pet.BabyDuration+pet.ChildDuration), 1)
}

// fakeStore is an in-memory pet.Store for tests, recording every Save.
type fakeStore struct {
	saved   []pet.Pet
	saveErr error
	loadPet pet.Pet
	loadOK  bool
	loadErr error
}

func (f *fakeStore) Load() (pet.Pet, bool, error) { return f.loadPet, f.loadOK, f.loadErr }

func (f *fakeStore) Save(p pet.Pet) error {
	f.saved = append(f.saved, p)
	return f.saveErr
}

var born = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func newScreen(t *testing.T, initial pet.Pet, store pet.Store) tui.Screen {
	t.Helper()
	s := next.New(initial, store)
	require.Equal(t, tui.NextScreenID, s.ID())
	return s
}

// sizedScreen builds a Screen at the game's minimum body size (80x23 — 80x24
// less the App's help-bar row); every test in this file renders at that one
// size, so it isn't a parameter.
func sizedScreen(t *testing.T, initial pet.Pet, store pet.Store) tui.Screen {
	t.Helper()
	s := newScreen(t, initial, store)
	s, _ = s.Update(tea.WindowSizeMsg{Width: 80, Height: 23})
	return s
}

// advanceAnim feeds n animation ticks at the given time, returning the
// updated Screen.
func advanceAnim(t *testing.T, s tui.Screen, now time.Time, n int) tui.Screen {
	t.Helper()
	for i := 0; i < n; i++ {
		var cmd tea.Cmd
		s, cmd = s.Update(anim.TickMsg{Time: now})
		require.NotNil(t, cmd, "every anim tick should reschedule the next")
	}
	return s
}

func TestInitStartsBothClocks(t *testing.T) {
	t.Parallel()

	cmd := newScreen(t, pet.New(born), &fakeStore{}).Init()
	require.NotNil(t, cmd)

	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	require.True(t, ok, "expected a tea.BatchMsg starting both clocks, got %T", msg)
	require.Len(t, batch, 2)
}

func TestScreenIsNotScrollable(t *testing.T) {
	t.Parallel()
	assert.False(t, newScreen(t, pet.New(born), &fakeStore{}).Scrollable())
}

func TestViewShowsEggBeforeHatchAndBabyAfter(t *testing.T) {
	t.Parallel()

	s := sizedScreen(t, pet.New(born), &fakeStore{})

	beforeHatch := stripANSI(s.View())
	assert.Contains(t, beforeHatch, ".--.", "Egg art should be shown before the hatch boundary")
	assert.NotContains(t, beforeHatch, "( o o)", "Baby art should not appear yet")

	s = advanceAnim(t, s, born.Add(pet.EggDuration), 1)
	afterHatch := stripANSI(s.View())
	assert.Contains(t, afterHatch, "( o o)", "Baby art should be shown after the hatch boundary")
	assert.NotContains(t, afterHatch, ".--.", "Egg art should no longer appear")
}

func TestViewDoesNotUnHatchIfTheClockGoesBackwards(t *testing.T) {
	t.Parallel()

	s := sizedScreen(t, pet.New(born), &fakeStore{})
	s = advanceAnim(t, s, born.Add(pet.EggDuration), 1)
	require.Contains(t, stripANSI(s.View()), "( o o)", "should be a Baby after hatching")

	// A corrected system clock ticking backwards must not un-hatch the Pet.
	s = advanceAnim(t, s, born, 1)

	view := stripANSI(s.View())
	assert.Contains(t, view, "( o o)", "should still show Baby art")
	assert.NotContains(t, view, ".--.", "should not revert to Egg art")
}

func TestViewShowsChildAfterBabyDuration(t *testing.T) {
	t.Parallel()

	s := sizedScreen(t, pet.New(born), &fakeStore{})
	s = advanceAnim(t, s, born.Add(pet.EggDuration), 1)
	require.Contains(t, stripANSI(s.View()), "( o o)", "should be a Baby before BabyDuration has elapsed")

	s = advanceAnim(t, s, born.Add(pet.EggDuration+pet.BabyDuration), 1)
	view := stripANSI(s.View())
	assert.Contains(t, view, "( ^  ^)", "Child art should be shown after BabyDuration has elapsed")
	assert.NotContains(t, view, "( o o)", "Baby art should no longer appear")
}

func TestViewShowsTeenAfterChildDuration(t *testing.T) {
	t.Parallel()

	s := sizedScreen(t, pet.New(born), &fakeStore{})
	s = advanceAnim(t, s, born.Add(pet.EggDuration+pet.BabyDuration), 1)
	require.Contains(t, stripANSI(s.View()), "( ^  ^)", "should be a Child before ChildDuration has elapsed")

	s = advanceAnim(t, s, born.Add(pet.EggDuration+pet.BabyDuration+pet.ChildDuration), 1)
	view := stripANSI(s.View())
	assert.Contains(t, view, "^==^", "Teen art should be shown after ChildDuration has elapsed")
	assert.NotContains(t, view, "( ^  ^)", "Child art should no longer appear")
}

func TestTeenWalkCycleAlternatesPoses(t *testing.T) {
	t.Parallel()

	teenAt := born.Add(pet.EggDuration + pet.BabyDuration + pet.ChildDuration)
	s := sizedScreen(t, pet.New(born), &fakeStore{})

	var sawRightLimbPose, sawLeftLimbPose bool
	for i := 0; i < 64; i++ {
		s = advanceAnim(t, s, teenAt, 1)
		view := stripANSI(s.View())
		if strings.Contains(view, ")>") {
			sawRightLimbPose = true
		}
		if strings.Contains(view, "<(") {
			sawLeftLimbPose = true
		}
	}
	assert.True(t, sawRightLimbPose, "Teen's walk cycle should show its right-limb pose at some point")
	assert.True(t, sawLeftLimbPose, "Teen's walk cycle should show its left-limb pose at some point")
}

func TestIconBarAndCareActionsRemainAvailableForTeen(t *testing.T) {
	t.Parallel()

	p := pet.New(born)
	p.Happiness = 1
	s := teenScreen(t, p, &fakeStore{})

	view := stripANSI(s.View())
	assert.Contains(t, view, "Feed")
	assert.Contains(t, view, "Play")
	assert.Contains(t, view, "Clean")

	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	ns, ok := s.(*next.Screen)
	require.True(t, ok)
	assert.Equal(t, 2, ns.Pet().Happiness, "Play should still work once the Pet is a Teen")
}

func TestMouseClickActivatesIconsForTeen(t *testing.T) {
	t.Parallel()

	p := pet.New(born)
	p.Happiness = 1
	s := teenScreen(t, p, &fakeStore{})

	s = clickZone(t, s, next.PlayZoneID)

	ns, ok := s.(*next.Screen)
	require.True(t, ok)
	assert.Equal(t, 2, ns.Pet().Happiness, "clicking Play should still work once the Pet is a Teen")
}

func TestShortHelpAdvertisesIconBarHintsForTeenToo(t *testing.T) {
	t.Parallel()

	teen, ok := teenScreen(t, pet.New(born), &fakeStore{}).(tui.HelpProvider)
	require.True(t, ok)
	assert.NotEmpty(t, teen.ShortHelp())
}

func TestIconBarAndCareActionsRemainAvailableForChild(t *testing.T) {
	t.Parallel()

	p := pet.New(born)
	p.Happiness = 1
	s := childScreen(t, p, &fakeStore{})

	view := stripANSI(s.View())
	assert.Contains(t, view, "Feed")
	assert.Contains(t, view, "Play")
	assert.Contains(t, view, "Clean")

	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	ns, ok := s.(*next.Screen)
	require.True(t, ok)
	assert.Equal(t, 2, ns.Pet().Happiness, "Play should still work once the Pet is a Child")
}

func TestMouseClickActivatesIconsForChild(t *testing.T) {
	t.Parallel()

	p := pet.New(born)
	p.Happiness = 1
	s := childScreen(t, p, &fakeStore{})

	s = clickZone(t, s, next.PlayZoneID)

	ns, ok := s.(*next.Screen)
	require.True(t, ok)
	assert.Equal(t, 2, ns.Pet().Happiness, "clicking Play should still work once the Pet is a Child")
}

func TestShortHelpAdvertisesIconBarHintsForChildToo(t *testing.T) {
	t.Parallel()

	child, ok := childScreen(t, pet.New(born), &fakeStore{}).(tui.HelpProvider)
	require.True(t, ok)
	assert.NotEmpty(t, child.ShortHelp())
}

func TestViewShowsTheStageLabelForEggBabyChildAndTeen(t *testing.T) {
	t.Parallel()

	s := sizedScreen(t, pet.New(born), &fakeStore{})
	assert.Contains(t, stripANSI(s.View()), "Egg", "should show the Egg label before Hatch")

	s = advanceAnim(t, s, born.Add(pet.EggDuration), 1)
	assert.Contains(t, stripANSI(s.View()), "Baby", "should show the Baby label after Hatch")

	s = advanceAnim(t, s, born.Add(pet.EggDuration+pet.BabyDuration), 1)
	assert.Contains(t, stripANSI(s.View()), "Child", "should show the Child label once grown")

	s = advanceAnim(t, s, born.Add(pet.EggDuration+pet.BabyDuration+pet.ChildDuration), 1)
	assert.Contains(t, stripANSI(s.View()), "Teen", "should show the Teen label once grown further")
}

func TestHungerAndHappinessMetersShowTheRightPips(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		value int
		want  string
	}{
		"empty": {0, "[----]"},
		"one":   {1, "[*---]"},
		"half":  {2, "[**--]"},
		"three": {3, "[***-]"},
		"full":  {4, "[****]"},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			p := pet.New(born)
			p.Hunger = tt.value
			p.Happiness = tt.value

			view := stripANSI(sizedScreen(t, p, &fakeStore{}).View())
			assert.Contains(t, view, tt.want)
		})
	}
}

func TestViewShowsAgeAndWeight(t *testing.T) {
	t.Parallel()

	s := sizedScreen(t, pet.New(born), &fakeStore{})
	view := stripANSI(s.View())

	assert.Contains(t, view, "Day 0")
	assert.Contains(t, view, "Weight 2g")
}

func TestBeatAdvancesThePetAndReissuesBeatAndSave(t *testing.T) {
	t.Parallel()

	store := &fakeStore{}
	s := newScreen(t, pet.New(born), store)

	later := born.Add(pet.HungerDecayInterval)
	s, cmd := s.Update(pet.BeatMsg{Time: later})

	require.NotNil(t, cmd, "a Beat should re-issue the next Beat and a save")
	ns, ok := s.(*next.Screen)
	require.True(t, ok)
	assert.Equal(t, pet.MaxStat-1, ns.Pet().Hunger, "the Beat should have advanced the held Pet")

	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	require.True(t, ok, "expected a tea.BatchMsg, got %T", msg)
	require.Len(t, batch, 2)

	// batch[0] is the re-issued pet.Beat() (see next.Screen.Update) — deliberately
	// not invoked here, since it would block for the real BeatInterval. batch[1]
	// is the SaveCmd; invoking just that is enough to prove the save fires.
	batch[1]()
	require.Len(t, store.saved, 1)
	assert.Equal(t, pet.MaxStat-1, store.saved[0].Hunger)
}

func TestOnQuitSavesTheCurrentPetViaTheStore(t *testing.T) {
	t.Parallel()

	store := &fakeStore{}
	s := newScreen(t, pet.New(born), store)

	qh, ok := s.(tui.QuitHandler)
	require.True(t, ok, "Next Screen should implement tui.QuitHandler")

	cmd := qh.OnQuit()
	require.NotNil(t, cmd)
	cmd()

	require.Len(t, store.saved, 1)
}

// TestOnQuitDiscardsAFailedSaveWithoutPanicking proves a Store whose Save
// fails on quit still leaves the app able to exit cleanly, matching
// pet.SaveCmd's documented "a failed save is not fatal" contract.
func TestOnQuitDiscardsAFailedSaveWithoutPanicking(t *testing.T) {
	t.Parallel()

	store := &fakeStore{saveErr: errors.New("disk full")}
	s := newScreen(t, pet.New(born), store)

	qh, ok := s.(tui.QuitHandler)
	require.True(t, ok)

	cmd := qh.OnQuit()
	require.NotNil(t, cmd)

	assert.NotPanics(t, func() {
		assert.Nil(t, cmd())
	})
	assert.Len(t, store.saved, 1, "Save should still have been attempted")
}

func TestScreenNowIsSeededFromInitialLastSeenAt(t *testing.T) {
	t.Parallel()

	// A duration of exactly one HungerDecayInterval both moves LastSeenAt
	// forward by a whole step (Advance only ever consumes whole steps, so a
	// fractional interval would leave it at born) and stays well short of
	// EggDuration+BabyDuration — this test wants Baby art, not Child art.
	initial := pet.New(born).Advance(born.Add(pet.HungerDecayInterval)) // LastSeenAt moves to born+3m
	s := sizedScreen(t, initial, &fakeStore{})

	// The hatch boundary (EggDuration after CreatedAt=born) has passed
	// relative to the seeded now (born+3m), so the Screen should already
	// show the Baby art on its very first render — before any anim.TickMsg.
	view := stripANSI(s.View())
	assert.Contains(t, view, "( o o)")
}

var ansiSeq = regexp.MustCompile("\x1b\\[[0-9;]*m")

// stripANSI removes SGR colour sequences so plain-text art and labels can be
// matched, the same helper the Welcome Screen's tests use.
func stripANSI(s string) string { return ansiSeq.ReplaceAllString(s, "") }

func TestIconBarHiddenUntilBabyStage(t *testing.T) {
	t.Parallel()

	s := sizedScreen(t, pet.New(born), &fakeStore{})
	view := stripANSI(s.View())

	assert.NotContains(t, view, "Play")
	assert.NotContains(t, view, "Clean")

	hp, ok := s.(tui.HelpProvider)
	require.True(t, ok)
	assert.Empty(t, hp.ShortHelp(), "no care-action hints while the Pet is an Egg")
}

// TestIconBarKeyIsNoopBeforeHatch proves a Care-action hotkey pressed while
// the Pet is still an Egg does nothing — there is nothing to act on yet.
func TestIconBarKeyIsNoopBeforeHatch(t *testing.T) {
	t.Parallel()

	s := sizedScreen(t, pet.New(born), &fakeStore{})

	s, cmd := s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})

	assert.Nil(t, cmd)
	ns, ok := s.(*next.Screen)
	require.True(t, ok)
	assert.Equal(t, pet.New(born).Happiness, ns.Pet().Happiness, "a hotkey before hatching should not run any Care action")
}

// TestIconBarMouseIsNoopBeforeHatch mirrors TestIconBarKeyIsNoopBeforeHatch
// for a mouse click.
func TestIconBarMouseIsNoopBeforeHatch(t *testing.T) {
	t.Parallel()

	s := sizedScreen(t, pet.New(born), &fakeStore{})

	s, cmd := s.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: 0, Y: 0})

	assert.Nil(t, cmd)
	ns, ok := s.(*next.Screen)
	require.True(t, ok)
	assert.Equal(t, pet.New(born).Happiness, ns.Pet().Happiness, "a click before hatching should not run any Care action")
}

func TestIconBarAppearsAfterHatch(t *testing.T) {
	t.Parallel()

	view := stripANSI(babyScreen(t, pet.New(born), &fakeStore{}).View())

	assert.Contains(t, view, "Feed")
	assert.Contains(t, view, "Play")
	assert.Contains(t, view, "Clean")
}

func TestIconSelectionCyclesAndWraps(t *testing.T) {
	t.Parallel()

	s := babyScreen(t, pet.New(born), &fakeStore{})

	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRight})
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRight})
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRight}) // wraps back to the first icon
	ns, ok := s.(*next.Screen)
	require.True(t, ok)
	assert.Equal(t, 0, ns.Selected())

	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyLeft}) // wraps to the last icon
	ns, ok = s.(*next.Screen)
	require.True(t, ok)
	assert.Equal(t, 2, ns.Selected())
}

func TestEnterActivatesTheSelectedIcon(t *testing.T) {
	t.Parallel()

	p := pet.New(born)
	p.LastCleanedAt = born.Add(-pet.MessInterval) // already messy
	s := babyScreen(t, p, &fakeStore{})

	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRight}) // Feed -> Play
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRight}) // Play -> Clean
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyEnter})

	ns, ok := s.(*next.Screen)
	require.True(t, ok)
	assert.False(t, ns.Pet().HasMess(born.Add(pet.EggDuration)), "Enter on Clean should clear the Mess")
}

func TestHotkeysActivateDirectlyFromAnySelection(t *testing.T) {
	t.Parallel()

	s := babyScreen(t, pet.New(born), &fakeStore{})

	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})

	ns, ok := s.(*next.Screen)
	require.True(t, ok)
	assert.Equal(t, pet.MaxStat, ns.Pet().Happiness, "already full, but Play should still have run")
	assert.Contains(t, stripANSI(s.View()), "plays happily")
}

func TestPlayIncreasesHappiness(t *testing.T) {
	t.Parallel()

	p := pet.New(born)
	p.Happiness = 1
	s := babyScreen(t, p, &fakeStore{})

	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})

	ns, ok := s.(*next.Screen)
	require.True(t, ok)
	assert.Equal(t, 2, ns.Pet().Happiness)
}

func TestFlourishClearsAfterItsDuration(t *testing.T) {
	t.Parallel()

	s := babyScreen(t, pet.New(born), &fakeStore{})
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	require.Contains(t, stripANSI(s.View()), "tidied up")

	s = advanceAnim(t, s, born.Add(pet.EggDuration+time.Second), anim.FPS*3)

	assert.NotContains(t, stripANSI(s.View()), "tidied up")
}

func TestMessIndicatorRendersOnlyWhenMessy(t *testing.T) {
	t.Parallel()

	clean := babyScreen(t, pet.New(born), &fakeStore{})
	assert.NotContains(t, stripANSI(clean.View()), "(___)")

	p := pet.New(born)
	p.LastCleanedAt = born.Add(-pet.MessInterval)
	messy := babyScreen(t, p, &fakeStore{})
	assert.Contains(t, stripANSI(messy.View()), "(___)")
}

func TestShortHelpAdvertisesIconBarHintsOnlyOnceHatched(t *testing.T) {
	t.Parallel()

	egg, ok := sizedScreen(t, pet.New(born), &fakeStore{}).(tui.HelpProvider)
	require.True(t, ok)
	assert.Empty(t, egg.ShortHelp())

	baby, ok := babyScreen(t, pet.New(born), &fakeStore{}).(tui.HelpProvider)
	require.True(t, ok)
	assert.NotEmpty(t, baby.ShortHelp())
}

// TestShortHelpHasNoBlankBindings guards against a binding with no help text
// (e.g. a Right-arrow key that only exists as a Left binding's second key)
// slipping into a ShortHelp() list, where bubbles/help would render it as a
// stray empty bullet regardless of whether it has visible help text.
func TestShortHelpHasNoBlankBindings(t *testing.T) {
	t.Parallel()

	assertNoBlankBindings := func(t *testing.T, s tui.Screen) {
		t.Helper()
		hp, ok := s.(tui.HelpProvider)
		require.True(t, ok)
		for _, b := range hp.ShortHelp() {
			assert.NotEmpty(t, b.Help().Key, "binding with empty help text in ShortHelp()")
		}
	}

	icons := babyScreen(t, pet.New(born), &fakeStore{})
	assertNoBlankBindings(t, icons)

	feedChoice, _ := icons.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	assertNoBlankBindings(t, feedChoice)
}

// iconZone scans the screen until the named zone resolves (bubblezone
// records positions on a worker goroutine) and returns its bounds, the same
// pattern welcome_test.go's beginZone helper uses.
func iconZone(t *testing.T, s tui.Screen, id string) *zone.ZoneInfo {
	t.Helper()
	var z *zone.ZoneInfo
	require.Eventually(t, func() bool {
		zone.Scan(s.View())
		z = zone.Get(id)
		return !z.IsZero()
	}, 2*time.Second, 10*time.Millisecond, "zone %q should resolve after a render", id)
	return z
}

// clickZone clicks the centre of the named zone and returns the updated
// Screen.
func clickZone(t *testing.T, s tui.Screen, id string) tui.Screen {
	t.Helper()
	z := iconZone(t, s, id)
	s, _ = s.Update(tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
		X:      (z.StartX + z.EndX) / 2,
		Y:      z.StartY,
	})
	return s
}

func TestMouseClickActivatesTheMatchingIcon(t *testing.T) {
	t.Run("clicking Play activates it like the p hotkey would", func(t *testing.T) {
		p := pet.New(born)
		p.Happiness = 1
		s := babyScreen(t, p, &fakeStore{})

		s = clickZone(t, s, next.PlayZoneID)

		ns, ok := s.(*next.Screen)
		require.True(t, ok)
		assert.Equal(t, 2, ns.Pet().Happiness)
	})

	t.Run("clicking Clean activates it regardless of the current selection", func(t *testing.T) {
		p := pet.New(born)
		p.LastCleanedAt = born.Add(-pet.MessInterval)
		s := babyScreen(t, p, &fakeStore{})

		s = clickZone(t, s, next.CleanZoneID)

		ns, ok := s.(*next.Screen)
		require.True(t, ok)
		assert.False(t, ns.Pet().HasMess(born.Add(pet.EggDuration)))
	})

	t.Run("a click outside every zone does nothing", func(t *testing.T) {
		s := babyScreen(t, pet.New(born), &fakeStore{})
		iconZone(t, s, next.PlayZoneID) // ensure zones are known, so this is a real miss

		s, cmd := s.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: 0, Y: 0})
		assert.Nil(t, cmd)
		ns, ok := s.(*next.Screen)
		require.True(t, ok)
		assert.Equal(t, pet.New(born).Happiness, ns.Pet().Happiness)
	})

	t.Run("a release on the Play zone does nothing", func(t *testing.T) {
		p := pet.New(born)
		p.Happiness = 1
		s := babyScreen(t, p, &fakeStore{})
		z := iconZone(t, s, next.PlayZoneID)

		s, cmd := s.Update(tea.MouseMsg{Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft, X: (z.StartX + z.EndX) / 2, Y: z.StartY})
		assert.Nil(t, cmd)
		ns, ok := s.(*next.Screen)
		require.True(t, ok)
		assert.Equal(t, 1, ns.Pet().Happiness, "only a press should activate Play, not a release")
	})

	t.Run("a right-button press on the Play zone does nothing", func(t *testing.T) {
		p := pet.New(born)
		p.Happiness = 1
		s := babyScreen(t, p, &fakeStore{})
		z := iconZone(t, s, next.PlayZoneID)

		s, cmd := s.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonRight, X: (z.StartX + z.EndX) / 2, Y: z.StartY})
		assert.Nil(t, cmd)
		ns, ok := s.(*next.Screen)
		require.True(t, ok)
		assert.Equal(t, 1, ns.Pet().Happiness, "only a left-button press should activate Play")
	})
}

func TestActivatingFeedOpensTheChooserWithoutChangingStats(t *testing.T) {
	t.Parallel()

	p := pet.New(born)
	p.Hunger, p.Happiness = 1, 1
	s := babyScreen(t, p, &fakeStore{})

	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})

	ns, ok := s.(*next.Screen)
	require.True(t, ok)
	assert.Equal(t, 1, ns.Pet().Hunger)
	assert.Equal(t, 1, ns.Pet().Happiness)
	view := stripANSI(s.View())
	assert.Contains(t, view, "Meal")
	assert.Contains(t, view, "Snack")
}

func TestFeedChoiceMealAndSnack(t *testing.T) {
	t.Parallel()

	t.Run("Meal increases Hunger only, via hotkey", func(t *testing.T) {
		p := pet.New(born)
		p.Hunger, p.Happiness = 1, 1
		s := babyScreen(t, p, &fakeStore{})

		s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
		s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})

		ns, ok := s.(*next.Screen)
		require.True(t, ok)
		assert.Equal(t, 2, ns.Pet().Hunger)
		assert.Equal(t, 1, ns.Pet().Happiness)
		assert.Equal(t, pet.BaseWeight, ns.Pet().Weight)
		assert.Contains(t, stripANSI(s.View()), "munch munch")
	})

	t.Run("Snack increases Happiness and Weight, via Left/Right and Enter", func(t *testing.T) {
		p := pet.New(born)
		p.Hunger, p.Happiness = 1, 1
		s := babyScreen(t, p, &fakeStore{})

		s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
		s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRight}) // Meal -> Snack
		s, _ = s.Update(tea.KeyMsg{Type: tea.KeyEnter})

		ns, ok := s.(*next.Screen)
		require.True(t, ok)
		assert.Equal(t, 1, ns.Pet().Hunger)
		assert.Equal(t, 2, ns.Pet().Happiness)
		assert.Equal(t, pet.BaseWeight+1, ns.Pet().Weight)
		assert.Contains(t, stripANSI(s.View()), "nom nom")
	})

	t.Run("Snack increases Happiness and Weight, via Left wrapping from Meal", func(t *testing.T) {
		p := pet.New(born)
		p.Hunger, p.Happiness = 1, 1
		s := babyScreen(t, p, &fakeStore{})

		s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}) // opens on Meal
		s, _ = s.Update(tea.KeyMsg{Type: tea.KeyLeft})                      // wraps Meal -> Snack
		s, _ = s.Update(tea.KeyMsg{Type: tea.KeyEnter})

		ns, ok := s.(*next.Screen)
		require.True(t, ok)
		assert.Equal(t, 1, ns.Pet().Hunger)
		assert.Equal(t, 2, ns.Pet().Happiness)
		assert.Equal(t, pet.BaseWeight+1, ns.Pet().Weight)
	})
}

func TestEscCancelsTheFeedChooserWithoutAStatChange(t *testing.T) {
	t.Parallel()

	p := pet.New(born)
	p.Hunger, p.Happiness = 1, 1
	s := babyScreen(t, p, &fakeStore{})

	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyEsc})

	ns, ok := s.(*next.Screen)
	require.True(t, ok)
	assert.Equal(t, 1, ns.Pet().Hunger)
	assert.Equal(t, 1, ns.Pet().Happiness)
	view := stripANSI(s.View())
	assert.NotContains(t, view, "Meal")
	assert.Contains(t, view, "Play", "should be back at the main icon bar")
}

func TestFeedChooserInputsAreIndependentOfTheOuterIconBar(t *testing.T) {
	t.Parallel()

	p := pet.New(born)
	p.Happiness = 1
	s := babyScreen(t, p, &fakeStore{})

	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})

	// While the chooser is open, the outer bar's Play/Clean hotkeys do
	// nothing, and re-pressing 'f' does nothing either.
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})

	ns, ok := s.(*next.Screen)
	require.True(t, ok)
	assert.Equal(t, 1, ns.Pet().Happiness, "p/c/f must be no-ops while the Feed chooser is open")
	assert.Contains(t, stripANSI(s.View()), "Meal", "the chooser should still be open")
}

func TestShortHelpSwitchesToFeedChoiceHintsWhileOpen(t *testing.T) {
	t.Parallel()

	s := babyScreen(t, pet.New(born), &fakeStore{})
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})

	hp, ok := s.(tui.HelpProvider)
	require.True(t, ok)
	bindings := hp.ShortHelp()

	found := map[string]bool{}
	for _, b := range bindings {
		found[b.Help().Key] = true
	}
	assert.True(t, found["m"], "expected the Meal hint while the chooser is open")
	assert.True(t, found["s"], "expected the Snack hint while the chooser is open")
}

func TestMouseClickActivatesFeedChoice(t *testing.T) {
	t.Run("clicking Snack applies it", func(t *testing.T) {
		p := pet.New(born)
		p.Happiness = 1
		s := babyScreen(t, p, &fakeStore{})
		s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})

		s = clickZone(t, s, next.SnackZoneID)

		ns, ok := s.(*next.Screen)
		require.True(t, ok)
		assert.Equal(t, 2, ns.Pet().Happiness)
		assert.Equal(t, pet.BaseWeight+1, ns.Pet().Weight)
	})

	t.Run("clicking Feed opens the chooser", func(t *testing.T) {
		s := babyScreen(t, pet.New(born), &fakeStore{})

		s = clickZone(t, s, next.FeedZoneID)

		assert.Contains(t, stripANSI(s.View()), "Meal")
	})

	t.Run("a click outside both Meal and Snack does nothing", func(t *testing.T) {
		p := pet.New(born)
		p.Happiness = 1
		s := babyScreen(t, p, &fakeStore{})
		s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
		iconZone(t, s, next.MealZoneID) // ensure the chooser's zones are known, so this is a real miss

		s, cmd := s.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: 0, Y: 0})

		assert.Nil(t, cmd)
		ns, ok := s.(*next.Screen)
		require.True(t, ok)
		assert.Equal(t, 1, ns.Pet().Happiness, "a miss should not feed")
		assert.Contains(t, stripANSI(s.View()), "Meal", "the chooser should still be open")
	})
}

// TestViewHeightIsStableAcrossMessAndFlourishStateChanges guards against a
// regression where the Mess indicator and the Care-action flourish were only
// appended to the view when present, so the rest of the centred stack (art,
// meters, icon bar) visibly shifted by a row whenever either appeared or
// cleared — unlike the art, which reserves a fixed height for exactly this
// reason.
func TestViewHeightIsStableAcrossMessAndFlourishStateChanges(t *testing.T) {
	t.Parallel()

	lineCount := func(view string) int { return strings.Count(view, "\n") + 1 }

	clean := babyScreen(t, pet.New(born), &fakeStore{})
	baseline := lineCount(clean.View())

	p := pet.New(born)
	p.LastCleanedAt = born.Add(-pet.MessInterval)
	messy := babyScreen(t, p, &fakeStore{})
	assert.Equal(t, baseline, lineCount(messy.View()), "a visible Mess indicator should not change the view's height")

	flourishing, _ := clean.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	require.Contains(t, stripANSI(flourishing.View()), "plays happily")
	assert.Equal(t, baseline, lineCount(flourishing.View()), "a visible flourish should not change the view's height")
}

// TestIconBarMouseIsSafeBeforeAnyManagerExists proves updateIconBarMouse
// guards against a nil zone.DefaultManager — the state a Screen would see if
// a click ever arrived before bubblezone had been initialised — the same way
// the Welcome Screen's isBeginClick does; see welcome_test.go's
// TestZoneGuardsAreSafeBeforeAnyManagerExists.
//
// Deliberately not t.Parallel(): it mutates the shared zone.DefaultManager, so
// it must run in the serial phase before any parallel test could read it.
func TestIconBarMouseIsSafeBeforeAnyManagerExists(t *testing.T) {
	saved := zone.DefaultManager
	zone.DefaultManager = nil
	defer func() { zone.DefaultManager = saved }()

	p := pet.New(born)
	p.Happiness = 1
	s := babyScreen(t, p, &fakeStore{})

	s, cmd := s.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: 0, Y: 0})

	assert.Nil(t, cmd)
	ns, ok := s.(*next.Screen)
	require.True(t, ok)
	assert.Equal(t, 1, ns.Pet().Happiness, "a click must not activate anything while no zone manager exists to resolve it")
}
