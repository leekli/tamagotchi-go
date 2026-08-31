# Full-bleed Screens opt out of the scrolling viewport

Phase 1 left one question open: should a Screen's `View` return its content at
natural height (and let the App's viewport scroll when it overflows the body
area), or should it fill the body area itself? Phase 2 answers it per Screen
rather than globally, keeping the `Scrollable` opt-out from ADR-0003.

The Welcome Screen is **full-bleed**: it is authored to fit the minimum 80×24
terminal, it centres a fixed vertical stack in the body area with
`lipgloss.Place`, and it animates every frame. Wrapping that in a viewport would
add a scrollbar it never needs and a second source of truth for its size. So it
returns `Scrollable() == false` and the App frames it without a viewport; the
existing resize notice still guards terminals below 80×24.

Text Screens whose content can genuinely outgrow the body area — the Next Screen
today, longer informational Screens later — keep `Scrollable() == true` and
render at natural height. The contract in `internal/tui` already describes both
modes; no code there changed. New Screens choose: authored-to-fit and animated
→ full-bleed; free-flowing text → scrollable.
