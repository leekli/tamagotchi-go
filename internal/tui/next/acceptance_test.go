package next_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	snack, meal, play, clean, cure func(t *testing.T, s tui.Screen) tui.Screen
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
		cure:  func(_ *testing.T, s tui.Screen) tui.Screen { return typeKeys(s, 'u') },
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
		cure:  func(t *testing.T, s tui.Screen) tui.Screen { return clickZone(t, s, next.CureZoneID) },
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

// TestAcceptance_CareMistakes drives Care mistakes through the Screen's own
// clocks: the Beat advances the Pet and counts a mistake when a Stat's grace
// window expires, a Care action inside the window prevents it, and nothing about
// the tally shows on screen. The Pet is clean, so Hunger (starting at 1) empties
// at 3:00 and its 4-minute window expires at 7:00.
func TestAcceptance_CareMistakes(t *testing.T) {
	t.Parallel()

	starving := func() pet.Pet {
		p := pet.New(born)
		p.LastCleanedAt = born.Add(24 * time.Hour)
		p.Hunger = 1
		return p
	}
	beatAt := func(s tui.Screen, at time.Duration) tui.Screen {
		s, _ = s.Update(pet.BeatMsg{Time: born.Add(at)})
		return s
	}

	t.Run("given a Stat is left Empty when its grace window expires then one Care mistake is counted, and only then", func(t *testing.T) {
		s := advanceAnim(t, sizedScreen(t, starving(), &fakeStore{}), born.Add(6*time.Minute), 1)

		s = beatAt(s, 7*time.Minute-time.Nanosecond)
		assert.Zero(t, currentPet(t, s).CareMistakes, "the window has not yet expired")
		s = beatAt(s, 7*time.Minute)
		assert.Equal(t, 1, currentPet(t, s).CareMistakes)
		s = beatAt(s, 12*time.Minute)
		assert.Equal(t, 1, currentPet(t, s).CareMistakes, "the same spell is not counted again")
	})

	t.Run("given a Stat is Empty when the player refills it inside the window then no mistake is counted for that spell", func(t *testing.T) {
		s := advanceAnim(t, sizedScreen(t, starving(), &fakeStore{}), born.Add(6*time.Minute+30*time.Second), 1)

		s = typeKeys(s, 'f', 'm') // a Meal at 6:30, 3:30 into the spell
		require.Equal(t, 1, currentPet(t, s).Hunger, "sanity check: the Meal refilled Hunger")

		s = beatAt(s, 8*time.Minute) // past the old 7:00 deadline
		assert.Zero(t, currentPet(t, s).CareMistakes)
	})

	t.Run("the tally is not shown to the player while the Pet is alive", func(t *testing.T) {
		clean := babyScreen(t, pet.New(born), &fakeStore{})

		owing := pet.New(born)
		owing.CareMistakes = 3
		owing.HungerEmpty = pet.EmptySpell{Since: born, GraceEndsAt: born.Add(pet.GraceWindow)}
		withTally := babyScreen(t, owing, &fakeStore{})

		assert.Equal(t, visibleText(clean.View()), visibleText(withTally.View()),
			"a Care mistake tally should make no difference to what is drawn")
	})
}

// TestAcceptance_RestartClearsTheCareMistakeTallyAndEmptySpells: Restart
// replaces the Pet outright, so nothing of the previous life's neglect carries
// into the new one.
func TestAcceptance_RestartClearsTheCareMistakeTallyAndEmptySpells(t *testing.T) {
	t.Parallel()

	t.Run("given a Pet that died owing mistakes and an open Empty spell when the player restarts then the new Pet has none", func(t *testing.T) {
		died := pet.New(born)
		died.CareMistakes = 3
		died.HungerEmpty = pet.EmptySpell{Since: born, GraceEndsAt: born.Add(pet.GraceWindow)}
		died.HappinessEmpty = pet.EmptySpell{Since: born}
		died.SickSince = born
		died.LastCuredAt = born
		s := deathScreen(t, died, &fakeStore{})

		s = typeKeys(s, 'x') // not the Restart key: nothing should change
		require.Equal(t, 3, currentPet(t, s).CareMistakes)
		s, _ = s.Update(tea.KeyMsg{Type: tea.KeyEnter})

		restarted := currentPet(t, s)
		assert.Zero(t, restarted.CareMistakes)
		assert.Equal(t, pet.EmptySpell{}, restarted.HungerEmpty)
		assert.Equal(t, pet.EmptySpell{}, restarted.HappinessEmpty)
		assert.True(t, restarted.SickSince.IsZero(), "and no Sickness")
		assert.True(t, restarted.LastCuredAt.IsZero())
	})
}

// sickFromBirth is a Pet whose Mess appeared and then went unattended long
// enough that it is Sick from the moment of birth on.
func sickFromBirth() pet.Pet {
	p := pet.New(born)
	p.LastCleanedAt = born.Add(-(pet.MessInterval + pet.SickAfterMess))
	return p
}

// TestAcceptance_Cure: Cure ends a Sick Pet's Sickness, does nothing but say so
// for a healthy one, and is reached the same way by keyboard and by mouse.
//
// Deliberately not t.Parallel(): the mouse variants share the global bubblezone
// manager, like the package's other mouse tests.
func TestAcceptance_Cure(t *testing.T) {
	hatchedAt := born.Add(pet.EggDuration)

	for name, in := range careInputs {
		t.Run(name, func(t *testing.T) {
			t.Run("given a Sick Pet, when it is Cured, then it is no longer Sick and the Screen says so", func(t *testing.T) {
				s := babyScreen(t, sickFromBirth(), &fakeStore{})
				require.True(t, currentPet(t, s).Sick(hatchedAt), "sanity check: the Pet is Sick")
				require.Contains(t, visibleText(s.View()), "Sick", "sanity check: and says so")

				s = in.cure(t, s)

				assert.False(t, currentPet(t, s).Sick(hatchedAt))
				view := visibleText(s.View())
				assert.Contains(t, view, "feels better")
				assert.NotContains(t, view, "Sick", "the Sick indicator clears with the Sickness")
			})

			t.Run("given a healthy Pet, when Cure is used, then nothing changes but the Screen says it feels fine", func(t *testing.T) {
				s := babyScreen(t, pet.New(born), &fakeStore{})
				before := currentPet(t, s)

				s = in.cure(t, s)

				after := currentPet(t, s)
				assert.Contains(t, visibleText(s.View()), "feels fine")
				assert.True(t, after.SickSince.IsZero())
				assert.True(t, after.LastCuredAt.IsZero(), "a Cure on a healthy Pet must not touch the sickness clock")
				assert.Equal(t, before.Hunger, after.Hunger)
				assert.Equal(t, before.Happiness, after.Happiness)
				assert.Equal(t, before.Weight, after.Weight)
			})

			t.Run("given a Sick Pet, when it is Cleaned, then it is still Sick", func(t *testing.T) {
				s := babyScreen(t, sickFromBirth(), &fakeStore{})

				s = in.clean(t, s)

				assert.True(t, currentPet(t, s).Sick(hatchedAt), "Clean removes the Mess but does not cure")
				assert.Contains(t, visibleText(s.View()), "Sick")
			})
		})
	}
}

// TestTheSickIndicatorFillsItsReservedRowAndNothingMoves: Sickness shows a text
// indicator in the STATS row reserved for it (readable without colour), and
// showing it moves nothing, since the row was already there, blank.
func TestTheSickIndicatorFillsItsReservedRowAndNothingMoves(t *testing.T) {
	t.Parallel()

	healthy := babyScreen(t, pet.New(born), &fakeStore{})
	sick := babyScreen(t, sickFromBirth(), &fakeStore{})

	assert.NotContains(t, visibleText(healthy.View()), "Sick")
	require.Contains(t, visibleText(sick.View()), "Sick")

	healthyTop, healthyLast := occupiedRows(t, healthy.View())
	sickTop, sickLast := occupiedRows(t, sick.View())
	assert.Equal(t, healthyTop, sickTop)
	assert.Equal(t, healthyLast, sickLast)

	left, right := occupiedColumns(sick.View())
	assert.Equal(t, envelopeWidth, right-left, "the indicator fits inside the STATS panel")
}

// TestTheDeathPanelShowsNoSickIndicator: once the Pet has died nothing about its
// Sickness means anything, so the Death panel leaves it out.
func TestTheDeathPanelShowsNoSickIndicator(t *testing.T) {
	t.Parallel()

	view := visibleText(deathScreen(t, sickFromBirth(), &fakeStore{}).View())

	assert.NotContains(t, view, "Sick")
}

// TestTheCureTabIsAlwaysVisibleOnceHatched: the fourth Icon bar tab appears with
// the other three from the moment the Pet hatches, healthy or not, and not
// before.
func TestTheCureTabIsAlwaysVisibleOnceHatched(t *testing.T) {
	t.Parallel()

	assert.NotContains(t, visibleText(sizedScreen(t, pet.New(born), &fakeStore{}).View()), "Cure", "no Icon bar before Hatch")
	assert.Contains(t, visibleText(babyScreen(t, pet.New(born), &fakeStore{}).View()), "Cure", "visible while healthy")
	assert.Contains(t, visibleText(babyScreen(t, sickFromBirth(), &fakeStore{}).View()), "Cure", "and while Sick")
	assert.Contains(t, visibleText(adultScreen(t, caredForUntil(adultAt), &fakeStore{}).View()), "Cure", "at every Stage")
}

// TestTheFourTabBarFillsTheWholeIconRow: four 9-column tabs and three 2-column
// gaps are exactly the Icon bar row's 42 columns.
func TestTheFourTabBarFillsTheWholeIconRow(t *testing.T) {
	t.Parallel()

	view := visibleText(babyScreen(t, pet.New(born), &fakeStore{}).View())
	var tabLine string
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, "Feed") {
			tabLine = line
		}
	}
	require.NotEmpty(t, tabLine)

	left, right := lineSpan(tabLine)

	assert.Equal(t, 42, right-left)
}

// TestTheHelpBarAdvertisesCureAndStillFitsTheMinimumTerminal: the App renders
// the Screen's hints plus its own Quit binding on one row of the 80-column
// minimum terminal.
func TestTheHelpBarAdvertisesCureAndStillFitsTheMinimumTerminal(t *testing.T) {
	t.Parallel()

	screen, ok := babyScreen(t, pet.New(born), &fakeStore{}).(tui.HelpProvider)
	require.True(t, ok)
	bindings := append(screen.ShortHelp(), tui.DefaultKeyMap().Quit)

	bar := help.New().ShortHelpView(bindings)

	assert.Contains(t, bar, "cure")
	assert.LessOrEqual(t, lipgloss.Width(bar), 80, "the help bar must fit one row of the minimum terminal")
}

// TestAcceptance_ACureResumesTheGraceClockFromTheScreensClock: Cure moves each
// pending grace deadline by the time it was paused, and that needs the Pet's
// spells and Sickness already recorded up to the moment of the Cure. Between
// Beats they are not, so the Screen must advance the Pet first (as it does for
// every Care action). Here no Beat runs before the Cure at 8:00: Hunger emptied
// at 3:00 and the Pet fell Sick at 6:00 (see pet's sickness_grace_test.go), so
// its resumed window expires at 9:00. A Cure applied without advancing first
// would leave the spell unrecorded, and the mistake would be counted at 7:00.
func TestAcceptance_ACureResumesTheGraceClockFromTheScreensClock(t *testing.T) {
	t.Parallel()

	t.Run("given a Sick Pet with an Empty Stat when it is Cured before any Beat has run then the mistake is counted a resumed window later", func(t *testing.T) {
		p := pet.New(born)
		p.LastCleanedAt = born.Add(-4 * time.Minute) // Mess at 1:00, Sick at 6:00
		p.Hunger = 1                                 // Empty at 3:00
		s := advanceAnim(t, sizedScreen(t, p, &fakeStore{}), born.Add(8*time.Minute), 1)

		s = typeKeys(s, 'u')
		require.False(t, currentPet(t, s).Sick(born.Add(8*time.Minute)), "sanity check: the Pet was Cured")

		s, _ = s.Update(pet.BeatMsg{Time: born.Add(9*time.Minute - time.Nanosecond)})
		assert.Zero(t, currentPet(t, s).CareMistakes, "just before the resumed window expires")
		s, _ = s.Update(pet.BeatMsg{Time: born.Add(9 * time.Minute)})
		assert.Equal(t, 1, currentPet(t, s).CareMistakes, "three minutes before the Sickness plus one after")
	})
}

// starvedPet is a clean Pet never fed: Hunger is Empty at 12:00 and it starves
// at exactly 22:00.
func starvedPet() pet.Pet {
	p := pet.New(born)
	p.LastCleanedAt = born.Add(24 * time.Hour)
	return p
}

// TestAcceptance_TheDeathPanelNamesTheCause: the Death panel shows why the Pet
// died in the row reserved for it, and it still fits its fixed dimensions.
func TestAcceptance_TheDeathPanelNamesTheCause(t *testing.T) {
	t.Parallel()

	t.Run("Starvation, recorded by a Beat", func(t *testing.T) {
		s := advanceAnim(t, sizedScreen(t, starvedPet(), &fakeStore{}), born.Add(22*time.Minute), 1)
		s, _ = s.Update(pet.BeatMsg{Time: born.Add(22 * time.Minute)})

		view := visibleText(s.View())

		assert.Contains(t, view, "Death")
		assert.Contains(t, view, "Cause: Starvation")
		assert.Contains(t, view, "hatch a new Egg", "the Restart prompt is still there")
		first, last := occupiedRows(t, s.View())
		assert.Equal(t, envelopeHeight, last-first+1, "and the panel is still its fixed height")
	})

	t.Run("Sickness, recorded by a Beat", func(t *testing.T) {
		s := advanceAnim(t, sizedScreen(t, pet.New(born), &fakeStore{}), born.Add(18*time.Minute), 1)
		s, _ = s.Update(pet.BeatMsg{Time: born.Add(18 * time.Minute)})

		view := visibleText(s.View())

		assert.Contains(t, view, "Cause: Sickness")
		assert.NotContains(t, view, "Sick\n", "the Sick indicator is gone with the STATS panel")
		assert.Contains(t, view, "hatch a new Egg")
	})

	t.Run("Neglect, when Care mistakes had shortened the Adult", func(t *testing.T) {
		// Three mistakes: the Adult lasts 14 minutes, so the Pet dies at 54:30. Its
		// record reaches 50 minutes, so the death falls inside the Beat's window.
		p := pet.New(born)
		p.CareMistakes = 3
		last := born.Add(50 * time.Minute)
		p.LastSeenAt, p.HappinessLastSeenAt, p.LastCleanedAt = last, last, last
		diesAt := born.Add(54*time.Minute + 30*time.Second)
		s := advanceAnim(t, sizedScreen(t, p, &fakeStore{}), diesAt, 1)
		s, _ = s.Update(pet.BeatMsg{Time: diesAt})

		view := visibleText(s.View())

		assert.Contains(t, view, "Cause: Neglect")
		assert.Contains(t, view, "Care mistakes: 3")
	})

	t.Run("every Death panel shows the tally, in its reserved row", func(t *testing.T) {
		s := advanceAnim(t, sizedScreen(t, starvedPet(), &fakeStore{}), born.Add(22*time.Minute), 1)
		s, _ = s.Update(pet.BeatMsg{Time: born.Add(22 * time.Minute)})
		p := currentPet(t, s)
		require.NotZero(t, p.CareMistakes, "sanity check: Happiness's neglected window earned a mistake before it starved")

		assert.Contains(t, visibleText(s.View()), fmt.Sprintf("Care mistakes: %d", p.CareMistakes))
		first, last := occupiedRows(t, s.View())
		assert.Equal(t, envelopeHeight, last-first+1, "and the panel is still its fixed height")
	})

	t.Run("Old age, though never recorded (a save from before deaths were)", func(t *testing.T) {
		s := deathScreen(t, pet.New(born), &fakeStore{})

		view := visibleText(s.View())

		assert.Contains(t, view, "Cause: Old age")
		assert.Contains(t, view, "Care mistakes: 0")
	})

	t.Run("no cause is shown while the Pet is alive", func(t *testing.T) {
		assert.NotContains(t, visibleText(babyScreen(t, pet.New(born), &fakeStore{}).View()), "Cause")
	})
}

// TestAcceptance_StarvationArrivesOnTheBeat: the Pet is alive right up to the
// Beat that reaches its 22:00, then dead. The Screen's own clock is at 22:00
// throughout, so what changes is the Pet's record.
func TestAcceptance_StarvationArrivesOnTheBeat(t *testing.T) {
	t.Parallel()

	s := advanceAnim(t, sizedScreen(t, starvedPet(), &fakeStore{}), born.Add(22*time.Minute), 1)

	s, _ = s.Update(pet.BeatMsg{Time: born.Add(22*time.Minute - time.Nanosecond)})
	assert.Contains(t, visibleText(s.View()), "Hunger", "alive a nanosecond before")
	assert.True(t, currentPet(t, s).DiedAt.IsZero())

	s, _ = s.Update(pet.BeatMsg{Time: born.Add(22 * time.Minute)})
	assert.NotContains(t, visibleText(s.View()), "Hunger", "the Meters give way to the Death panel")
	assert.Equal(t, pet.Starvation, currentPet(t, s).Cause)
}

// TestAcceptance_ACareActionAfterTheTrueMomentOfDeathFindsThePetDead: the Screen
// advances the Pet before a Care action, so a Meal pressed after the Pet has
// truly starved, but before the next Beat, finds it already dead and does
// nothing: no mercy window.
func TestAcceptance_ACareActionAfterTheTrueMomentOfDeathFindsThePetDead(t *testing.T) {
	t.Parallel()

	s := advanceAnim(t, sizedScreen(t, starvedPet(), &fakeStore{}), born.Add(22*time.Minute+5*time.Second), 1)
	require.True(t, currentPet(t, s).DiedAt.IsZero(), "sanity check: no Beat has recorded the death yet")

	s = typeKeys(s, 'f', 'm')

	p := currentPet(t, s)
	assert.Zero(t, p.Hunger, "the Meal was not given")
	assert.True(t, born.Add(22*time.Minute).Equal(p.DiedAt), "the death is recorded at its true moment")
	assert.Equal(t, pet.Starvation, p.Cause)
	assert.Contains(t, visibleText(s.View()), "Cause: Starvation")
}

// TestAcceptance_EveryCareActionFindsAPetThatHasTrulyDiedAlreadyDead extends the
// Meal case above to the other Care actions: Play, Clean and Cure, pressed after
// the Pet has starved but before any Beat has recorded it, do nothing either.
func TestAcceptance_EveryCareActionFindsAPetThatHasTrulyDiedAlreadyDead(t *testing.T) {
	t.Parallel()

	for name, key := range map[string]rune{"Play": 'p', "Clean": 'c', "Cure": 'u'} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			before := starvedPet()
			s := advanceAnim(t, sizedScreen(t, before, &fakeStore{}), born.Add(22*time.Minute+5*time.Second), 1)

			s = typeKeys(s, key)

			p := currentPet(t, s)
			assert.Equal(t, pet.Starvation, p.Cause, "the Pet was found dead")
			assert.Zero(t, p.Happiness, "Play was not given")
			assert.True(t, before.LastCleanedAt.Equal(p.LastCleanedAt), "nor was it Cleaned")
			assert.True(t, p.LastCuredAt.IsZero(), "nor Cured")
		})
	}
}

// TestAcceptance_DeathMidInteractionResetsTheInteractionState: if the Pet dies
// while a Care action's state is live (here Play is selected, with its flourish
// showing), that state is cleared, so nothing of it outlives the Pet. The
// selection is what can be observed from outside: the Death panel draws neither
// the Icon bar, the Feed chooser nor a flourish, but one function resets all
// of them together.
func TestAcceptance_DeathMidInteractionResetsTheInteractionState(t *testing.T) {
	t.Parallel()

	s := advanceAnim(t, sizedScreen(t, starvedPet(), &fakeStore{}), born.Add(21*time.Minute), 1)
	s = typeKeys(s, 'p') // selects Play, with its flourish showing
	ns, ok := s.(*next.Screen)
	require.True(t, ok)
	require.Equal(t, 1, ns.Selected(), "sanity check: Play is selected")

	s = advanceAnim(t, s, born.Add(22*time.Minute), 1)
	s, _ = s.Update(pet.BeatMsg{Time: born.Add(22 * time.Minute)})

	ns, ok = s.(*next.Screen)
	require.True(t, ok)
	require.Equal(t, pet.Starvation, ns.Pet().Cause, "sanity check: the Pet has died")
	assert.Zero(t, ns.Selected(), "the selection was reset")
}

// TestAcceptance_RestartAfterEveryCause: Restart works after every cause of
// Death, by keyboard and by mouse. It saves at once, so quitting straight after
// keeps the Egg, and it produces a completely fresh Egg with every field, the
// death record included, back to zero.
//
// Deliberately not t.Parallel(): the mouse variants share the global bubblezone
// manager, like the package's other mouse tests.
func TestAcceptance_RestartAfterEveryCause(t *testing.T) {
	restartedAt := born.Add(2 * time.Hour)

	dead := map[string]pet.Pet{
		"Old age, never recorded": pet.New(born),
		"Old age, recorded": func() pet.Pet {
			p := pet.New(born)
			p.DiedAt, p.Cause = born.Add(60*time.Minute+30*time.Second), pet.OldAge
			return p
		}(),
		"Starvation": func() pet.Pet {
			p := starvedPet().Advance(born.Add(30 * time.Minute))
			require.Equal(t, pet.Starvation, p.Cause, "sanity check")
			return p
		}(),
		"Sickness": func() pet.Pet {
			p := pet.New(born).Advance(born.Add(30 * time.Minute)) // never cleaned: dies of Sickness at 18:00
			require.Equal(t, pet.Sickness, p.Cause, "sanity check")
			return p
		}(),
		"Neglect": func() pet.Pet {
			p := pet.New(born)
			p.CareMistakes = 3
			p.DiedAt, p.Cause = born.Add(54*time.Minute+30*time.Second), pet.Neglect
			return p
		}(),
	}

	for cause, initial := range dead {
		for _, input := range []string{"keyboard", "mouse"} {
			t.Run(cause+" by "+input, func(t *testing.T) {
				store := &fakeStore{}
				s := advanceAnim(t, sizedScreen(t, initial, store), restartedAt, 1)
				require.Contains(t, visibleText(s.View()), "Cause:", "sanity check: the Death panel is showing")

				var cmd tea.Cmd
				if input == "mouse" {
					z := iconZone(t, s, next.RestartZoneID)
					s, cmd = s.Update(tea.MouseMsg{
						Action: tea.MouseActionPress, Button: tea.MouseButtonLeft,
						X: (z.StartX + z.EndX) / 2, Y: z.StartY,
					})
				} else {
					s, cmd = s.Update(tea.KeyMsg{Type: tea.KeyEnter})
				}

				assert.Equal(t, pet.New(restartedAt), currentPet(t, s),
					"a completely fresh Egg, with no trace of the death, the tally or the previous life")
				require.NotNil(t, cmd, "Restart returns a save command")
				cmd()
				require.NotEmpty(t, store.saved, "and it saved at once, so quitting straight after keeps the Egg")
				assert.Equal(t, pet.New(restartedAt), store.saved[len(store.saved)-1])
			})
		}
	}
}

// TestAcceptance_ADeadPetIsDeadOnTheVeryFirstFrame: a Pet that died long before
// the game was launched has its clock frozen at the moment of death. The Screen
// draws its first frame before any animation tick, from a clock seeded from the
// Pet's record, so that frame must already be the Death panel, and not the living
// Pet it would be if the clock were seeded from Hunger's whole-step anchor, which
// for a Starvation is a step short of the death.
func TestAcceptance_ADeadPetIsDeadOnTheVeryFirstFrame(t *testing.T) {
	t.Parallel()

	died := starvedPet().Advance(born.Add(3 * time.Hour))
	require.Equal(t, pet.Starvation, died.Cause, "sanity check")
	require.True(t, died.LastSeenAt.Before(died.DiedAt), "sanity check: Hunger's anchor is short of the death")

	s := sizedScreen(t, died, &fakeStore{}) // no animation tick has run

	view := visibleText(s.View())
	assert.Contains(t, view, "Cause: Starvation")
	assert.NotContains(t, view, "Hunger", "not even one frame of a living Pet's Meters")
}
