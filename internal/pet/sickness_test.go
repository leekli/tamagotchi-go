package pet_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/leekli/tamagotchi-go/internal/pet"
)

// These tests pin Sickness and Cure. A Pet never cleaned gets a Mess at 5:00
// (MessInterval after birth) and falls Sick SickAfterMess (5 minutes) later, at
// 10:00. Only Cure ends Sickness; Clean removes the Mess but does not cure.

func TestMessLeftForSickAfterMessPastItsOnsetMakesThePetSick(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	sickAt := born.Add(10 * time.Minute)
	require.Equal(t, 5*time.Minute, pet.SickAfterMess, "the worked timings assume a 5-minute wait after Mess appears")
	p := pet.New(born)

	t.Run("exactly at that instant, read at the Screen's clock", func(t *testing.T) {
		t.Parallel()
		assert.False(t, p.Sick(sickAt.Add(-time.Nanosecond)))
		assert.True(t, p.Sick(sickAt))
	})

	t.Run("exactly at that instant once advanced", func(t *testing.T) {
		t.Parallel()
		assert.False(t, p.Advance(sickAt.Add(-time.Nanosecond)).Sick(sickAt.Add(-time.Nanosecond)))
		assert.True(t, p.Advance(sickAt).Sick(sickAt))
	})

	t.Run("however the time is advanced the Pet fell Sick at the same instant", func(t *testing.T) {
		t.Parallel()
		oneLong := p.Advance(born.Add(time.Hour))

		walked := p
		for now := born.Add(pet.BeatInterval); !now.After(born.Add(time.Hour)); now = now.Add(pet.BeatInterval) {
			walked = walked.Advance(now)
		}

		assert.True(t, sickAt.Equal(oneLong.SickSince), "one long catch-up: fell Sick at %v", oneLong.SickSince.Sub(born))
		assert.True(t, sickAt.Equal(walked.SickSince), "20-second Beats: fell Sick at %v", walked.SickSince.Sub(born))
	})
}

func TestACleanBeforeTheOnsetCancelsAPendingSickness(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cleanedAt := born.Add(9*time.Minute + 59*time.Second) // a second before it would have fallen Sick
	p := pet.New(born).Advance(cleanedAt).Clean(cleanedAt)

	// Cleaned at 9:59, the new Mess appears at 14:59 and the Pet is Sick 5
	// minutes after that, at 19:59.
	sickAt := cleanedAt.Add(pet.MessInterval + pet.SickAfterMess)
	assert.False(t, p.Advance(born.Add(15*time.Minute)).Sick(born.Add(15*time.Minute)), "the original 10:00 was cancelled")
	assert.False(t, p.Advance(sickAt.Add(-time.Nanosecond)).Sick(sickAt.Add(-time.Nanosecond)))
	assert.True(t, p.Advance(sickAt).Sick(sickAt))
}

func TestCleanDoesNotCureSicknessButCureDoes(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	at := born.Add(11 * time.Minute)
	sick := pet.New(born).Advance(at)
	require.True(t, sick.Sick(at), "sanity check: the Pet is Sick at 11:00")

	cleaned := sick.Clean(at)
	assert.True(t, cleaned.Sick(at), "Clean removes the Mess but does not cure")
	assert.True(t, cleaned.Sick(born.Add(time.Hour)), "and it stays Sick until Cured")

	cured := sick.Cure(at)
	assert.False(t, cured.Sick(at), "Cure ends Sickness at once")
}

func TestACureWhileTheMessIsStillThereRestartsTheSicknessClock(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	curedAt := born.Add(12 * time.Minute)
	cured := pet.New(born).Advance(curedAt).Cure(curedAt) // the Mess from 5:00 is still there

	sickAgain := curedAt.Add(pet.SickAfterMess) // another full wait, from the Cure
	assert.False(t, cured.Advance(sickAgain.Add(-time.Nanosecond)).Sick(sickAgain.Add(-time.Nanosecond)))
	assert.True(t, cured.Advance(sickAgain).Sick(sickAgain))
}

func TestACureAfterACleanWaitsForTheNewMess(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cleanedAt := born.Add(11 * time.Minute)
	curedAt := born.Add(12 * time.Minute)
	p := pet.New(born).Advance(cleanedAt).Clean(cleanedAt) // Sick since 10:00, Mess now gone
	p = p.Advance(curedAt).Cure(curedAt)

	// The new Mess appears at 16:00 (5 minutes after the Clean), well after the
	// Cure, so it is that Mess's own 5 minutes that count: Sick again at 21:00.
	sickAgain := cleanedAt.Add(pet.MessInterval + pet.SickAfterMess)
	assert.False(t, p.Advance(sickAgain.Add(-time.Nanosecond)).Sick(sickAgain.Add(-time.Nanosecond)))
	assert.True(t, p.Advance(sickAgain).Sick(sickAgain))
}

func TestCureOnAHealthyPetChangesNothing(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	at := born.Add(9 * time.Minute) // Mess is pending but the Pet is not yet Sick
	healthy := pet.New(born).Advance(at)
	require.False(t, healthy.Sick(at))

	assert.Equal(t, healthy, healthy.Cure(at), "a Cure on a healthy Pet is a no-op")

	// In particular it must not delay the sickness that is already pending, or
	// pressing Cure would be a way to avoid it.
	sickAt := born.Add(10 * time.Minute)
	assert.True(t, healthy.Cure(at).Advance(sickAt).Sick(sickAt))
}

// TestCureWorksOnASickPetThatHasNotBeenAdvancedYet: the Screen reads Sickness at
// its own clock, which can be ahead of the Pet's last Advance, so Cure must act
// on a Pet that is Sick as of now even if no Advance has recorded it.
func TestCureWorksOnASickPetThatHasNotBeenAdvancedYet(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	at := born.Add(12 * time.Minute)
	notAdvanced := pet.New(born)
	require.True(t, notAdvanced.Sick(at), "sanity check: Sick as of 12:00 without an Advance")

	cured := notAdvanced.Cure(at)

	assert.False(t, cured.Sick(at))
	sickAgain := at.Add(pet.SickAfterMess)
	assert.False(t, cured.Sick(sickAgain.Add(-time.Nanosecond)))
	assert.True(t, cured.Sick(sickAgain))
}

func TestASickPetCanStillBeFedAndPlayedWithAndDecayIsUnchanged(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	at := born.Add(11 * time.Minute)
	sick := pet.New(born).Advance(at)
	require.True(t, sick.Sick(at))

	// Decay: an ignored Pet's Hunger is 1 and Happiness 0 at 11:00 (Hunger empties
	// at 12:00, Happiness at 8:30), the same timeline as before Sickness existed.
	assert.Equal(t, 1, sick.Hunger)
	assert.Equal(t, 0, sick.Happiness)

	assert.Equal(t, 2, sick.Feed(pet.Meal).Hunger, "a Sick Pet still eats")
	assert.Equal(t, 1, sick.Feed(pet.Snack).Happiness, "and takes a Snack")
	assert.Equal(t, 1, sick.Play().Happiness, "and plays")
	assert.True(t, sick.Feed(pet.Meal).Sick(at), "none of which cures it")
}
