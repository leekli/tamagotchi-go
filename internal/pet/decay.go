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
// Happiness decays at AcceleratedHappinessDecayInterval, rather than
// HappinessDecayInterval, whenever the Pet HasMess or is Overfed as of now —
// a single rate for the whole elapsed window, chosen by the state at the end
// of it, and shared between both causes rather than compounding when both
// apply at once. A window that straddles the exact moment either condition
// would have started partway through is not split into two rates: that would
// double the complexity of the partial-step accounting below for a boundary
// nobody is watching in real time.
//
// Hunger and Happiness are decayed against their own anchors (LastSeenAt and
// HappinessLastSeenAt respectively) rather than one shared one: their
// intervals can now differ (Happiness speeds up while HasMess or Overfed),
// and a single shared anchor advanced by whichever stat consumed less would
// silently re-apply a stat's already-counted step on every subsequent call
// until the other stat caught up — see docs/adr/0006's update note.
func (p Pet) Advance(now time.Time) Pet {
	if now.Before(p.LastSeenAt) || now.Before(p.HappinessLastSeenAt) {
		// The clock went backwards (e.g. a corrected system clock). Ignore
		// rather than produce a negative elapsed duration.
		return p
	}

	// Advance each anchor only by the decay steps actually consumed, not all
	// the way to now: the Next Screen's Beat fires far more often than a
	// decay interval (seconds vs. minutes), so resetting an anchor to now on
	// every call would discard each Beat's sub-interval progress before it
	// ever accumulated into a whole step.
	var hungerConsumed, happinessConsumed time.Duration
	p.Hunger, hungerConsumed = decayStat(p.Hunger, now.Sub(p.LastSeenAt), HungerDecayInterval)
	p.LastSeenAt = p.LastSeenAt.Add(hungerConsumed)

	happinessInterval := HappinessDecayInterval
	if p.HasMess(now) || p.Overfed() {
		happinessInterval = AcceleratedHappinessDecayInterval
	}
	p.Happiness, happinessConsumed = decayStat(p.Happiness, now.Sub(p.HappinessLastSeenAt), happinessInterval)
	p.HappinessLastSeenAt = p.HappinessLastSeenAt.Add(happinessConsumed)

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
