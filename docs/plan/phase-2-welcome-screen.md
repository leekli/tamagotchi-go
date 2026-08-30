# Phase 2 — Welcome Screen

Goal: replace the Welcome Screen stub with the real thing — an ASCII **Wordmark**
with a one-pass **shine sweep**, a wandering **Character** (Marutchi), and a
pulsing begin prompt — without changing the router. Fully tested.

Branch: `phase-2-welcome-screen`. Starts after Phase 1 CI is green.

## Foundations

- [ ] `internal/anim`: injectable clock + frame source; `tickMsg` on `tea.Tick`
      at ~15 FPS; easing/interpolation helpers. Deterministic under test.
- [ ] `internal/art`: `//go:embed` loader for `.txt` art; golden-file friendly.
- [ ] Add `bubblezone`; wrap `App.View`/screen views so named click zones work.

## Wordmark

- [ ] Hand-author `internal/art/wordmark.txt` — the word "Tamagotchi", ~6 rows,
      ≤ 72 columns, styled after the rounded 1997 logo.
- [ ] Render in a single adaptive teal (`Palette.Shell`).
- [ ] Shine sweep: 2–3 column brighter band (`Palette.Highlight`), left→right,
      ~800 ms, linear, once on Screen entry, then static. Not replayed on resize.
- [ ] Unit-test the sweep frame maths against the injected clock; golden-file the
      first, mid, and final frames.

## Character (Marutchi)

- [ ] Hand-author 2 walk frames + mirrored turn in `internal/art/`.
- [ ] Wander: horizontal back-and-forth in a ~40-column centred box below the
      Wordmark; ±1-row bob; ~1 cell per 2–3 frames; faces travel direction;
      starts centred, walks right first; loops forever.
- [ ] Unit-test position/facing as a pure function of frame count.

## Begin prompt

- [ ] Text "Press Enter or click to begin", slow ~1 s brightness pulse.
- [ ] Register a `bubblezone` region for exactly the prompt text; only a click in
      that zone advances (tighten Phase 1's click-anywhere).
- [ ] Keep Enter and Ctrl+Q/Ctrl+C behaviour.

## Layout

- [ ] Centred vertical stack: Wordmark → gap → Character box → gap → prompt.
- [ ] Decide: scrollable Screens render natural height (viewport scrolls when the
      content is taller than the body area) vs. fill the area. Update
      `internal/tui` and ADR if the contract changes.
- [ ] Verify the resize notice and viewport scroll still behave with real content.

## Tests

- [ ] Unit: sweep maths, character path, prompt pulse, art loader.
- [ ] Integration (`teatest`): sweep completes; character moves between frames;
      Enter and a prompt-zone click both navigate; a click outside the zone does
      not.
- [ ] Golden files for representative frames; documented `-update` workflow.
- [ ] Acceptance: one given/when/then group per Welcome Screen sentence in the
      brief.
- [ ] Coverage stays ≥ the ratcheted threshold.

## Docs

- [ ] Update `CONTEXT.md` if new terms appear (e.g. frame source, walk cycle).
- [ ] ADR if the Screen-render contract or an added dependency warrants one.
- [ ] README: replace the demo-GIF TODO with a real recording; flip the Phase 2
      row to done.
