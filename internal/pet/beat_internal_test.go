package pet

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewBeatMsgCarriesTheGivenTime(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	msg, ok := newBeatMsg(now).(BeatMsg)

	assert.True(t, ok, "expected BeatMsg")
	assert.Equal(t, now, msg.Time)
}
