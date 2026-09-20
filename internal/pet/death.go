package pet

import "time"

// CauseOfDeath is why a Pet died.
type CauseOfDeath int

const (
	// NotDead is the zero value: the Pet is alive.
	NotDead CauseOfDeath = iota
	// OldAge is a death from the Adult Stage running out.
	OldAge
	// Starvation is a death from Hunger left Empty for StarvationInterval.
	Starvation
	// Sickness is a death from being left Sick for SickDeathInterval without a
	// Cure.
	Sickness
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
	default:
		return ""
	}
}

// lifespanEnd is when the Pet dies of old age: the end of its Adult Stage.
func (p Pet) lifespanEnd() time.Time {
	return p.CreatedAt.Add(EggDuration + BabyDuration + ChildDuration + TeenDuration + AdultDuration)
}

// earliestDeath reports the moment the Pet dies, if it does by now, and why. It
// is read from a Pet already advanced to now, so that the Empty spell that will
// starve it and the Sickness that will kill it are on record. The earliest
// instant wins. Causes falling at the very same instant go in the spec's order:
// Starvation, Sickness, Neglect, Old age. Candidates are therefore considered in
// that order, and a later one only replaces an earlier only if it is strictly
// sooner.
func (p Pet) earliestDeath(now time.Time) (at time.Time, cause CauseOfDeath, dies bool) {
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
	consider(p.lifespanEnd(), OldAge)
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
	default:
		return ""
	}
}

// causeFromName reads a cause back from the save file. A name it does not
// recognise reads as NotDead, which Pet.withLoadDefaults turns into OldAge for a
// Pet with a recorded death.
func causeFromName(name string) CauseOfDeath {
	switch name {
	case "old_age":
		return OldAge
	case "starvation":
		return Starvation
	case "sickness":
		return Sickness
	default:
		return NotDead
	}
}
