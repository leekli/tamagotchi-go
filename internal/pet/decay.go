package pet

import (
	"math"
	"time"
)

// happinessAcceleration is how many times faster Happiness Decays while the
// Pet HasMess or is Overfed. AcceleratedHappinessDecayInterval must divide
// HappinessDecayInterval exactly, so that progress toward a point stays a
// whole number of nanoseconds however a window is split.
const happinessAcceleration = int64(HappinessDecayInterval / AcceleratedHappinessDecayInterval)

// maxHappinessAccrual bounds the progress a single constant-rate piece of a
// window can add. time.Duration saturates at about 292 years, so a save file
// that lost its timestamps (read as the zero time) yields a window far larger
// than any real one, and doubling it for the accelerated rate would overflow.
// Half the range is many thousands of times more than Happiness could ever
// lose, so clamping to it changes nothing that matters.
const maxHappinessAccrual = time.Duration(math.MaxInt64 / 2)

// Advance is the one function that moves time forward. It is pure and
// deterministic given p and now — it never calls time.Now() itself — so
// Decay is testable without sleeping. It serves two callers identically: the
// offline catch-up applied once at load, and the Next Screen's periodic Beat
// while the game is running.
//
// Advance is exact: it gives the same result however the window up to now is
// sliced, so one long catch-up after a long absence equals the same span
// advanced in many short Beats. Callers that apply a Care action at some
// instant must Advance to that instant first, so the action lands on a Pet
// that is up to date rather than retroactively changing the rate over time
// that has already passed.
//
// Advance applies uniformly regardless of Stage: it does not special-case
// StageEgg. In practice EggDuration is short enough that this rarely
// matters, but it's a deliberate simplification — coupling Advance to Stage
// would tie together two things this package otherwise keeps independent.
//
// Hunger Decays at one fixed rate, so it needs only a whole-step anchor
// (LastSeenAt): the window is divided into whole steps and any remainder is
// left to accumulate on the next call, which is already exact.
//
// Happiness's rate is not fixed. It runs at HappinessDecayInterval, or at
// AcceleratedHappinessDecayInterval — a single rate shared by both causes,
// not compounding — while the Pet HasMess or is Overfed. Advance therefore
// splits its window wherever that can change and accrues each piece at its
// own rate, into HappinessProgress: time spent at the accelerated rate counts
// double toward the next point. Partial progress is thereby carried across a
// rate change proportionally and exactly: a point that was two thirds done
// when Mess appeared is still two thirds done, and needs only the remaining
// third at the faster rate. The only rate change that happens by itself with
// the passage of time is Mess appearing MessInterval after LastCleanedAt;
// Care actions (Clean, and the Weight changes of Snack and Play) are the
// others, which is why they must follow an Advance to their own instant.
//
// Advance also keeps each Stat's Empty spell (see EmptySpell): it records the
// exact instant a Stat reaches 0, and counts a Care mistake at the exact
// instant the spell's grace window expires, however many Beats or how long a
// catch-up that falls inside. A Stat already at 0 with no recorded spell (a
// save from before spells existed) gets a fresh window starting at now.
//
// It likewise records the exact instant the Pet fell Sick, if it did in this
// window (see Pet.Sick).
func (p Pet) Advance(now time.Time) Pet {
	if now.Before(p.LastSeenAt) || now.Before(p.HappinessLastSeenAt) {
		// The clock went backwards (e.g. a corrected system clock). Ignore
		// rather than produce a negative elapsed duration.
		return p
	}

	p = p.startUnrecordedSpells(now)
	p = p.decayHunger(now)
	p = p.decayHappiness(now)
	p = p.beginSickness(now)
	return p.countLapsedWindows(now)
}

// decayHunger applies Hunger Decay up to now, and records the instant Hunger
// reaches 0 if it does so in this window.
//
// The anchor advances only by the decay steps actually consumed, not all the
// way to now: the Next Screen's Beat fires far more often than a decay interval
// (seconds vs. minutes), so resetting the anchor to now on every call would
// discard each Beat's sub-interval progress before it ever accumulated into a
// whole step. Because steps land on a fixed grid from the anchor, the instant
// Hunger reaches 0 is exactly the anchor plus one interval per point it had.
func (p Pet) decayHunger(now time.Time) Pet {
	anchor, before := p.LastSeenAt, p.Hunger

	var consumed time.Duration
	p.Hunger, consumed = decayStat(before, now.Sub(anchor), HungerDecayInterval)
	p.LastSeenAt = anchor.Add(consumed)

	if before > 0 && p.Hunger == 0 {
		p.HungerEmpty = beginAt(anchor.Add(time.Duration(before) * HungerDecayInterval))
	}
	return p
}

// decayHappiness applies Happiness Decay from HappinessLastSeenAt up to now,
// one constant-rate piece at a time. Each piece ends at the next moment the
// rate could change (or at now), so the loop is the place to add any further
// event that changes Happiness's rate: it only needs to report the time of
// its next change from nextRateChange.
func (p Pet) decayHappiness(now time.Time) Pet {
	cursor := p.HappinessLastSeenAt
	progress := p.HappinessProgress
	points := p.Happiness
	var emptiedAt time.Time // when Happiness reached 0 in this window, if it did

	for cursor.Before(now) {
		end := now
		if change, ok := p.nextRateChange(cursor); ok && change.Before(end) {
			end = change
		}
		rate := time.Duration(p.happinessRate(cursor))

		before := progress
		progress += min(end.Sub(cursor), maxHappinessAccrual/rate) * rate
		steps := int(progress / HappinessDecayInterval)
		progress %= HappinessDecayInterval

		if points > 0 && steps >= points {
			// The last point goes when progress reaches `points` whole intervals:
			// that much work from where this piece began, done at this piece's
			// rate. Rounded up to a whole nanosecond, so the instant is one the
			// Pet has actually reached.
			need := time.Duration(points)*HappinessDecayInterval - before
			emptiedAt = cursor.Add((need + rate - 1) / rate)
		}
		points = max(points-steps, 0)
		cursor = end
	}

	p.Happiness = points
	p.HappinessProgress = progress
	p.HappinessLastSeenAt = now
	if !emptiedAt.IsZero() {
		p.HappinessEmpty = beginAt(emptiedAt)
	}
	return p
}

// happinessRate is how many times normal-rate progress accrues per unit of
// real time at the instant at: 1, or happinessAcceleration while the Pet
// HasMess or is Overfed.
func (p Pet) happinessRate(at time.Time) int64 {
	if p.HasMess(at) || p.Overfed() {
		return happinessAcceleration
	}
	return 1
}

// nextRateChange reports the earliest moment after cursor at which
// Happiness's rate changes of its own accord, if any: Mess appearing. Weight
// (and so Overfed) and LastCleanedAt only ever change through Care actions,
// which are applied between Advance calls, so they never change the rate
// partway through a window.
func (p Pet) nextRateChange(cursor time.Time) (time.Time, bool) {
	onset := p.LastCleanedAt.Add(MessInterval)
	return onset, cursor.Before(onset)
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
