package pet_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/leekli/tamagotchi-go/internal/pet"
)

func TestNewSetsFieldsFromNow(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	p := pet.New(now)

	assert.Equal(t, now, p.CreatedAt)
	assert.Equal(t, now, p.LastSeenAt)
	assert.Equal(t, pet.MaxStat, p.Hunger)
	assert.Equal(t, pet.MaxStat, p.Happiness)
	assert.Equal(t, pet.BaseWeight, p.Weight)
}

func TestStageBeforeAtAndAfterEggDuration(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p := pet.New(born)

	tests := map[string]struct {
		now  time.Time
		want pet.Stage
	}{
		"just born":         {born, pet.StageEgg},
		"just before hatch": {born.Add(pet.EggDuration - time.Nanosecond), pet.StageEgg},
		"exactly at hatch":  {born.Add(pet.EggDuration), pet.StageBaby},
		"well after hatch":  {born.Add(pet.EggDuration + time.Hour), pet.StageBaby},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, p.Stage(tt.now))
		})
	}
}

func TestAgeIsElapsedSinceCreation(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p := pet.New(born)

	assert.Equal(t, 90*time.Minute, p.Age(born.Add(90*time.Minute)))
}
