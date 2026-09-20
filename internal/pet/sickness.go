package pet

import "time"

// Sick reports whether the Pet is Sick as of now. It falls Sick SickAfterMess
// after its Mess appears, and stays Sick until Cured: cleaning up removes the
// Mess but does not cure. Sickness is read at now, not only from what Advance
// has recorded, because the Next Screen reads it on its own faster clock, which
// can be ahead of the Pet's last Advance.
func (p Pet) Sick(now time.Time) bool {
	return !p.SickSince.IsZero() || !p.sicknessOnset().After(now)
}

// Cure ends a Sick Pet's Sickness as of now, in a single dose. On a Pet that is
// not Sick it does nothing at all, and in particular does not touch the
// sickness clock: otherwise pressing Cure while a Mess is pending would delay
// the Sickness it is meant to answer.
//
// A Pet cured while its Mess is still there is Sick again SickAfterMess after
// the Cure, not at once, so the player has time to clean up. Like every Care
// action, Cure follows an Advance to now.
func (p Pet) Cure(now time.Time) Pet {
	if !p.Sick(now) {
		return p
	}
	p.SickSince = time.Time{}
	p.LastCuredAt = now
	return p
}

// sicknessOnset is when the Pet would fall Sick if nothing changed: SickAfterMess
// after the later of the Mess appearing and the last Cure. A Clean before then
// moves the Mess later and so cancels it.
func (p Pet) sicknessOnset() time.Time {
	clockStart := p.LastCleanedAt.Add(MessInterval)
	if p.LastCuredAt.After(clockStart) {
		clockStart = p.LastCuredAt
	}
	return clockStart.Add(SickAfterMess)
}

// beginSickness records the exact instant the Pet fell Sick, if it did by now.
// A Mess only ever changes through Care actions, which follow an Advance, so
// within one Advance window the Mess is constant and the onset is exact.
func (p Pet) beginSickness(now time.Time) Pet {
	if p.SickSince.IsZero() {
		if onset := p.sicknessOnset(); !onset.After(now) {
			p.SickSince = onset
		}
	}
	return p
}
