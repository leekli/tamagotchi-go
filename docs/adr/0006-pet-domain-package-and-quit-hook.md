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
