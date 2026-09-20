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

### Layout

**Panel**:
A labelled, bordered region within a Screen's layout, grouping related content and setting it apart from sibling Panels. Captioned only when a sibling Panel on the same Screen needs distinguishing from it; a Screen's sole Panel stays uncaptioned.
_Avoid_: box, card, widget, container

**PET panel**:
The Panel showing the Pet's art and Stage label on the Next Screen, for every Stage except Death.
_Avoid_: art panel, creature panel

**STATS panel**:
The Panel showing the Pet's Hunger and Happiness Meters, Mess (when present), and Age/Weight, beside the PET panel on the Next Screen, for every Stage except Death.
_Avoid_: info panel, meter panel

**Death panel**:
The single, uncaptioned Panel shown once the Pet has reached Death, consolidating its art, Stage label, Age/Weight, and the Restart prompt — never split into a PET panel and STATS panel the way earlier Stages are.
_Avoid_: a stretched PET panel

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
The Pet's life stage: Egg, Baby, Child, Teen, Adult, or Death. Adult is the
last Stage the Pet grows into; Death follows it automatically after a fixed
interval, the same way every earlier Stage transition already works. Shown
on the Next Screen as a bare-word label beneath the Pet's art.
_Avoid_: level, form (informal use of "Marutchi form" for the Character is fine)

**Hatch**:
The one-time transition from Egg to Baby, a fixed real-time interval after
the Pet is first created (`CreatedAt`, i.e. birth) — not the moment the Pet
struct is instantiated in memory on every load.
_Avoid_: spawn, level up

**Stat**:
One of the Pet's numeric attributes: Hunger, Happiness, or Weight. Hunger and
Happiness Decay over time; Weight never Decays — it only ever changes as the
direct result of a Care action (Snack increases it, Play decreases it), between
a floor of its birth weight and a fixed maximum.
_Avoid_: meter (that is the pip display), stat point

**Decay**:
The automatic, time-driven reduction of Hunger and Happiness. Decay is a pure
function of elapsed wall-clock time, not of frames or player action, so it
keeps moving whether or not the game is running. It is also exact: one long
absence and the same span passed in many short steps leave the Pet identical.
Happiness Decays faster while the Pet has Mess or is Overfed; when that rate
changes partway through a point (Mess appearing, or a Care action ending it),
the progress already made toward the next point is kept in proportion rather
than restarted.
_Avoid_: drain, tick down

**Save file**:
The single JSON file holding the Pet's persisted state between runs.
_Avoid_: save slot, profile (there is only ever one Pet, one save)

**Mess**:
The uncleaned state the Pet is left in once enough time has passed since it
was last cleaned. While present it depresses Happiness Decay further; the
Clean Care action removes it. Deliberately not called "poop" — keep the term
politely abstract in code, comments, and UI copy; the small pile glyph shown
on screen can still read as one.
_Avoid_: poop

**Overfed**:
The state a Pet enters once its Weight reaches a threshold above its birth
weight, from choosing Snack too often. While present it depresses Happiness
Decay further, the same way Mess does — the two share one accelerated rate
rather than compounding when both apply at once. The Play Care action is
what brings Weight back down and resolves it. Deliberately named for the
cause (too much Snack), not the Pet's body.
_Avoid_: fat, chubby, overweight

**Empty**:
A Hunger or Happiness Stat at 0.
_Avoid_: depleted, drained

**Empty spell**:
One unbroken run of a Stat being Empty. It begins the instant Decay takes the Stat to 0 and ends when a Care action lifts it above 0; a later run is a new spell.
_Avoid_: episode, neglect episode

**Grace window**:
The fixed interval a Stat may stay Empty, counted from the start of its Empty spell, before that spell costs a Care mistake.
_Avoid_: timeout, deadline

**Care mistake**:
The cost of an Empty spell whose Grace window expires without the Stat being refilled: one per spell, per Stat, however long the spell lasts. Mistakes are counted on a single tally kept from birth.
_Avoid_: care miss, strike, penalty

**Attention call**:
The Pet's signal that a Stat is Empty, in one of two phases: the Grace window still running, or lapsed (the Care mistake counted). Only an Empty Hunger or Happiness calls; a Mess does not.
_Avoid_: alert, notification

**Care action**:
A player-triggered command on the Next Screen that changes the Pet's Stats:
Feed, Play, or Clean.
_Avoid_: command, verb

**Icon bar**:
The row of selectable Care-action icons the Next Screen shows once the Pet
has hatched into a Baby, navigated by keyboard or mouse. It disappears again
once the Pet has reached Death.
_Avoid_: menu bar, toolbar

**Meal** / **Snack**:
The two choices offered by the Feed Care action. Meal restores Hunger with no
side effect; Snack restores Happiness but adds Weight.

**Death**:
The terminal Stage a Pet reaches after living as an Adult for a fixed
real-time interval. Once reached, Care actions and their Icon bar are no
longer available, and Mess no longer shows even if present — the only
action left to the player is a Restart.
_Avoid_: dead, dying, game over

**Restart**:
The player-triggered action, available only once the Pet has reached Death,
that discards it and creates a brand new Egg in its place — a full
replacement of the single Save file's contents, not a partial reset of any
one Stat. Distinct from Hatch: Hatch is the automatic, time-driven
Egg→Baby transition; Restart is the one manual action that produces a fresh
Egg to begin with.
_Avoid_: reset, retry, new game
