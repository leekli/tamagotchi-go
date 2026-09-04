# Plan 02 — Core care actions (Feed, Play, Clean)

**Status:** proposed, not yet implemented
**Depends on:** [`01-pet-domain-model-and-next-screen.md`](01-pet-domain-model-and-next-screen.md)
must be fully implemented first — this plan adds no new packages, only new behaviour on
top of `internal/pet` and `internal/tui/next` as that plan leaves them.

## 1. Purpose

Give the player something to *do*. Plan 01 makes the Pet exist and decay; this plan adds
the three verbs that let the player intervene: **Feed**, **Play**, and **Clean**. This is
the loop the original 1997 hardware was built around — stats fall, the player acts on an
icon menu, stats rise — and finishing it is what turns the Next Screen from a diorama
into a game.

Read Plan 01 in full before starting: this plan assumes `pet.Pet`, `pet.Advance`,
`pet.Store`, `pet.TickInterval`/`Tick`/`TickMsg`, `tui.QuitHandler`, and the Next
Screen's layout and two-clock design (`anim.TickMsg` for cosmetics, `pet.TickMsg` for
simulation) all exist exactly as that plan specifies. Do not re-derive or restructure
them here — extend them.

## 2. Vocabulary this plan introduces

Add to `CONTEXT.md`'s "The Pet" section (created by Plan 01).

| Term | Meaning |
|---|---|
| **Care action** | One of Feed, Play, or Clean — a player-triggered command that changes the Pet's stats. |
| **Meal** / **Snack** | The two Feed choices. Meal restores Hunger with no side effect; Snack restores Happiness but adds Weight. |
| **Mess** | The uncleaned state the Pet is left in after enough time passes; while present it depresses Happiness decay further, and Clean removes it. (Deliberately not called "poop" in code or UI copy — keep the term politely abstract; ASCII art can still be a small pile glyph.) |
| **Icon bar** | The row of selectable care-action icons the Next Screen now shows, navigated by keyboard or mouse. |

## 3. Non-goals (explicitly out of scope for this plan)

- **Medicine and Discipline** icons, and any sickness/health model. Nothing in this plan
  makes the Pet sick — Mess affects Happiness decay only, not a health stat, because no
  health stat exists yet (Plan 01 deliberately didn't add one; don't add one implicitly
  here either).
- **A real minigame** for Play. Play is a stat action with a short reaction animation,
  not a separate interactive minigame — that's a larger feature with its own input
  handling and deserves its own plan if wanted later.
- **Cooldowns / rate limits** on actions (the original hardware limited how often
  feeding "counted"). Skip this for the first playable loop; note it as a candidate
  follow-up in `CONTEXT.md` or a code comment only if it's genuinely easy to leave a hook
  for — don't build unused infrastructure for it.
- **Evolution.** Weight and care history influencing the Pet's eventual adult form is a
  later feature; this plan only makes Weight move.

## 4. Design decisions

### 4.1 Extending `internal/pet`

Add to the existing `Pet` struct (no schema version bump needed if you add fields with
sensible zero values that round-trip through the existing JSON tags — but check: adding
`Mess bool` defaults to `false` on old save files, which is exactly the desired
behaviour for a save written before this plan existed, so this is safe. If any new field
would *not* have a safe zero-value default, bump `schema_version` and write a migration
step in `FileStore.Load` instead of guessing — do not skip this check).

```go
type Pet struct {
    // ... existing fields from Plan 01 ...
    Mess bool // true once enough time has passed without cleaning
}
```

New constants, named and documented like Plan 01's decay intervals:
- `MessInterval` — real-time duration after which `Mess` becomes `true` if not already.
- `MessHappinessMultiplier` (or equivalent) — how much faster Happiness decays while
  `Mess` is `true`. A simple approach: while `Mess`, treat `HappinessDecayInterval` as
  shorter (e.g. halved) inside `Advance`. Pick something that visibly matters within a
  play session, consistent with Plan 01's guidance on tuning decay for session-length
  play rather than real-hardware timescales.

Extend `(p Pet) Advance(now time.Time) Pet` (pure, same discipline as Plan 01):
- Set `Mess = true` once `elapsed-since-last-clean >= MessInterval`. This needs a way to
  know when the Pet was last cleaned — add `LastCleanedAt time.Time` (defaults to
  `CreatedAt` for a freshly-hatched Pet and for any pre-Plan-02 save file, which is the
  correct "never cleaned yet" reading).
- Apply the faster Happiness decay while `Mess` is `true`, as above.

New pure mutators, each returning a new `Pet` (never mutating in place — matches Plan
01's style and `internal/anim`'s pure-function convention):

```go
type FoodKind int

const (
    Meal FoodKind = iota
    Snack
)

func (p Pet) Feed(kind FoodKind) Pet
func (p Pet) Play() Pet
func (p Pet) Clean(now time.Time) Pet
```

Behaviour, all capped at `MaxStat` (never overflowing the 4-pip meter):
- `Feed(Meal)`: `Hunger = min(Hunger+1, MaxStat)`. No Weight change.
- `Feed(Snack)`: `Happiness = min(Happiness+1, MaxStat)` **and** `Weight += 1` (uncapped,
  or capped at a generous ceiling if you want to avoid unbounded growth — either is
  defensible; document whichever you choose in a comment, since "why doesn't Weight cap
  like the others" is a natural question for the next reader).
- `Play()`: `Happiness = min(Happiness+1, MaxStat)`.
- `Clean(now)`: `Mess = false`, `LastCleanedAt = now`. No direct stat change — the
  benefit is stopping the accelerated Happiness decay, not an instant boost.

These are called from the Next Screen's `Update` in response to a confirmed menu
selection (§4.3) — never from `Advance`, and never as a side effect of any tick. Keep
"time passing" and "the player acted" as two clearly separate code paths, exactly as
Plan 01 keeps `anim.TickMsg` and `pet.TickMsg` separate.

### 4.2 Art

New embedded files in `internal/art/`, ASCII-only, sized to sit comfortably in the
existing charBox-style footprint used by the Welcome Screen's Character (see
`internal/tui/welcome/character.go` for the pattern, not the exact box — the Next Screen
has its own layout from Plan 01):

- `mess.txt` — a small pile glyph shown near the Pet while `Mess` is true. Keep it tiny
  (a handful of characters) — it's a status indicator, not a set piece.
- Optional but recommended for "closely mimicking the original P1" per the project
  brief: a brief one- or two-frame "eating" pose and a brief "happy/playing" pose for the
  Baby stage, reusing the existing mirror/idle-bob techniques from Plan 01 rather than a
  full new animation system. If authoring new poses feels like scope creep when you get
  there, it is acceptable to fall back to the existing idle Baby art plus a text-only
  reaction (e.g. a brief "*munch*" / "*plays happily*" line) — the stat change and menu
  interaction are the deliverable, the celebratory animation is polish. Use judgement and
  say in the PR description which you chose and why.

### 4.3 Icon bar UI (`internal/tui/next`)

This is the largest piece of new UI work in the project so far — plan it carefully
against the existing conventions rather than inventing new ones:

- **Layout:** a horizontal row of three icons/labels — `Feed`, `Play`, `Clean` — below
  the Pet and its stat meters, still inside the full-bleed, centred stack Plan 01 set up
  with `lipgloss.Place`/`lipgloss.JoinVertical`.
- **Keyboard navigation:** a `selected int` field on `Screen` cycling 0..2, moved by
  Left/Right (or `h`/`l`) and wrapping at the ends; Enter activates the selected icon.
  This mirrors the original hardware's two-button "select, then confirm" interaction
  while staying idiomatic for a TUI (arrow keys + Enter is the standard pattern; do not
  invent a third input mode).
- **Direct hotkeys:** `f`, `p`, `c` as accelerators that both move the selection to and
  immediately activate the corresponding icon, for players who don't want to navigate.
- **Mouse:** each icon is its own `zone.Mark`ed region (follow `internal/tui/welcome/prompt.go`'s
  `BeginZoneID` pattern exactly — a package-level exported `const ...ZoneID = "next.feed"`
  etc., a `zone.Mark` wrapping the rendered icon in `View`, and a `zone.Get(id).InBounds(msg)`
  check in `Update`'s `tea.MouseMsg` case, with the same `zone.DefaultManager == nil`
  nil-guard the Welcome Screen uses for a click that arrives before the first scan). A
  click on any icon activates it directly regardless of current keyboard `selected` state
  — consistent with how the Begin prompt already works.
- **Feed's sub-choice:** activating Feed does not immediately feed — it opens an inline
  two-option chooser, "Meal" / "Snack", using the identical selection pattern (Left/Right
  + Enter, plus direct `m`/`s` hotkeys, plus its own two mouse zones). Esc cancels back to
  the main icon bar without changing any stat. Model this as a small explicit UI state on
  the Screen (e.g. `menu menuState` where `menuState` is `menuIcons` or `menuFeedChoice`)
  rather than overloading `selected`'s meaning — keep the two menus' index spaces
  separate so a bug in one can't silently corrupt the other.
- **Help bar:** implement `tui.HelpProvider.ShortHelp()` (already used by the Welcome
  Screen) to surface the current context's keys — the icon bar's hints when at the top
  level, the Meal/Snack hints when the Feed sub-menu is open.
- **Feedback:** after an action resolves (Meal/Snack chosen, Play or Clean activated),
  return to the main icon bar and show the stat change take effect immediately in the
  meters. A short animation or text flourish (§4.2) is welcome but the state transition
  must be correct with or without it.

### 4.4 What does *not* change

- The two-clock split from Plan 01 (`anim.TickMsg` for cosmetics, `pet.TickMsg` for
  decay) is untouched — care actions are a third, purely event-driven path (`tea.KeyMsg`
  / `tea.MouseMsg`), not a new tick.
- Persistence: the existing periodic `pet.SaveCmd` from Plan 01 already captures the Pet
  after any mutation on the next simulation tick, and `OnQuit()` already flushes on quit.
  No new save-triggering is needed — but do double check that a care action doesn't need
  an *immediate* save to feel right (e.g. if a player feeds the Pet then kills the
  process before the next simulation tick, is losing that one action acceptable?
  Given Plan 01 already accepts up to one `TickInterval` of loss on crash — as opposed to
  clean quit, which is covered by `OnQuit` — treating a care action the same way is
  consistent and requires no new code. If you decide immediate-save-per-action is worth
  the extra I/O, that's a reasonable call too — just make it deliberately and say so in
  the PR description, don't leave it as an accidental inconsistency.)

## 5. Testing plan

Same five layers as Plan 01 (`CLAUDE.md` §Testing rules); this plan's tests build
directly on Plan 01's fixtures and fakes rather than inventing new ones.

**Unit — `internal/pet`:**
- `Feed(Meal)` / `Feed(Snack)`: stat deltas, `MaxStat` capping (table-driven: starting
  at 0, at `MaxStat-1`, at `MaxStat`), Snack's Weight increase.
- `Play()`: Happiness delta and capping.
- `Clean(now)`: `Mess` clears, `LastCleanedAt` updates.
- `Advance` extended cases: `Mess` becomes `true` after `MessInterval` since
  `LastCleanedAt`; Happiness decays faster while `Mess` is `true` than while it's
  `false` (a direct comparison test, not just "some decay happened"); a pre-Plan-02 save
  (zero-value `Mess`/`LastCleanedAt` semantics — i.e. `LastCleanedAt` defaulting to
  `CreatedAt`) behaves sanely on first load, not as if freshly cleaned at `time.Time{}`.

**Unit — `internal/tui/next`:**
- Selection state: Left/Right cycles and wraps across exactly the 3 icons; Enter on each
  activates the right action; `f`/`p`/`c` hotkeys activate directly from any starting
  `selected` value.
- Feed sub-menu: activating Feed opens it without changing stats; Meal/Snack selection
  applies the right mutator; Esc cancels without a stat change; the sub-menu's own
  Left/Right/Enter/hotkeys are independent of the outer icon bar's state.
- Mouse: a click on each icon zone activates it exactly like the matching key would (use
  the same `beginZone`-style test helper pattern as
  `internal/tui/welcome/welcome_test.go`, adapted for three zones instead of one); a click
  outside all zones does nothing.
- `ShortHelp()` returns the icon-bar hints at the top level and the Meal/Snack hints
  while that sub-menu is open.
- Mess renders when `Pet.Mess` is true and not otherwise.

**Integration (`teatest`):**
- A full sequence: launch, wait past `EggDuration` (via `pet.TickMsg`s, not sleeping),
  select Feed via keyboard, choose Snack, and assert both Happiness and Weight moved
  in the rendered view.
- Same for Play and for Clean-clears-Mess, driven the same way.
- A full mouse-only run of the same sequence, to prove keyboard and mouse paths reach
  identical end states (matches the project's "keyboard and mouse throughout" feature
  claim in `README.md`).

**Acceptance (given/when/then):** at minimum, one group per action:
- Given the Feed icon is selected, when Meal is chosen, then Hunger increases and Weight
  does not.
- Given the Feed icon is selected, when Snack is chosen, then Happiness and Weight both
  increase.
- Given the Play icon is activated, then Happiness increases.
- Given the Pet has a Mess, when Clean is activated, then the Mess indicator disappears
  and Happiness decay returns to its normal rate.
- Given a Mess is present and uncleaned, when time passes, then Happiness falls faster
  than it would without a Mess.

**Smoke test:** extend the existing pty-driven smoke test (or add a case) to drive at
least one full care action (e.g. select Feed, choose Meal, confirm) through the real
binary and assert the rendered terminal output reflects it — proving the icon bar's key
and render pipeline works end-to-end, not just against fakes.

## 6. Coverage, lint, and formatting

Same gate as Plan 01, no exceptions:
- `golangci-lint run ./...` clean. The Next Screen's `Update` is likely to grow past
  `funlen`'s 80-line/50-statement limit once it handles two menus plus mouse/keyboard —
  split the icon-bar and Feed-sub-menu handling into their own unexported methods rather
  than requesting a `//nolint`.
- `go test -race -covermode=atomic ./...` green; coverage thresholds in
  `.testcoverage.yml` still met (they ratchet upward — if this plan's changes push a
  package's coverage up, that's expected and fine; don't lower the file/package/total
  numbers).
- UK English in all new prose/help text/comments.

## 7. Documentation updates

- **`CONTEXT.md`**: add the terms from §2. Note the deliberate "Mess", not "poop",
  naming choice inline so a future contributor doesn't casually rename it.
- **`README.md`**: update "Features" (the full care loop now exists), the "Controls"
  table (add the new keys — Left/Right or `h`/`l`, Enter, `f`/`p`/`c`, `m`/`s`, Esc —
  described in terms of what they do on the Next Screen, matching the existing table's
  style), and confirm "Project layout" still matches (no new packages expected in this
  plan, only new files inside `internal/pet` and `internal/tui/next`).
- No new ADR is expected for this plan — it extends the seams Plan 01's ADR-0006 already
  established (the icon-bar/menu-selection pattern is ordinary Screen-local state, not a
  new architectural boundary). If, while implementing, the icon-bar/menu pattern turns
  out to be complex enough that a future Screen would want to reuse it as a shared
  component, that extraction is worth its own short ADR at that point — don't
  pre-emptively extract or document a generic "menu component" now on the strength of a
  single use site.

## 8. Deliverables checklist

- [ ] `internal/pet`: `Mess`, `LastCleanedAt` fields; `MessInterval` and the Happiness
      decay-multiplier constant; `Advance` extended; `FoodKind`, `Feed`, `Play`, `Clean`.
- [ ] `internal/art/mess.txt` (new); optional eating/playing pose art per §4.2.
- [ ] `internal/tui/next`: icon bar (keyboard, hotkeys, mouse), Feed sub-menu, updated
      `View`, updated `ShortHelp`.
- [ ] Full test suite per §5, all passing, coverage gate green.
- [ ] `CONTEXT.md`, `README.md` updated per §7.
- [ ] `golangci-lint run ./...`, `go test -race -covermode=atomic ./...`, and the
      coverage tool all pass locally before pushing (`lefthook`'s `pre-push` gate).
