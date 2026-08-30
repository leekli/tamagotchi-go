# Phase 1 — Scaffold

Goal: a project that builds, tests, lints, and launches to a working (art-free)
Welcome Screen that navigates to a placeholder Next Screen and quits cleanly —
with the full quality gate in place. No Tamagotchi art or animation yet.

Branch: `phase-1-scaffold`. Done when every box is ticked and CI is green.

## Module & layout

- [x] `go mod init github.com/leekli/tamagotchi-go`, `go 1.26.6`
- [x] `cmd/tamagotchi-go/main.go` — one-liner over `internal/cli`
- [x] `internal/cli` — flag parsing (`--version`, `--no-color`, `--help`),
      `NO_COLOR` support, program start behind a testable seam
- [x] `.gitignore` covers build output and coverage files

## Screen router (`internal/tui`)

- [x] `Screen` interface: `ID`, `Init`, `Update`, `View`, `Scrollable`
- [x] `ScreenID` constants + `AllScreenIDs` for wiring tests
- [x] `NavigateMsg` + `Navigate` command
- [x] `App` root model: owns active Screen, routes messages, replace-semantics
      navigation, panics on an unregistered start screen
- [x] Shared chrome: resize notice below 80×24, scrolling viewport (opt-out via
      `Scrollable`), help bar on the last row
- [x] `App` forwards a body-area `WindowSizeMsg` (height minus help bar) to Screens
- [x] `KeyMap` with Ctrl+Q / Ctrl+C quit; `HelpProvider` optional interface
- [x] Adaptive `Palette` + `Styles` in one file

## Screens

- [x] `internal/tui/welcome` — stub: plain title + begin prompt, Enter/click →
      Next Screen, contributes a help hint, scrollable
- [x] `internal/tui/next` — placeholder text, ignores input, scrollable

## Tests

- [x] Unit: `App` (navigation, quit, resize notice, viewport scroll, help bar,
      unknown-ID ignored, start-panic), both Screens, `Navigate`, `cli.run`
- [x] Integration: `teatest` welcome → next → quit; resize-notice recovery
- [x] Smoke: build the binary, drive it through a pty, Enter then Ctrl+Q
- [x] Wiring: every `ScreenID` resolves to a factory that builds the right Screen
- [x] Coverage ≥ 90% total (currently ~97%)

## Quality gate

- [x] `.golangci.yml` (v2, curated strict set + gofumpt/goimports) — `0 issues`
- [x] `lefthook.yml` — `pre-commit` (fmt, vet), `pre-push` (fmt, vet, lint, race
      test, govulncheck)
- [x] `.testcoverage.yml` — thresholds, `main.go` excluded
- [x] `.github/workflows/ci.yml` — lint, test (ubuntu + macos) + coverage gate +
      artifact, build, govulncheck, gitleaks, `go mod tidy` check
- [ ] CI green on the opened PR

## Docs

- [x] `CONTEXT.md` glossary
- [x] `docs/adr/0001`–`0003`
- [x] `CLAUDE.md` (incl. no-AI-attribution commit rule)
- [x] `README.md` (badges, controls, layout, trademark disclaimer)
- [x] `docs/plan/phase-1-scaffold.md`, `docs/plan/phase-2-welcome-screen.md`

## Deliberately deferred to Phase 2

- `internal/art` (embedded ASCII), `internal/anim` (clock/frame source)
- `bubblezone` for precise click targets — Phase 1 accepts click-anywhere on the
  Welcome Screen
- Whether scrollable Screens render natural height vs. filling the body area
