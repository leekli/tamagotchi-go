package next_test

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/leekli/tamagotchi-go/internal/pet"
	"github.com/leekli/tamagotchi-go/internal/tui"
	"github.com/leekli/tamagotchi-go/internal/tui/next"
)

// These tests map one given/when/then group to each user-facing requirement
// of the Play and Clean Care actions, mirroring
// internal/tui/welcome/acceptance_test.go's style.

func TestAcceptance_Feed(t *testing.T) {
	t.Parallel()

	t.Run("given the Feed icon is selected, when Meal is chosen, then Hunger increases and Weight does not", func(t *testing.T) {
		p := pet.New(born)
		p.Hunger = 1
		s := babyScreen(t, p, &fakeStore{})

		s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
		s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})

		ns, ok := s.(*next.Screen)
		require.True(t, ok)
		assert.Equal(t, 2, ns.Pet().Hunger)
		assert.Equal(t, pet.BaseWeight, ns.Pet().Weight)
	})

	t.Run("given the Feed icon is selected, when Snack is chosen, then Happiness and Weight both increase", func(t *testing.T) {
		p := pet.New(born)
		p.Happiness = 1
		s := babyScreen(t, p, &fakeStore{})

		s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
		s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})

		ns, ok := s.(*next.Screen)
		require.True(t, ok)
		assert.Equal(t, 2, ns.Pet().Happiness)
		assert.Equal(t, pet.BaseWeight+1, ns.Pet().Weight)
	})
}

func TestAcceptance_Play(t *testing.T) {
	t.Parallel()

	t.Run("given the Play icon is activated then Happiness increases", func(t *testing.T) {
		p := pet.New(born)
		p.Happiness = 1
		s := babyScreen(t, p, &fakeStore{})

		s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})

		ns, ok := s.(*next.Screen)
		require.True(t, ok)
		assert.Equal(t, 2, ns.Pet().Happiness)
	})
}

func TestAcceptance_Clean(t *testing.T) {
	t.Parallel()

	t.Run("given the Pet has a Mess, when Clean is activated, then the Mess indicator disappears and Happiness decay returns to its normal rate", func(t *testing.T) {
		p := pet.New(born)
		p.LastCleanedAt = born.Add(-pet.MessInterval)
		s := babyScreen(t, p, &fakeStore{})
		require.Contains(t, stripANSI(s.View()), "(___)", "should start with the Mess indicator visible")

		s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})

		assert.NotContains(t, stripANSI(s.View()), "(___)")

		ns, ok := s.(*next.Screen)
		require.True(t, ok)
		assert.False(t, ns.Pet().HasMess(born.Add(pet.EggDuration)))
	})
}

func TestAcceptance_UncleanedMessAcceleratesHappinessDecay(t *testing.T) {
	t.Parallel()

	t.Run("given a Mess is present and uncleaned, when time passes, then Happiness falls faster than it would without one", func(t *testing.T) {
		elapsed := pet.HappinessDecayInterval + pet.AcceleratedHappinessDecayInterval

		clean := pet.New(born).Advance(born.Add(elapsed))

		messyStart := pet.New(born)
		messyStart.LastCleanedAt = born.Add(-pet.MessInterval)
		messy := messyStart.Advance(born.Add(elapsed))

		assert.Less(t, messy.Happiness, clean.Happiness)
	})
}

// TestAcceptance_OverfedRecoveryLoop proves the whole Weight feedback loop
// end to end: Snack can push a Pet into Overfed, Overfed has the same
// accelerated-decay consequence as an uncleaned Mess, and Play is the way
// back out of it.
func TestAcceptance_OverfedRecoveryLoop(t *testing.T) {
	t.Parallel()

	t.Run("given a Pet becomes Overfed via repeated Snacks, when time passes, then Happiness decays faster; and once Play brings Weight back under the threshold, decay returns to normal", func(t *testing.T) {
		elapsed := pet.HappinessDecayInterval + pet.AcceleratedHappinessDecayInterval

		notOverfed := pet.New(born).Advance(born.Add(elapsed))

		overfedStart := pet.New(born)
		for overfedStart.Weight < pet.OverfedThreshold {
			overfedStart = overfedStart.Feed(pet.Snack)
		}
		require.True(t, overfedStart.Overfed(), "should be Overfed after enough Snacks")

		overfed := overfedStart.Advance(born.Add(elapsed))
		assert.Less(t, overfed.Happiness, notOverfed.Happiness,
			"an Overfed Pet should lose more Happiness over the same elapsed time")

		recovered := overfedStart
		for recovered.Overfed() {
			recovered = recovered.Play()
		}
		require.False(t, recovered.Overfed(), "enough Play should resolve Overfed")

		recoveredAdvanced := recovered.Advance(born.Add(elapsed))
		assert.Equal(t, notOverfed.Happiness, recoveredAdvanced.Happiness,
			"once Overfed is resolved, Happiness should decay at the normal rate again")
	})
}

// careInput is one way a player triggers each Care action: the same actions by
// keyboard and by mouse must reach the same state.
type careInput struct {
	snack, meal, play, clean func(t *testing.T, s tui.Screen) tui.Screen
}

func typeKeys(s tui.Screen, runes ...rune) tui.Screen {
	for _, r := range runes {
		s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	return s
}

func currentPet(t *testing.T, s tui.Screen) pet.Pet {
	t.Helper()
	ns, ok := s.(*next.Screen)
	require.True(t, ok)
	return ns.Pet()
}

var careInputs = map[string]careInput{
	"keyboard": {
		snack: func(_ *testing.T, s tui.Screen) tui.Screen { return typeKeys(s, 'f', 's') },
		meal:  func(_ *testing.T, s tui.Screen) tui.Screen { return typeKeys(s, 'f', 'm') },
		play:  func(_ *testing.T, s tui.Screen) tui.Screen { return typeKeys(s, 'p') },
		clean: func(_ *testing.T, s tui.Screen) tui.Screen { return typeKeys(s, 'c') },
	},
	"mouse": {
		snack: func(t *testing.T, s tui.Screen) tui.Screen {
			return clickZone(t, clickZone(t, s, next.FeedZoneID), next.SnackZoneID)
		},
		meal: func(t *testing.T, s tui.Screen) tui.Screen {
			return clickZone(t, clickZone(t, s, next.FeedZoneID), next.MealZoneID)
		},
		play:  func(t *testing.T, s tui.Screen) tui.Screen { return clickZone(t, s, next.PlayZoneID) },
		clean: func(t *testing.T, s tui.Screen) tui.Screen { return clickZone(t, s, next.CleanZoneID) },
	},
}

// TestAcceptance_CareActionsApplyToAnUpToDatePet: the Screen only advances the
// Pet on its 20-second Beat, so between Beats the Pet lags the Screen's clock.
// A Care action must first bring the Pet up to the Screen's time, so it lands
// on a current Pet and any change it makes to a Decay rate applies from that
// moment, not back over the time since the last Beat. Each case runs no Beat
// before the action, then feeds Beats at exact instants either side of the
// point the corrected timing predicts.
//
// Deliberately not parallel: the mouse variants share the global bubblezone
// manager, like the package's other mouse tests.
func TestAcceptance_CareActionsApplyToAnUpToDatePet(t *testing.T) {
	beatAt := func(s tui.Screen, at time.Duration) tui.Screen {
		s, _ = s.Update(pet.BeatMsg{Time: born.Add(at)})
		return s
	}

	for name, in := range careInputs {
		t.Run(name, func(t *testing.T) {
			t.Run("given a Pet a Snack short of Overfed, when a Snack is given at 1:30, then Happiness decays at the faster rate only from then on", func(t *testing.T) {
				p := pet.New(born)
				p.Weight = pet.OverfedThreshold - 1
				s := advanceAnim(t, sizedScreen(t, p, &fakeStore{}), born.Add(90*time.Second), 1)

				s = in.snack(t, s)
				require.True(t, currentPet(t, s).Overfed(), "the Snack should have made the Pet Overfed")

				// Half a point was made at the normal rate by 1:30; the other half
				// takes 45s at the doubled rate, so the point lands at exactly 2:15.
				s = beatAt(s, 2*time.Minute+15*time.Second-time.Nanosecond)
				assert.Equal(t, pet.MaxStat, currentPet(t, s).Happiness)
				s = beatAt(s, 2*time.Minute+15*time.Second)
				assert.Equal(t, pet.MaxStat-1, currentPet(t, s).Happiness)
			})

			t.Run("given a Pet Overfed from birth, when Play is used at 0:45, then Happiness decays at the normal rate only from then on", func(t *testing.T) {
				p := pet.New(born)
				p.Weight = pet.OverfedThreshold
				s := advanceAnim(t, sizedScreen(t, p, &fakeStore{}), born.Add(45*time.Second), 1)

				s = in.play(t, s)
				require.False(t, currentPet(t, s).Overfed(), "Play should have resolved the Overfed state")

				// Half a point was made at the doubled rate by 0:45; the other half
				// takes 90s at the normal rate, so the point lands at exactly 2:15.
				s = beatAt(s, 2*time.Minute+15*time.Second-time.Nanosecond)
				assert.Equal(t, pet.MaxStat, currentPet(t, s).Happiness)
				s = beatAt(s, 2*time.Minute+15*time.Second)
				assert.Equal(t, pet.MaxStat-1, currentPet(t, s).Happiness)
			})

			t.Run("given a messy Pet, when it is Cleaned at 1:00, then Happiness decays at the normal rate only from then on", func(t *testing.T) {
				p := pet.New(born)
				p.LastCleanedAt = born.Add(-pet.MessInterval) // messy from the start
				s := advanceAnim(t, sizedScreen(t, p, &fakeStore{}), born.Add(time.Minute), 1)

				s = in.clean(t, s)

				// Two thirds of a point were made at the doubled rate by 1:00; the
				// last third takes 60s at the normal rate: the point lands at 2:00.
				s = beatAt(s, 2*time.Minute-time.Nanosecond)
				assert.Equal(t, pet.MaxStat, currentPet(t, s).Happiness)
				s = beatAt(s, 2*time.Minute)
				assert.Equal(t, pet.MaxStat-1, currentPet(t, s).Happiness)
			})

			t.Run("given a Decay step has come due but no Beat has run, when a Meal is given, then it behaves as if the step had already happened", func(t *testing.T) {
				p := pet.New(born)
				p.Hunger = 2
				s := advanceAnim(t, sizedScreen(t, p, &fakeStore{}), born.Add(pet.HungerDecayInterval+10*time.Second), 1)

				s = in.meal(t, s)

				assert.Equal(t, 2, currentPet(t, s).Hunger, "one point lost to the due step, one point gained from the Meal")
			})
		})
	}
}

// TestAcceptance_ADeadPetIgnoresCareActionsBeforeTheNextBeat: a Care action
// pressed after the Pet's true moment of death, but before any Beat has run,
// finds it already dead. There is no Icon bar to click at Death, so this is
// keyboard only.
func TestAcceptance_ADeadPetIgnoresCareActionsBeforeTheNextBeat(t *testing.T) {
	t.Parallel()

	t.Run("given the Pet has passed its death age but no Beat has run, when Care keys are pressed, then nothing changes", func(t *testing.T) {
		initial := pet.New(born)
		initial.Hunger = 1
		initial.Happiness = 1
		s := deathScreen(t, initial, &fakeStore{})

		s = typeKeys(s, 'f', 'm', 'f', 's', 'p', 'c')

		assert.Equal(t, initial, currentPet(t, s),
			"a dead Pet should be neither fed, played with, cleaned, nor advanced")
	})
}

// TestCareActionsNeitherSaveNorRescheduleTheBeat pins that the Beat still owns
// the periodic save and its own rescheduling: advancing the Pet inside a Care
// action must not add a command of its own.
func TestCareActionsNeitherSaveNorRescheduleTheBeat(t *testing.T) {
	t.Parallel()

	for name, key := range map[string]rune{"Play": 'p', "Clean": 'c'} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			s := babyScreen(t, pet.New(born), &fakeStore{})

			_, cmd := s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{key}})

			assert.Nil(t, cmd)
		})
	}

	t.Run("Feed", func(t *testing.T) {
		t.Parallel()

		s := typeKeys(babyScreen(t, pet.New(born), &fakeStore{}), 'f')

		_, cmd := s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})

		assert.Nil(t, cmd)
	})
}
