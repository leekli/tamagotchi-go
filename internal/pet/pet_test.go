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
		"well after growing into Child":  {born.Add(pet.EggDuration + pet.BabyDuration + time.Minute), pet.StageChild},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, p.Stage(tt.now))
		})
	}
}

func TestStageBeforeAtAndAfterChildDuration(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p := pet.New(born)

	tests := map[string]struct {
		now  time.Time
		want pet.Stage
	}{
		"just grew into Child":          {born.Add(pet.EggDuration + pet.BabyDuration), pet.StageChild},
		"just before growing into Teen": {born.Add(pet.EggDuration + pet.BabyDuration + pet.ChildDuration - time.Nanosecond), pet.StageChild},
		"exactly at growing into Teen":  {born.Add(pet.EggDuration + pet.BabyDuration + pet.ChildDuration), pet.StageTeen},
		"well after growing into Teen":  {born.Add(pet.EggDuration + pet.BabyDuration + pet.ChildDuration + time.Minute), pet.StageTeen},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, p.Stage(tt.now))
		})
	}
}

func TestStageBeforeAtAndAfterTeenDuration(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p := pet.New(born)

	tests := map[string]struct {
		now  time.Time
		want pet.Stage
	}{
		"just grew into Teen":            {born.Add(pet.EggDuration + pet.BabyDuration + pet.ChildDuration), pet.StageTeen},
		"just before growing into Adult": {born.Add(pet.EggDuration + pet.BabyDuration + pet.ChildDuration + pet.TeenDuration - time.Nanosecond), pet.StageTeen},
		"exactly at growing into Adult":  {born.Add(pet.EggDuration + pet.BabyDuration + pet.ChildDuration + pet.TeenDuration), pet.StageAdult},
		"well after growing into Adult":  {born.Add(pet.EggDuration + pet.BabyDuration + pet.ChildDuration + pet.TeenDuration + time.Minute), pet.StageAdult},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, p.Stage(tt.now))
		})
	}
}

func TestStageBeforeAtAndAfterAdultDuration(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p := pet.New(born)

	tests := map[string]struct {
		now  time.Time
		want pet.Stage
	}{
		"just grew into Adult":          {born.Add(pet.EggDuration + pet.BabyDuration + pet.ChildDuration + pet.TeenDuration), pet.StageAdult},
		"just before dying":             {born.Add(pet.EggDuration + pet.BabyDuration + pet.ChildDuration + pet.TeenDuration + pet.AdultDuration - time.Nanosecond), pet.StageAdult},
		"exactly at dying":              {born.Add(pet.EggDuration + pet.BabyDuration + pet.ChildDuration + pet.TeenDuration + pet.AdultDuration), pet.StageDeath},
		"well after dying, still Death": {born.Add(pet.EggDuration + pet.BabyDuration + pet.ChildDuration + pet.TeenDuration + pet.AdultDuration + 24*time.Hour), pet.StageDeath},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, p.Stage(tt.now))
		})
	}
}

func TestOverfedBeforeAtAndAfterThreshold(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		weight int
		want   bool
	}{
		"just below threshold": {pet.OverfedThreshold - 1, false},
		"exactly at threshold": {pet.OverfedThreshold, true},
		"well above threshold": {pet.OverfedThreshold + 10, true},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			p := pet.New(born)
			p.Weight = tt.weight

			assert.Equal(t, tt.want, p.Overfed())
		})
	}
}

func TestStageHatched(t *testing.T) {
	t.Parallel()

	assert.False(t, pet.StageEgg.Hatched())
	assert.True(t, pet.StageBaby.Hatched())
	assert.True(t, pet.StageChild.Hatched())
	assert.True(t, pet.StageTeen.Hatched())
	assert.True(t, pet.StageAdult.Hatched())
	// A Pet that has reached Death is still Hatched — it did, historically —
	// even though Care actions are no longer available; see
	// TestStageCareAvailable.
	assert.True(t, pet.StageDeath.Hatched())
}

func TestStageCareAvailable(t *testing.T) {
	t.Parallel()

	assert.False(t, pet.StageEgg.CareAvailable())
	assert.True(t, pet.StageBaby.CareAvailable())
	assert.True(t, pet.StageChild.CareAvailable())
	assert.True(t, pet.StageTeen.CareAvailable())
	assert.True(t, pet.StageAdult.CareAvailable())
	assert.False(t, pet.StageDeath.CareAvailable())
}

func TestStageString(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Egg", pet.StageEgg.String())
	assert.Equal(t, "Baby", pet.StageBaby.String())
	assert.Equal(t, "Child", pet.StageChild.String())
	assert.Equal(t, "Teen", pet.StageTeen.String())
	assert.Equal(t, "Adult", pet.StageAdult.String())
	assert.Equal(t, "Death", pet.StageDeath.String())
}

func TestAgeIsElapsedSinceCreation(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p := pet.New(born)

	assert.Equal(t, 5*time.Minute, p.Age(born.Add(5*time.Minute)))
}

func TestAgeFreezesAtDeath(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p := pet.New(born)

	deathAge := pet.EggDuration + pet.BabyDuration + pet.ChildDuration + pet.TeenDuration + pet.AdultDuration

	assert.Equal(t, deathAge, p.Age(born.Add(deathAge)), "Age at the exact moment of death should be the fixed death age")
	assert.Equal(t, deathAge, p.Age(born.Add(deathAge+7*24*time.Hour)), "Age should stay frozen long after death, not keep climbing")
}
