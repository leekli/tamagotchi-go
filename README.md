# Tamagotchi Go

[![CI](https://github.com/leekli/tamagotchi-go/actions/workflows/ci.yml/badge.svg)](https://github.com/leekli/tamagotchi-go/actions/workflows/ci.yml)
![Coverage](https://img.shields.io/badge/coverage-%E2%89%A593%25-brightgreen)
![Go](https://img.shields.io/badge/go-1.26-00ADD8)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue)](LICENSE)

A command-line pet game in the style of the original first-generation Tamagotchi
toy (1996–1997), built in Go with the [Bubble Tea](https://github.com/charmbracelet/bubbletea)
TUI framework.

```
   ____   ____  _    _  ____   ____   ____   ____   ____  _    _  ____
 (_  _) ( __ ) |\  /| ( __ ) / ___) / __ \ (_  _) / ___) | |  | (_  _)
    ||   | || | | \/ | | || | | / _  | |  |   ||   | / __ | |__|   ||
    ||   |(__)| | /\ | |(__)| | (_ ) | |  |   ||   | \__  |  __|   ||
    ||   | || | |/  \| | || | \ __/  \ __ /   ||   \ ___) | |  |  _||_
   ()   |_||_| |_||_| |_||_|  \__)   \__/    ()    \__)  |_|  | (____)

                             ,--.
                            ( o o)
                             >`-'

              Press Enter or click to begin
```


## Features

- A screen-routed TUI that clears the terminal on entry and restores it on exit.
- A hand-authored ASCII wordmark with a one-pass shine sweep, a wandering
  animated Character, and a pulsing begin prompt — all on a deterministic,
  test-driven frame clock (`internal/anim`).
- Keyboard **and** mouse throughout; clicks are matched to precise on-screen
  regions with [bubblezone](https://github.com/lrstanley/bubblezone).
- Scrolls gracefully in small terminals; shows a resize hint below 80×24.
- Adaptive colour that reads on light and dark terminals, with `NO_COLOR` and
  `--no-color` support.

## Controls

| Key | Action |
|-----|--------|
| <kbd>Enter</kbd>, or a click on the begin prompt | Begin (advance from the Welcome Screen) |
| <kbd>↑</kbd> <kbd>↓</kbd> <kbd>PgUp</kbd> <kbd>PgDn</kbd> <kbd>Home</kbd> <kbd>End</kbd> / wheel | Scroll (on scrollable Screens) |
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
tests written as given/when/then subtests. Animation is deterministic — tests
feed `anim.TickMsg`s rather than sleeping. The coverage gate
(`.testcoverage.yml`) requires ≥ 93% overall and ratchets upward.

Representative shine-sweep frames are locked with golden files under
`internal/tui/welcome/testdata/`. After an intentional change to the wordmark
or the sweep, regenerate them:

```sh
go test ./internal/tui/welcome -run TestWordmarkGoldenFrames -update
```

## Architecture

The entrypoint (`cmd/tamagotchi-go`) is a one-liner over `internal/cli`, which
parses flags and starts the program. `internal/tui` holds the `App` — an
Elm-style root model that owns exactly one active `Screen`, routes messages to
it, and applies navigation. Navigation **replaces** the active Screen; Screens
communicate only by emitting a `NavigateMsg` naming a `ScreenID`, so they never
depend on each other. The `App` wraps each Screen in shared chrome: a resize
notice when the terminal is too small, an optional scrolling viewport, and a
one-line help bar. It also scans each composed frame for
[bubblezone](https://github.com/lrstanley/bubblezone) markers so Screens can
name precise click targets.

Two small support packages back the Welcome Screen: `internal/anim` (a
fixed-rate frame clock and easing helpers, injectable so animation is
deterministic under test) and `internal/art` (a `//go:embed` loader for the
hand-authored ASCII art, plus a left-right mirror helper).

Decisions with lasting consequences are recorded in
[`docs/adr/`](docs/adr/). The domain vocabulary is defined in
[`CONTEXT.md`](CONTEXT.md).

## Project layout

```
cmd/tamagotchi-go/      entrypoint
internal/cli/           command-line argument wiring
internal/tui/           App router, Screen interface, shared styles and keys
internal/tui/welcome/   Welcome Screen (wordmark, shine sweep, Character, prompt)
internal/tui/next/      Next Screen (placeholder)
internal/anim/          fixed-rate frame clock and easing helpers
internal/art/           embedded ASCII art loader and mirror helper
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
