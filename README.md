# Tamagotchi Go

[![CI](https://github.com/leekli/tamagotchi-go/actions/workflows/ci.yml/badge.svg)](https://github.com/leekli/tamagotchi-go/actions/workflows/ci.yml)
![Coverage](https://img.shields.io/badge/coverage-%E2%89%A590%25-brightgreen)
![Go](https://img.shields.io/badge/go-1.26-00ADD8)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue)](LICENSE)

A command-line pet game in the style of the original first-generation Tamagotchi
toy (1996–1997), built in Go with the [Bubble Tea](https://github.com/charmbracelet/bubbletea)
TUI framework.

<!-- TODO: demo GIF of the Welcome Screen once Phase 2 lands -->

## Status

Early development, built in phases:

| Phase | Scope | State |
|-------|-------|-------|
| 1 | Project scaffold, screen router, functional-stub Welcome Screen, placeholder Next Screen | In progress |
| 2 | Welcome Screen art: ASCII wordmark + shine sweep, animated character, begin prompt | Planned |
| Later | Pet-care mechanics | Not started |

See [`docs/plan/`](docs/plan/) for the task breakdowns.

## Features

- A screen-routed TUI that clears the terminal on entry and restores it on exit.
- Keyboard **and** mouse throughout.
- Scrolls gracefully in small terminals; shows a resize hint below 80×24.
- Adaptive colour that reads on light and dark terminals, with `NO_COLOR` and
  `--no-color` support.

## Controls

| Key | Action |
|-----|--------|
| <kbd>Enter</kbd> / left click | Begin (advance from the Welcome Screen) |
| <kbd>↑</kbd> <kbd>↓</kbd> <kbd>PgUp</kbd> <kbd>PgDn</kbd> <kbd>Home</kbd> <kbd>End</kbd> / wheel | Scroll |
| <kbd>Ctrl</kbd>+<kbd>Q</kbd> (or <kbd>Ctrl</kbd>+<kbd>C</kbd>) | Quit |

## Requirements

- Go 1.26 or newer
- A terminal at least 80×24, with ANSI support

## Install and run

From source:

```sh
git clone https://github.com/leekli/tamagotchi-go.git
cd tamagotchi-go
go run ./cmd/tamagotchi-go
```

Or install the binary onto your `PATH`:

```sh
go install github.com/leekli/tamagotchi-go/cmd/tamagotchi-go@latest
tamagotchi-go
```

Flags: `--version`, `--no-color`, `--help`.

## Testing

```sh
go test ./...                                   # unit + integration
go test -race -covermode=atomic ./...           # the full run, as CI does it
go test -short ./...                             # skip the binary smoke test
go test -covermode=atomic -coverprofile=cover.out ./... && go tool cover -func=cover.out
```

Layers: unit tests on pure functions and each Screen's `Update`/`View`;
integration tests driving the real Bubble Tea program with
[`teatest`](https://github.com/charmbracelet/x/tree/main/exp/teatest); a smoke
test that builds the binary and drives it through a pseudo-terminal; acceptance
tests written as given/when/then subtests. The coverage gate
(`.testcoverage.yml`) requires ≥ 90% overall and ratchets upward.

## Architecture

The entrypoint (`cmd/tamagotchi-go`) is a one-liner over `internal/cli`, which
parses flags and starts the program. `internal/tui` holds the `App` — an
Elm-style root model that owns exactly one active `Screen`, routes messages to
it, and applies navigation. Navigation **replaces** the active Screen; Screens
communicate only by emitting a `NavigateMsg` naming a `ScreenID`, so they never
depend on each other. The `App` wraps each Screen in shared chrome: a resize
notice when the terminal is too small, an optional scrolling viewport, and a
one-line help bar.

Decisions with lasting consequences are recorded in
[`docs/adr/`](docs/adr/). The domain vocabulary is defined in
[`CONTEXT.md`](CONTEXT.md).

## Project layout

```
cmd/tamagotchi-go/      entrypoint
internal/cli/           command-line argument wiring
internal/tui/           App router, Screen interface, shared styles and keys
internal/tui/welcome/   Welcome Screen
internal/tui/next/      Next Screen (placeholder)
docs/adr/               architecture decision records
docs/plan/              per-phase task breakdowns
.github/workflows/      CI pipeline
```

## Contributing

`lefthook install` sets up the pre-commit and pre-push hooks. Commits follow
[Conventional Commits](https://www.conventionalcommits.org/). See
[`CLAUDE.md`](CLAUDE.md) for the working agreement.

## Licence

[MIT](LICENSE).

---

*Unofficial fan project. Not affiliated with, endorsed by, or sponsored by
Bandai. "Tamagotchi" is a registered trademark of Bandai.*
