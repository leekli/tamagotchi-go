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
