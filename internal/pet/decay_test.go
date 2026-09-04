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
			assert.Equal(t, now, advanced.LastSeenAt)
		})
	}
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
