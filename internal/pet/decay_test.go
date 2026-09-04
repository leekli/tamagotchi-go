package pet_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

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

func TestAdvanceIgnoresClockGoingBackwards(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p := pet.New(born).Advance(born.Add(time.Hour)) // some decay applied, LastSeenAt moved on

	rewound := p.Advance(born) // now is before LastSeenAt

	assert.Equal(t, p, rewound, "a clock that went backwards should leave the Pet unchanged")
}

func TestAdvanceHungerAndHappinessNeverGoNegative(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p := pet.New(born)

	advanced := p.Advance(born.Add(1000 * pet.HungerDecayInterval))

	assert.Equal(t, 0, advanced.Hunger)
	assert.Equal(t, 0, advanced.Happiness)
}
