# Pet domain package and the save-on-quit hook

ADR-0002 deferred a `game`/domain package to "the change that first needs it," with
its own ADR when that happens. The Next Screen's real content is that change:
a Pet with a life stage, stats, and time-driven decay is genuine domain logic,
not presentation, so it gets its own package — `internal/pet` — rather than
living inside the Screen that renders it.

`Pet.Advance(now)` is pure and takes `now` as an argument rather than calling
`time.Now()` itself, mirroring `internal/anim`'s existing rationale for the
animation clock: a Screen advances state by feeding it timestamps, so decay,
hatching, and offline catch-up are all testable without sleeping. The same
function serves both callers — the one-time catch-up applied at load and the
Next Screen's periodic simulation step — because decay is one algorithm
regardless of who's asking for it to run.

Persistence is a `Store` interface with a file-backed implementation, not a
concrete type baked into the Screen. This keeps `Store` swappable for tests
(an in-memory fake needs no real file) and keeps I/O out of `Screen.Update`,
which `internal/tui`'s contract already forbids; all loading happens once at
startup in `internal/cli`, and all saving from a running Screen happens via a
deferred `tea.Cmd`.

The router gains `tui.QuitHandler`, a small optional interface — the same
shape as the existing `HelpProvider` pattern — letting a Screen run a command
before the App quits. It's a generic App-level addition, not a Next-Screen
special case: any future Screen with state to flush can implement it. It
depends on one constraint worth recording because it isn't visible from the
router code alone: `OnQuit` only fires for a quit that reaches the App's own
`Ctrl+C` handling. A Screen that handles its own local quit key (as the
Welcome Screen does with `Esc`) and returns `tea.Quit` directly bypasses the
App entirely, so it bypasses `OnQuit` too. The Next Screen has no such key
today, which is why `OnQuit` is reliable here — a future one added without
also calling `OnQuit()` would silently break save-on-quit.

Finally, the Next Screen's periodic simulation step is named `pet.Beat`, not
`pet.Tick`. `internal/anim`'s `Tick`/`TickMsg` already own that name for the
~15/sec animation clock, and reusing it for a second clock fifty times slower
would blur two genuinely different things every time either is mentioned in
code or conversation.

## Update: Mess follows the same derive-don't-store pattern as Stage

The Core Care Actions feature (Feed, Play, Clean) added a Mess concept: the
Pet is left in an uncleaned state once enough time has passed since it was
last cleaned, and Happiness decays faster while that's true. The working plan
for that feature proposed a stored, mutated `Mess bool` field, updated inside
`Advance`.

Implementation instead derived it — `HasMess(now time.Time) bool`, computed
from a `LastCleanedAt` timestamp — with no `Mess` field on `Pet` at all. This
is the same reasoning already recorded above for `Stage`: a derived value
can't drift out of sync with the timestamp it's based on, whereas a stored
flag mutated on each `Advance` call could, in principle, disagree with what
the timestamp implies. It also simplified the pre-feature save file question:
with no field to add, there was nothing to default on old saves beyond
`LastCleanedAt` itself (defaulted to `CreatedAt`, meaning "never cleaned
since birth").

## Update: `Advance` is exact, so long catch-ups and short Beats agree

`Advance` originally chose one Happiness Decay rate for the whole window it was
given, by whether the Pet had Mess (or was Overfed) at the end of it, and its
doc comment recorded that as a deliberate simplification. That made the result
depend on how the same span was sliced: an untouched Pet reached Happiness 0 at
6:00 when caught up in one call but at 7:30 when advanced in 20-second Beats.
That was tolerable while Decay only drove a meter, but neglect consequences
(Care mistakes, sickness, early Death) are triggered by the exact moment a Stat
empties, so the simplification is retired.

`Advance` now splits its window wherever Happiness's rate can change on its own
(Mess appearing `MessInterval` after the last Clean) and accrues each piece at
its own rate. Rather than re-anchoring a step's start time on each rate change,
which needs a proportional rescale that rounds at odd nanoseconds, Happiness
keeps a `HappinessProgress`: how far it is toward its next point, measured in
normal-rate time. The accelerated rate simply counts double toward the next
point, in whole-number nanosecond arithmetic, so partial progress carries across
a rate change proportionally and exactly, and any slicing of the same span gives
the same Pet. `AcceleratedHappinessDecayInterval` must therefore divide
`HappinessDecayInterval` evenly, which a test pins. Hunger's rate never changes,
so it keeps its whole-step anchor unchanged.

The other rate changes, Clean and the Weight changes of Snack and Play, are
made by Care actions between `Advance` calls. The contract is therefore that a
Care action follows an `Advance` to its own instant, so it lands on a Pet that
is up to date rather than changing the rate over time already passed. This is a
contract of the domain package; making the Next Screen honour it is a separate
change.

`HappinessProgress` is a new optional Save file field (`happiness_progress_ns`).
A save without it reads as no partial progress, and a negative value from a
corrupted save is clamped to zero on load, so no `schema_version` bump was
needed.
