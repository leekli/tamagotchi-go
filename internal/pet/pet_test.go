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
	assert.Equal(t, now, p.HappinessLastSeenAt)
	assert.Equal(t, now, p.LastCleanedAt)
	assert.Equal(t, pet.MaxStat, p.Hunger)
	assert.Equal(t, pet.MaxStat, p.Happiness)
	assert.Equal(t, pet.BaseWeight, p.Weight)
}

func TestHasMessBeforeAtAndAfterMessInterval(t *testing.T) {
	t.Parallel()

	cleanedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p := pet.New(cleanedAt)

	tests := map[string]struct {
		now  time.Time
		want bool
	}{
		"just cleaned":           {cleanedAt, false},
		"just before Mess":       {cleanedAt.Add(pet.MessInterval - time.Nanosecond), false},
		"exactly at Mess":        {cleanedAt.Add(pet.MessInterval), true},
		"well past MessInterval": {cleanedAt.Add(pet.MessInterval + time.Hour), true},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, p.HasMess(tt.now))
		})
	}
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
		"well after hatch":  {born.Add(pet.EggDuration + time.Minute), pet.StageBaby},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, p.Stage(tt.now))
		})
	}
}

func TestStageBeforeAtAndAfterBabyDuration(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p := pet.New(born)

	tests := map[string]struct {
		now  time.Time
		want pet.Stage
	}{
		"just hatched into Baby":         {born.Add(pet.EggDuration), pet.StageBaby},
		"just before growing into Child": {born.Add(pet.EggDuration + pet.BabyDuration - time.Nanosecond), pet.StageBaby},
		"exactly at growing into Child":  {born.Add(pet.EggDuration + pet.BabyDuration), pet.StageChild},
		"well after growing into Child":  {born.Add(pet.EggDuration + pet.BabyDuration + time.Hour), pet.StageChild},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, p.Stage(tt.now))
		})
	}
}

func TestStageHatched(t *testing.T) {
	t.Parallel()

	assert.False(t, pet.StageEgg.Hatched())
	assert.True(t, pet.StageBaby.Hatched())
	assert.True(t, pet.StageChild.Hatched())
}

func TestStageString(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Egg", pet.StageEgg.String())
	assert.Equal(t, "Baby", pet.StageBaby.String())
	assert.Equal(t, "Child", pet.StageChild.String())
}

func TestAgeIsElapsedSinceCreation(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p := pet.New(born)

	assert.Equal(t, 90*time.Minute, p.Age(born.Add(90*time.Minute)))
}
