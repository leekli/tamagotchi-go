# Care history is stored, not derived

ADR-0006 derives state from timestamps wherever it can (`Stage` from `CreatedAt`, `HasMess` from `LastCleanedAt`), so nothing can drift out of step with what it is derived from. Neglect cannot be handled that way: when a Stat emptied depends on the care given along the way, and whether a Care mistake has already been counted is history that neither the Stat's current value nor `CreatedAt` can recover. So the Pet stores it: a lifetime `CareMistakes` tally, and per Stat an `EmptySpell` holding when the current Empty spell began and when its grace window expires. `Advance` is the only thing that opens a spell or counts a mistake, at the exact instants they happen (see ADR-0006's update on exactness); Care actions only close spells. What can still be derived is: the Attention call is read from a spell and the time, never stored.

A spell holds two instants, not one flag plus a start. The start is kept untouched because later features (starvation) measure from it. The grace deadline is zeroed once its mistake has been counted, which is what makes a spell cost at most one mistake, and it can be moved later without disturbing the start (a Sick Pet's grace clock is paused this way).

## Considered options

- **A `counted` flag beside a single start time.** Works for one mistake per spell, but leaves nowhere to move the deadline, so pausing the grace clock would need a further field.
- **Replaying history to recompute the tally.** Impossible: the Pet keeps no event log, and the Care actions between two saves are not recorded. An event log would be a new layer with no other use (ADR-0002).

## Consequences

- The new state is added to the Save file as optional fields with load defaults and no `schema_version` bump. A save with a Stat already at 0 but no recorded spell gets a fresh grace window from its first `Advance`, so upgrading never counts time from before it.
- `Stage` will stop being purely derivable once an early Death is recorded; that change amends this ADR.
