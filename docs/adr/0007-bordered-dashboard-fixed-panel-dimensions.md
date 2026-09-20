# Fixed per-panel dimensions for the Bordered Dashboard layout

The Bordered Dashboard redesign wraps both Screens' content in bordered
Panels instead of today's bare centred stack. Rather than let each Panel
size itself to whatever content a given render happens to produce — which
is exactly what produced misaligned borders in the design exploration
itself — every Panel type gets one fixed width and height, derived from
its real worst-case content, and reused verbatim on every state that shows
it: PET 18×10 (Adult's 14-column art, the widest of any Stage), STATS
23×10 (the Hunger/Happiness Meter row), the Icon bar row 31 columns
(shared by the 3-tab and 2-tab menus), the Death panel 43×15, and
Welcome's card 75×18. The PET+STATS row and the Death panel both total
43×15; this is treated as one deliberate content envelope for the whole
Next Screen rather than a coincidence, so a state that consolidates
Panels still occupies the same footprint as one that shows them side by
side.

Consequences: a future addition to any state must fit inside its Panel's
fixed dimensions or trigger a deliberate re-derivation of the constant —
and everything that shares it — rather than silently reflowing.

## Update: the envelope grows to 43×17 to make room for neglect

The consequences above anticipated this: the Pet is gaining a Sick indicator, an
Attention call, a cause of Death, a Care mistake tally and a fourth Care action
(Cure). Each needs a place on screen, and every existing row was already used, so
the constants are re-derived deliberately, in one change, rather than squeezing
new content in or letting it reflow.

- PET and STATS are 12 rows tall (was 10), still 18 and 23 columns wide. STATS
  gains two content rows (8 in all): one reserved for the Sick indicator and one
  for the Attention call, blank until they exist. The Mess row and both spacer
  rows stay. PET's art and label simply centre in the taller panel.
- The Death panel is 17 rows (was 15), still 43 wide, with two of its 13 content
  rows reserved blank for the cause of Death and the Care mistake tally.
- The Icon bar row is 42 columns (was 31): four 9-column tabs and three 2-column
  gaps. The three-tab bar and the two-tab Meal/Snack chooser both centre within
  it, so switching between them still shifts nothing.
- The shared envelope is therefore 43×17: the PET+STATS row (12) plus the gap (1),
  the tab row (3) and the flourish row (1), and the Death panel, which is as tall
  as that whole composite. It still fits the 80×24 minimum terminal's 23-row body
  with room to spare, so the minimum does not change.

The reserved rows are blank on purpose: this change adds no visible content, so the
screen looks as it did, only taller, and later changes fill rows in without moving
anything. Because `View` is centred to the whole body, a raw line count is the same
for every state; the tests therefore measure the occupied rows and columns to prove
each state, Egg included, occupies the same envelope.

## Update: the Attention call fills its reserved row

Every row reserved above is now in use: the Sick indicator, the cause of Death
and the Care mistake tally arrived with their features, and the Attention call is
the last. The STATS row reserved for it now carries it: `(!) Hungry`,
`(!) Sad` or both, and `(!!) HUNGRY & SAD` once the Grace windows have lapsed. The
longest form is 17 columns, inside the panel's 19-column content width, so it needs
no wrapping and shifts nothing; a unit test pins that every combination fits. The
Sick indicator's row above it and the spacer below stay as they were.
