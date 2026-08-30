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
The Screen the player reaches from the Welcome Screen. For now it holds placeholder
text and nothing else; its real content is a later feature.
_Avoid_: Placeholder Screen (describes its current state, not its identity), Game Screen

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

**Pet**:
The creature the player will raise, feed, and care for. A later feature — absent from
the first two delivery phases.
_Avoid_: Tamagotchi (ambiguous with the application's name), Character, creature
