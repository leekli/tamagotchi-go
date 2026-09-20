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
)

// String returns the cause's display name, shown on the Death panel. It is empty
// for NotDead.
func (c CauseOfDeath) String() string {
	switch c {
	case OldAge:
		return "Old age"
	case Starvation:
		return "Starvation"
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
// starve it is on record. When causes fall at the very same instant the spec's
// order applies: Starvation before Old age.
func (p Pet) earliestDeath(now time.Time) (at time.Time, cause CauseOfDeath, dies bool) {
	if p.Hunger == 0 && !p.HungerEmpty.Since.IsZero() {
		if starves := p.HungerEmpty.Since.Add(StarvationInterval); !starves.After(now) {
			at, cause, dies = starves, Starvation, true
		}
	}
	if old := p.lifespanEnd(); !old.After(now) && (!dies || old.Before(at)) {
		at, cause, dies = old, OldAge, true
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
	default:
		return NotDead
	}
}
