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
)

const (
	// MaxStat is the top of the Hunger/Happiness range, matching the
	// four-pip meters of the original hardware.
	MaxStat = 4

	// BaseWeight is the Pet's starting Weight, set once at birth by New.
	// Weight is fixed for now — nothing changes it afterwards — but it
	// stays a Stat in its own right, because a later feature makes it
	// dynamic. Don't "fix" the fact that it never moves.
	BaseWeight = 2

	// EggDuration is how long the Pet stays an Egg before it Hatches.
	// Deliberately short — tens of seconds, not hours — to give the player
	// an early "it's alive" moment on first launch, mirroring the original
	// hardware's power-on hatch rather than its hours-long real-world pacing.
	EggDuration = 30 * time.Second

	// HungerDecayInterval is the wall-clock duration per one-point Hunger
	// Decay step. Deliberately on the order of single-digit minutes, not the
	// original hardware's hours: this is a CLI game played in short
	// sessions, so a player who watches the Next Screen for a while should
	// actually see a stat move. Don't "correct" this back to real-hardware
	// timing.
	HungerDecayInterval = 3 * time.Minute
	// HappinessDecayInterval is the same, for Happiness.
	HappinessDecayInterval = 3 * time.Minute
)

// Pet is the creature the player raises.
type Pet struct {
	// CreatedAt is when the Pet was first born; it drives Age and the
	// Egg→Baby Hatch.
	CreatedAt time.Time
	// LastSeenAt is the wall-clock time Decay was last applied up to; it
	// drives the offline catch-up applied on load.
	LastSeenAt time.Time
	// Hunger ranges 0 (starving) .. MaxStat (full).
	Hunger int
	// Happiness ranges 0 (unhappy) .. MaxStat (happy).
	Happiness int
	// Weight is set once at birth to BaseWeight. Fixed for now; a later
	// feature makes it dynamic.
	Weight int
}

// New returns a freshly born Pet: an Egg, full Hunger and Happiness, and
// BaseWeight, as of now.
func New(now time.Time) Pet {
	return Pet{
		CreatedAt:  now,
		LastSeenAt: now,
		Hunger:     MaxStat,
		Happiness:  MaxStat,
		Weight:     BaseWeight,
	}
}

// Stage reports the Pet's life stage as of now. It is derived from
// CreatedAt rather than stored, so it can never drift out of sync with it.
func (p Pet) Stage(now time.Time) Stage {
	if now.Sub(p.CreatedAt) >= EggDuration {
		return StageBaby
	}
	return StageEgg
}

// Age reports how long the Pet has been alive as of now.
func (p Pet) Age(now time.Time) time.Duration {
	return now.Sub(p.CreatedAt)
}
