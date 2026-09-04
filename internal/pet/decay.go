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
func (p Pet) Advance(now time.Time) Pet {
	if now.Before(p.LastSeenAt) {
		// The clock went backwards (e.g. a corrected system clock). Ignore
		// rather than produce a negative elapsed duration.
		return p
	}

	elapsed := now.Sub(p.LastSeenAt)
	p.Hunger = decayStat(p.Hunger, elapsed, HungerDecayInterval)
	p.Happiness = decayStat(p.Happiness, elapsed, HappinessDecayInterval)
	p.LastSeenAt = now
	return p
}

// decayStat reduces a stat by one whole point per interval elapsed, floored
// at zero.
func decayStat(stat int, elapsed, interval time.Duration) int {
	stat -= int(elapsed / interval)
	if stat < 0 {
		return 0
	}
	return stat
}
