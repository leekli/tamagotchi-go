# Tamagotchi Go

[![CI](https://github.com/leekli/tamagotchi-go/actions/workflows/ci.yml/badge.svg)](https://github.com/leekli/tamagotchi-go/actions/workflows/ci.yml)
![Coverage](https://img.shields.io/badge/coverage-%E2%89%A593%25-brightgreen)
![Go](https://img.shields.io/badge/go-1.26-00ADD8)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue)](LICENSE)

A command-line pet game in the style of the original first-generation Tamagotchi
toy (1996–1997), built in Go with the [Bubble Tea](https://github.com/charmbracelet/bubbletea)
TUI framework.

<div style="display: flex;">
  <img src="https://cdn-icons-png.flaticon.com/512/9480/9480360.png" width="64" height="64">
  <img src="https://cdn-icons-png.flaticon.com/512/2532/2532690.png" width="64" height="64">
  <img src="https://cdn-icons-png.flaticon.com/512/2532/2532579.png" width="64" height="64">
</div> 

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
```

> [!NOTE]
> This project is built using `Claude Code`. This is a project to aid my own learning of using Agentic AI tooling 🤖

## Features

- A living, persisted Pet on the Next Screen: it's born an Egg, hatches into a
  Baby, grows into a Child, a Teen, and an Adult, and eventually dies of old
  age, or sooner of starvation or untreated sickness if you neglect it — its
  current Stage always shown as a label beneath its art — while its
  Hunger and Happiness decay over real elapsed time, whether or not the game
  is running. State persists between runs (`internal/pet`). Once it has
  died, a panel says why and hatching a brand new Egg is one keypress or click
  away, whatever the cause and even if it died while the game was closed.
- A full Care loop once the Pet has hatched: Feed it a Meal or a Snack, Play
  with it, Clean up a Mess it's left uncleaned for too long, or Cure it — each
  reachable by keyboard or mouse, with immediate feedback in the stat meters,
  and unaffected by growing from Baby all the way to Adult (once the Pet has
  died, there's nothing left to Feed, Play with, or Clean). Play also works
  off Weight gained from Snacking; let it climb too high and the Pet becomes
  Overfed, which speeds up Happiness decay the same way an uncleaned Mess
  does.
- Neglect can make the Pet Sick: leave a Mess long enough and a `[+] Sick`
  indicator appears in its STATS panel. Cleaning up does not cure it — only the
  Cure action does (the fourth Icon bar tab, or <kbd>u</kbd>), and a Pet Cured
  while its Mess is still there falls Sick again a while later. A Sick Pet can
  still be fed and played with, but left Sick for eight minutes without a Cure
  it dies of Sickness.
- Neglect is tracked behind the scenes: leave Hunger or Happiness at 0 for
  longer than a short grace window and the Pet racks up a Care mistake (one per
  Stat per Empty spell, on a lifetime tally that isn't shown to you). Refilling
  the Stat in time — a Meal for Hunger, a Snack or Play for Happiness — spares
  you, and a Sick Pet's grace window is paused until you Cure it. The tally
  isn't shown and has no consequence yet; it is groundwork for what neglect
  will cost. What does have a cost already: a Pet whose Hunger stays at 0 for
  ten minutes starves, even while Sick. One never cleaned dies of Sickness
  first, at 18 minutes old.
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

| Key                                                                                              | Action                                                                                 |
| ------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------- |
| <kbd>Enter</kbd>, or a click on the begin prompt                                                 | Begin (advance from the Welcome Screen)                                                |
| <kbd>↑</kbd> <kbd>↓</kbd> <kbd>PgUp</kbd> <kbd>PgDn</kbd> <kbd>Home</kbd> <kbd>End</kbd> / wheel | Scroll (on scrollable Screens)                                                         |
| <kbd>←</kbd> <kbd>→</kbd> / <kbd>h</kbd> <kbd>l</kbd>, or a click on an icon                     | Select a Care action or Feed's Meal/Snack choice (Next Screen, once the Pet has hatched) |
| <kbd>Enter</kbd>                                                                                 | Activate the selected icon or choice (Next Screen)                                     |
| <kbd>f</kbd> <kbd>p</kbd> <kbd>c</kbd> <kbd>u</kbd>                                              | Feed, Play, Clean, or Cure directly, from any selection (Next Screen)                  |
| <kbd>m</kbd> <kbd>s</kbd>                                                                        | Choose Meal or Snack directly, once Feed's chooser is open (Next Screen)               |
| <kbd>Esc</kbd>                                                                                   | Cancel Feed's Meal/Snack chooser (Next Screen); quit (Welcome Screen)                  |
| <kbd>Ctrl</kbd>+<kbd>C</kbd>                                                                     | Quit (any Screen)                                                                      |

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

Flags: `--version`, `--no-color`, `--save-path` (override the save file
location, mainly for testing), `--help`.

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

```mermaid
flowchart TD

subgraph group_entry["Entry and Wiring"]
  node_cmd["CLI Binary<br/>[main.go]"]
  node_cli["CLI Runner<br/>[cli.go]"]
end

subgraph group_orchestration["TUI Orchestration"]
  node_app["TUI App<br/>[app.go]"]
end

subgraph group_screens["Game Screens"]
  node_welcome["Welcome Screen<br/>[welcome.go]"]
  node_next["Next Screen<br/>[next.go]"]
end

subgraph group_domain["Pet Domain"]
  node_pet["Pet Lifecycle<br/>[pet.go]"]
  node_simulation["Decay Simulation<br/>[decay.go]"]
  node_care["Care Actions<br/>[actions.go]"]
  node_neglect["Neglect Tracking<br/>[neglect.go]"]
  node_store[("Save File<br/>[store.go]")]
end

subgraph group_support["Rendering Support"]
  node_anim["Frame Clock<br/>[anim.go]"]
  node_art["ASCII Art<br/>[art.go]"]
  node_clicks["Click Targets<br/>[zone.go]"]
  node_styles["Adaptive Styles<br/>[styles.go]"]
end

node_player(("Player"))

node_player -->|"launches"| node_cmd
node_cmd -->|"invokes"| node_cli
node_cli -->|"loads save"| node_store
node_store -->|"returns Pet"| node_cli
node_cli -->|"creates or advances"| node_pet
node_cli -->|"builds and starts"| node_app
node_app -->|"routes messages"| node_welcome
node_app -->|"routes messages"| node_next
node_welcome -->|"emits navigation"| node_app
node_welcome -->|"schedules ticks"| node_anim
node_welcome -->|"loads artwork"| node_art
node_welcome -->|"reads clicks"| node_clicks
node_next -->|"schedules ticks"| node_anim
node_next -->|"loads artwork"| node_art
node_next -->|"reads stage"| node_pet
node_next -->|"advances time"| node_simulation
node_next -->|"applies care"| node_care
node_next -->|"saves Pet"| node_store
node_next -->|"reads clicks"| node_clicks
node_simulation -->|"counts lapses"| node_neglect
node_app -->|"scans frames"| node_clicks
node_app -->|"uses chrome"| node_styles
node_welcome -->|"uses styling"| node_styles
node_next -->|"uses styling"| node_styles

click node_cmd "https://github.com/leekli/tamagotchi-go/blob/main/cmd/tamagotchi-go/main.go"
click node_cli "https://github.com/leekli/tamagotchi-go/blob/main/internal/cli/cli.go"
click node_app "https://github.com/leekli/tamagotchi-go/blob/main/internal/tui/app.go"
click node_welcome "https://github.com/leekli/tamagotchi-go/blob/main/internal/tui/welcome/welcome.go"
click node_next "https://github.com/leekli/tamagotchi-go/blob/main/internal/tui/next/next.go"
click node_pet "https://github.com/leekli/tamagotchi-go/blob/main/internal/pet/pet.go"
click node_simulation "https://github.com/leekli/tamagotchi-go/blob/main/internal/pet/decay.go"
click node_care "https://github.com/leekli/tamagotchi-go/blob/main/internal/pet/actions.go"
click node_neglect "https://github.com/leekli/tamagotchi-go/blob/main/internal/pet/neglect.go"
click node_store "https://github.com/leekli/tamagotchi-go/blob/main/internal/pet/store.go"
click node_anim "https://github.com/leekli/tamagotchi-go/blob/main/internal/anim/anim.go"
click node_art "https://github.com/leekli/tamagotchi-go/blob/main/internal/art/art.go"
click node_clicks "https://github.com/leekli/tamagotchi-go/blob/main/internal/tui/zone.go"
click node_styles "https://github.com/leekli/tamagotchi-go/blob/main/internal/tui/styles.go"

classDef toneNeutral fill:#f8fafc,stroke:#334155,stroke-width:1.5px,color:#0f172a
classDef toneBlue fill:#dbeafe,stroke:#2563eb,stroke-width:1.5px,color:#172554
classDef toneAmber fill:#fef3c7,stroke:#d97706,stroke-width:1.5px,color:#78350f
classDef toneMint fill:#dcfce7,stroke:#16a34a,stroke-width:1.5px,color:#14532d
classDef toneRose fill:#ffe4e6,stroke:#e11d48,stroke-width:1.5px,color:#881337
classDef toneIndigo fill:#e0e7ff,stroke:#4f46e5,stroke-width:1.5px,color:#312e81
classDef toneTeal fill:#ccfbf1,stroke:#0f766e,stroke-width:1.5px,color:#134e4a
class node_cmd,node_cli toneBlue
class node_app toneAmber
class node_welcome,node_next toneMint
class node_pet,node_simulation,node_care,node_neglect,node_store toneRose
class node_anim,node_art,node_clicks,node_styles,node_player toneIndigo
```

Two small support packages back the Welcome Screen: `internal/anim` (a
fixed-rate frame clock and easing helpers, injectable so animation is
deterministic under test) and `internal/art` (a `//go:embed` loader for the
hand-authored ASCII art, plus a left-right mirror helper).

`internal/pet` is the game's first domain package: the `Pet` type, its
derived Stage and Age, a pure `Advance(now)` Decay function, and a `Store`
interface (file-backed in production) for persistence. It performs no direct
I/O itself — loading happens once at startup in `internal/cli`, and saving
from the running Next Screen happens via a deferred command, on its own slow
simulation clock and again on quit.

Decisions with lasting consequences are recorded in
[`docs/adr/`](docs/adr/). The domain vocabulary is defined in
[`CONTEXT.md`](CONTEXT.md).

## Project layout

```
cmd/tamagotchi-go/      entrypoint
internal/cli/           command-line argument wiring
internal/tui/           App router, Screen interface, shared styles and keys
internal/tui/welcome/   Welcome Screen (wordmark, shine sweep, Character, prompt)
internal/tui/next/      Next Screen (the Pet: art, meters, age, weight, Care actions)
internal/pet/           Pet domain model: Stage, Decay, Care actions, and persistence
internal/anim/          fixed-rate frame clock and easing helpers
internal/art/           embedded ASCII art loader and mirror helper
docs/adr/               architecture decision records
.github/workflows/      CI pipeline
```

## Licence

[MIT](LICENSE).

---

_Unofficial fan project. Not affiliated with, endorsed by, or sponsored by
Bandai. "Tamagotchi" is a registered trademark of Bandai._
