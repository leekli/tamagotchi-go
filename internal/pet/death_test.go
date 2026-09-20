package pet_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/leekli/tamagotchi-go/internal/pet"
)

// These tests pin recorded Death. Timings are worked by hand: a Pet kept clean
// decays Hunger one point per 3 minutes, so it is Empty at 12:00 and, with a
// 10-minute starvation interval, dies of Starvation at 22:00. Old age comes at
// 60:30 (30s + 10 + 15 + 15 + 20 minutes). Every death records its moment and
// cause, after which the Pet no longer changes.

const oldAge = 60*time.Minute + 30*time.Second

// keptFedUntil advances p in 5-minute steps to at, giving it two Meals at each
// step, which is enough to keep Hunger up: a Pet that will not starve.
func keptFedUntil(p pet.Pet, born, at time.Time) pet.Pet {
	for now := born.Add(5 * time.Minute); !now.After(at); now = now.Add(5 * time.Minute) {
		p = p.Advance(now).Feed(pet.Meal).Feed(pet.Meal)
	}
	return p
}

func TestHungerEmptyForTheStarvationIntervalKillsAtThatExactInstant(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p := cleanPetBornAt(born) // Hunger Empty at 12:00
	diesAt := born.Add(22 * time.Minute)
	require.Equal(t, 10*time.Minute, pet.StarvationInterval, "the worked timings assume a 10-minute starvation interval")

	t.Run("alive a nanosecond before, dead at the instant", func(t *testing.T) {
		t.Parallel()
		before := p.Advance(diesAt.Add(-time.Nanosecond))
		assert.True(t, before.DiedAt.IsZero())
		assert.NotEqual(t, pet.StageDeath, before.Stage(diesAt.Add(-time.Nanosecond)))
		assert.Equal(t, pet.NotDead, before.CauseAt(diesAt.Add(-time.Nanosecond)))

		died := p.Advance(diesAt)
		assert.True(t, diesAt.Equal(died.DiedAt), "died at %v", died.DiedAt.Sub(born))
		assert.Equal(t, pet.Starvation, died.Cause)
		assert.Equal(t, pet.StageDeath, died.Stage(diesAt))
		assert.Equal(t, pet.Starvation, died.CauseAt(diesAt))
	})

	t.Run("however the time is advanced the Pet died at the same instant", func(t *testing.T) {
		t.Parallel()
		oneLong := p.Advance(born.Add(3 * time.Hour))

		walked := p
		for now := born.Add(pet.BeatInterval); !now.After(born.Add(3 * time.Hour)); now = now.Add(pet.BeatInterval) {
			walked = walked.Advance(now)
		}

		assert.True(t, diesAt.Equal(oneLong.DiedAt), "one long catch-up: died at %v", oneLong.DiedAt.Sub(born))
		assert.True(t, diesAt.Equal(walked.DiedAt), "20-second Beats: died at %v", walked.DiedAt.Sub(born))
		assert.Equal(t, oneLong, walked, "and both are the same frozen Pet")
	})
}

func TestARefillJustBeforeStarvationSavesThePet(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p := cleanPetBornAt(born) // Hunger Empty at 12:00, would starve at 22:00

	saved := p.Advance(born.Add(20*time.Minute + 59*time.Second)).Feed(pet.Meal)
	require.Equal(t, 1, saved.Hunger)

	// Hunger's steps stay on a 3-minute grid, so it is Empty again at 21:00, and
	// that second spell is what may starve the Pet: at 31:00.
	assert.True(t, saved.Advance(born.Add(31*time.Minute-time.Nanosecond)).DiedAt.IsZero())
	died := saved.Advance(born.Add(31 * time.Minute))
	assert.True(t, born.Add(31*time.Minute).Equal(died.DiedAt), "the first, refilled spell is forgotten")
	assert.Equal(t, pet.Starvation, died.Cause)
}

func TestHappinessAtZeroNeverKillsThePet(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p := cleanPetBornAt(born)
	p.Happiness = 0

	p = keptFedUntil(p, born, born.Add(55*time.Minute))

	assert.True(t, p.DiedAt.IsZero(), "an Empty Happiness alone does not kill")
	assert.Zero(t, p.Happiness)
	assert.NotZero(t, p.CareMistakes, "though it does cost Care mistakes")
}

func TestAPetThatLivesItsWholeLifeDiesOfOldAge(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p := keptFedUntil(cleanPetBornAt(born), born, born.Add(60*time.Minute))
	require.True(t, p.DiedAt.IsZero(), "sanity check: still alive at 60:00")

	died := p.Advance(born.Add(2 * time.Hour))

	assert.True(t, born.Add(oldAge).Equal(died.DiedAt), "died at %v", died.DiedAt.Sub(born))
	assert.Equal(t, pet.OldAge, died.Cause)
	assert.Equal(t, pet.StageDeath, died.Stage(born.Add(2*time.Hour)))
}

func TestADeadPetNeverChangesAgain(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	diesAt := born.Add(22 * time.Minute)
	died := cleanPetBornAt(born).Advance(diesAt)
	require.False(t, died.DiedAt.IsZero(), "sanity check")

	assert.Equal(t, died, died.Advance(born.Add(time.Hour)), "Advance leaves a dead Pet exactly as it was")
	assert.Equal(t, died, died.Advance(born.Add(100*time.Hour)))
	assert.Zero(t, died.Hunger, "Stats froze at the moment of death")
	assert.Zero(t, died.Happiness)

	assert.Equal(t, 22*time.Minute, died.Age(born.Add(5*time.Hour)), "Age stops at the moment of death")
	assert.Equal(t, 22*time.Minute, died.Age(diesAt))
}

// TestACauseTieGoesToStarvation: the spec's order for causes that fall at the
// very same instant is Starvation, Sickness, Neglect, Old age.
func TestACauseTieGoesToStarvation(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := born.Add(oldAge)
	p := pet.New(born)
	p.Hunger = 0
	p.HungerEmpty = pet.EmptySpell{Since: end.Add(-pet.StarvationInterval)} // starves exactly at old age
	p.LastSeenAt, p.HappinessLastSeenAt = end.Add(-5*time.Minute), end.Add(-5*time.Minute)
	p.LastCleanedAt = end

	died := p.Advance(end)

	assert.True(t, end.Equal(died.DiedAt))
	assert.Equal(t, pet.Starvation, died.Cause)
}

// TestASaveFromBeforeDeathsWereRecordedStillDiesOfAge: a Pet loaded from a save
// written before Death was recorded has no DiedAt. Once past its age it is dead
// by the age-based rule at once, and the next Advance records that death as Old
// age at the moment it happened.
func TestASaveFromBeforeDeathsWereRecordedStillDiesOfAge(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p := pet.New(born)
	p.LastSeenAt, p.HappinessLastSeenAt, p.LastCleanedAt = born.Add(59*time.Minute), born.Add(59*time.Minute), born.Add(59*time.Minute)
	now := born.Add(2 * time.Hour)

	assert.Equal(t, pet.StageDeath, p.Stage(now), "dead by the age-based rule, with nothing recorded")
	assert.Equal(t, pet.OldAge, p.CauseAt(now), "and its cause reads as Old age")
	assert.Equal(t, oldAge, p.Age(now))

	died := p.Advance(now)

	assert.True(t, born.Add(oldAge).Equal(died.DiedAt))
	assert.Equal(t, pet.OldAge, died.Cause)
	assert.Equal(t, died, died.Advance(now.Add(time.Hour)))
}

func TestACauseOfDeathHasAReadableName(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Old age", pet.OldAge.String())
	assert.Equal(t, "Starvation", pet.Starvation.String())
	assert.Empty(t, pet.NotDead.String())
}

// TestADeadPetStaysFrozenEvenAfterACareAction: the domain does not stop a Care
// action being applied to a dead Pet (the Screen offers none), but Advance is the
// only thing that moves time, and it must leave a dead Pet alone. Re-deriving the
// same death would happen to reproduce an untouched Pet, so this pins the
// guard on the one case where it matters: a Pet that has been changed since.
func TestADeadPetStaysFrozenEvenAfterACareAction(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	died := cleanPetBornAt(born).Advance(born.Add(22 * time.Minute))
	require.False(t, died.DiedAt.IsZero(), "sanity check")

	fed := died.Feed(pet.Meal)
	require.Equal(t, 1, fed.Hunger, "sanity check: the Meal was applied")

	later := fed.Advance(born.Add(2 * time.Hour))

	assert.Equal(t, fed, later, "no time passes for a dead Pet: Hunger does not decay again")
}

// TestASaveAlreadyPastItsDeathInstantIsRecordedWithoutRewindingItsStats: a Pet
// from a save written before deaths were recorded can have been advanced past its
// age by the old code, which kept decaying Stats after death. Its Stats cannot be
// rewound to the moment of death, so they are left as saved; the death is still
// recorded at the moment it happened.
func TestASaveAlreadyPastItsDeathInstantIsRecordedWithoutRewindingItsStats(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p := pet.New(born)
	saved := born.Add(2 * time.Hour) // long after old age at 60:30
	p.LastSeenAt, p.HappinessLastSeenAt, p.LastCleanedAt = saved, saved, saved
	p.Hunger, p.Happiness = 2, 1

	died := p.Advance(born.Add(3 * time.Hour))

	assert.True(t, born.Add(oldAge).Equal(died.DiedAt), "recorded at the moment it happened")
	assert.Equal(t, pet.OldAge, died.Cause)
	assert.Equal(t, 2, died.Hunger, "Stats stay as saved")
	assert.Equal(t, 1, died.Happiness)
}

// TestARecordKnowsTheLatestInstantItReaches: the Next Screen seeds its clock from
// it, before its first tick. An advanced Pet's record reaches the last Advance
// (Hunger's own anchor can trail it by nearly a step), and a dead Pet's reaches
// the moment it died, however long ago that was, since nothing advances it again.
func TestARecordKnowsTheLatestInstantItReaches(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("a fresh Pet reaches its birth", func(t *testing.T) {
		t.Parallel()
		assert.True(t, born.Equal(pet.New(born).RecordedUntil()))
	})

	t.Run("an advanced Pet reaches its last Advance, not Hunger's whole-step anchor", func(t *testing.T) {
		t.Parallel()
		advanced := pet.New(born).Advance(born.Add(4 * time.Minute))
		require.True(t, advanced.LastSeenAt.Before(born.Add(4*time.Minute)), "sanity check: Hunger's anchor trails")

		assert.True(t, born.Add(4*time.Minute).Equal(advanced.RecordedUntil()))
	})

	t.Run("a dead Pet reaches the moment it died", func(t *testing.T) {
		t.Parallel()
		diesAt := born.Add(22 * time.Minute)
		died := cleanPetBornAt(born).Advance(born.Add(3 * time.Hour))
		require.True(t, diesAt.Equal(died.DiedAt), "sanity check")
		require.True(t, died.LastSeenAt.Before(diesAt), "sanity check: Hunger's anchor is a step short of it")

		assert.True(t, diesAt.Equal(died.RecordedUntil()))
	})
}
