package next_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/leekli/tamagotchi-go/internal/pet"
	"github.com/leekli/tamagotchi-go/internal/tui/next"
)

// These tests map one given/when/then group to each user-facing requirement
// of the Play and Clean Care actions, mirroring
// internal/tui/welcome/acceptance_test.go's style.

func TestAcceptance_Play(t *testing.T) {
	t.Parallel()

	t.Run("given the Play icon is activated then Happiness increases", func(t *testing.T) {
		p := pet.New(born)
		p.Happiness = 1
		s := babyScreen(t, p, &fakeStore{})

		s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})

		ns, ok := s.(*next.Screen)
		require.True(t, ok)
		assert.Equal(t, 2, ns.Pet().Happiness)
	})
}

func TestAcceptance_Clean(t *testing.T) {
	t.Parallel()

	t.Run("given the Pet has a Mess, when Clean is activated, then the Mess indicator disappears and Happiness decay returns to its normal rate", func(t *testing.T) {
		p := pet.New(born)
		p.LastCleanedAt = born.Add(-pet.MessInterval)
		s := babyScreen(t, p, &fakeStore{})
		require.Contains(t, stripANSI(s.View()), "(___)", "should start with the Mess indicator visible")

		s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})

		assert.NotContains(t, stripANSI(s.View()), "(___)")

		ns, ok := s.(*next.Screen)
		require.True(t, ok)
		assert.False(t, ns.Pet().HasMess(born.Add(pet.EggDuration)))
	})
}

func TestAcceptance_UncleanedMessAcceleratesHappinessDecay(t *testing.T) {
	t.Parallel()

	t.Run("given a Mess is present and uncleaned, when time passes, then Happiness falls faster than it would without one", func(t *testing.T) {
		elapsed := pet.HappinessDecayInterval + pet.MessHappinessDecayInterval

		clean := pet.New(born).Advance(born.Add(elapsed))

		messyStart := pet.New(born)
		messyStart.LastCleanedAt = born.Add(-pet.MessInterval)
		messy := messyStart.Advance(born.Add(elapsed))

		assert.Less(t, messy.Happiness, clean.Happiness)
	})
}
