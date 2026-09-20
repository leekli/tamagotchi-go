package pet_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/leekli/tamagotchi-go/internal/pet"
)

func TestAdvance(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		elapsed       time.Duration
		wantHunger    int
		wantHappiness int
	}{
		"zero elapsed":   {0, pet.MaxStat, pet.MaxStat},
		"partial step":   {pet.HungerDecayInterval - time.Second, pet.MaxStat, pet.MaxStat},
		"exact step":     {pet.HungerDecayInterval, pet.MaxStat - 1, pet.MaxStat - 1},
		"multiple steps": {2 * pet.HungerDecayInterval, pet.MaxStat - 2, pet.MaxStat - 2},
		"floors at zero": {10 * pet.HungerDecayInterval, 0, 0},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			p := pet.New(born)
			// This table is about generic elapsed-duration decay mechanics,
			// not the Mess interaction (covered separately by
			// TestAdvanceDecaysHappinessFasterWithAMess and
			// TestAdvanceSplitsTheWindowAtMessOnset) — clean it well past every
			// elapsed value below so HasMess never trips and Happiness
			// decays at the same rate as Hunger throughout.
			p.LastCleanedAt = born.Add(24 * time.Hour)
			now := born.Add(tt.elapsed)
			advanced := p.Advance(now)

			assert.Equal(t, tt.wantHunger, advanced.Hunger)
			assert.Equal(t, tt.wantHappiness, advanced.Happiness)
			assert.Equal(t, pet.BaseWeight, advanced.Weight, "Weight never decays")

			// LastSeenAt only advances by whole decay steps actually
			// consumed, not all the way to now: any sub-interval remainder is
			// preserved for the next Advance call (see
			// TestAdvanceAccumulatesAcrossRepeatedShortCalls).
			wantConsumed := (tt.elapsed / pet.HungerDecayInterval) * pet.HungerDecayInterval
			assert.Equal(t, born.Add(wantConsumed), advanced.LastSeenAt)
		})
	}
}

// TestAdvanceIgnoredPetTimeline pins the worked timeline from the neglect spec
// for a Pet that is never cared for: Mess appears at 5:00, which doubles
// Happiness's Decay rate from then on while keeping the two thirds of a step
// already made by 5:00, so Happiness runs 3 at 3:00, 2 at 5:30, 1 at 7:00 and
// 0 at 8:30. Hunger's rate never changes: 0 at 12:00. Every instant is checked
// one nanosecond either side of its boundary, in a single Advance call.
func TestAdvanceIgnoredPetTimeline(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	at := func(minutes, seconds int) time.Duration {
		return time.Duration(minutes)*time.Minute + time.Duration(seconds)*time.Second
	}

	tests := map[string]struct {
		elapsed       time.Duration
		wantHunger    int
		wantHappiness int
	}{
		"just before the first step":        {at(3, 0) - time.Nanosecond, 4, 4},
		"3:00 first step of both":           {at(3, 0), 3, 3},
		"just before Happiness speeds up":   {at(5, 30) - time.Nanosecond, 3, 3},
		"5:30 the carried step lands":       {at(5, 30), 3, 2},
		"6:00 Hunger's second step":         {at(6, 0), 2, 2},
		"7:00 Happiness at the fast rate":   {at(7, 0), 2, 1},
		"just before Happiness is empty":    {at(8, 30) - time.Nanosecond, 2, 1},
		"8:30 Happiness is empty":           {at(8, 30), 2, 0},
		"9:00 Hunger's third step":          {at(9, 0), 1, 0},
		"just before Hunger is empty":       {at(12, 0) - time.Nanosecond, 1, 0},
		"12:00 Hunger is empty":             {at(12, 0), 0, 0},
		"long afterwards both stay at zero": {at(12, 0) + time.Hour, 0, 0},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			advanced := pet.New(born).Advance(born.Add(tt.elapsed))

			assert.Equal(t, tt.wantHunger, advanced.Hunger, "Hunger")
			assert.Equal(t, tt.wantHappiness, advanced.Happiness, "Happiness")
		})
	}
}

// TestAdvanceAccumulatesAcrossRepeatedShortCalls guards against the
// regression where every Advance call reset LastSeenAt straight to now: that
// discarded each call's sub-interval progress, so repeated short calls (the
// Next Screen's Beat fires every 20s, far shorter than the 3-minute decay
// interval) never accumulated into a whole decay step at all.
func TestAdvanceAccumulatesAcrossRepeatedShortCalls(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p := pet.New(born)
	// This test is about accumulating sub-interval progress across repeated
	// short calls, not the Mess interaction — keep it clean throughout.
	p.LastCleanedAt = born.Add(24 * time.Hour)

	const step = 20 * time.Second
	now := born
	for range 30 { // 30 * 20s = 10 minutes of continuous play
		now = now.Add(step)
		p = p.Advance(now)
	}

	wantSteps := int(10 * time.Minute / pet.HungerDecayInterval)
	assert.Equal(t, pet.MaxStat-wantSteps, p.Hunger)
	assert.Equal(t, pet.MaxStat-wantSteps, p.Happiness)
}

// TestAdvanceAccumulatesCorrectlyAcrossRepeatedShortCallsWhileMessy is a
// regression guard on Hunger and Happiness decaying against separate
// anchors: with a shared anchor advanced by the smaller of the two stats'
// consumed durations, Happiness's own already-applied step would be
// recomputed from the stale anchor and re-subtracted on every subsequent
// Beat until Hunger (the slower stat once messy) finally caught up to its
// own first step — collapsing Happiness to 0 far faster than
// AcceleratedHappinessDecayInterval intends.
func TestAdvanceAccumulatesCorrectlyAcrossRepeatedShortCallsWhileMessy(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p := pet.New(born)
	p.LastCleanedAt = born.Add(-pet.MessInterval) // messy from the start

	const step = pet.BeatInterval
	now := born
	for range 9 { // 9 * 20s = 3 minutes of continuous play
		now = now.Add(step)
		p = p.Advance(now)
	}

	wantSteps := int(3 * time.Minute / pet.AcceleratedHappinessDecayInterval)
	require.Equal(t, 2, wantSteps, "sanity check on the expected step count")
	assert.Equal(t, pet.MaxStat-wantSteps, p.Happiness)
}

func TestAdvanceIgnoresClockGoingBackwards(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p := pet.New(born).Advance(born.Add(time.Hour)) // some decay applied, LastSeenAt moved on

	rewound := p.Advance(born) // now is before LastSeenAt

	assert.Equal(t, p, rewound, "a clock that went backwards should leave the Pet unchanged")
}

// TestAdvanceIgnoresAClockBehindOnlyTheHappinessAnchor covers the gap between
// the two anchors. After an Advance to 4:00, Hunger's anchor sits back at 3:00
// (its last whole step) while Happiness's has moved on to 4:00 itself. A time
// between the two is ahead of one anchor but behind the other, and must still
// be treated as the clock going backwards: applying it would move Happiness's
// anchor backwards and corrupt its progress.
func TestAdvanceIgnoresAClockBehindOnlyTheHappinessAnchor(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p := pet.New(born).Advance(born.Add(4 * time.Minute))
	require.True(t, p.LastSeenAt.Before(p.HappinessLastSeenAt), "sanity check: the anchors should differ")

	between := p.LastSeenAt.Add(30 * time.Second)
	require.True(t, between.Before(p.HappinessLastSeenAt), "sanity check: the time should be behind Happiness's anchor")

	assert.Equal(t, p, p.Advance(between))
}

// TestAdvanceDecaysHappinessFasterWithAMess is a direct comparison, not just
// "some decay happened": the same elapsed time must cost a messy Pet more
// Happiness than a clean one.
func TestAdvanceDecaysHappinessFasterWithAMess(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	elapsed := pet.HappinessDecayInterval + pet.AcceleratedHappinessDecayInterval

	clean := pet.New(born).Advance(born.Add(elapsed))

	messyStart := pet.New(born)
	messyStart.LastCleanedAt = born.Add(-pet.MessInterval) // already messy at born
	messy := messyStart.Advance(born.Add(elapsed))

	assert.Less(t, messy.Happiness, clean.Happiness,
		"a Pet with a Mess should lose more Happiness over the same elapsed time")
}

// TestAdvanceSplitsTheWindowAtMessOnset replaces the test that used to lock in
// the retired "one rate for the whole window, chosen by whether the Pet HasMess
// at now" rule. A window that straddles the moment Mess appears must be split
// there: only the part after it decays at the faster rate. Here the window
// runs from birth to 6:30, so Mess appears at 5:00 and Happiness is at 2 (the
// old rule wrongly gave 0, applying the fast rate from birth).
func TestAdvanceSplitsTheWindowAtMessOnset(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	advanced := pet.New(born).Advance(born.Add(pet.MessInterval + 90*time.Second))

	assert.Equal(t, 2, advanced.Happiness)
}

// The next three tests pin proportional carry across the rate changes a Care
// action causes. Each applies its action straight after an Advance to the
// instant of the action, then checks the very next Happiness point lands
// exactly when the unfinished fraction of the step predicts. A restart of the
// step (or a retroactive rate change) would land it minutes away.

// TestAdvanceCarriesProgressAcrossClean: a Pet messy from birth has made 60s
// at the double rate by 1:00, i.e. two thirds of a step. Clean drops the rate
// to normal, so the remaining third takes 60s more: the point lands at 2:00.
func TestAdvanceCarriesProgressAcrossClean(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p := pet.New(born)
	p.LastCleanedAt = born.Add(-pet.MessInterval) // messy from the start

	cleanedAt := born.Add(time.Minute)
	p = p.Advance(cleanedAt).Clean(cleanedAt)

	assert.Equal(t, pet.MaxStat, p.Advance(born.Add(2*time.Minute-time.Nanosecond)).Happiness)
	assert.Equal(t, pet.MaxStat-1, p.Advance(born.Add(2*time.Minute)).Happiness)
}

// TestAdvanceCarriesProgressAcrossOverfedStarting: a Pet one Snack short of
// Overfed has made 60s at the normal rate by 1:00, one third of a step. A Snack
// makes it Overfed, doubling the rate, so the remaining two thirds take 60s:
// the point lands at 2:00.
func TestAdvanceCarriesProgressAcrossOverfedStarting(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p := pet.New(born)
	p.Weight = pet.OverfedThreshold - 1

	snackAt := born.Add(time.Minute)
	p = p.Advance(snackAt).Feed(pet.Snack)
	require.True(t, p.Overfed(), "the Snack should have made the Pet Overfed")

	assert.Equal(t, pet.MaxStat, p.Advance(born.Add(2*time.Minute-time.Nanosecond)).Happiness)
	assert.Equal(t, pet.MaxStat-1, p.Advance(born.Add(2*time.Minute)).Happiness)
}

// TestAdvanceCarriesProgressAcrossOverfedEnding: an Overfed Pet has made 45s
// at the double rate by 0:45, i.e. 90s of normal-rate progress: half a step.
// Play resolves the Overfed state, halving the rate, so the remaining half
// takes 90s: the point lands at 2:15.
func TestAdvanceCarriesProgressAcrossOverfedEnding(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p := pet.New(born)
	p.Weight = pet.OverfedThreshold

	playAt := born.Add(45 * time.Second)
	p = p.Advance(playAt).Play()
	require.False(t, p.Overfed(), "Play should have resolved the Overfed state")

	assert.Equal(t, pet.MaxStat, p.Advance(born.Add(135*time.Second-time.Nanosecond)).Happiness)
	assert.Equal(t, pet.MaxStat-1, p.Advance(born.Add(135*time.Second)).Happiness)
}

// TestAdvanceDecaysHappinessFasterWhenOverfed mirrors
// TestAdvanceDecaysHappinessFasterWithAMess for Overfed instead of Mess.
func TestAdvanceDecaysHappinessFasterWhenOverfed(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	elapsed := pet.HappinessDecayInterval + pet.AcceleratedHappinessDecayInterval

	notOverfed := pet.New(born).Advance(born.Add(elapsed))

	overfedStart := pet.New(born)
	overfedStart.Weight = pet.OverfedThreshold
	overfed := overfedStart.Advance(born.Add(elapsed))

	assert.Less(t, overfed.Happiness, notOverfed.Happiness,
		"an Overfed Pet should lose more Happiness over the same elapsed time")
}

// TestAdvanceDoesNotCompoundMessAndOverfed proves a Pet that is both Messy
// and Overfed at once decays Happiness at the same single accelerated rate
// as either condition alone, not a faster combined one.
func TestAdvanceDoesNotCompoundMessAndOverfed(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	elapsed := pet.HappinessDecayInterval + pet.AcceleratedHappinessDecayInterval

	messyOnly := pet.New(born)
	messyOnly.LastCleanedAt = born.Add(-pet.MessInterval)
	messyAdvanced := messyOnly.Advance(born.Add(elapsed))

	both := pet.New(born)
	both.LastCleanedAt = born.Add(-pet.MessInterval)
	both.Weight = pet.OverfedThreshold
	bothAdvanced := both.Advance(born.Add(elapsed))

	assert.Equal(t, messyAdvanced.Happiness, bothAdvanced.Happiness,
		"Messy+Overfed together should decay Happiness at the same rate as Messy alone, not faster")
}

func TestAdvanceHungerAndHappinessNeverGoNegative(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p := pet.New(born)

	advanced := p.Advance(born.Add(1000 * pet.HungerDecayInterval))

	assert.Equal(t, 0, advanced.Hunger)
	assert.Equal(t, 0, advanced.Happiness)
}
