package next_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/leekli/tamagotchi-go/internal/pet"
	"github.com/leekli/tamagotchi-go/internal/tui"
)

// These tests pin the Attention call's cue in the STATS panel: it names each
// Empty Stat (Hungry to be Fed, Sad to be Played with), wears a different phase
// once a Stat's grace window has lapsed, and is absent while the Pet is Sick.
// The Screens are all hatched, so its clock stands at hatchedAt.

var hatchedAt = born.Add(pet.EggDuration)

// runningSpell is an Empty spell whose grace window still has three minutes to
// run at hatchedAt; lapsedSpell is one whose window expired and was counted.
func runningSpell() pet.EmptySpell {
	return pet.EmptySpell{Since: hatchedAt.Add(-time.Minute), GraceEndsAt: hatchedAt.Add(pet.GraceWindow - time.Minute)}
}

func lapsedSpell() pet.EmptySpell {
	return pet.EmptySpell{Since: hatchedAt.Add(-pet.GraceWindow - time.Minute)}
}

// emptyPet is a clean Pet, current as of hatchedAt, with the given Stats Empty
// in the given spells (nil leaves that Stat full).
func emptyPet(hunger, happiness *pet.EmptySpell) pet.Pet {
	p := caredForUntil(hatchedAt)
	if hunger != nil {
		p.Hunger, p.HungerEmpty = 0, *hunger
	}
	if happiness != nil {
		p.Happiness, p.HappinessEmpty = 0, *happiness
	}
	return p
}

// attentionRow returns the STATS panel row the cue occupies: the line, cut down
// to the panel's contents, that carries the "(!" glyph, or "" if none does.
func attentionRow(view string) string {
	for _, line := range strings.Split(visibleText(view), "\n") {
		if i := strings.Index(line, "(!"); i >= 0 {
			return strings.TrimSpace(strings.Trim(strings.TrimSpace(line[i:]), "│"))
		}
	}
	return ""
}

func TestTheAttentionCallNamesTheEmptyStatsAndTheirPhase(t *testing.T) {
	t.Parallel()

	running, lapsed := runningSpell(), lapsedSpell()
	tests := map[string]struct {
		pet  pet.Pet
		want string
	}{
		"Hunger, window running":                 {emptyPet(&running, nil), "(!) Hungry"},
		"Happiness, window running":              {emptyPet(nil, &running), "(!) Sad"},
		"both, windows running":                  {emptyPet(&running, &running), "(!) Hungry & Sad"},
		"Hunger, window lapsed":                  {emptyPet(&lapsed, nil), "(!!) HUNGRY"},
		"Happiness, window lapsed":               {emptyPet(nil, &lapsed), "(!!) SAD"},
		"both, windows lapsed":                   {emptyPet(&lapsed, &lapsed), "(!!) HUNGRY & SAD"},
		"Hunger lapsed, Happiness still running": {emptyPet(&lapsed, &running), "(!!) HUNGRY & Sad"},
		"Happiness lapsed, Hunger still running": {emptyPet(&running, &lapsed), "(!!) Hungry & SAD"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, attentionRow(babyScreen(t, tc.pet, &fakeStore{}).View()))
		})
	}
}

func TestNoAttentionCallWhileNoStatIsEmpty(t *testing.T) {
	t.Parallel()

	assert.Empty(t, attentionRow(babyScreen(t, emptyPet(nil, nil), &fakeStore{}).View()))
	assert.Empty(t, attentionRow(babyScreen(t, pet.New(born), &fakeStore{}).View()))
}

// TestTheAttentionCallFollowsTheScreensClockFromRunningToLapsed: the wording
// changes at the very instant the grace window expires, and stays that way.
func TestTheAttentionCallFollowsTheScreensClockFromRunningToLapsed(t *testing.T) {
	t.Parallel()

	spell := runningSpell()
	s := babyScreen(t, emptyPet(&spell, nil), &fakeStore{})
	require.Equal(t, "(!) Hungry", attentionRow(s.View()))

	s = advanceAnim(t, s, spell.GraceEndsAt.Add(-time.Nanosecond), 1)
	assert.Equal(t, "(!) Hungry", attentionRow(s.View()), "a nanosecond before the deadline")

	s = advanceAnim(t, s, spell.GraceEndsAt, 1)
	assert.Equal(t, "(!!) HUNGRY", attentionRow(s.View()), "at the deadline")

	s = advanceAnim(t, s, spell.GraceEndsAt.Add(5*time.Minute), 1) // short of starvation, 10 minutes into the spell
	assert.Equal(t, "(!!) HUNGRY", attentionRow(s.View()), "and it stays until the Stat is refilled")
}

// TestTheAttentionCallClearsWhenTheStatIsRefilled, in either phase, by either
// Care action.
func TestTheAttentionCallClearsWhenTheStatIsRefilled(t *testing.T) {
	t.Parallel()

	running, lapsed := runningSpell(), lapsedSpell()
	for name, spell := range map[string]pet.EmptySpell{"window running": running, "window lapsed": lapsed} {
		t.Run("Hunger, "+name, func(t *testing.T) {
			t.Parallel()
			s := babyScreen(t, emptyPet(&spell, nil), &fakeStore{})
			require.NotEmpty(t, attentionRow(s.View()), "sanity check")

			assert.Empty(t, attentionRow(typeKeys(s, 'f', 'm').View()))
		})
		t.Run("Happiness, "+name, func(t *testing.T) {
			t.Parallel()
			s := babyScreen(t, emptyPet(nil, &spell), &fakeStore{})
			require.NotEmpty(t, attentionRow(s.View()), "sanity check")

			assert.Empty(t, attentionRow(typeKeys(s, 'p').View()))
		})
	}

	t.Run("refilling one Stat leaves the other's call", func(t *testing.T) {
		t.Parallel()
		// Update changes a Screen in place, so each Care action gets its own.
		played := typeKeys(babyScreen(t, emptyPet(&running, &lapsed), &fakeStore{}), 'p')
		fed := typeKeys(babyScreen(t, emptyPet(&running, &lapsed), &fakeStore{}), 'f', 'm')

		assert.Equal(t, "(!) Hungry", attentionRow(played.View()), "Happiness played with, Hunger still calls")
		assert.Equal(t, "(!!) SAD", attentionRow(fed.View()), "Hunger fed, Happiness still calls")
	})
}

// sickAndEmpty is a Pet that is Sick and has Hunger Empty, its window running.
func sickAndEmpty() pet.Pet {
	p := sickFromBirth()
	p.Hunger = 0
	p.HungerEmpty = pet.EmptySpell{Since: born.Add(10 * time.Second), GraceEndsAt: born.Add(10*time.Second + pet.GraceWindow)}
	return p
}

// TestTheAttentionCallIsAbsentWhileSickAndReturnsAfterACure, by keyboard and by
// mouse. Deliberately not t.Parallel(): the mouse variants share the global
// bubblezone manager, like the package's other mouse tests.
func TestTheAttentionCallIsAbsentWhileSickAndReturnsAfterACure(t *testing.T) {
	for name, in := range careInputs {
		t.Run(name, func(t *testing.T) {
			s := babyScreen(t, sickAndEmpty(), &fakeStore{})
			require.True(t, currentPet(t, s).Sick(hatchedAt), "sanity check: Sick")
			require.Zero(t, currentPet(t, s).Hunger, "sanity check: and Hunger is Empty")
			require.Contains(t, visibleText(s.View()), "Sick", "sanity check: and says so")

			assert.Empty(t, attentionRow(s.View()), "a Sick Pet does not call for attention")

			s = in.cure(t, s)

			assert.Equal(t, "(!) Hungry", attentionRow(s.View()), "Cured, with Hunger still Empty, it calls again")
		})
	}
}

// TestTheAttentionCallFillsItsReservedRowAndNothingMoves: the cue sits in the
// blank row the STATS panel holds for it, two rows above the first Meter, and
// every state stays inside the same 43x17 envelope.
func TestTheAttentionCallFillsItsReservedRowAndNothingMoves(t *testing.T) {
	t.Parallel()

	running, lapsed := runningSpell(), lapsedSpell()
	healthy := babyScreen(t, emptyPet(nil, nil), &fakeStore{})
	healthyFirst, healthyLast := occupiedRows(t, healthy.View())
	rowOf := func(view, needle string) int {
		for i, line := range strings.Split(visibleText(view), "\n") {
			if strings.Contains(line, needle) {
				return i
			}
		}
		t.Fatalf("no row contains %q in:\n%s", needle, visibleText(view))
		return -1
	}

	states := map[string]pet.Pet{
		"Hunger running": emptyPet(&running, nil),
		"both running":   emptyPet(&running, &running),
		"both lapsed":    emptyPet(&lapsed, &lapsed),
		"mixed":          emptyPet(&lapsed, &running),
	}
	for name, p := range states {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			s := babyScreen(t, p, &fakeStore{})
			view := s.View()

			first, last := occupiedRows(t, view)
			left, right := occupiedColumns(view)
			assert.Equal(t, healthyFirst, first)
			assert.Equal(t, healthyLast, last)
			assert.Equal(t, envelopeWidth, right-left, "the cue fits inside the STATS panel")
			assert.Equal(t, rowOf(view, "Hunger")-2, rowOf(view, "(!"), "the cue is in the reserved row above the Meters")
			assert.Equal(t, len(strings.Split(visibleText(healthy.View()), "\n")), len(strings.Split(visibleText(view), "\n")), "and no row is added")
		})
	}
}

// TestTheAttentionCallIsPlainASCIIAndNeverRingsTheBell: copy is ASCII-safe, and
// nothing in a frame carries a bell.
func TestTheAttentionCallIsPlainASCIIAndNeverRingsTheBell(t *testing.T) {
	t.Parallel()

	lapsed := lapsedSpell()
	view := babyScreen(t, emptyPet(&lapsed, &lapsed), &fakeStore{}).View()

	require.NotEmpty(t, attentionRow(view), "sanity check")
	for _, r := range attentionRow(view) {
		assert.Less(t, r, rune(128), "%q is not ASCII", r)
	}
	assert.NotContains(t, view, "\a")
}

// TestTheDeathPanelShowsNoAttentionCall: a dead Pet calls for nothing.
func TestTheDeathPanelShowsNoAttentionCall(t *testing.T) {
	t.Parallel()

	s := deathScreen(t, starvedPet(), &fakeStore{})
	require.Equal(t, tui.NextScreenID, s.ID())

	assert.Empty(t, attentionRow(s.View()))
	assert.NotContains(t, strings.ToLower(visibleText(s.View())), "hungry")
}
