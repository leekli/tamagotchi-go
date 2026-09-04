# Tamagotchi Go

A CLI/TUI game in the style of the original first-generation Tamagotchi toy (1996–1997),
written in Go with the Bubble Tea / Charm ecosystem. This document is the project
glossary: it fixes the words we use so code, docs, and conversation agree.

## Language

### Screens and navigation

**Screen**:
A full-terminal state the player occupies. Exactly one Screen is active at a time;
new features are added as new Screens.
_Avoid_: Scene, View, Page, Route

**Welcome Screen**:
The first Screen shown on launch: the ASCII wordmark, the animated Character, and a
prompt to begin. Navigating away from it hands control to the Next Screen.
_Avoid_: Splash (informal use is fine), Home, Menu, Landing

**Next Screen**:
The Screen the player reaches from the Welcome Screen. It shows the Pet: its
Stage art with a small idle animation, its Hunger and Happiness meters, and
its Age and Weight.
_Avoid_: Placeholder Screen (describes its former state, not its identity), Game Screen

### On-screen art

**Wordmark**:
The word "Tamagotchi" rendered as hand-authored ASCII art, coloured at runtime.
_Avoid_: logo, title, banner, header

**Shine sweep**:
The single left-to-right highlight pass that travels across the Wordmark once when
the Welcome Screen appears, then stops.
_Avoid_: shine, sheen, shimmer, shame, wipe, glint

**Character**:
The small, non-interactive animated ASCII creature on the Welcome Screen. It is
decorative: the player cannot act on it. Distinct from the Pet.
_Avoid_: sprite, mascot, avatar, pet

**Marutchi**:
The specific Character shown on the Welcome Screen: the round first-generation
form. Named after the toy's まるっち. Used when we mean this particular creature
rather than the Character role.
_Avoid_: blob, baby, Marutchy

**Begin prompt**:
The "Press Enter or click to begin" line beneath the Character, with its slow
brightness pulse. It is the only click target on the Welcome Screen.
_Avoid_: start button, CTA, call to action

### Animation

**Frame**:
One animation step. The frame clock advances one frame per tick, ~15 per second.
On-screen motion is a pure function of the frame count, so it is deterministic
under test.
_Avoid_: tick (that is the message), step (ambiguous with the walk cycle)

**Frame source** / **animation clock**:
`internal/anim`: the fixed-rate `Tick` command and `TickMsg`. A Screen animates
by advancing a counter on each `TickMsg` and re-issuing `Tick`. Tests feed
`TickMsg`s instead of sleeping.
_Avoid_: timer, ticker, game loop

**Walk cycle**:
The two hand-authored Marutchi poses (`walk-1`, `walk-2`) alternated as the
Character moves, plus their mirror image for the opposite facing.
_Avoid_: walk animation, gait, frames (bare)

**Bob**:
The ±1-row vertical wobble applied to the Character as it walks.
_Avoid_: hop, bounce, jump

**Shine sweep** (already defined above): the one-pass highlight across the
Wordmark.

**Click zone**:
A named rectangular region registered with `bubblezone` that a mouse event can
be tested against. The begin prompt is the Welcome Screen's only click zone.
_Avoid_: hitbox, hotspot, target (bare)

**Pet**:
The creature the player raises. Distinct from the Character (decorative,
Welcome Screen only).
_Avoid_: Tamagotchi (ambiguous with the application's name), Character, creature

### The Pet

**Stage**:
The Pet's life stage: Egg or Baby, for now. (Child/Teen/Adult evolution is a
later feature.)
_Avoid_: level, form (informal use of "Marutchi form" for the Character is fine)

**Hatch**:
The one-time transition from Egg to Baby, a fixed real-time interval after
the Pet is first created (`CreatedAt`, i.e. birth) — not the moment the Pet
struct is instantiated in memory on every load.
_Avoid_: spawn, level up

**Stat**:
One of the Pet's numeric attributes: Hunger, Happiness, or Weight. Hunger and
Happiness Decay over time; Weight is fixed at birth for now — a later feature
makes it dynamic.
_Avoid_: meter (that is the pip display), stat point

**Decay**:
The automatic, time-driven reduction of Hunger and Happiness. Decay is a pure
function of elapsed wall-clock time, not of frames or player action, so it
keeps moving whether or not the game is running.
_Avoid_: drain, tick down

**Save file**:
The single JSON file holding the Pet's persisted state between runs.
_Avoid_: save slot, profile (there is only ever one Pet, one save)
