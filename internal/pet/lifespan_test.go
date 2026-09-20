package pet_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/leekli/tamagotchi-go/internal/pet"
)

// These tests pin how Care mistakes shorten life. Only the Adult Stage shrinks:
// it begins at 40:30 whatever the tally, and lasts 20 minutes minus 2 per Care
// mistake, never below 5. Its end is the end of life: Old age with no mistakes,
// Neglect with any. A mistake that lands after the shortened end has already
// passed kills the Pet at the moment it is counted, never retroactively.

const adultBegins = 40*time.Minute + 30*time.Second

// wellCaredThrough looks after p every 4 minutes up to until: a Clean (so no Mess
// and no Sickness), two Meals and two Plays (so neither Stat ever empties, and no
// Care mistake accrues by itself). Its tally is then only what the test gives it.
func wellCaredThrough(p pet.Pet, born time.Time, until time.Duration) pet.Pet {
	for at := 4 * time.Minute; at <= until; at += 4 * time.Minute {
		now := born.Add(at)
		p = p.Advance(now).Clean(now).Feed(pet.Meal).Feed(pet.Meal).Play().Play()
	}
	return p
}

func TestAdultLastsItsNormalSpanMinusTwoMinutesPerMistakeDownToTheFloor(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	require.Equal(t, 2*time.Minute, pet.AdultMistakePenalty, "the worked timings assume 2 minutes per mistake")
	require.Equal(t, 5*time.Minute, pet.AdultMinDuration, "and a 5-minute floor")

	tests := map[string]struct {
		mistakes  int
		adultSpan time.Duration
		cause     pet.CauseOfDeath
	}{
		"none: the normal 20 minutes":             {0, 20 * time.Minute, pet.OldAge},
		"one: 18 minutes":                         {1, 18 * time.Minute, pet.Neglect},
		"three: 14 minutes":                       {3, 14 * time.Minute, pet.Neglect},
		"seven: 6 minutes":                        {7, 6 * time.Minute, pet.Neglect},
		"eight would be 4, but the floor is 5":    {8, 5 * time.Minute, pet.Neglect},
		"many more still leave the floor":         {50, 5 * time.Minute, pet.Neglect},
		"an implausible tally cannot overflow it": {1 << 40, 5 * time.Minute, pet.Neglect},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			p := pet.New(born)
			p.CareMistakes = tt.mistakes
			diesAt := born.Add(adultBegins + tt.adultSpan)

			// Looked after well past the death, so only the lifespan can end it.
			died := wellCaredThrough(p, born, 64*time.Minute).Advance(born.Add(2 * time.Hour))

			assert.True(t, diesAt.Equal(died.DiedAt), "died at %v, want %v", died.DiedAt.Sub(born), diesAt.Sub(born))
			assert.Equal(t, tt.cause, died.Cause)
			assert.Equal(t, tt.mistakes, died.CareMistakes, "and the tally is untouched")
		})
	}
}

func TestAPetWithNoMistakesLivesItsWholeLifeAndDiesOfOldAge(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p := wellCaredThrough(pet.New(born), born, 60*time.Minute)
	require.True(t, p.DiedAt.IsZero(), "sanity check: alive at 60:00")
	require.Zero(t, p.CareMistakes, "sanity check: well cared for, so no mistakes")

	died := p.Advance(born.Add(2 * time.Hour))

	assert.True(t, born.Add(oldAge).Equal(died.DiedAt), "with a zero tally the timeline is unchanged: 60:30")
	assert.Equal(t, pet.OldAge, died.Cause)
}

func TestAMistakeCountedInAnEarlierStageShortensTheAdultToo(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	// Cared for in every way except Play, so Happiness empties at 12:00 (a Child) and
	// its window expires at 16:00: one mistake, long before Adult begins.
	p := pet.New(born)
	for at := 4 * time.Minute; at <= 64*time.Minute; at += 4 * time.Minute {
		now := born.Add(at)
		p = p.Advance(now).Clean(now).Feed(pet.Meal).Feed(pet.Meal)
	}

	died := p.Advance(born.Add(2 * time.Hour))

	assert.Equal(t, 1, died.CareMistakes, "the one mistake, counted at 16:00")
	assert.True(t, born.Add(adultBegins+18*time.Minute).Equal(died.DiedAt), "so the Adult lasted 18 minutes, not 20")
	assert.Equal(t, pet.Neglect, died.Cause)
}

func TestTheAdultIsAlwaysSeenHoweverManyMistakes(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p := pet.New(born)
	p.CareMistakes = 50
	endsAt := born.Add(adultBegins + pet.AdultMinDuration) // 45:30

	assert.Equal(t, pet.StageTeen, p.Stage(born.Add(adultBegins-time.Nanosecond)), "Adult still begins at 40:30")
	assert.Equal(t, pet.StageAdult, p.Stage(born.Add(adultBegins)))
	assert.Equal(t, pet.StageAdult, p.Stage(endsAt.Add(-time.Nanosecond)), "and lasts at least its floor")
	assert.Equal(t, pet.StageDeath, p.Stage(endsAt))
}

func TestAMistakeThatLandsAfterTheShortenedEndKillsAtTheMomentItIsCounted(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	at := func(minutes int) time.Time { return born.Add(time.Duration(minutes) * time.Minute) }

	// One mistake so far, so the Adult would end at 58:30. Happiness has been Empty
	// since 54:00 and its window expires at 58:00, where the second mistake makes
	// the Adult 16 minutes: it would have ended at 56:30, two minutes before that.
	p := pet.New(born)
	p.CareMistakes = 1
	p.Hunger, p.Happiness = pet.MaxStat, 0
	p.HappinessEmpty = pet.EmptySpell{Since: at(54), GraceEndsAt: at(58)}
	p.LastSeenAt, p.HappinessLastSeenAt, p.LastCleanedAt = at(57), at(57), at(57)

	assert.True(t, p.Advance(at(58).Add(-time.Nanosecond)).DiedAt.IsZero(), "alive until the mistake lands")

	died := p.Advance(at(59))

	assert.True(t, at(58).Equal(died.DiedAt), "died at %v: the moment the mistake was counted, not at 56:30", died.DiedAt.Sub(born))
	assert.Equal(t, pet.Neglect, died.Cause)
	assert.Equal(t, 2, died.CareMistakes, "and the mistake that killed it is on the tally")
}

func TestALifespanCanEndBetweenTwoMistakesInOneWindow(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	at := func(minutes int) time.Time { return born.Add(time.Duration(minutes) * time.Minute) }

	// One mistake so far: the Adult would end at 58:30. Hunger's window expires at
	// 55:00 (a second mistake, so the Adult would end at 56:30) and Happiness's at
	// 57:00. The Pet dies at 56:30, between them, and the 57:00 mistake is never
	// counted.
	p := pet.New(born)
	p.CareMistakes = 1
	p.Hunger, p.Happiness = 0, 0
	p.HungerEmpty = pet.EmptySpell{Since: at(51), GraceEndsAt: at(55)}
	p.HappinessEmpty = pet.EmptySpell{Since: at(53), GraceEndsAt: at(57)}
	p.LastSeenAt, p.HappinessLastSeenAt, p.LastCleanedAt = at(54), at(54), at(54)

	oneLong := p.Advance(at(60))

	walked := p
	for now := at(54).Add(pet.BeatInterval); !now.After(at(60)); now = now.Add(pet.BeatInterval) {
		walked = walked.Advance(now)
	}

	for name, died := range map[string]pet.Pet{"one long catch-up": oneLong, "20-second Beats": walked} {
		assert.True(t, born.Add(56*time.Minute+30*time.Second).Equal(died.DiedAt), "%s: died at %v", name, died.DiedAt.Sub(born))
		assert.Equal(t, pet.Neglect, died.Cause, name)
		assert.Equal(t, 2, died.CareMistakes, "%s: the 55:00 mistake counted, the 57:00 one never", name)
	}
	assert.Equal(t, oneLong, walked)
}

func TestAMistakeWhileTheAdultHasTimeToSpareOnlyBringsTheEndCloser(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	at := func(minutes int) time.Time { return born.Add(time.Duration(minutes) * time.Minute) }

	// No mistakes so far (end 60:30). One lands at 50:00: the end moves to 58:30,
	// still ahead, so the Pet lives on and dies at 58:30, of Neglect.
	p := pet.New(born)
	p.Hunger, p.Happiness = pet.MaxStat, 0
	p.HappinessEmpty = pet.EmptySpell{Since: at(46), GraceEndsAt: at(50)}
	p.LastSeenAt, p.HappinessLastSeenAt, p.LastCleanedAt = at(49), at(49), at(49)

	died := p.Advance(at(62))

	assert.True(t, born.Add(58*time.Minute+30*time.Second).Equal(died.DiedAt))
	assert.Equal(t, pet.Neglect, died.Cause)
	assert.Equal(t, 1, died.CareMistakes)
}

// TestAPetPastItsShortenedAgeIsDeadWithoutAnyAdvance: Stage, Age and the cause read
// the tally too, so a Pet whose Advance has not run yet (or a save from before
// deaths were recorded) is dead once past its shortened age, of Neglect.
func TestAPetPastItsShortenedAgeIsDeadWithoutAnyAdvance(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p := pet.New(born)
	p.CareMistakes = 3 // the Adult lasts 14 minutes: 54:30
	endsAt := born.Add(adultBegins + 14*time.Minute)

	assert.Equal(t, pet.StageAdult, p.Stage(endsAt.Add(-time.Nanosecond)))
	assert.Equal(t, pet.StageDeath, p.Stage(endsAt))
	assert.Equal(t, pet.Neglect, p.CauseAt(endsAt.Add(time.Hour)))
	assert.Equal(t, pet.NotDead, p.CauseAt(endsAt.Add(-time.Nanosecond)))
	assert.Equal(t, endsAt.Sub(born), p.Age(endsAt.Add(time.Hour)), "and Age stops at the shortened end")
}

func TestNeglectFollowsStarvationAndSicknessInTheSpecsOrderOnATie(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := born.Add(adultBegins + 18*time.Minute) // one mistake: 58:30

	t.Run("Starvation before Neglect", func(t *testing.T) {
		t.Parallel()
		p := pet.New(born)
		p.CareMistakes = 1
		p.Hunger = 0
		p.HungerEmpty = pet.EmptySpell{Since: end.Add(-pet.StarvationInterval)}
		p.LastSeenAt, p.HappinessLastSeenAt, p.LastCleanedAt = end.Add(-time.Minute), end.Add(-time.Minute), end.Add(-time.Minute)

		died := p.Advance(end)

		assert.True(t, end.Equal(died.DiedAt))
		assert.Equal(t, pet.Starvation, died.Cause)
	})

	t.Run("Sickness before Neglect", func(t *testing.T) {
		t.Parallel()
		p := pet.New(born)
		p.CareMistakes = 1
		p.SickSince = end.Add(-pet.SickDeathInterval)
		p.LastCleanedAt = p.SickSince.Add(-pet.MessInterval - pet.SickAfterMess)
		p.LastSeenAt, p.HappinessLastSeenAt = end.Add(-time.Minute), end.Add(-time.Minute)

		died := p.Advance(end)

		assert.True(t, end.Equal(died.DiedAt))
		assert.Equal(t, pet.Sickness, died.Cause)
	})
}

func TestNeglectHasAReadableNameThatSurvivesASaveAndLoad(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Neglect", pet.Neglect.String())
}
