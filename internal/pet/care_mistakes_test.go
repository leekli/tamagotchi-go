package pet_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/leekli/tamagotchi-go/internal/pet"
)

// These tests pin Care mistakes: a Stat left Empty (at 0) for the grace window
// earns one mistake, once per Empty spell, per Stat. Timings are worked by hand
// from the Decay rules: a Pet kept clean decays both Stats one point per 3
// minutes, so a Stat that starts at 1 empties at 3:00, and one that starts full
// empties at 12:00. The grace window is 4 minutes.

// cleanPetBornAt is a fresh Pet that will not get a Mess for a day, so
// Happiness decays at the same normal rate as Hunger throughout.
func cleanPetBornAt(born time.Time) pet.Pet {
	p := pet.New(born)
	p.LastCleanedAt = born.Add(24 * time.Hour)
	return p
}

func TestAnEmptyHungerSpellEarnsOneMistakeWhenItsGraceWindowExpires(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p := cleanPetBornAt(born)
	p.Hunger = 1 // Empty at 3:00, so its grace window expires at 7:00
	// Happiness starts full, so it does not empty (12:00) until well after.

	require.Equal(t, 4*time.Minute, pet.GraceWindow, "the worked timings below assume a 4-minute window")

	assert.Zero(t, p.Advance(born.Add(7*time.Minute-time.Nanosecond)).CareMistakes, "just before the window expires")
	assert.Equal(t, 1, p.Advance(born.Add(7*time.Minute)).CareMistakes, "at the exact instant it expires")
	assert.Equal(t, 1, p.Advance(born.Add(15*time.Minute)).CareMistakes,
		"one spell earns one mistake however long it lasts")
}

func TestAnEmptyHungerSpellEarnsOneMistakeHoweverItIsAdvanced(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p := cleanPetBornAt(born)
	p.Hunger = 1

	oneLong := p.Advance(born.Add(15 * time.Minute))

	walked := p
	for now := born.Add(pet.BeatInterval); !now.After(born.Add(15 * time.Minute)); now = now.Add(pet.BeatInterval) {
		walked = walked.Advance(now)
	}

	assert.Equal(t, 1, oneLong.CareMistakes)
	assert.Equal(t, oneLong.CareMistakes, walked.CareMistakes, "20-second Beats should count the same mistake once")
}

func TestHungerAndHappinessEmptySpellsAreJudgedIndependently(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("both Empty together earn two mistakes", func(t *testing.T) {
		t.Parallel()
		p := cleanPetBornAt(born) // both empty at 12:00, both windows expire at 16:00

		assert.Zero(t, p.Advance(born.Add(16*time.Minute-time.Nanosecond)).CareMistakes)
		assert.Equal(t, 2, p.Advance(born.Add(16*time.Minute)).CareMistakes)
	})

	t.Run("only Happiness Empty earns one", func(t *testing.T) {
		t.Parallel()
		p := cleanPetBornAt(born)
		p.Happiness = 1 // Empty at 3:00, window expires at 7:00

		assert.Zero(t, p.Advance(born.Add(7*time.Minute-time.Nanosecond)).CareMistakes)
		assert.Equal(t, 1, p.Advance(born.Add(7*time.Minute)).CareMistakes)
	})

	t.Run("Stats emptying at different times are counted at their own instants", func(t *testing.T) {
		t.Parallel()
		p := cleanPetBornAt(born)
		p.Hunger = 1    // Empty 3:00, expires 7:00
		p.Happiness = 2 // Empty 6:00, expires 10:00

		assert.Equal(t, 1, p.Advance(born.Add(9*time.Minute)).CareMistakes, "only Hunger's window has expired")
		assert.Equal(t, 2, p.Advance(born.Add(10*time.Minute)).CareMistakes, "now Happiness's has too")
	})
}

func TestRefillingBeforeTheWindowExpiresEarnsNoMistake(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p := cleanPetBornAt(born)
	p.Hunger = 1 // Empty at 3:00; the window would expire at 7:00

	refilledAt := born.Add(6*time.Minute + 59*time.Second)
	p = p.Advance(refilledAt).Feed(pet.Meal)
	require.Equal(t, 1, p.Hunger)

	// The first spell's 7:00 deadline passes with Hunger refilled, so it must not
	// be counted. Advancing to just past it is what shows that: a Pet that kept
	// the stale spell would count it here, whereas jumping straight to a later
	// time can hide it, because a new spell overwrites the old.
	p = p.Advance(born.Add(8 * time.Minute))
	require.Zero(t, p.CareMistakes, "the refilled spell's deadline passed")

	// Hunger's own steps keep landing every 3 minutes from birth, so it is Empty
	// again at 9:00 and that second spell's window expires at 13:00. Nothing was
	// counted for the first spell, which was cut short.
	assert.Zero(t, p.Advance(born.Add(13*time.Minute-time.Nanosecond)).CareMistakes)
	assert.Equal(t, 1, p.Advance(born.Add(13*time.Minute)).CareMistakes, "only the second spell counts")
}

// TestAnyRefillEndsAnEmptySpell: Meal, Snack and Play each lift the Stat they
// feed above 0, ending its spell; a later spell can then earn its own mistake.
func TestAnyRefillEndsAnEmptySpell(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		emptyStat func(*pet.Pet)
		refill    func(pet.Pet) pet.Pet
	}{
		"Meal refills Hunger":     {func(p *pet.Pet) { p.Hunger = 1 }, func(p pet.Pet) pet.Pet { return p.Feed(pet.Meal) }},
		"Snack refills Happiness": {func(p *pet.Pet) { p.Happiness = 1 }, func(p pet.Pet) pet.Pet { return p.Feed(pet.Snack) }},
		"Play refills Happiness":  {func(p *pet.Pet) { p.Happiness = 1 }, func(p pet.Pet) pet.Pet { return p.Play() }},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			p := cleanPetBornAt(born)
			tt.emptyStat(&p) // Empty at 3:00, first window would expire at 7:00

			// Refilled at 6:30, 3:30 into the spell and before its deadline. The Stat
			// is not Empty again until 9:00 (its steps keep landing every 3 minutes
			// from birth), after the old deadline, so passing 7:00 in between is what
			// shows the old spell really ended. The second spell runs 9:00 to 13:00.
			p = tt.refill(p.Advance(born.Add(6*time.Minute + 30*time.Second)))
			p = p.Advance(born.Add(8 * time.Minute))
			require.Zero(t, p.CareMistakes, "the first spell ended at the refill, so its 7:00 deadline earns nothing")

			assert.Zero(t, p.Advance(born.Add(13*time.Minute-time.Nanosecond)).CareMistakes)
			assert.Equal(t, 1, p.Advance(born.Add(13*time.Minute)).CareMistakes,
				"a later spell earns its own mistake")
		})
	}
}

func TestALaterSpellCanEarnAnotherMistake(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p := cleanPetBornAt(born)
	p.Hunger = 1 // first spell: Empty 3:00, expires 7:00

	p = p.Advance(born.Add(8 * time.Minute)) // the first mistake is counted
	require.Equal(t, 1, p.CareMistakes)
	p = p.Feed(pet.Meal) // 8:00: refilled, ending the spell (Hunger 0 -> 1)

	// Hunger's next step lands at 9:00 (steps stay on a 3-minute grid from birth),
	// so a second spell runs 9:00 to 13:00.
	assert.Equal(t, 1, p.Advance(born.Add(12*time.Minute+59*time.Second)).CareMistakes)
	assert.Equal(t, 2, p.Advance(born.Add(13*time.Minute)).CareMistakes)
}

func TestTheAttentionCallReportsEmptyStatsAndWhetherTheirWindowIsRunning(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	at := born.Add

	t.Run("Hunger walks through none, running, lapsed, and none again once refilled", func(t *testing.T) {
		t.Parallel()
		p := cleanPetBornAt(born)
		p.Hunger = 1 // Empty 3:00, window until 7:00

		state := func(now time.Time) pet.Attention { return p.Advance(now).HungerAttention(now) }

		assert.Equal(t, pet.AttentionNone, state(at(3*time.Minute-time.Nanosecond)), "not Empty yet")
		assert.Equal(t, pet.AttentionWindowRunning, state(at(3*time.Minute)), "Empty, window just opened")
		assert.Equal(t, pet.AttentionWindowRunning, state(at(7*time.Minute-time.Nanosecond)), "window still running")
		assert.Equal(t, pet.AttentionLapsed, state(at(7*time.Minute)), "window lapsed")
		assert.Equal(t, pet.AttentionLapsed, state(at(10*time.Minute)), "and it stays lapsed until refilled")

		refilled := p.Advance(at(8 * time.Minute)).Feed(pet.Meal)
		assert.Equal(t, pet.AttentionNone, refilled.HungerAttention(at(8*time.Minute)), "refilled")
	})

	t.Run("Happiness is reported separately from Hunger", func(t *testing.T) {
		t.Parallel()
		p := cleanPetBornAt(born)
		p.Happiness = 1 // only Happiness empties at 3:00

		advanced := p.Advance(at(4 * time.Minute))

		assert.Equal(t, pet.AttentionWindowRunning, advanced.HappinessAttention(at(4*time.Minute)))
		assert.Equal(t, pet.AttentionNone, advanced.HungerAttention(at(4*time.Minute)), "Hunger is not Empty")
		assert.True(t, advanced.NeedsAttention(at(4*time.Minute)))
		assert.False(t, p.Advance(at(2*time.Minute)).NeedsAttention(at(2*time.Minute)), "nothing is Empty yet")
	})
}

// TestAStatAlreadyEmptyWithNoRecordedStartGetsAFreshWindow covers a Pet loaded
// from a save written before Care mistakes existed: it can have a Stat at 0 with
// no record of when it emptied. Upgrading must not punish it retroactively, so
// its window starts at the first Advance, not at whenever it really emptied.
func TestAStatAlreadyEmptyWithNoRecordedStartGetsAFreshWindow(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p := cleanPetBornAt(born)
	p.Hunger = 0 // Empty, with no recorded start

	assert.Equal(t, pet.AttentionWindowRunning, p.HungerAttention(born.Add(20*time.Minute)),
		"before any Advance the window is treated as just opening")

	firstAdvance := born.Add(20 * time.Minute)
	advanced := p.Advance(firstAdvance)

	// Happiness ran out at 12:00 in those 20 minutes and its window expired at 16:00, so
	// that one mistake is genuine. Hunger's fresh window opens at the Advance.
	assert.Equal(t, 1, advanced.CareMistakes, "only Happiness's spell, which really did lapse")
	assert.Equal(t, pet.AttentionWindowRunning, advanced.HungerAttention(firstAdvance))
	assert.Equal(t, 1, advanced.Advance(firstAdvance.Add(pet.GraceWindow-time.Nanosecond)).CareMistakes,
		"Hunger's fresh window has not expired")
	assert.Equal(t, 2, advanced.Advance(firstAdvance.Add(pet.GraceWindow)).CareMistakes,
		"it expires a full window after the first Advance")
}

// TestAnEmptySpellBeginsAtTheInstantTheStatReachesZero pins that a spell's
// recorded start is exactly when the Stat reaches 0: at that instant the Stat
// is 0, and one nanosecond before it still has its last point. Happiness at
// the doubled rate with an odd amount of progress left is the case that
// needs rounding, and a start rounded the wrong way would name an instant at
// which the Stat had not yet emptied.
func TestAnEmptySpellBeginsAtTheInstantTheStatReachesZero(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	reaches := 20 * time.Minute

	tests := map[string]struct {
		prepare func(*pet.Pet)
		stat    func(pet.Pet) int
		since   func(pet.Pet) time.Time
	}{
		"Hunger": {
			prepare: func(p *pet.Pet) { p.Hunger = 2 },
			stat:    func(p pet.Pet) int { return p.Hunger },
			since:   func(p pet.Pet) time.Time { return p.HungerEmpty.Since },
		},
		"Happiness at the normal rate": {
			prepare: func(p *pet.Pet) { p.Happiness = 2 },
			stat:    func(p pet.Pet) int { return p.Happiness },
			since:   func(p pet.Pet) time.Time { return p.HappinessEmpty.Since },
		},
		"Happiness at the doubled rate with an odd amount of progress left": {
			prepare: func(p *pet.Pet) {
				p.Happiness = 1
				p.Weight = pet.OverfedThreshold       // the doubled rate
				p.HappinessProgress = time.Nanosecond // so the work left, and its half, is odd
			},
			stat:  func(p pet.Pet) int { return p.Happiness },
			since: func(p pet.Pet) time.Time { return p.HappinessEmpty.Since },
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			p := cleanPetBornAt(born)
			tt.prepare(&p)

			since := tt.since(p.Advance(born.Add(reaches)))
			require.False(t, since.IsZero(), "the Stat should have emptied within the window")

			assert.Zero(t, tt.stat(p.Advance(since)), "at the recorded start the Stat is 0")
			assert.NotZero(t, tt.stat(p.Advance(since.Add(-time.Nanosecond))), "one nanosecond earlier it is not")
		})
	}
}

// TestAHappinessAlreadyEmptyWithNoRecordedStartGetsAFreshWindow is the mirror of
// the Hunger case above: a Pet from a pre-spells save with Happiness at 0.
func TestAHappinessAlreadyEmptyWithNoRecordedStartGetsAFreshWindow(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p := cleanPetBornAt(born)
	p.Happiness = 0 // Empty, with no recorded start

	// 14 minutes in: Hunger emptied at 12:00 and its own window expires at 16:00,
	// before the 22:00 at which it would starve the Pet.
	firstAdvance := born.Add(14 * time.Minute)
	assert.Equal(t, pet.AttentionWindowRunning, p.HappinessAttention(firstAdvance))

	advanced := p.Advance(firstAdvance)

	assert.Zero(t, advanced.CareMistakes, "nothing has lapsed yet, and Happiness's time at 0 before the upgrade is not held against it")
	assert.Equal(t, pet.AttentionWindowRunning, advanced.HappinessAttention(firstAdvance))

	// Hunger's genuine mistake lands at 16:00; Happiness's fresh window, a full
	// GraceWindow from the first Advance, expires at 18:00.
	assert.Equal(t, 1, advanced.Advance(firstAdvance.Add(pet.GraceWindow-time.Nanosecond)).CareMistakes, "only Hunger's, so far")
	assert.Equal(t, 2, advanced.Advance(firstAdvance.Add(pet.GraceWindow)).CareMistakes,
		"Happiness's window is a full GraceWindow from the first Advance")
}
