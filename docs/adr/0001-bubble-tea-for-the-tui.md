# Bubble Tea (Charm ecosystem) for the TUI

The project brief mandates a terminal UI built with Bubble Tea and its sibling
libraries (Lip Gloss for styling, Bubbles for reusable components). We accept this:
Bubble Tea's Elm-style `Model`/`Update`/`View` loop gives pure, testable rendering
functions, and the ecosystem ships a golden-file test harness (`x/exp/teatest`).
The main alternative, `tview`/`tcell`, uses mutable widget trees that are harder to
unit-test. All Charm dependencies share a release cadence, limiting version drift.
