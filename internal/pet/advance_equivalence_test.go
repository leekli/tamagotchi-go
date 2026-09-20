package pet_test

import (
	"cmp"
	"fmt"
	"math/rand/v2"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/leekli/tamagotchi-go/internal/pet"
)

// These tests pin the property Advance exists to provide: it is exact, so one
// long catch-up after a long absence gives precisely the same Pet as the same
// span advanced in many short Beats, however the Beats fall and whatever Care
// actions land in between. A whole-Pet comparison is deliberate: "identical
// results" means no stored difference at all, since any leftover difference
// would surface later as a different Decay step.
//
// Every Care action is applied the way the Next Screen applies it: Advance to
// the action's instant first, then act.
//
// Probes are kept sparse on purpose. The "one long catch-up" path only differs
// from the Beat path where a single call spans a rate change, so probes every
// few seconds would split the long path into short ones and hide a bug.

// careAction is one Care action applied at an offset from birth.
type careAction struct {
	at    time.Duration
	name  string
	apply func(p pet.Pet, now time.Time) pet.Pet
}

// careKinds are the Care actions a schedule draws from: each one can change
// Happiness's Decay rate (Snack and Play through Overfed, Clean through Mess)
// or leaves it alone (Meal).
var careKinds = []struct {
	name  string
	apply func(p pet.Pet, now time.Time) pet.Pet
}{
	{"meal", func(p pet.Pet, _ time.Time) pet.Pet { return p.Feed(pet.Meal) }},
	{"snack", func(p pet.Pet, _ time.Time) pet.Pet { return p.Feed(pet.Snack) }},
	{"play", func(p pet.Pet, _ time.Time) pet.Pet { return p.Play() }},
	{"clean", func(p pet.Pet, now time.Time) pet.Pet { return p.Clean(now) }},
}

func namedAction(name string, at time.Duration) careAction {
	for _, k := range careKinds {
		if k.name == name {
			return careAction{at: at, name: name, apply: k.apply}
		}
	}
	panic("unknown care action " + name)
}

// event is something that happens at an offset from birth: a Care action, or
// (when act is nil) a probe that records the Pet as it stands at that instant.
type event struct {
	at  time.Duration
	act *careAction
}

// simulate runs start (born at born) through the Care actions and returns the
// Pet as it stood at each probe offset. Comparing snapshots along the way,
// not just the final Pet, matters: once Hunger and Happiness have both hit 0
// a wrong path can still finish in the same place, hiding the difference.
// With beat 0 the Pet is advanced only to the actions and probes themselves
// (one long catch-up between them); otherwise each span is also walked in
// beat-sized Advance calls, finishing with a call at the exact instant.
func simulate(start pet.Pet, born time.Time, actions []careAction, probes []time.Duration, beat time.Duration) []pet.Pet {
	events := make([]event, 0, len(actions)+len(probes))
	for i := range actions {
		events = append(events, event{at: actions[i].at, act: &actions[i]})
	}
	for _, at := range probes {
		events = append(events, event{at: at})
	}
	slices.SortStableFunc(events, func(a, b event) int { return cmp.Compare(a.at, b.at) })

	p, now := start, born
	advanceTo := func(target time.Time) {
		for beat > 0 && now.Add(beat).Before(target) {
			now = now.Add(beat)
			p = p.Advance(now)
		}
		now = target
		p = p.Advance(now)
	}

	snapshots := make([]pet.Pet, 0, len(probes))
	for _, e := range events {
		advanceTo(born.Add(e.at))
		if e.act != nil {
			p = e.act.apply(p, now)
			continue
		}
		snapshots = append(snapshots, p)
	}
	return snapshots
}

func describe(actions []careAction) string {
	parts := make([]string, len(actions))
	for i, a := range actions {
		parts[i] = fmt.Sprintf("%s@%v", a.name, a.at)
	}
	return fmt.Sprint(parts)
}

// beatSizes mixes sizes that divide Decay intervals evenly with ones that land
// on no boundary at all, so an off-by-a-fraction bug cannot hide.
var beatSizes = []time.Duration{
	time.Second,
	13*time.Second + 7*time.Millisecond,
	pet.BeatInterval,
	61 * time.Second,
	5 * time.Minute,
}

func TestAdvanceScriptedTimelinesMatchOneLongCatchUp(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	messyAtBirth := func(p pet.Pet) pet.Pet {
		p.LastCleanedAt = born.Add(-pet.MessInterval)
		return p
	}
	overfedAtBirth := func(p pet.Pet) pet.Pet {
		p.Weight = pet.OverfedThreshold
		return p
	}
	unchanged := func(p pet.Pet) pet.Pet { return p }

	tests := map[string]struct {
		prepare func(pet.Pet) pet.Pet
		actions []careAction
		probes  []time.Duration
	}{
		"an ignored Pet, Mess appearing partway through": {
			prepare: unchanged, probes: []time.Duration{6*time.Minute + 30*time.Second, 9 * time.Minute, 15 * time.Minute},
		},
		"cleaned just before Mess would appear": {
			prepare: unchanged, probes: []time.Duration{7*time.Minute + 30*time.Second, 12 * time.Minute, 15 * time.Minute},
			actions: []careAction{namedAction("clean", 4*time.Minute+59*time.Second)},
		},
		"cleaned while already messy": {
			prepare: messyAtBirth, probes: []time.Duration{90 * time.Second, 4 * time.Minute, 10 * time.Minute},
			actions: []careAction{namedAction("clean", time.Minute)},
		},
		"Snacks into Overfed, then Play out of it": {
			prepare: unchanged, probes: []time.Duration{150 * time.Second, 8 * time.Minute, 12 * time.Minute, 20 * time.Minute},
			actions: []careAction{
				namedAction("snack", time.Minute), namedAction("snack", 2*time.Minute),
				namedAction("snack", 3*time.Minute), namedAction("play", 9*time.Minute),
				namedAction("play", 10*time.Minute), namedAction("play", 11*time.Minute),
			},
		},
		"Overfed and messy together, Play then Clean": {
			prepare: func(p pet.Pet) pet.Pet { return overfedAtBirth(messyAtBirth(p)) },
			actions: []careAction{namedAction("play", 2*time.Minute), namedAction("clean", 4*time.Minute)},
			probes:  []time.Duration{3 * time.Minute, 5 * time.Minute, 12 * time.Minute},
		},
		"a long absence": {
			prepare: unchanged, probes: []time.Duration{20 * time.Minute, 6 * time.Hour},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			start := tt.prepare(pet.New(born))
			oneLong := simulate(start, born, tt.actions, tt.probes, 0)

			for _, beat := range beatSizes {
				assert.Equal(t, oneLong, simulate(start, born, tt.actions, tt.probes, beat),
					"advancing in %v Beats should match one long catch-up", beat)
			}
		})
	}
}

// TestAdvanceBeatsMatchOneLongCatchUpAtEveryBeat walks an ignored Pet forward
// one real Beat at a time and checks, after each, that it equals a fresh Pet
// caught up in a single call to that same instant.
func TestAdvanceBeatsMatchOneLongCatchUpAtEveryBeat(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p := pet.New(born)

	for k := 1; k <= 45; k++ { // 15 minutes of Beats
		now := born.Add(time.Duration(k) * pet.BeatInterval)
		p = p.Advance(now)
		require.Equal(t, pet.New(born).Advance(now), p, "after Beat %d (%v)", k, now.Sub(born))
	}
}

// scenario is one randomly generated Pet, Care schedule and set of probe
// instants. lowStats starts Hunger and Happiness anywhere from 1 to full (so
// Empty spells begin and lapse within the run), rather than at full or one
// short of it; horizonSpan bounds how long the run lasts, above a 5-minute floor.
type scenario struct {
	start   pet.Pet
	actions []careAction
	probes  []time.Duration
}

func newScenario(seed uint64, born time.Time, lowStats bool, horizonSpan time.Duration) scenario {
	// A fixed-seed PRNG, so a failing schedule reproduces exactly.
	r := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))

	horizon := 5*time.Minute + time.Duration(r.Int64N(int64(horizonSpan)))
	actions := make([]careAction, r.IntN(13))
	for i := range actions {
		kind := careKinds[r.IntN(len(careKinds))]
		actions[i] = careAction{
			at:    time.Duration(r.Int64N(int64(horizon))),
			name:  kind.name,
			apply: kind.apply,
		}
	}
	slices.SortStableFunc(actions, func(a, b careAction) int { return cmp.Compare(a.at, b.at) })

	probes := make([]time.Duration, 0, 4)
	for range 3 {
		probes = append(probes, time.Duration(r.Int64N(int64(horizon))))
	}
	probes = append(probes, horizon)

	start := pet.New(born)
	if lowStats {
		start.Hunger = 1 + r.IntN(pet.MaxStat)
		start.Happiness = 1 + r.IntN(pet.MaxStat)
	} else {
		start.Hunger = pet.MaxStat - r.IntN(2)
		start.Happiness = pet.MaxStat - r.IntN(2)
	}
	start.Weight = pet.BaseWeight + r.IntN(pet.MaxWeight-pet.BaseWeight+1)
	// Last cleaned within the previous 4 minutes, so Mess usually appears
	// partway through the run rather than before or long after it.
	start.LastCleanedAt = born.Add(-time.Duration(r.Int64N(int64(pet.MessInterval - time.Minute))))
	start.LastSeenAt = born.Add(-time.Duration(r.Int64N(int64(pet.HungerDecayInterval))))
	start.HappinessProgress = time.Duration(r.Int64N(int64(pet.HappinessDecayInterval)))

	return scenario{start: start, actions: actions, probes: probes}
}

// TestAdvanceIsExactUnderRandomCareSchedules generalises the scripted cases
// over many seeded random Pets and Care schedules. The seeds are fixed, so a
// failure reproduces exactly; the failure message names the schedule.
func TestAdvanceIsExactUnderRandomCareSchedules(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	for seed := uint64(1); seed <= 300; seed++ {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()

			sc := newScenario(seed, born, false, 25*time.Minute)
			r := rand.New(rand.NewPCG(seed, seed))
			horizon := sc.probes[len(sc.probes)-1]

			oneLong := simulate(sc.start, born, sc.actions, sc.probes, 0)

			for range 2 {
				beat := beatSizes[r.IntN(len(beatSizes))]
				assert.Equal(t, oneLong, simulate(sc.start, born, sc.actions, sc.probes, beat),
					"beat %v, schedule %s, probes %v, horizon %v", beat, describe(sc.actions), sc.probes, horizon)
			}
		})
	}
}

// TestAdvanceCareMistakesAreExactUnderRandomSchedules is the same property for
// the Care mistake tally and Empty spells, with Stats that start low and runs
// long enough for spells to begin, lapse, be cut short by Care actions and
// begin again. A whole-Pet comparison covers the tally, both spells and the
// Attention state at once. The run also counts the mistakes it saw, so the test
// cannot pass vacuously on schedules that never earn one.
func TestAdvanceCareMistakesAreExactUnderRandomSchedules(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var withMistakes, mistakes atomic.Int64

	t.Run("seeds", func(t *testing.T) {
		for seed := uint64(1); seed <= 300; seed++ {
			t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
				t.Parallel()

				sc := newScenario(seed, born, true, 45*time.Minute)
				r := rand.New(rand.NewPCG(seed, seed+1))

				oneLong := simulate(sc.start, born, sc.actions, sc.probes, 0)
				final := oneLong[len(oneLong)-1]
				if final.CareMistakes > 0 {
					withMistakes.Add(1)
					mistakes.Add(int64(final.CareMistakes))
				}

				for range 2 {
					beat := beatSizes[r.IntN(len(beatSizes))]
					assert.Equal(t, oneLong, simulate(sc.start, born, sc.actions, sc.probes, beat),
						"beat %v, schedule %s, probes %v", beat, describe(sc.actions), sc.probes)
				}
			})
		}
	})

	// The parallel seeds above have all finished by here.
	t.Logf("%d of 300 schedules earned a Care mistake, %d in all", withMistakes.Load(), mistakes.Load())
	assert.GreaterOrEqual(t, withMistakes.Load(), int64(150),
		"at least half of the schedules should earn a Care mistake, or the property is barely exercised")
	assert.GreaterOrEqual(t, mistakes.Load(), int64(300), "and between them a good number of mistakes")
}

// TestHappinessAccelerationDividesTheDecayInterval guards the arithmetic the
// exactness above depends on: progress toward a Happiness point accrues at a
// whole-number multiple of real time, so it never needs rounding.
func TestHappinessAccelerationDividesTheDecayInterval(t *testing.T) {
	t.Parallel()

	assert.Zero(t, pet.HappinessDecayInterval%pet.AcceleratedHappinessDecayInterval,
		"AcceleratedHappinessDecayInterval must divide HappinessDecayInterval exactly")
}
