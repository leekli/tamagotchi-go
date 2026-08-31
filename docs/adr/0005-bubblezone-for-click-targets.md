# bubblezone for click targets

Phase 1 accepted a click *anywhere* as "begin", with a TODO to tighten it.
Phase 2 needs a click to count only when it lands on the begin prompt itself.
Computing that rectangle by hand means tracking the prompt's position through
centring and padding on every resize — fragile, and re-derived in tests.

We add `github.com/lrstanley/bubblezone`. A Screen wraps a clickable fragment in
`zone.Mark(id, s)`; the App's root `View` runs the composed frame through
`zone.Scan`, which records each marked region's real coordinates and strips the
markers from the output. On a `tea.MouseMsg` the Screen asks
`zone.Get(id).InBounds(msg)`. The manager is process-wide (`zone.NewGlobal`,
started once from `NewApp`); lookups are nil-safe, so a click that arrives
before the first scan is simply ignored.

This is a presentation-layer dependency that removes hand-maintained geometry,
consistent with ADR-0002 (add a seam when a feature needs it). The marker
sequences are zero-width and ignored by Lip Gloss's width maths, so layout is
unaffected. If a future Screen needs many zones it can namespace ids with
`zone.NewPrefix()`.
