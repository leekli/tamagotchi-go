package pet

import "time"

// EmptySpell records one Stat's current Empty spell: an unbroken run of the Stat
// being at 0, beginning the instant a Decay step took it there and ending when a
// Care action lifts it above 0. The zero value means no spell.
//
// It holds two instants rather than one flag so that a spell's start (which
// later features, such as starvation, measure from) and its pending Care
// mistake stay independent: the grace deadline can be counted, or later moved,
// without disturbing when the spell began.
type EmptySpell struct {
	// Since is when the Stat reached 0. Zero when there is no recorded spell,
	// including for a Stat already at 0 in a save file written before spells
	// were recorded, until the next Advance gives it a fresh start.
	Since time.Time
	// GraceEndsAt is when the spell's GraceWindow expires and it costs a Care
	// mistake. Zero once that mistake has been counted, so a spell can only ever
	// cost one, and when there is no spell.
	GraceEndsAt time.Time
}

// beginAt returns a spell that began at at, with its full GraceWindow ahead.
func beginAt(at time.Time) EmptySpell {
	return EmptySpell{Since: at, GraceEndsAt: at.Add(GraceWindow)}
}

// Attention is the state of one Stat's Attention call: whether the Stat is
// Empty and, if so, whether its grace window is still running.
type Attention int

const (
	// AttentionNone means the Stat is not Empty.
	AttentionNone Attention = iota
	// AttentionWindowRunning means the Stat is Empty and its grace window has
	// not yet expired: the player can still refill it without a Care mistake.
	AttentionWindowRunning
	// AttentionLapsed means the Stat is Empty and its grace window has expired,
	// so the spell has cost its Care mistake. It stays lapsed until the Stat is
	// refilled.
	AttentionLapsed
)

// HungerAttention reports Hunger's Attention call as of now. A Sick Pet does not
// call for attention, so it is AttentionNone while the Pet is Sick.
func (p Pet) HungerAttention(now time.Time) Attention {
	return attention(p.Hunger, p.HungerEmpty, p.Sick(now), now)
}

// HappinessAttention reports Happiness's Attention call as of now, likewise
// suppressed while the Pet is Sick.
func (p Pet) HappinessAttention(now time.Time) Attention {
	return attention(p.Happiness, p.HappinessEmpty, p.Sick(now), now)
}

// NeedsAttention reports whether either Stat is Empty, in either phase.
func (p Pet) NeedsAttention(now time.Time) bool {
	return p.HungerAttention(now) != AttentionNone || p.HappinessAttention(now) != AttentionNone
}

func attention(stat int, spell EmptySpell, sick bool, now time.Time) Attention {
	switch {
	case stat > 0, sick:
		return AttentionNone
	case spell.Since.IsZero():
		// Empty with no recorded start: a save from before spells were recorded.
		// Its fresh window opens at the next Advance, so until then it is running.
		return AttentionWindowRunning
	case spell.GraceEndsAt.IsZero() || !now.Before(spell.GraceEndsAt):
		return AttentionLapsed
	default:
		return AttentionWindowRunning
	}
}

// startUnrecordedSpells gives a Stat that is already at 0 but has no recorded
// spell a fresh one starting at now. That only happens for a Pet loaded from a
// save written before spells were recorded; opening its window at the first
// Advance, not at whenever it really emptied, means upgrading never punishes a
// Pet retroactively.
func (p Pet) startUnrecordedSpells(now time.Time) Pet {
	if p.Hunger == 0 && p.HungerEmpty.Since.IsZero() {
		p.HungerEmpty = beginAt(now)
	}
	if p.Happiness == 0 && p.HappinessEmpty.Since.IsZero() {
		p.HappinessEmpty = beginAt(now)
	}
	return p
}

// countLapsedWindows counts a Care mistake for each Stat whose grace window has
// expired by now and not yet been counted.
func (p Pet) countLapsedWindows(now time.Time) Pet {
	if windowLapsed(p.HungerEmpty, p.SickSince, now) {
		p.CareMistakes++
		p.HungerEmpty.GraceEndsAt = time.Time{}
	}
	if windowLapsed(p.HappinessEmpty, p.SickSince, now) {
		p.CareMistakes++
		p.HappinessEmpty.GraceEndsAt = time.Time{}
	}
	return p
}

// windowLapsed reports whether s's grace window has expired by now and should be
// counted. The grace clock does not run while the Pet is Sick, so a window that
// would expire after the Sickness began (sickSince, zero when the Pet is well)
// is left pending, to be resumed by a Cure. One that expired at or before the
// Sickness began ran its full course while the Pet was well, and still counts.
func windowLapsed(s EmptySpell, sickSince, now time.Time) bool {
	if s.GraceEndsAt.IsZero() || now.Before(s.GraceEndsAt) {
		return false
	}
	return sickSince.IsZero() || !s.GraceEndsAt.After(sickSince)
}

// resumedAfterSickness returns s with its pending grace deadline moved later by
// the time it spent paused: from the later of the Sickness beginning and the
// spell beginning, to the Cure. A spell that was running when the Pet fell Sick
// thereby keeps the grace time it had left; one that began during the Sickness
// gets a fresh window from the Cure. Only the deadline moves: when the spell
// began does not, because time Empty is not paused by Sickness. A deadline that
// is zero (no spell, or its mistake already counted) or that fell at or before
// the Sickness began is left alone.
func (s EmptySpell) resumedAfterSickness(sickSince, curedAt time.Time) EmptySpell {
	if !s.GraceEndsAt.After(sickSince) {
		return s
	}
	pausedFrom := sickSince
	if s.Since.After(pausedFrom) {
		pausedFrom = s.Since
	}
	s.GraceEndsAt = s.GraceEndsAt.Add(curedAt.Sub(pausedFrom))
	return s
}

// endRefilledSpells ends the Empty spell of any Stat a Care action has lifted
// above 0. Called by every Care action that can raise a Stat.
func (p Pet) endRefilledSpells() Pet {
	if p.Hunger > 0 {
		p.HungerEmpty = EmptySpell{}
	}
	if p.Happiness > 0 {
		p.HappinessEmpty = EmptySpell{}
	}
	return p
}
