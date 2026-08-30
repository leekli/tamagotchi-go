package tui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/leekli/tamagotchi-go/internal/tui"
)

func TestNavigateCommandEmitsNavigateMsg(t *testing.T) {
	t.Parallel()

	cmd := tui.Navigate(tui.NextScreenID)
	require.NotNil(t, cmd)

	msg := cmd()
	nav, ok := msg.(tui.NavigateMsg)
	require.True(t, ok, "expected a NavigateMsg, got %T", msg)
	assert.Equal(t, tui.NextScreenID, nav.To)
}

func TestAllScreenIDsAreUnique(t *testing.T) {
	t.Parallel()

	seen := map[tui.ScreenID]bool{}
	for _, id := range tui.AllScreenIDs() {
		assert.False(t, seen[id], "duplicate ScreenID %q", id)
		seen[id] = true
	}
	assert.NotEmpty(t, tui.AllScreenIDs())
}
