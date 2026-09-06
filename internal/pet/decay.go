package pet

import "time"

// Advance is the one function that moves time forward. It is pure and
// deterministic given p and now — it never calls time.Now() itself — so
// Decay is testable without sleeping. It serves two callers identically: the
// offline catch-up applied once at load, and the Next Screen's periodic Beat
// while the game is running.
//
// Advance applies uniformly regardless of Stage: it does not special-case
// StageEgg. In practice EggDuration is short enough that this rarely
// matters, but it's a deliberate simplification — coupling Advance to Stage
// would tie together two things this package otherwise keeps independent.
//
// Happiness decays at MessHappinessDecayInterval, rather than
// HappinessDecayInterval, whenever the Pet HasMess as of now — a single rate
// for the whole elapsed window, chosen by the state at the end of it. A
// window that straddles the exact moment a Mess would have appeared partway
// through is not split into two rates: that would double the complexity of
// the partial-step accounting below for a boundary nobody is watching in
// real time.
func (p Pet) Advance(now time.Time) Pet {
	if now.Before(p.LastSeenAt) {
		// The clock went backwards (e.g. a corrected system clock). Ignore
		// rather than produce a negative elapsed duration.
		return p
	}

	elapsed := now.Sub(p.LastSeenAt)

	happinessInterval := HappinessDecayInterval
	if p.HasMess(now) {
		happinessInterval = MessHappinessDecayInterval
	}

	var hungerConsumed, happinessConsumed time.Duration
	p.Hunger, hungerConsumed = decayStat(p.Hunger, elapsed, HungerDecayInterval)
	p.Happiness, happinessConsumed = decayStat(p.Happiness, elapsed, happinessInterval)

	// Advance LastSeenAt only by the smaller of the two consumed durations,
	// not all the way to now. The Next Screen's Beat fires far more often
	// than a decay interval (seconds vs. minutes), so resetting LastSeenAt to
	// now on every call would discard each Beat's sub-interval progress
	// before it ever accumulated into a whole step — Hunger/Happiness would
	// then only ever move via a single large offline catch-up, never while
	// the game is actually running.
	consumed := hungerConsumed
	if happinessConsumed < consumed {
		consumed = happinessConsumed
	}
	p.LastSeenAt = p.LastSeenAt.Add(consumed)
	return p
}

// decayStat reduces a stat by one whole point per interval elapsed, floored
// at zero, and reports how much of elapsed was actually consumed by whole
// steps (steps * interval) — Advance uses this to preserve any leftover
// sub-interval progress instead of discarding it.
func decayStat(stat int, elapsed, interval time.Duration) (newStat int, consumed time.Duration) {
	steps := int(elapsed / interval)
	stat -= steps
	if stat < 0 {
		stat = 0
	}
	return stat, time.Duration(steps) * interval
}
