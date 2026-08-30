# Screen router with replace semantics

Screens are modelled behind a `Screen` interface (`Init`, `Update`, `View`, `ID`). A
root `App` model holds the single active screen, forwards messages to it, and acts on
navigation messages a screen emits (e.g. `NavigateMsg{To: ScreenID}`). Screens never
reference each other directly — they name a destination by `ScreenID` and the router
resolves it.

Navigation **replaces** the active screen; there is no history stack. Current flows
are linear (Welcome → Next). If back-navigation or modal overlays are needed later, a
stack will be added then and recorded in a follow-up ADR. A stack now would be
structure with no user-facing behaviour to justify it.
