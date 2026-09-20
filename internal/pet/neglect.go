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

// HungerAttention reports Hunger's Attention call as of now.
func (p Pet) HungerAttention(now time.Time) Attention {
	return attention(p.Hunger, p.HungerEmpty, now)
}

// HappinessAttention reports Happiness's Attention call as of now.
func (p Pet) HappinessAttention(now time.Time) Attention {
	return attention(p.Happiness, p.HappinessEmpty, now)
}

// NeedsAttention reports whether either Stat is Empty, in either phase.
func (p Pet) NeedsAttention(now time.Time) bool {
	return p.HungerAttention(now) != AttentionNone || p.HappinessAttention(now) != AttentionNone
}

func attention(stat int, spell EmptySpell, now time.Time) Attention {
	switch {
	case stat > 0:
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
	if windowLapsed(p.HungerEmpty, now) {
		p.CareMistakes++
		p.HungerEmpty.GraceEndsAt = time.Time{}
	}
	if windowLapsed(p.HappinessEmpty, now) {
		p.CareMistakes++
		p.HappinessEmpty.GraceEndsAt = time.Time{}
	}
	return p
}

func windowLapsed(s EmptySpell, now time.Time) bool {
	return !s.GraceEndsAt.IsZero() && !now.Before(s.GraceEndsAt)
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
