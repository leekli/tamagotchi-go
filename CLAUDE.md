# CLAUDE.md

Guidance for AI agents working in this repository. Humans should read
`README.md`; the domain glossary is `CONTEXT.md`; architecture decisions are in
`docs/adr/`.

## What this project is

A CLI/TUI game in the style of the original first-generation Tamagotchi toy
(1996–1997), written in Go with the Bubble Tea / Charm ecosystem. It is built in
phases (`docs/plan/`): Phase 1 is the scaffold and a functional-stub Welcome
Screen; Phase 2 fills in the Welcome Screen's art and animation. Pet-care
mechanics come later.

## Commands

| Task | Command |
|------|---------|
| Run the game | `go run ./cmd/tamagotchi-go` |
| Build | `go build ./...` |
| Unit + integration tests | `go test ./...` |
| Full test run (as CI) | `go test -race -covermode=atomic ./...` |
| Skip the slow binary smoke test | `go test -short ./...` |
| Coverage report | `go test -covermode=atomic -coverprofile=cover.out ./... && go tool cover -func=cover.out` |
| Update golden files | `go test ./internal/tui/welcome -run TestWordmarkGoldenFrames -update` |
| Lint (blocking) | `golangci-lint run ./...` |
| Auto-format | `golangci-lint fmt ./...` |
| Vulnerability scan | `go run golang.org/x/vuln/cmd/govulncheck@latest ./...` |
| Install git hooks | `lefthook install` |

## The pre-push gate

`lefthook.yml` defines two hooks. `pre-commit` runs `gofmt` + `go vet`.
`pre-push` runs the full gate: format check, `go vet`, `golangci-lint`, race
tests, `govulncheck`. **Never bypass a hook with `--no-verify`.** If a check is
wrong, fix the check and say why in the commit.

## Architecture

- **Grow the boundaries** (ADR-0002): build the seams that pay off now, no empty
  layers. A domain package arrives with the first feature that needs it.
- **Screen router** (ADR-0003): every full-terminal state implements the
  `tui.Screen` interface (`ID`/`Init`/`Update`/`View`/`Scrollable`). The
  `tui.App` root model owns one active Screen, routes messages to it, and acts on
  `tui.NavigateMsg`. Navigation **replaces** the active Screen — there is no
  history stack. Screens never import one another; they name a destination by
  `ScreenID`.
- `App` frames every Screen with shared chrome: a resize notice below 80×24, an
  optional scrolling viewport, and a help bar on the last row. It forwards a
  `WindowSizeMsg` carrying the body area (height minus the help-bar row), and
  runs each composed frame through `zone.Scan` (bubblezone) so Screens can name
  click targets with `zone.Mark` / `zone.Get(id).InBounds` (ADR-0005).
- **Full-bleed vs scrollable** (ADR-0004): a Screen authored to fit 80×24 that
  paints every cell returns `Scrollable() == false` (the Welcome Screen);
  free-flowing text Screens that can overflow return `true` (the Next Screen).
- **Animation**: `internal/anim` is the frame clock — `Tick`/`TickMsg` at
  `anim.FPS` (~15). A Screen advances an integer frame counter on each `TickMsg`
  and re-issues `Tick`; all motion is a pure function of that counter, so tests
  drive it by sending `TickMsg`s, never by sleeping.
- **Art**: `internal/art` embeds the hand-authored `.txt` files (`//go:embed`)
  and exposes `Load`/`MustLoad`, `Width`, and `Mirror`. Art is ASCII only.
- Layout: `cmd/tamagotchi-go` (entrypoint), `internal/cli` (arg wiring),
  `internal/tui` (router + shared chrome), `internal/tui/<screen>` (one package
  per Screen), `internal/anim` (frame clock), `internal/art` (embedded art).

## Testing rules

- Five layers: **unit** (pure funcs, each Screen's `Update`/`View`),
  **integration** (`teatest`, scripted input, real `tea.Program`), **smoke**
  (built binary under a pty), **acceptance** (given/when/then subtests per
  user-facing requirement). No Gherkin.
- Assertions use `testify` (`require` to stop the test, `assert` to continue).
- **No `time.Sleep` in tests as a synchronisation tool.** Wait on a condition
  (`teatest.WaitFor`, `require.Eventually`, a polling helper). Animation advances
  by feeding `anim.TickMsg`s, so frame maths is asserted directly.
- Golden files: representative shine-sweep frames live under
  `internal/tui/welcome/testdata/`. Regenerate intentionally with
  `go test ./internal/tui/welcome -run TestWordmarkGoldenFrames -update`; never
  `-update` blindly to make a failing test pass.
- Coverage gate: `.testcoverage.yml`, total ≥ 93%, ratcheting upward. `main.go`
  is excluded. Do not lower a threshold.

## Code style

- `gofumpt` + `goimports` (local prefix `github.com/leekli/tamagotchi-go`),
  applied by `golangci-lint fmt`.
- Prose (comments, docs, help strings) is **UK English** ("colour",
  "behaviour"). Library identifiers such as `lipgloss.Center` and the
  conventional `--no-color` flag stay as they are.
- An inline `//nolint` needs a specific linter name and an explanation.

## Commit & PR rules

- **Conventional Commits**: `feat:`, `fix:`, `chore:`, `docs:`, `test:`, `ci:`,
  `refactor:`, `build:`. Imperative mood, lower-case subject.
- **No AI attribution anywhere.** Do not add `Co-Authored-By: Claude`, a
  "Generated with Claude Code" line, or any Claude/Anthropic mention to commit
  messages or pull request descriptions.
- Work happens on a branch per phase/feature; `main` stays green.
- Use the vocabulary from `CONTEXT.md` in code and prose. If you need a new term,
  add it there in the same change.

## Agent skills

### Issue tracker

Issues and specs live as GitHub issues in `leekli/tamagotchi-go`, managed via the
`gh` CLI. See `docs/agents/issue-tracker.md`.

### Domain docs

Single-context: `CONTEXT.md` and `docs/adr/` at the repo root. See
`docs/agents/domain.md`.
