# Phase 2 — Welcome Screen

Goal: replace the Welcome Screen stub with the real thing — an ASCII **Wordmark**
with a one-pass **shine sweep**, a wandering **Character** (Marutchi), and a
pulsing begin prompt — without changing the router. Fully tested.

Branch: `phase-2-welcome-screen`. Starts after Phase 1 CI is green.

## Foundations

- [x] `internal/anim`: injectable clock + frame source; `TickMsg` on `tea.Tick`
      at ~15 FPS; easing/interpolation helpers (`Lerp`, `Clamp01`, `Triangle`,
      `Pulse`, `Elapsed`). Deterministic under test.
- [x] `internal/art`: `//go:embed` loader for `.txt` art (`Load`/`MustLoad`),
      `Width`, `Mirror`; golden-file friendly.
- [x] Add `bubblezone`; `App.View` runs `zone.Scan` so named click zones work.

## Wordmark

- [x] Hand-author `internal/art/wordmark.txt` — the word "Tamagotchi", 6 rows,
      ≤ 72 columns (69), rounded style.
- [x] Render in a single adaptive teal (`Palette.Shell`).
- [x] Shine sweep: 3-column brighter band (`Palette.Highlight`), left→right,
      ~800 ms, linear, once on Screen entry, then static. Not replayed on resize
      (it is a pure function of the frame counter, which resizes never touch).
- [x] Unit-test the sweep frame maths against the injected clock; golden-file the
      first, mid, and final frames.

## Character (Marutchi)

- [x] Hand-author 2 walk frames (`marutchi-walk-1.txt`, `marutchi-walk-2.txt`);
      the opposite facing is `art.Mirror` of those.
- [x] Wander: horizontal back-and-forth in a 40-column centred box below the
      Wordmark; ±1-row bob; ~1 cell per 2 frames; faces travel direction; starts
      centred, walks right first; loops forever.
- [x] Unit-test position/facing as a pure function of frame count.

## Begin prompt

- [x] Text "Press Enter or click to begin", ~1 s raised-cosine brightness pulse
      quantised to three steps.
- [x] Register a `bubblezone` region for exactly the prompt text; only a click in
      that zone advances (Phase 1's click-anywhere is gone).
- [x] Keep Enter and Ctrl+Q/Ctrl+C behaviour.

## Layout

- [x] Centred vertical stack: Wordmark → gap → Character box → gap → prompt.
- [x] Decision: the Welcome Screen is **full-bleed** — it fills the body area and
      returns `Scrollable() == false`. Scrollable text Screens (Next) still
      render at natural height. Recorded in ADR-0004; no `internal/tui` change.
- [x] Verified the resize notice still fires below 80×24 and the viewport path
      still works for the Next Screen.

## Tests

- [x] Unit: sweep maths, character path, prompt pulse, art loader/mirror, anim
      helpers.
- [x] Integration (`teatest`): the Character wanders with no input; Enter and a
      centred prompt click both navigate; a corner click does not.
- [x] Golden files for the first/mid/final sweep frames; `-update` workflow
      documented in `README.md` and `CLAUDE.md`.
- [x] Acceptance: one given/when/then group per Welcome Screen sentence in the
      brief (`internal/tui/welcome/acceptance_test.go`).
- [x] Coverage 96.5% total; gate raised 90 → 93.

## Docs

- [x] `CONTEXT.md`: added Marutchi, Begin prompt, Frame, Frame source, Walk
      cycle, Bob, Click zone.
- [x] ADR-0004 (full-bleed Screens) and ADR-0005 (bubblezone).
- [x] `README.md`: text preview of the Welcome Screen (terminal recording still
      TODO — needs a human at a real terminal); Phase 1 and 2 rows flipped to
      Complete.
