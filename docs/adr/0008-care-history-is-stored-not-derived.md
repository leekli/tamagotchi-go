# Care history is stored, not derived

ADR-0006 derives state from timestamps wherever it can (`Stage` from `CreatedAt`, `HasMess` from `LastCleanedAt`), so nothing can drift out of step with what it is derived from. Neglect cannot be handled that way: when a Stat emptied depends on the care given along the way, and whether a Care mistake has already been counted is history that neither the Stat's current value nor `CreatedAt` can recover. So the Pet stores it: a lifetime `CareMistakes` tally, and per Stat an `EmptySpell` holding when the current Empty spell began and when its grace window expires. `Advance` is the only thing that opens a spell or counts a mistake, at the exact instants they happen (see ADR-0006's update on exactness); Care actions only close spells. What can still be derived is: the Attention call is read from a spell and the time, never stored. (The Next Screen reads it at its own animation clock, so its wording turns from running to lapsed at the grace deadline itself, while the mistake is recorded by the next `Advance`, on the following Beat: up to one Beat later.)

A spell holds two instants, not one flag plus a start. The start is kept untouched because later features (starvation) measure from it. The grace deadline is zeroed once its mistake has been counted, which is what makes a spell cost at most one mistake, and it can be moved later without disturbing the start (a Sick Pet's grace clock is paused this way).

## Considered options

- **A `counted` flag beside a single start time.** Works for one mistake per spell, but leaves nowhere to move the deadline, so pausing the grace clock would need a further field.
- **Replaying history to recompute the tally.** Impossible: the Pet keeps no event log, and the Care actions between two saves are not recorded. An event log would be a new layer with no other use (ADR-0002).

## Consequences

- The new state is added to the Save file as optional fields with load defaults and no `schema_version` bump. A save with a Stat already at 0 but no recorded spell gets a fresh grace window from its first `Advance`, so upgrading never counts time from before it.
- `Stage` stops being purely derivable once an early Death is recorded; see the update on recorded death below.

## Update: Sickness is stored for the same reason

Sickness follows the same reasoning. It is caused by a Mess left uncleaned, but
it outlasts that cause: cleaning up removes the Mess and the Pet is still Sick
until Cured. So `SickSince` is stored, and `Advance` records it at the exact
instant the Pet fell Sick. A Cure stamps `LastCuredAt`, which moves the start of
the sickness clock, so a Pet Cured while its Mess is still there is Sick again a
full interval later rather than at once. Only what the Pet would be if nothing
changed is derived: `Sick(now)` also reports a Pet as Sick from its Mess and its
last Cure, so the Screen, which reads at its own faster clock, never lags an
Advance. A Cure on a Pet that is not Sick must not stamp `LastCuredAt`, or
pressing it would delay the Sickness it is meant to answer.

## Update: Sickness pauses the grace clock by sliding the deadline

Sickness pauses each Empty spell's grace clock, and the two-instant spell is what
makes that cheap. While the Pet is Sick nothing is counted, and the pending
deadline is left where it is. A Cure moves every pending deadline later by the
time its spell spent Sick, measured from the later of the Sickness beginning and
the spell beginning: a spell that was running keeps the grace time it had left,
and one that began during the Sickness gets a full window from the Cure. The
spell's start is never touched, because time spent Empty is not paused and
starvation will measure from it. A window that had already expired when the
Sickness began ran its course while the Pet was well and is still counted, and
one that expires at the very instant it begins counts too. Because a Cure can only
move deadlines that an `Advance` has already recorded, it, like every Care action,
must follow an `Advance` to its own instant.

## Update: Death is recorded, so Stage is no longer purely derived

An early Death, from Starvation, cannot be derived from `CreatedAt` either, so
`Advance` now records every death, Old age included, as `DiedAt` and a cause, at
the exact instant it happened. `Stage` and `Age` read that record when there is
one: from `DiedAt` on the Pet is in Stage Death and its Age stops at the moment
of death. The age-based rule stays as the fallback, so a Pet with no record (a
save from before deaths were recorded, or one whose `Advance` has not yet run) is
still dead once past its age, its cause reading as Old age; the first `Advance`
then records it. The cause is saved by name, not by number, so reordering the
constants can never change what an existing save means.

Two consequences shape `Advance`. To find a death it advances the Pet to now
first, so that the Empty spell that will starve it is on record, takes the
earliest death that falls by then (Starvation before Old age on an exact tie),
and then advances again, only as far as that moment, so that its Stats and Care
mistakes read as of the death and not of the time the game happened to notice. And
from then on `Advance` returns the Pet unchanged: a dead Pet has no time left to
pass. The Screen notices a death on the next Beat, or sooner when a Care action
advances the Pet first, in which case the action finds it dead and does nothing.

## Update: Sickness kills, and the first launch after an upgrade is not special

A Pet left Sick for `SickDeathInterval` without a Cure now dies of Sickness, at
that exact instant. `earliestDeath` considers Starvation, then Sickness, then Old
age, and a later cause replaces an earlier one only if it is strictly sooner, so
an exact tie goes to the earlier in the spec's order. A Pet never cleaned is Sick
at 10:00 and dead at 18:00, ahead of the 22:00 at which Hunger would starve it.

The Next Screen seeds its clock from `Pet.RecordedUntil`, the latest instant the
Pet's own record reaches, and reads the real clock at once from `Init`. Hunger's
whole-step anchor is the wrong seed for a Pet that died long ago: dead Pets are
frozen, so it stays a step short of the moment of death, and a first frame drawn
from it would show a Starvation as still alive.

**Known limitation, accepted.** A save written before neglect was tracked has no
record of any of it, so its first launch applies the whole time the game was
closed under the new rules, exactly as for any other offline gap: a Pet still alive
under the old rules that was left idle for about 22 minutes (or with a Mess
unattended for about 18) is found dead, with its cause shown. Special-casing
that needs a marker that says "this save predates neglect", which means a schema
version bump and a one-off migration, for saves that are, as of this decision,
essentially the maintainer's own: any save more than an hour old was already dead
of old age. The outcome is not destructive, since a Pet lives about an hour by
design and Restart is one keypress. If the game is ever distributed to other
players, revisit it then, with the schema bump and a real migration path.

## Update: the tally shortens life, so the end of life is not fixed either

Each Care mistake now shortens the Adult Stage, so when the Pet's life ends depends
on the stored tally, which itself changes over time. `Stage`, `Age` and the search
for a death all read the end from the tally, `lifespanEnd`, instead of a constant.
The Adult always begins at the same instant; only its end moves, and never below
`AdultMinDuration`, so the Pet is always seen as an Adult.

This makes the tally's *timing* matter, not just its value: a mistake that lands
after the shortened end has already passed must kill the Pet at the moment it is
counted, never before it, or a Pet the player was still looking after would be
found to have died in the past. So `Advance` cannot judge a window against its
final tally. `advanceLiving` reports the instants at which it counted mistakes,
and `lifespanDeath` walks the window one piece at a time between them, each with
the tally as it stood there. A window holds at most two such instants (one per
Stat, since no Care action can end a spell within it), so the walk is short. When
the Pet dies, `Advance` re-advances only as far as that moment, so the tally on
the Death panel includes the mistake that killed it and none that came after.
A death that would come from the end of the Adult Stage is Neglect when any mistake
is on the tally, and Old age when none is.
