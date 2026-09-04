package pet_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/leekli/tamagotchi-go/internal/pet"
)

func TestBeatIntervalIsMuchSlowerThanTheAnimationClock(t *testing.T) {
	t.Parallel()
	assert.Greater(t, pet.BeatInterval, time.Second)
}

func TestBeatReturnsACommand(t *testing.T) {
	t.Parallel()

	// Deliberately not invoked here: BeatInterval is tens of seconds, and
	// calling the command would block the test for that long. Its message
	// shape is exercised by feeding pet.BeatMsg directly into a Screen's
	// Update, the same way anim.TickMsg is fed in rather than waited on.
	cmd := pet.Beat()
	require.NotNil(t, cmd)
}
