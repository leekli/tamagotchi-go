package pet_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/leekli/tamagotchi-go/internal/pet"
)

// These tests pin death from untreated Sickness. A Pet never cleaned gets a Mess
// at 5:00 and falls Sick at 10:00; left Sick for 8 minutes it dies at 18:00, four
// minutes before Hunger, Empty from 12:00, would have starved it at 22:00.

func TestSickForTheDeathIntervalWithoutACureKillsAtThatExactInstant(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p := pet.New(born) // never cleaned: Sick at 10:00
	diesAt := born.Add(18 * time.Minute)
	require.Equal(t, 8*time.Minute, pet.SickDeathInterval, "the worked timings assume an 8-minute interval")

	t.Run("alive a nanosecond before, dead at the instant", func(t *testing.T) {
		t.Parallel()
		before := p.Advance(diesAt.Add(-time.Nanosecond))
		assert.True(t, before.DiedAt.IsZero())
		assert.True(t, before.Sick(diesAt.Add(-time.Nanosecond)))

		died := p.Advance(diesAt)
		assert.True(t, diesAt.Equal(died.DiedAt), "died at %v", died.DiedAt.Sub(born))
		assert.Equal(t, pet.Sickness, died.Cause)
		assert.Equal(t, pet.StageDeath, died.Stage(diesAt))
		assert.True(t, born.Add(10*time.Minute).Equal(died.SickSince), "it had been Sick since 10:00")
	})

	t.Run("however the time is advanced the Pet died at the same instant", func(t *testing.T) {
		t.Parallel()
		oneLong := p.Advance(born.Add(3 * time.Hour))

		walked := p
		for now := born.Add(pet.BeatInterval); !now.After(born.Add(3 * time.Hour)); now = now.Add(pet.BeatInterval) {
			walked = walked.Advance(now)
		}

		assert.True(t, diesAt.Equal(oneLong.DiedAt))
		assert.True(t, diesAt.Equal(walked.DiedAt))
		assert.Equal(t, oneLong, walked)
	})
}

func TestAFullyIgnoredPetDiesOfSicknessAheadOfStarvation(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	died := pet.New(born).Advance(born.Add(2 * time.Hour))

	assert.True(t, born.Add(18*time.Minute).Equal(died.DiedAt), "at 18:00, though Starvation would have come at 22:00")
	assert.Equal(t, pet.Sickness, died.Cause)
}

func TestAPetCleanedButNeverFedStarvesInsteadAndIsNeverSick(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p := pet.New(born)
	for at := 4 * time.Minute; at <= 24*time.Minute; at += 4 * time.Minute {
		p = p.Advance(born.Add(at)).Clean(born.Add(at)) // a Clean every 4 minutes: the Mess never has time to make it Sick
	}

	died := p.Advance(born.Add(time.Hour))

	assert.True(t, born.Add(22*time.Minute).Equal(died.DiedAt))
	assert.Equal(t, pet.Starvation, died.Cause)
	assert.True(t, died.SickSince.IsZero(), "the Pet never fell Sick")
}

func TestACureInTimePreventsTheDeathAndTheClockRestarts(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	feed := func(p pet.Pet, at time.Duration) pet.Pet {
		return p.Advance(born.Add(at)).Feed(pet.Meal).Feed(pet.Meal)
	}

	// Kept fed throughout, so only Sickness can kill it. Sick from 10:00, it would
	// die at 18:00; a Cure at 17:59 stops that. The Mess is still there, so it is
	// Sick again 5 minutes later, at 22:59, and dies 8 minutes after that: 30:59.
	p := pet.New(born)
	for _, at := range []time.Duration{5 * time.Minute, 10 * time.Minute, 15 * time.Minute} {
		p = feed(p, at)
	}
	p = p.Advance(born.Add(17*time.Minute + 59*time.Second)).Cure(born.Add(17*time.Minute + 59*time.Second))
	require.True(t, p.DiedAt.IsZero())

	assert.True(t, p.Advance(born.Add(18*time.Minute)).DiedAt.IsZero(), "the original 18:00 passes with the Pet Cured")

	for _, at := range []time.Duration{20 * time.Minute, 25 * time.Minute, 30 * time.Minute} {
		p = feed(p, at)
	}
	diesAt := born.Add(30*time.Minute + 59*time.Second)
	assert.True(t, p.Advance(diesAt.Add(-time.Nanosecond)).DiedAt.IsZero(), "still alive a nanosecond before")
	died := p.Advance(diesAt.Add(time.Second))

	assert.True(t, diesAt.Equal(died.DiedAt), "the restarted clock: Sick again at 22:59, dead at 30:59")
	assert.Equal(t, pet.Sickness, died.Cause)
}

// TestStarvationIsNotPausedByASickness replaces the earlier version of this
// test, which used a Pet never Cured. Such a Pet now dies of Sickness at 18:00,
// before starvation could show. The claim is the same: time Empty is not paused
// by Sickness. Here the Pet is Sick from 10:00 to 17:59 (Cured just in time), and
// Hunger has been Empty since 12:00, so if Sickness paused the starvation clock it
// would live well past 22:00.
func TestStarvationIsNotPausedByASickness(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cureAt := born.Add(17*time.Minute + 59*time.Second)
	p := pet.New(born).Advance(cureAt)
	require.True(t, p.Sick(cureAt), "sanity check: Sick, with Hunger Empty since 12:00")
	p = p.Cure(cureAt)

	died := p.Advance(born.Add(2 * time.Hour))

	assert.True(t, born.Add(22*time.Minute).Equal(died.DiedAt), "starved 10 minutes after Hunger emptied, all the same")
	assert.Equal(t, pet.Starvation, died.Cause)
}

// TestCausesFallingAtTheSameInstantGoInTheSpecsOrder: Starvation, Sickness,
// Neglect, Old age. Each pair of neighbours is checked with both causes due at
// the very same instant.
func TestCausesFallingAtTheSameInstantGoInTheSpecsOrder(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("Starvation before Sickness", func(t *testing.T) {
		t.Parallel()
		at := born.Add(30 * time.Minute)
		p := pet.New(born)
		p.Hunger = 0
		p.HungerEmpty = pet.EmptySpell{Since: at.Add(-pet.StarvationInterval)}
		p.SickSince = at.Add(-pet.SickDeathInterval)
		p.LastCleanedAt = p.SickSince.Add(-pet.MessInterval - pet.SickAfterMess) // consistent with being Sick then
		p.LastSeenAt, p.HappinessLastSeenAt = at.Add(-5*time.Minute), at.Add(-5*time.Minute)

		died := p.Advance(at)

		assert.True(t, at.Equal(died.DiedAt))
		assert.Equal(t, pet.Starvation, died.Cause)
	})

	t.Run("Sickness before Old age", func(t *testing.T) {
		t.Parallel()
		end := born.Add(oldAge)
		p := pet.New(born)
		p.SickSince = end.Add(-pet.SickDeathInterval)
		p.LastCleanedAt = p.SickSince.Add(-pet.MessInterval - pet.SickAfterMess)
		p.LastSeenAt, p.HappinessLastSeenAt = end.Add(-2*time.Minute), end.Add(-2*time.Minute)

		died := p.Advance(end)

		assert.True(t, end.Equal(died.DiedAt))
		assert.Equal(t, pet.Sickness, died.Cause)
	})
}

func TestTheEarliestCauseWinsWhateverItIs(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("Starvation first when the Pet falls Sick late", func(t *testing.T) {
		t.Parallel()
		p := pet.New(born)
		p.LastCleanedAt = born.Add(15 * time.Minute) // cleaned at 15:00: Sick at 25:00, dead by it at 33:00

		died := p.Advance(born.Add(2 * time.Hour))

		assert.True(t, born.Add(22*time.Minute).Equal(died.DiedAt))
		assert.Equal(t, pet.Starvation, died.Cause)
	})

	t.Run("Sickness first when it comes before old age", func(t *testing.T) {
		t.Parallel()
		p := pet.New(born) // kept fed, never cleaned
		for at := 5 * time.Minute; at <= 15*time.Minute; at += 5 * time.Minute {
			p = p.Advance(born.Add(at)).Feed(pet.Meal).Feed(pet.Meal)
		}

		died := p.Advance(born.Add(2 * time.Hour))

		assert.True(t, born.Add(18*time.Minute).Equal(died.DiedAt))
		assert.Equal(t, pet.Sickness, died.Cause)
	})
}

func TestSicknessHasAReadableNameThatSurvivesASaveAndLoad(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Sickness", pet.Sickness.String())
}
