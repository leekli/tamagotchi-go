package pet_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/leekli/tamagotchi-go/internal/pet"
)

var actionsBorn = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func TestFeedMeal(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		startHunger int
		wantHunger  int
	}{
		"from zero":      {0, 1},
		"below the cap":  {pet.MaxStat - 1, pet.MaxStat},
		"already at cap": {pet.MaxStat, pet.MaxStat},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			p := pet.New(actionsBorn)
			p.Hunger = tt.startHunger
			startWeight := p.Weight

			fed := p.Feed(pet.Meal)

			assert.Equal(t, tt.wantHunger, fed.Hunger)
			assert.Equal(t, startWeight, fed.Weight, "Meal must not change Weight")
			assert.Equal(t, p.Happiness, fed.Happiness, "Meal must not change Happiness")
		})
	}
}

func TestFeedSnack(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		startHappiness int
		wantHappiness  int
	}{
		"from zero":      {0, 1},
		"below the cap":  {pet.MaxStat - 1, pet.MaxStat},
		"already at cap": {pet.MaxStat, pet.MaxStat},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			p := pet.New(actionsBorn)
			p.Happiness = tt.startHappiness
			startWeight := p.Weight
			startHunger := p.Hunger

			fed := p.Feed(pet.Snack)

			assert.Equal(t, tt.wantHappiness, fed.Happiness)
			assert.Equal(t, startWeight+1, fed.Weight, "Snack adds Weight, uncapped")
			assert.Equal(t, startHunger, fed.Hunger, "Snack must not change Hunger")
		})
	}
}

func TestFeedSnackWeightIsUncapped(t *testing.T) {
	t.Parallel()

	p := pet.New(actionsBorn)
	for range 10 {
		p = p.Feed(pet.Snack)
	}

	assert.Equal(t, pet.BaseWeight+10, p.Weight)
}

func TestPlay(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		startHappiness int
		wantHappiness  int
	}{
		"from zero":      {0, 1},
		"below the cap":  {pet.MaxStat - 1, pet.MaxStat},
		"already at cap": {pet.MaxStat, pet.MaxStat},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			p := pet.New(actionsBorn)
			p.Happiness = tt.startHappiness
			startHunger := p.Hunger
			startWeight := p.Weight

			played := p.Play()

			assert.Equal(t, tt.wantHappiness, played.Happiness)
			assert.Equal(t, startHunger, played.Hunger, "Play must not change Hunger")
			assert.Equal(t, startWeight, played.Weight, "Play must not change Weight")
		})
	}
}

func TestCleanClearsMessAndUpdatesLastCleanedAt(t *testing.T) {
	t.Parallel()

	p := pet.New(actionsBorn)
	messyAt := actionsBorn.Add(pet.MessInterval)
	require := assert.New(t)
	require.True(p.HasMess(messyAt), "should have a Mess once MessInterval has elapsed")

	cleaned := p.Clean(messyAt)

	require.False(cleaned.HasMess(messyAt), "cleaning should clear the Mess immediately")
	require.Equal(messyAt, cleaned.LastCleanedAt)
}

func TestCleanMakesNoDirectStatChange(t *testing.T) {
	t.Parallel()

	p := pet.New(actionsBorn)
	p.Hunger, p.Happiness = 1, 1

	cleaned := p.Clean(actionsBorn.Add(time.Minute))

	assert.Equal(t, p.Hunger, cleaned.Hunger)
	assert.Equal(t, p.Happiness, cleaned.Happiness)
	assert.Equal(t, p.Weight, cleaned.Weight)
}
