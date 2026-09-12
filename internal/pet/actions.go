package pet

import "time"

// FoodKind is the choice offered by the Feed Care action.
type FoodKind int

const (
	// Meal restores Hunger with no other effect.
	Meal FoodKind = iota
	// Snack restores Happiness, but adds Weight.
	Snack
)

// Feed applies kind's effect and returns the updated Pet. Meal restores one
// point of Hunger, capped at MaxStat, with no other effect. Snack restores
// one point of Happiness, capped at MaxStat, and adds one point of Weight —
// uncapped here, since Weight's own ceiling is behavioural rather than a
// hard cap: reaching OverfedThreshold accelerates Happiness decay, and Play
// is the Care action that brings Weight back down.
func (p Pet) Feed(kind FoodKind) Pet {
	switch kind {
	case Snack:
		p.Happiness = min(p.Happiness+1, MaxStat)
		p.Weight++
	case Meal:
		fallthrough
	default:
		p.Hunger = min(p.Hunger+1, MaxStat)
	}
	return p
}

// Play restores one point of Happiness, capped at MaxStat, and reduces
// Weight by one point, floored at BaseWeight — a real counterbalance to
// Snack's Weight cost, so a Care action exists to bring Weight back down.
func (p Pet) Play() Pet {
	p.Happiness = min(p.Happiness+1, MaxStat)
	p.Weight = max(p.Weight-1, BaseWeight)
	return p
}

// Clean records that the Pet was cleaned as of now, ending any Mess. It
// makes no direct stat change — its only effect is stopping the accelerated
// Happiness Decay HasMess causes.
func (p Pet) Clean(now time.Time) Pet {
	p.LastCleanedAt = now
	return p
}
