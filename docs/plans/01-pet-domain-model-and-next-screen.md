# Plan 01 — Pet domain model & the Next Screen's real content

**Status:** proposed, not yet implemented
**Depends on:** nothing — this is the foundation
**Blocks:** [`02-core-care-actions.md`](02-core-care-actions.md), which assumes everything in
this plan already exists

## 1. Purpose

Introduce the game's first domain package, `internal/pet`, and give the Next Screen
(`internal/tui/next`) its real content: a living, persisted creature the player can see,
whose stats change over real elapsed time whether or not the game is running — the
defining trait of a first-generation Tamagotchi.

This plan deliberately stops short of any player action. There is no Feed, Play, or
Clean here — the pet simply exists, is born, grows from an Egg into a Baby, decays over
time, and survives a restart. Plan 02 adds the verbs. Splitting it this way keeps this
plan's surface small enough to get the persistence and decay model right before any UI
complexity sits on top of it.

Read `CONTEXT.md`, `CLAUDE.md`, and `docs/adr/0002-pragmatic-architecture-grow-the-boundaries.md`
before starting — this plan follows their conventions throughout and does not repeat their
reasoning.

## 2. Vocabulary this plan introduces

Add these to `CONTEXT.md` under a new "The Pet" section (see §7). Use exactly this
vocabulary in code, comments, and commit messages.

| Term | Meaning |
|---|---|
| **Pet** | The creature the player raises. Already reserved in `CONTEXT.md`; this plan is the feature that makes it real. Distinct from the **Character** (decorative, Welcome Screen only). |
| **Stage** | The Pet's life stage: **Egg** or **Baby** in this plan. (Child/Teen/Adult evolution is a later feature — do not build it now.) |
| **Hatch** | The one-time transition from Egg to Baby, a fixed real-time interval after the Pet is first created. |
| **Stat** | One of the Pet's numeric attributes: **Hunger**, **Happiness**, **Weight**. Hunger and Happiness decay over time; Weight is fixed at birth in this plan — a later feature makes it dynamic. |
| **Decay** | The automatic, time-driven reduction of Hunger and Happiness. Decay is a pure function of elapsed wall-clock time, not of frames or player action. |
| **Save file** | The single JSON file holding the Pet's persisted state between runs. |

Do not introduce "Home Screen" or rename the Next Screen. `CONTEXT.md` already defines
the Next Screen's identity ("the Screen the player reaches from the Welcome Screen") and
explicitly lists "Placeholder Screen" and "Game Screen" as names to avoid — this plan is
the "later feature" its own glossary entry refers to. Keep the package at
`internal/tui/next`.

## 3. Non-goals (explicitly out of scope for this plan)

- Any player-facing action (feed, play, clean, discipline, medicine). Plan 02 territory.
- Sickness/health, poop/mess, and evolution beyond Egg→Baby. None of these are stored or
  stubbed here — adding unused fields "for later" is exactly the speculative design
  `CLAUDE.md` and ADR-0002 warn against. When a future feature needs them, it adds them
  and writes its own save-schema migration.
- A minigame of any kind.
- Sound.
- Multiple save slots / multiple pets.

## 4. Design decisions

### 4.1 Package layout

New package `internal/pet` — pure domain logic (data + functions), no direct file or
network I/O in its core type. This is the "change that first needs it" ADR-0002 was
waiting for: write a short ADR for it (§8).

```
internal/pet/
  pet.go          // Pet struct, Stage, stat constants, New(), pure mutators
  pet_test.go
  decay.go        // Advance(now) — the pure time-driven decay function
  decay_test.go
  store.go        // Store interface, FileStore implementation, DefaultSavePath()
  store_test.go
```

### 4.2 The `Pet` type

```go
type Stage int

const (
    StageEgg Stage = iota
    StageBaby
)

type Pet struct {
    CreatedAt  time.Time // when the Pet was first born; drives Age and the Egg→Baby hatch
    LastSeenAt time.Time // wall-clock time decay was last applied up to; drives offline catch-up
    Hunger     int       // 0 (starving) .. MaxStat (full)
    Happiness  int       // 0 (unhappy) .. MaxStat (happy)
    Weight     int       // starts at BaseWeight; only this plan's birth (New) sets it, nothing
                          // decays it — fixed for now, a later feature makes it dynamic
}
```

Stats are plain `int`, not a hidden struct — `CLAUDE.md` favours the simplest thing that
works, and there's no behaviour yet that needs encapsulation beyond the pure functions
below.

Constants (tune during implementation, but name and document them so Plan 02 and later
tuning passes have one place to change):

- `MaxStat = 4` — matches the four-pip meters of the original hardware.
- `BaseWeight = 2` — the Pet's starting weight, set once at birth (`New`). Fixed for this
  plan — nothing changes it afterwards; Weight stays a **Stat** in the glossary, but a
  later feature makes it dynamic. Say so in the doc comment on `Weight` itself, so nobody
  "fixes" the fact that it never moves.
- `EggDuration = 30 * time.Second` — how long the Pet stays an Egg before hatching.
  Deliberately short — it exists to give the player an early "it's alive" moment on first
  launch, mirroring the real toy's power-on hatch, not to gate play.
- `HungerDecayInterval = HappinessDecayInterval = 3 * time.Minute` — wall-clock duration
  per one-point decay step.

**Why these numbers:** the original hardware decayed stats over hours because the toy
stayed in the player's pocket all day. This game is a CLI program people run for minutes
at a session. Copying 1997's timings verbatim would mean nothing visibly happens in a
normal play session, which is a worse experience than the toy it's imitating. The
3-minute decay interval is deliberately on the order of single-digit minutes so a player
who watches the Next Screen for a little while actually sees the stat change — say so in
a code comment next to the constants so nobody "corrects" it back to real-hardware timing
later.

Pure functions on `Pet` (value receiver, returns a new `Pet` — no mutation in place,
consistent with `internal/anim`'s and `internal/art`'s pure-function style):

- `New(now time.Time) Pet` — a freshly hatched-from-nothing Egg: `CreatedAt` and
  `LastSeenAt` set to `now`, `Hunger`/`Happiness` at `MaxStat`, `Weight` at `BaseWeight`.
- `(p Pet) Stage(now time.Time) Stage` — derived, not stored: `StageEgg` until
  `now.Sub(p.CreatedAt) >= EggDuration`, then `StageBaby` forever after. Deriving it
  avoids a second piece of state that could drift out of sync with `CreatedAt`.
- `(p Pet) Age(now time.Time) time.Duration` — `now.Sub(p.CreatedAt)`.
- `(p Pet) Advance(now time.Time) Pet` — see `decay.go` below.

### 4.3 Decay (`decay.go`)

`Advance(now)` is the one function that moves time forward. It must be pure and
deterministic given `p` and `now` — no `time.Now()` calls inside it — so it is testable
without sleeping, the same discipline `internal/anim` uses for animation frames.

```go
func (p Pet) Advance(now time.Time) Pet
```

Behaviour:

1. If `now` is before `p.LastSeenAt`, return `p` unchanged (clock went backwards —
   ignore rather than panic or produce a negative elapsed duration; this is a boundary
   condition worth a named test case, not an error path).
2. Compute `elapsed := now.Sub(p.LastSeenAt)`.
3. Reduce `Hunger` by `elapsed / HungerDecayInterval` whole steps, floored at 0.
4. Reduce `Happiness` by `elapsed / HappinessDecayInterval` whole steps, floored at 0.
5. `Weight` is untouched — nothing in this plan changes it after hatch.
6. Set `LastSeenAt = now` on the result.
7. Stage is not stored, so hatching falls out of `Stage(now)` automatically and needs no
   handling here. This applies uniformly regardless of Stage — `Advance` does not
   special-case `StageEgg`. In practice `EggDuration` is short enough that it rarely
   matters, but it's a deliberate simplification: coupling `Advance` to `Stage` would tie
   together two things this plan otherwise keeps independent.

This same function serves two callers: the offline catch-up on load (§4.4) and the
Next Screen's periodic tick while the game is running (§4.5) — one algorithm, exercised
identically both ways, which is exactly the point of keeping it pure and separate from
both the file I/O and the TUI.

### 4.4 Persistence (`store.go`)

```go
type Store interface {
    Load() (Pet, bool, error) // ok=false, err=nil means "no save file yet"
    Save(Pet) error
}

type FileStore struct { Path string }

func NewFileStore(path string) FileStore
func DefaultSavePath() (string, error) // os.UserConfigDir() + "tamagotchi-go/save.json"
```

Save format: JSON, with an explicit schema version so a future field addition (health,
mess, evolution stage, …) can migrate old files instead of failing to load them:

```json
{
  "schema_version": 1,
  "created_at": "...",
  "last_seen_at": "...",
  "hunger": 4,
  "happiness": 4,
  "weight": 2
}
```

`FileStore.Load` behaviour, all of which needs a test:

- File does not exist → `(Pet{}, false, nil)`. This is the normal first-run case, not an
  error.
- File exists and parses → `(pet, true, nil)`.
- File exists but is unreadable or fails to unmarshal (corrupt, truncated, wrong schema
  version) → return a descriptive `error`; **do not panic**. The caller (§4.5) treats any
  error the same as "no save file": log a one-line notice to stderr and start a fresh
  Pet. Losing a corrupt save is an acceptable, explicit trade-off for a hobby game; a
  hard crash on a corrupt file is not.

`FileStore.Save` writes atomically enough not to corrupt the file on a crash mid-write:
write to a temp file in the same directory, then rename over the target. Create the
parent directory (`os.MkdirAll`) if it doesn't exist yet.

**I/O discipline:** `internal/tui.Screen.Update` is documented as "implementations must
not perform I/O directly." `Store` methods therefore must never be called synchronously
from `next.Screen.Update`. All loading happens once, before the `tea.Program` starts, in
`internal/cli` (a wiring layer, not a Screen — see §4.6). All saving from inside a
running Screen happens via a `tea.Cmd` (see §4.5), which is the established Bubble Tea
pattern for deferred work — exactly how `anim.Tick` already defers its own timer.

### 4.5 The Next Screen (`internal/tui/next`)

Replace the placeholder in `internal/tui/next/next.go` with the real Screen. Its
constructor now takes what it needs instead of building nothing:

```go
func New(initial pet.Pet, store pet.Store) tui.Screen
```

This changes `next.New`'s signature, which breaks its use as a zero-arg
`tui.ScreenFactory` in `internal/cli.ScreenFactories()` — fix that at the call site with
a closure that captures the loaded Pet and Store (§4.6); `tui.ScreenFactory` itself does
not need to change.

State the Screen holds:

- `pet pet.Pet` — current state.
- `store pet.Store` — for the periodic save `tea.Cmd`.
- `now time.Time` — the Screen's own view of wall-clock time, seeded at construction from
  `initial.LastSeenAt` (the post-offline-catch-up timestamp `internal/cli` already
  computed) and refreshed on every `anim.TickMsg`. Drives `Stage`/`Age` rendering only —
  never `Advance` (see below).
- `width, height int` — from `tea.WindowSizeMsg`, as every other Screen does.
- `frame int` — an animation frame counter, advanced by `anim.TickMsg`, exactly like the
  Welcome Screen — for idle motion (an Egg wobble, a Baby bob), not for decay.

Two independent clocks drive this Screen, and they must not be conflated. Deliberately
*not* both named `Tick` — `CONTEXT.md`'s **Frame** entry already reserves "tick" to mean
the animation message specifically, and a second, much slower clock reusing that name
would blur two genuinely different things in code and conversation:

1. **`anim.Tick`** at `anim.FPS` — cosmetic animation only, identical usage to the
   Welcome Screen. Advances `frame`, and also refreshes `s.now` from `TickMsg.Time`.
2. **A new, much slower simulation clock** — define `pet.BeatInterval = 20 * time.Second`
   and a `pet.Beat() tea.Cmd` / `pet.BeatMsg{Time time.Time}` pair in the `pet` package,
   mirroring `anim.Tick`/`anim.TickMsg`'s shape exactly (same rationale: a Screen
   re-issues it from `Update` to keep it running, tests drive it by feeding `BeatMsg`
   instead of sleeping). On `pet.BeatMsg`, the Screen calls `s.pet = s.pet.Advance(msg.Time)`
   (pure), then returns `tea.Batch(pet.Beat(), pet.SaveCmd(s.store, s.pet))` where
   `SaveCmd` is a small `tea.Cmd` wrapper around `store.Save` — this is where persistence
   I/O actually happens, off the Update call stack.

**Rendering needs a finer-grained clock than the Beat.** `Stage(now)` decides Egg vs.
Baby art, and `EggDuration` is only 30 seconds — if the Screen only knew "now" from
`pet.BeatMsg` (every 20s), the visible hatch could lag the real boundary by up to a full
`BeatInterval`, undermining the "it's alive" moment this plan asks for. That's why `now`
is tracked separately from `anim.TickMsg` (fired ~15×/sec, already carries a wall-clock
`Time`) rather than derived from the Beat: `Stage` and `Age` render off `s.now`; only
`Advance` and the save run off the Beat.

Rendering (`View`): full-bleed, like the Welcome Screen — the Next Screen is about to
stop being "free-flowing placeholder text" and become an authored layout, so switch its
`Scrollable()` to `false` and lay it out with `lipgloss.Place` centred in the body area,
the same pattern `welcome.Screen.View` already uses. Update `docs/adr/0004-full-bleed-screens-opt-out-of-the-viewport.md`'s
"Text Screens whose content can genuinely outgrow the body area — the Next Screen
today…" sentence — it's describing the placeholder, and it stops being true; note that
in the ADR changelog rather than silently editing the historical decision text (add a
short "Update" section at the bottom, don't rewrite the original reasoning).

Show, at minimum:

- The Pet's art for its current `Stage` (art authoring in §4.7), with a small idle
  animation driven by `frame`.
- Hunger and Happiness as four-pip meters, e.g. `Hunger  [****]` / `Hunger  [**--]`,
  styled with the existing adaptive `tui.Palette` (`internal/tui/styles.go`) rather than
  inventing a new colour scheme.
- Age, in a human-scale unit derived from `Pet.Age(now)` (e.g. "Day 0" for the first 24h,
  "Day 1" after, …).
- Weight, as a plain number (unit is a naming decision for the implementer — pick one,
  e.g. "Weight 2g" or "Weight 2", and add it to `CONTEXT.md` if it needs a definition).
  It is fixed at `BaseWeight` for this plan — nothing changes it after birth; a later
  feature makes it dynamic, so don't design the display around it changing yet.

No icons/menu/help beyond the existing quit key are needed yet — that is Plan 02.

### 4.6 Wiring (`internal/cli`)

`internal/cli/cli.go`'s `run` function is the right place for the one synchronous,
startup-time load — it already does comparable one-shot setup (flag parsing, colour
profile). Add:

1. Resolve the save path: `pet.DefaultSavePath()`, with a `--save-path` flag override
   (`fs.String("save-path", "", "override the save file location (mainly for testing)")).
   This is cheap to add alongside the existing `--version`/`--no-color` flags and makes
   manual QA and any future scripted testing of persistence possible without touching a
   real user's config directory.
2. Build a `pet.FileStore` at that path.
3. `Load()` it. On `ok == false` or `err != nil` (see §4.4 for the corrupt-file case),
   log a one-line notice to `stderr` only in the error case, and start from `pet.New(time.Now())`.
4. On success, immediately call `.Advance(time.Now())` on the loaded Pet — this is the
   offline catch-up: decay is applied for the time elapsed since the game last ran,
   using the exact same pure function the running Screen uses per tick (§4.3).
5. Change `ScreenFactories()` to take the resolved `(pet.Pet, pet.Store)` as parameters
   (it stops being a zero-arg free function) and close over them for `next.New`:

   ```go
   func ScreenFactories(initial pet.Pet, store pet.Store) map[tui.ScreenID]tui.ScreenFactory {
       return map[tui.ScreenID]tui.ScreenFactory{
           tui.WelcomeScreenID: welcome.New,
           tui.NextScreenID:    func() tui.Screen { return next.New(initial, store) },
       }
   }
   ```

   Known, acceptable limitation: because `ScreenFactory` is still zero-arg and the
   closure captures `initial` once, re-navigating to the Next Screen would reset it to
   that captured value rather than resuming where the player left off. Nothing currently
   navigates away from the Next Screen (ADR-0003: replace semantics, no history stack;
   the Next Screen is presently a terminal destination), so this is not a live bug — just
   flag it with a code comment so nobody is surprised if a future feature adds a route
   back to the Welcome Screen.

### 4.7 Art (`internal/art`)

New embedded art file(s) in `internal/art/`, following the existing hand-authored,
ASCII-only convention (see `internal/art/art.go`'s doc comment: "Art is ASCII only").

- `egg.txt` — a P1-style egg: a simple rounded oval with a couple of spots/speckles, the
  way the original hardware's low-resolution egg sprite reads at a glance. Keep it small
  (roughly the footprint of the existing `marutchi-walk-*.txt` files) so it fits
  comfortably in the Next Screen's layout at 80×24.
- For the Baby stage, **reuse** the existing `marutchi-walk-1.txt` / `marutchi-walk-2.txt`
  via `art.MustLoad` and `art.Mirror` rather than authoring new art — `CONTEXT.md`
  already defines Marutchi as "the round first-generation form," and reusing it here
  gives the Pet visual continuity with the Welcome Screen's Character while satisfying
  the "as close as possible to the original P1" brief without new asset-authoring risk.
  A gentle idle bob (reuse `anim.Pulse` or a small triangle wave over `frame`, the same
  technique `welcome/character.go` uses) is enough motion for this plan; full walking
  around the Screen is not required here.

If new art is added, it needs the same golden-file discipline the Welcome Screen's
wordmark has *only if* you introduce frame-by-frame animation subtle enough to need
locking down visually — for the Egg (static) and the Baby idle bob (a simple, easily
assert-able offset), plain assertions on rendered output are sufficient; a new golden
file suite is not required for this plan.

### 4.8 `tui.QuitHandler` — save on quit

`internal/tui/app.go`'s `Update` intercepts `Ctrl+C` before the active Screen ever sees
it (`if key.Matches(msg, a.keys.Quit) { return a, tea.Quit }`), and relying solely on the
periodic simulation tick (§4.5) to persist means up to one `pet.TickInterval` of play
could be lost on quit. Add a small, optional interface — same shape as the existing
`HelpProvider` pattern in `internal/tui/keys.go`:

```go
// QuitHandler is an optional interface a Screen implements to run a command before
// the App quits — e.g. flushing persisted state.
type QuitHandler interface {
    OnQuit() tea.Cmd
}
```

Put it in `internal/tui/screen.go` next to the `Screen` interface it complements. Change
`App.Update`'s `Ctrl+C` branch:

```go
if key.Matches(msg, a.keys.Quit) {
    if qh, ok := a.current.(QuitHandler); ok {
        return a, tea.Sequence(qh.OnQuit(), tea.Quit)
    }
    return a, tea.Quit
}
```

`next.Screen` implements `OnQuit() tea.Cmd` by returning the same `pet.SaveCmd(s.store,
s.pet)` used by the periodic Beat. This is a small, generically useful App-level
addition (any future Screen with state to flush can use it), not a Next-Screen-specific
hack — call this out explicitly in the ADR (§8).

**Constraint this depends on:** `OnQuit` only fires because the Next Screen currently has
no local quit key of its own — every quit reaches it through the App's global `Ctrl+C`
branch (unlike the Welcome Screen, which handles its own `Esc` and returns `tea.Quit`
directly, bypassing the App entirely). This plan keeps the Next Screen `Ctrl+C`-only.
Record this as an explicit constraint in the ADR (§8): a future Screen-local quit key on
the Next Screen must call `OnQuit()` itself before `tea.Quit`, or save-on-quit silently
breaks.

## 5. Testing plan

Follow `CLAUDE.md`'s five layers. Every new piece of behaviour below needs a test before
this plan is done; none of this is optional polish.

**Unit — `internal/pet`:**
- `New`: fields set correctly from a fixed `now`.
- `Stage`: before/at/after `EggDuration` boundary (table-driven).
- `Age`: simple duration arithmetic.
- `Advance`: table-driven over elapsed durations — zero elapsed, partial step (no change
  yet), exact step, multiple steps, decay floors at 0 and does not go negative, `Weight`
  never changes, `LastSeenAt` updates to `now`, and the clock-went-backwards case from
  §4.3 step 1.
- `FileStore`: round-trip save→load in `t.TempDir()`; missing file returns `ok=false,
  err=nil`; corrupt/truncated file returns a non-nil error and does not panic; `Save`
  creates missing parent directories; a second `Save` overwrites cleanly (verifies the
  temp-file-then-rename doesn't leave stale temp files or fail on the second write).

**Unit — `internal/tui`:**
- `QuitHandler`: a fake Screen implementing it proves `App.Update` sequences its `OnQuit`
  command before `tea.Quit` on `Ctrl+C`; a Screen that does *not* implement it still
  quits cleanly (regression guard on the existing behaviour, using the existing
  `fakeScreen` pattern in `internal/tui/app_test.go`).

**Unit — `internal/tui/next`:**
- `View` renders the Egg art before `EggDuration` has elapsed and the Baby art after,
  driven entirely by feeding `anim.TickMsg`s with controlled `Time` values (not real
  sleeping) — this is what proves the Screen's own `now` (seeded from
  `initial.LastSeenAt`, refreshed per `anim.TickMsg`) is what `Stage`/`Age` render from,
  independent of whether any `pet.BeatMsg` has fired yet.
- Hunger/Happiness meters render the right number of filled/empty pips for given stat
  values (table-driven over 0..MaxStat).
- `Scrollable()` returns `false`.
- A `pet.BeatMsg` advances the held `pet.Pet` and re-issues both `pet.Beat()` and a save
  command (assert the returned `tea.Cmd` is non-nil / batches as expected — following the
  existing pattern for asserting `anim.Tick` re-issue in the Welcome Screen tests).
- `OnQuit()` returns a command that calls `Save` on the injected store (use a fake
  `pet.Store` — an in-memory map-backed one is enough, no need to touch a real file).

**Integration (`teatest`, real `tea.Program`):**
- Launching straight through Welcome → Next shows the Egg.
- Feeding the program enough `anim.TickMsg`s carrying times past `EggDuration` (not real
  time) crosses the boundary and the view switches to the Baby art.
- A fake/in-memory `pet.Store` records a `Save` call after a `pet.BeatMsg` fires.

**Smoke (`internal/cli/smoke_test.go`, builds the real binary under a pty):**
- Extend the existing smoke test (or add a case) that points `--save-path` at a temp
  file, launches the binary, advances past the Welcome Screen, quits, and asserts the
  save file now exists on disk with the expected JSON shape. This is the one place that
  proves the whole load → run → save-on-quit path works end to end through the real
  binary, not just through fakes.

**Acceptance (given/when/then, mirroring `internal/tui/welcome/acceptance_test.go`'s
style):** cover at least:
- Given no save file, when the game launches, then the Pet starts as a fresh Egg.
- Given a save file with elapsed offline time, when the game launches, then Hunger and
  Happiness reflect the offline decay.
- Given a corrupt save file, when the game launches, then it starts a fresh Egg instead
  of crashing.
- Given the Pet is an Egg, when `EggDuration` has elapsed, then it has hatched into a Baby.
- Given the game is running, when the player quits, then the current Pet state is saved.

## 6. Coverage, lint, and formatting

No exceptions from the standard gate in `CLAUDE.md`:
- `golangci-lint run ./...` clean — pay particular attention to `funlen` (80 lines / 50
  statements) and `cyclop` (max complexity 15) on `Advance` and the Next Screen's
  `Update`/`View`; split helpers out rather than requesting a `//nolint`.
- `go test -race -covermode=atomic ./...` green.
- `.testcoverage.yml`'s thresholds (75% file / 80% package / 93% total) must still pass;
  do not lower them. `internal/pet` in particular should be near 100% — it is pure logic
  with no excuse for untested branches.
- `gofumpt`/`goimports` via `golangci-lint fmt`.
- Prose (comments, doc strings, help text) in UK English, per `CLAUDE.md`.

## 7. Documentation updates

- **`CONTEXT.md`**: add "The Pet" section per §2's table. Update the existing **Pet**
  glossary entry to remove "A later feature — absent from the first two delivery
  phases" — it no longer is one. Update the **Next Screen** entry's "For now it holds
  placeholder text and nothing else; its real content is a later feature" sentence to
  describe what it now shows. Note in the **Stat** row that Weight is fixed at birth and
  does not change during play in this plan — a later feature makes it dynamic — so the
  glossary doesn't read as promising behaviour this plan doesn't deliver.
- **`README.md`**: update the "Features" bullet list (a living, persisted Pet on the
  Next Screen), the "Controls" table if any new key is added (there shouldn't be any in
  this plan — no player actions yet), and the "Project layout" section to list
  `internal/pet/`.
- **`docs/adr/0004-...md`**: append a short "Update" note (see §4.5) — do not rewrite the
  original decision text.

## 8. New ADR

Write `docs/adr/0006-pet-domain-package-and-quit-hook.md`. It should record, briefly (in
the house style of the existing ADRs — a few short paragraphs, not a template form):
- Why `internal/pet` is introduced now (ADR-0002 explicitly deferred this to "the change
  that first needs it," and asked for its own ADR when that happens).
- The pure-decay / injected-clock design (`Advance(now)`, no `time.Now()` inside the
  domain type) and why, mirroring `internal/anim`'s existing rationale.
- The `Store` interface and why persistence is a seam (testability, and keeping I/O out
  of `Screen.Update`).
- The `tui.QuitHandler` addition to the router and why it's a generic App-level hook
  rather than a Next-Screen special case, and the constraint it depends on: a Screen-local
  quit key (like the Welcome Screen's `Esc`) must call `OnQuit()` itself before
  `tea.Quit`, since it bypasses the App's `Ctrl+C` branch entirely. The Next Screen has no
  such key today, which is why `OnQuit` is reliable here — a future one added without
  calling `OnQuit()` would silently break save-on-quit.
- Why the simulation clock is named `pet.Beat`/`pet.BeatMsg` rather than reusing `Tick`:
  `anim.TickMsg` already owns that name for the ~15/sec animation frame clock in
  `CONTEXT.md`, and a second clock fifty times slower needs a different word, not the same
  one at a different rate.

## 9. Deliverables checklist

- [ ] `internal/pet` package: `Pet`, `Stage`, constants, `New`, `Stage()`, `Age()`,
      `Advance()`, `Store`, `FileStore`, `DefaultSavePath()`, `Beat`/`BeatMsg`, `SaveCmd`.
- [ ] `internal/tui/screen.go`: `QuitHandler` interface.
- [ ] `internal/tui/app.go`: `Ctrl+C` branch sequences `OnQuit()` when implemented.
- [ ] `internal/tui/next`: real `Screen` per §4.5, replacing the placeholder.
- [ ] `internal/cli/cli.go`: `--save-path` flag, startup load + offline `Advance`,
      `ScreenFactories` takes `(pet.Pet, pet.Store)`.
- [ ] `internal/art/egg.txt` (new); Baby stage reuses existing Marutchi art.
- [ ] Full test suite per §5, all passing, coverage gate green.
- [ ] `CONTEXT.md`, `README.md` updated per §7.
- [ ] `docs/adr/0004-...md` updated with a short note.
- [ ] `docs/adr/0006-pet-domain-package-and-quit-hook.md` added.
- [ ] `golangci-lint run ./...`, `go test -race -covermode=atomic ./...`, and the
      coverage tool all pass locally before pushing (`lefthook`'s `pre-push` gate).
