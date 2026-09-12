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
			// TestAdvancePicksTheMessRateAsOfNow) — clean it well past every
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

// TestAdvancePicksTheMessRateAsOfNow proves Advance uses a single decay rate
// for the whole elapsed window, chosen by whether the Pet HasMess at now —
// not an exact split at the moment the Mess would have appeared partway
// through the window.
func TestAdvancePicksTheMessRateAsOfNow(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p := pet.New(born)

	// The window [born, now] straddles the Mess boundary (MessInterval after
	// born), but the Pet HasMess by the end of it, so the whole window should
	// decay Happiness at the faster, messy rate.
	now := born.Add(pet.MessInterval + pet.AcceleratedHappinessDecayInterval)
	advanced := p.Advance(now)

	wantSteps := int((pet.MessInterval + pet.AcceleratedHappinessDecayInterval) / pet.AcceleratedHappinessDecayInterval)
	assert.Equal(t, pet.MaxStat-wantSteps, advanced.Happiness)
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
