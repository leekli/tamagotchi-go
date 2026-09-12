// Package pet is the game's pure domain model: the creature the player
// raises. Its core type performs no direct file or network I/O, and time
// only ever enters as an explicit, injected "now" so behaviour is
// deterministic under test — the same discipline internal/anim uses for
// animation frames.
package pet

import "time"

// Stage is the Pet's life stage.
type Stage int

const (
	// StageEgg is the Pet's stage from birth until EggDuration has elapsed.
	StageEgg Stage = iota
	// StageBaby is the Pet's stage once it has Hatched.
	StageBaby
	// StageChild is the Pet's stage once it has spent BabyDuration as a Baby.
	StageChild
	// StageTeen is the Pet's stage once it has spent ChildDuration as a Child.
	StageTeen
)

const (
	// MaxStat is the top of the Hunger/Happiness range, matching the
	// four-pip meters of the original hardware.
	MaxStat = 4

	// BaseWeight is the Pet's starting Weight, set once at birth by New.
	// Snack and Play move it up and down from there; BaseWeight also floors
	// how low Play can ever bring it. Unlike Hunger and Happiness, Weight
	// never Decays over elapsed time — it only ever changes as the direct
	// result of a Care action.
	BaseWeight = 2

	// OverfedThreshold is the Weight at or above which a Pet is Overfed,
	// accelerating Happiness decay the same way a Mess does. Set three
	// Snacks' worth above BaseWeight — enough headroom that the first Snack
	// or two isn't punished, mirroring the grace window MessInterval already
	// gives before a Mess appears.
	OverfedThreshold = BaseWeight + 3

	// MaxWeight is the highest Weight can ever reach — Snack simply stops
	// increasing it there, the same way Feed(Meal) already stops increasing
	// Hunger at MaxStat. Comfortably above OverfedThreshold so Overfed still
	// means something well before the hard ceiling, and small enough that
	// Weight's on-screen display never needs more than two digits.
	MaxWeight = OverfedThreshold + 7

	// EggDuration is how long the Pet stays an Egg before it Hatches.
	// Deliberately short — tens of seconds, not hours — to give the player
	// an early "it's alive" moment on first launch, mirroring the original
	// hardware's power-on hatch rather than its hours-long real-world pacing.
	EggDuration = 30 * time.Second

	// BabyDuration is how long the Pet stays a Baby before it grows into a
	// Child, once it has Hatched. Chosen to be a few minutes — long enough to
	// feel like a genuine second milestone beyond the initial Hatch, short
	// enough that a patient single session still reaches it. Don't "correct"
	// this back to real-hardware timing.
	BabyDuration = 10 * time.Minute

	// ChildDuration is how long the Pet stays a Child before it grows into a
	// Teen, once it has grown into one. Continues BabyDuration's escalating
	// pacing: a third milestone that's still reachable within a patient
	// single session. Don't "correct" this back to real-hardware timing.
	ChildDuration = 15 * time.Minute

	// HungerDecayInterval is the wall-clock duration per one-point Hunger
	// Decay step. Deliberately on the order of single-digit minutes, not the
	// original hardware's hours: this is a CLI game played in short
	// sessions, so a player who watches the Next Screen for a while should
	// actually see a stat move. Don't "correct" this back to real-hardware
	// timing.
	HungerDecayInterval = 3 * time.Minute
	// HappinessDecayInterval is the same, for Happiness.
	HappinessDecayInterval = 3 * time.Minute

	// MessInterval is how long the Pet can go without being Cleaned before it
	// HasMess. Chosen to be visible within a single play session: long enough
	// that a freshly-hatched Baby isn't immediately messy, short enough that a
	// player who ignores it for a while sees the consequence.
	MessInterval = 5 * time.Minute
	// AcceleratedHappinessDecayInterval is the (shorter) HappinessDecayInterval
	// applied while the Pet HasMess or is Overfed — Happiness Decays twice as
	// fast under either cause, so neglecting either has a real, visible cost.
	// The two causes share one rate rather than each defining their own: when
	// both apply at once, Happiness still decays at this single rate, not an
	// even faster combined one.
	AcceleratedHappinessDecayInterval = HappinessDecayInterval / 2
)

// Pet is the creature the player raises.
type Pet struct {
	// CreatedAt is when the Pet was first born; it drives Age and the
	// Egg→Baby Hatch.
	CreatedAt time.Time
	// LastSeenAt is the wall-clock time Hunger Decay was last applied up to;
	// it drives the offline catch-up applied on load.
	LastSeenAt time.Time
	// HappinessLastSeenAt is the same, for Happiness specifically. Tracked
	// separately from LastSeenAt because Happiness's Decay interval changes
	// (faster) while the Pet HasMess: a single shared anchor can't correctly
	// serve two stats that step at different rates — see docs/adr/0006's
	// update note.
	HappinessLastSeenAt time.Time
	// LastCleanedAt is the wall-clock time the Pet was last Cleaned. It
	// defaults to CreatedAt for a freshly-hatched Pet, and — via a save-file
	// load default — for any pre-Mess save file too, both of which correctly
	// read as "never cleaned yet". HasMess derives from this rather than
	// storing a separate flag, so it can never drift out of sync with it.
	LastCleanedAt time.Time
	// Hunger ranges 0 (starving) .. MaxStat (full).
	Hunger int
	// Happiness ranges 0 (unhappy) .. MaxStat (happy).
	Happiness int
	// Weight starts at BaseWeight and moves only via Care actions (Snack
	// increases it, Play decreases it) between a floor of BaseWeight and a
	// ceiling of MaxWeight — never from elapsed time.
	Weight int
}

// New returns a freshly born Pet: an Egg, full Hunger and Happiness, and
// BaseWeight, as of now.
func New(now time.Time) Pet {
	return Pet{
		CreatedAt:           now,
		LastSeenAt:          now,
		HappinessLastSeenAt: now,
		LastCleanedAt:       now,
		Hunger:              MaxStat,
		Happiness:           MaxStat,
		Weight:              BaseWeight,
	}
}

// Stage reports the Pet's life stage as of now. It is derived from
// CreatedAt rather than stored, so it can never drift out of sync with it.
// Cases must stay ordered from the largest cumulative duration to the
// smallest: a future Stage added below Teen's case, rather than above it,
// would never be reached, since Teen's condition would already have matched.
func (p Pet) Stage(now time.Time) Stage {
	age := now.Sub(p.CreatedAt)
	switch {
	case age >= EggDuration+BabyDuration+ChildDuration:
		return StageTeen
	case age >= EggDuration+BabyDuration:
		return StageChild
	case age >= EggDuration:
		return StageBaby
	default:
		return StageEgg
	}
}

// Hatched reports whether s is any Stage other than Egg. It is the single
// condition governing whether the Next Screen's icon bar and Care actions
// are available, so a future additional Stage doesn't require re-auditing
// every comparison against a specific Stage value.
func (s Stage) Hatched() bool { return s != StageEgg }

// String returns Stage's display name, used by the Next Screen's on-screen
// Stage label.
func (s Stage) String() string {
	switch s {
	case StageBaby:
		return "Baby"
	case StageChild:
		return "Child"
	case StageTeen:
		return "Teen"
	default:
		return "Egg"
	}
}

// Age reports how long the Pet has been alive as of now.
func (p Pet) Age(now time.Time) time.Duration {
	return now.Sub(p.CreatedAt)
}

// HasMess reports whether the Pet currently has a Mess: uncleaned for at
// least MessInterval since it was last Cleaned. Derived from LastCleanedAt
// rather than stored as a separate mutated flag, the same reasoning Stage
// already applies to CreatedAt — see docs/adr/0006's update note.
func (p Pet) HasMess(now time.Time) bool {
	return now.Sub(p.LastCleanedAt) >= MessInterval
}

// Overfed reports whether the Pet's Weight has reached OverfedThreshold.
// Derived purely from the already-stored Weight value — the same
// derive-don't-store reasoning HasMess and Stage already apply — so there is
// nothing here that could drift out of sync, since Overfed carries no state
// of its own. Unlike HasMess, it takes no now: Weight isn't time-driven, so
// there's nothing to derive it against.
func (p Pet) Overfed() bool {
	return p.Weight >= OverfedThreshold
}

// withLoadDefaults normalises a Pet freshly unmarshalled from a save file
// that predates a field, into the value that reads correctly for "this never
// happened yet". A zero LastCleanedAt (absent from any save file written
// before Mess existed) means "never cleaned since birth", which is
// CreatedAt, not the zero time.Time — kept here, in the domain package,
// rather than in FileStore, so the invariant holds regardless of where a Pet
// is loaded from.
func withLoadDefaults(p Pet) Pet {
	if p.LastCleanedAt.IsZero() {
		p.LastCleanedAt = p.CreatedAt
	}
	if p.HappinessLastSeenAt.IsZero() {
		// Every save file written before Mess existed always advanced Hunger
		// and Happiness in lockstep, so LastSeenAt is exactly the point
		// Happiness Decay was applied up to as well.
		p.HappinessLastSeenAt = p.LastSeenAt
	}
	// A save file written before MaxWeight existed (when Snack's Weight gain
	// was uncapped) can carry a Weight outside today's valid range. Clamp it
	// here, at load, rather than leaving it out of range until the next Feed
	// or Play happens to correct it as a side effect of its own min/max.
	p.Weight = min(max(p.Weight, BaseWeight), MaxWeight)
	return p
}
