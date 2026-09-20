package pet_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/leekli/tamagotchi-go/internal/pet"
)

// These tests pin how Sickness pauses the grace clock: while a Pet is Sick an
// Empty spell's grace window does not run and no Care mistake is counted, and a
// Cure resumes it where it left off. The Pet below is arranged so the timeline
// is worked by hand:
//
//	1:00  Mess appears (cleaned 4 minutes before birth, so 1 minute after it)
//	3:00  Hunger empties (it starts at 1), so its window would expire at 7:00
//	6:00  the Pet falls Sick (5 minutes after the Mess), 3 minutes into that spell
//	6:30  Happiness empties, while the Pet is Sick (the Mess doubles its Decay rate)
//	8:00  the Pet is Cured, with the Mess still there (Sick again at 13:00)
//
// Hunger's clock ran 3 minutes, paused for 2, and has 1 minute left: it expires
// at 9:00. Happiness's spell began during the Sickness, so its clock only starts
// at the Cure: it expires at 12:00.
func pausePetBornAt(born time.Time) pet.Pet {
	p := pet.New(born)
	p.LastCleanedAt = born.Add(-4 * time.Minute)
	p.Hunger = 1
	return p
}

func TestSicknessPausesTheGraceClockAndACureResumesIt(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	at := func(minutes, seconds int) time.Time {
		return born.Add(time.Duration(minutes)*time.Minute + time.Duration(seconds)*time.Second)
	}

	// The same Pet advanced to the Cure in one long catch-up, and in 20-second
	// Beats, must behave identically.
	oneLong := pausePetBornAt(born).Advance(at(8, 0)).Cure(at(8, 0))

	walked := pausePetBornAt(born)
	for now := born.Add(pet.BeatInterval); !now.After(at(8, 0)); now = now.Add(pet.BeatInterval) {
		walked = walked.Advance(now)
	}
	walked = walked.Advance(at(8, 0)).Cure(at(8, 0))

	for name, cured := range map[string]pet.Pet{"one long catch-up": oneLong, "20-second Beats": walked} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			require.Zero(t, cured.CareMistakes, "sanity check: nothing has been counted by the Cure, though 7:00 has passed")

			assert.Zero(t, cured.Advance(at(9, 0).Add(-time.Nanosecond)).CareMistakes, "Hunger: just before its resumed window expires")
			assert.Equal(t, 1, cured.Advance(at(9, 0)).CareMistakes, "Hunger: three minutes before Sickness plus one after, at 9:00")

			assert.Equal(t, 1, cured.Advance(at(12, 0).Add(-time.Nanosecond)).CareMistakes, "Happiness: not yet")
			assert.Equal(t, 2, cured.Advance(at(12, 0)).CareMistakes,
				"Happiness began while Sick, so its window starts at the 8:00 Cure and expires at 12:00")
		})
	}
}

func TestNoMistakesAreCountedWhileSickHoweverLongItLasts(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p := pausePetBornAt(born)

	assert.Zero(t, p.Advance(born.Add(7*time.Minute+30*time.Second)).CareMistakes,
		"Hunger's 7:00 window would have expired, but the Pet is Sick")
	assert.Zero(t, p.Advance(born.Add(3*time.Hour)).CareMistakes,
		"and however long the Pet stays Sick, none of the spells running through it cost a mistake")
}

func TestTheAttentionCallIsSuppressedWhileSickAndReturnsAfterACure(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	at := func(minutes int) time.Time { return born.Add(time.Duration(minutes) * time.Minute) }
	p := pausePetBornAt(born)

	sick := p.Advance(at(7))
	require.True(t, sick.Sick(at(7)), "sanity check")
	require.Zero(t, sick.Hunger, "sanity check: Hunger is Empty")
	assert.Equal(t, pet.AttentionNone, sick.HungerAttention(at(7)), "a Sick Pet does not call for attention")
	assert.False(t, sick.NeedsAttention(at(7)))

	cured := p.Advance(at(8)).Cure(at(8))
	assert.Equal(t, pet.AttentionWindowRunning, cured.HungerAttention(at(8)), "the call returns after a Cure, with time left on the window")
	assert.Equal(t, pet.AttentionWindowRunning, cured.HappinessAttention(at(8)), "and Happiness's window has just begun")
	assert.Equal(t, pet.AttentionLapsed, cured.Advance(at(9)).HungerAttention(at(9)), "Hunger's window expires at 9:00")
	assert.Equal(t, pet.AttentionNone, cured.Advance(at(13)).HungerAttention(at(13)), "and it is suppressed again once Sick again at 13:00")
}

func TestASpellThatBeginsWhileSickKeepsItsRealStart(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cured := pausePetBornAt(born).Advance(born.Add(8 * time.Minute)).Cure(born.Add(8 * time.Minute))

	// Only the grace deadline moves. When each spell began is untouched, since
	// later features (starvation) measure time Empty from it and must not pause.
	assert.True(t, born.Add(3*time.Minute).Equal(cured.HungerEmpty.Since), "Hunger has been Empty since 3:00")
	assert.True(t, born.Add(6*time.Minute+30*time.Second).Equal(cured.HappinessEmpty.Since),
		"Happiness has been Empty since 6:30, though its grace clock only started at the Cure")
}

func TestAWindowThatExpiredBeforeTheSicknessStillCounts(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("expired a minute before falling Sick", func(t *testing.T) {
		t.Parallel()
		p := pet.New(born)
		p.LastCleanedAt = born.Add(-2 * time.Minute) // Sick at 8:00
		p.Hunger = 1                                 // window expires at 7:00

		assert.Zero(t, p.Advance(born.Add(7*time.Minute-time.Nanosecond)).CareMistakes)
		assert.Equal(t, 1, p.Advance(born.Add(7*time.Minute)).CareMistakes,
			"the window expired while the Pet was still well, so it is not paused")
	})

	t.Run("expiring at the very instant it falls Sick", func(t *testing.T) {
		t.Parallel()
		p := pet.New(born)
		p.LastCleanedAt = born.Add(-3 * time.Minute) // Sick at 7:00
		p.Hunger = 1                                 // window expires at 7:00

		assert.Equal(t, 1, p.Advance(born.Add(7*time.Minute)).CareMistakes,
			"a window that completes as the Pet falls Sick has already run its course")
	})
}

func TestRefillingWhileSickEndsTheSpellSoItsResumedWindowNeverCounts(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p := pausePetBornAt(born).Advance(born.Add(7 * time.Minute)).Feed(pet.Meal) // Hunger refilled while Sick
	require.Equal(t, 1, p.Hunger)

	cured := p.Advance(born.Add(8 * time.Minute)).Cure(born.Add(8 * time.Minute))

	assert.Zero(t, cured.Advance(born.Add(10*time.Minute)).CareMistakes,
		"Hunger's old spell ended at the Meal, so nothing resumes at 9:00")
}
