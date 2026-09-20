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
	// StageAdult is the Pet's stage once it has spent TeenDuration as a
	// Teen. The Pet stops growing further once it reaches it.
	StageAdult
	// StageDeath is the Pet's stage once it has spent AdultDuration as an
	// Adult. It is the true last Stage: there is no Stage after it, and no
	// duration governing how long it lasts.
	StageDeath
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

	// TeenDuration is how long the Pet stays a Teen before it grows into an
	// Adult, once it has grown into one. Continues ChildDuration's
	// escalating pacing: the last milestone, still reachable within a
	// patient single session. Don't "correct" this back to real-hardware
	// timing.
	TeenDuration = 15 * time.Minute

	// AdultDuration is how long the Pet stays an Adult before it dies, once
	// it has grown into one. Deliberately longer than the growth-stage
	// durations before it, since Adult is meant to be lived in for a while,
	// not rushed through — but still a fixed, short-session-friendly number,
	// not the original hardware's much longer real-world lifespans. Don't
	// "correct" this back to real-hardware timing.
	AdultDuration = 20 * time.Minute

	// HungerDecayInterval is the wall-clock duration per one-point Hunger
	// Decay step. Deliberately on the order of single-digit minutes, not the
	// original hardware's hours: this is a CLI game played in short
	// sessions, so a player who watches the Next Screen for a while should
	// actually see a stat move. Don't "correct" this back to real-hardware
	// timing.
	HungerDecayInterval = 3 * time.Minute
	// HappinessDecayInterval is the same, for Happiness.
	HappinessDecayInterval = 3 * time.Minute

	// GraceWindow is how long a Stat may stay Empty (at 0) before that Empty
	// spell costs a Care mistake. Deliberately a little over one Decay step (3
	// minutes), so a single late refill is not punished, and far shorter than the
	// original hardware's 15 minutes, since the whole life here is about an hour.
	// Don't "correct" this back to real-hardware timing.
	GraceWindow = 4 * time.Minute

	// MessInterval is how long the Pet can go without being Cleaned before it
	// HasMess. Chosen to be visible within a single play session: long enough
	// that a freshly-hatched Baby isn't immediately messy, short enough that a
	// player who ignores it for a while sees the consequence.
	MessInterval = 5 * time.Minute
	// SickAfterMess is how long a Mess must be left, after it appears, before the
	// Pet falls Sick: 5 minutes, so a Pet never cleaned is Sick at 10:00. Long
	// enough that a player who cleans up within a few minutes of the Mess
	// appearing never sees it. Don't "correct" this back to real-hardware timing.
	SickAfterMess = 5 * time.Minute

	// StarvationInterval is how long Hunger may stay Empty before the Pet dies of
	// Starvation, counted from the start of the Empty spell. It is deliberately
	// longer than GraceWindow, so a Care mistake is counted well before it comes to
	// this, and Sickness does not pause it, unlike the grace clock. A Pet never
	// fed or cleaned is Sick long before, but it is Starvation that gets it first.
	// Don't "correct" this back to real-hardware timing.
	StarvationInterval = 10 * time.Minute

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
	// HappinessLastSeenAt is the wall-clock time Happiness Decay has been
	// applied up to. Unlike LastSeenAt it moves all the way to the time of
	// each Advance, because Happiness's Decay rate changes (faster while the
	// Pet HasMess or is Overfed): the progress already made toward the next
	// point is kept in HappinessProgress instead, so it survives a rate
	// change exactly — see docs/adr/0006's update notes.
	HappinessLastSeenAt time.Time
	// HappinessProgress is how far Happiness has got toward its next Decay
	// point as of HappinessLastSeenAt, measured in normal-rate time: a point
	// is lost each time it reaches HappinessDecayInterval, and it fills twice
	// as fast while the accelerated rate applies. It is always less than
	// HappinessDecayInterval after an Advance. Zero on any save file written
	// before it existed, which reads as "no partial progress yet".
	HappinessProgress time.Duration
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

	// CareMistakes is the lifetime tally of Care mistakes: one for each Empty
	// spell, per Stat, whose GraceWindow has expired. It is kept from birth and
	// only ever grows. Nothing shows it to the player yet.
	CareMistakes int
	// HungerEmpty and HappinessEmpty record each Stat's current Empty spell. They
	// are stored, not derived, because when a Stat emptied and whether its
	// mistake has been counted are history that cannot be recomputed from the
	// Stat's value and CreatedAt — see docs/adr/0008.
	HungerEmpty    EmptySpell
	HappinessEmpty EmptySpell

	// SickSince is when the Pet fell Sick, or the zero time if it is not. Sickness
	// is stored, not derived from the Mess that caused it, because it outlasts
	// that cause: cleaning up removes the Mess but does not cure — only Cure does.
	SickSince time.Time
	// DiedAt is the moment the Pet died, or the zero time while it is alive, and
	// Cause is why. Every death is recorded, including Old age, by Advance, at the
	// exact instant it happened: after that the Pet no longer changes, and Age
	// stops there. A save written before deaths were recorded has neither, and
	// is dead by the age-based rule alone (see Stage).
	DiedAt time.Time
	Cause  CauseOfDeath
	// LastCuredAt is when the Pet was last Cured while Sick, or the zero time. It
	// restarts the sickness clock: a Pet Cured while its Mess is still there is
	// Sick again SickAfterMess after the Cure, not at once.
	LastCuredAt time.Time
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

// Stage reports the Pet's life stage as of now. A recorded death (DiedAt) wins:
// the Pet is dead from that moment. Otherwise it is derived from CreatedAt
// rather than stored, so it can never drift out of sync with it, and that same
// age-based rule is what kills a Pet whose death was never recorded: one from a
// save written before deaths were recorded, or one whose Advance has not yet
// run. Cases must stay ordered from the largest cumulative duration to the
// smallest: a future Stage added below Teen's case, rather than above it,
// would never be reached, since Teen's condition would already have matched.
func (p Pet) Stage(now time.Time) Stage {
	if !p.DiedAt.IsZero() && !now.Before(p.DiedAt) {
		return StageDeath
	}
	age := now.Sub(p.CreatedAt)
	switch {
	case age >= EggDuration+BabyDuration+ChildDuration+TeenDuration+AdultDuration:
		return StageDeath
	case age >= EggDuration+BabyDuration+ChildDuration+TeenDuration:
		return StageAdult
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

// Hatched reports whether s is any Stage other than Egg. Kept separate from
// CareAvailable: a Pet that has reached Death is still Hatched — it did,
// historically — even though Care actions are no longer available.
func (s Stage) Hatched() bool { return s != StageEgg }

// CareAvailable reports whether Care actions can currently be taken: any
// Hatched Stage except Death. It is the single condition governing whether
// the Next Screen's icon bar and Care actions are available, so a future
// additional terminal Stage doesn't require re-auditing every comparison
// against a specific Stage value.
func (s Stage) CareAvailable() bool { return s.Hatched() && s != StageDeath }

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
	case StageAdult:
		return "Adult"
	case StageDeath:
		return "Death"
	default:
		return "Egg"
	}
}

// Age reports how long the Pet has been alive as of now. Once the Pet has
// reached Death, Age stops advancing and reports how long it lived: up to the
// recorded moment of death, or, for a death that was never recorded, the fixed
// elapsed time at which old age comes — every earlier Stage's duration summed,
// plus AdultDuration — rather than continuing to climb, since nothing about a
// dead Pet keeps changing.
func (p Pet) Age(now time.Time) time.Duration {
	if !p.DiedAt.IsZero() && !now.Before(p.DiedAt) {
		return p.DiedAt.Sub(p.CreatedAt)
	}
	if p.Stage(now) == StageDeath {
		return EggDuration + BabyDuration + ChildDuration + TeenDuration + AdultDuration
	}
	return now.Sub(p.CreatedAt)
}

// CauseAt reports why the Pet is dead as of now, or NotDead while it is alive. A
// recorded death names its own cause; one that was never recorded (see Stage)
// can only be old age.
func (p Pet) CauseAt(now time.Time) CauseOfDeath {
	switch {
	case !p.DiedAt.IsZero() && !now.Before(p.DiedAt):
		return p.Cause
	case p.Stage(now) == StageDeath:
		return OldAge
	default:
		return NotDead
	}
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
	// A negative HappinessProgress can only come from a corrupted or
	// hand-edited save file. Left alone, one below minus a whole interval
	// would make Advance hand Happiness points back without time passing.
	p.HappinessProgress = max(p.HappinessProgress, 0)
	// A negative tally can only come from a corrupted or hand-edited save file.
	p.CareMistakes = max(p.CareMistakes, 0)
	// A recorded death with no readable cause (a hand-edited or newer save file)
	// can only be read as old age.
	if !p.DiedAt.IsZero() && p.Cause == NotDead {
		p.Cause = OldAge
	}
	return p
}
