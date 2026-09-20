package pet

import "time"

// CauseOfDeath is why a Pet died.
type CauseOfDeath int

const (
	// NotDead is the zero value: the Pet is alive.
	NotDead CauseOfDeath = iota
	// OldAge is a death from the Adult Stage running out, when no Care mistake had
	// shortened it.
	OldAge
	// Starvation is a death from Hunger left Empty for StarvationInterval.
	Starvation
	// Sickness is a death from being left Sick for SickDeathInterval without a
	// Cure.
	Sickness
	// Neglect is a death from the Adult Stage running out after Care mistakes had
	// shortened it.
	Neglect
)

// String returns the cause's display name, shown on the Death panel. It is empty
// for NotDead.
func (c CauseOfDeath) String() string {
	switch c {
	case OldAge:
		return "Old age"
	case Starvation:
		return "Starvation"
	case Sickness:
		return "Sickness"
	case Neglect:
		return "Neglect"
	default:
		return ""
	}
}

// adultBegins is when the Pet's Adult Stage begins. Mistakes never move it: they
// shorten the Adult only.
func (p Pet) adultBegins() time.Time {
	return p.CreatedAt.Add(EggDuration + BabyDuration + ChildDuration + TeenDuration)
}

// adultSpan is how long the Adult Stage lasts with mistakes on the tally:
// AdultDuration less AdultMistakePenalty for each, never below AdultMinDuration.
// The shortening is capped before it is multiplied, so an absurd tally (from a
// corrupted save file) cannot overflow it.
func adultSpan(mistakes int) time.Duration {
	if mistakes <= 0 {
		return AdultDuration
	}
	if int64(mistakes) > int64((AdultDuration-AdultMinDuration)/AdultMistakePenalty) {
		return AdultMinDuration
	}
	return AdultDuration - time.Duration(mistakes)*AdultMistakePenalty
}

// lifespanEndFor is when the Pet's life ends, the end of its Adult Stage, if its
// tally stood at mistakes.
func (p Pet) lifespanEndFor(mistakes int) time.Time {
	return p.adultBegins().Add(adultSpan(mistakes))
}

// lifespanEnd is when the Pet's life ends with the tally it has now.
func (p Pet) lifespanEnd() time.Time { return p.lifespanEndFor(p.CareMistakes) }

// lifespanCause names the death that is the Adult Stage running out: Neglect if
// any Care mistake had shortened it, Old age if none had.
func lifespanCause(mistakes int) CauseOfDeath {
	if mistakes > 0 {
		return Neglect
	}
	return OldAge
}

// lifespanDeath reports when the Pet's life ends within the window from..now, if
// it does. The end depends on the tally, and the tally changes at the instants
// Care mistakes are counted (lapses, in order), so the window is walked one piece
// at a time, each with the tally as it stood there. A piece's end may already be
// past when a mistake lands and pushes it earlier; the Pet then dies at the
// moment that mistake is counted, never retroactively. Within the first piece,
// which begins at from with the tally the Pet already had, an end already behind
// it is simply reported (a save from before deaths were recorded).
func (p Pet) lifespanDeath(mistakes int, from time.Time, lapses []time.Time, now time.Time) (at time.Time, cause CauseOfDeath, dies bool) {
	segmentStart := from
	for i := 0; i <= len(lapses); i++ {
		segmentEnd := now
		if i < len(lapses) {
			segmentEnd = laterOf(lapses[i], from)
		}

		end := p.lifespanEndFor(mistakes)
		if i > 0 {
			end = laterOf(end, segmentStart) // a mistake cannot kill before it is counted
		}
		if !end.After(segmentEnd) {
			return end, lifespanCause(mistakes), true
		}

		segmentStart = segmentEnd
		mistakes++
	}
	return time.Time{}, NotDead, false
}

func laterOf(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}

// earliestDeath reports the moment the Pet dies, if it does by now, and why. It
// is read from a Pet already advanced to now, so that the Empty spell that will
// starve it and the Sickness that will kill it are on record; mistakes and from
// describe the window it was advanced across: the tally the Pet started it with,
// where it began, and the instants Care mistakes were counted in it. The earliest
// instant wins. Causes falling at the very same instant go in the spec's order:
// Starvation, Sickness, then the Adult Stage running out (Neglect or Old age).
// Candidates are therefore considered in that order, and a later one only
// replaces an earlier one if it is strictly sooner.
func (p Pet) earliestDeath(mistakes int, from time.Time, lapses []time.Time, now time.Time) (at time.Time, cause CauseOfDeath, dies bool) {
	consider := func(when time.Time, why CauseOfDeath) {
		if !when.After(now) && (!dies || when.Before(at)) {
			at, cause, dies = when, why, true
		}
	}

	if p.Hunger == 0 && !p.HungerEmpty.Since.IsZero() {
		consider(p.HungerEmpty.Since.Add(StarvationInterval), Starvation)
	}
	if !p.SickSince.IsZero() {
		consider(p.SickSince.Add(SickDeathInterval), Sickness)
	}
	if when, why, ok := p.lifespanDeath(mistakes, from, lapses, now); ok {
		consider(when, why)
	}
	return at, cause, dies
}

// causeName is a cause's stable name in the save file. It is not the display
// name: those may be reworded, while a save must keep meaning what it meant.
func causeName(c CauseOfDeath) string {
	switch c {
	case OldAge:
		return "old_age"
	case Starvation:
		return "starvation"
	case Sickness:
		return "sickness"
	case Neglect:
		return "neglect"
	default:
		return ""
	}
}

// causeFromName reads a cause back from the save file. A name it does not
// recognise reads as NotDead, which Pet.withLoadDefaults turns into the Adult
// Stage running out for a Pet with a recorded death.
func causeFromName(name string) CauseOfDeath {
	switch name {
	case "old_age":
		return OldAge
	case "starvation":
		return Starvation
	case "sickness":
		return Sickness
	case "neglect":
		return Neglect
	default:
		return NotDead
	}
}
