# Pragmatic architecture: grow the boundaries

The brief calls for Clean Architecture, but the application has no domain yet — it is
pure presentation until pet-care mechanics arrive much later. Rather than stand up
empty `domain/`, `usecase/`, and `adapter/` layers now, we establish only the seams
that carry their weight today: a `Screen` abstraction with a router so screens are
pluggable (see ADR-0003), pure `Update`/`View` functions with no I/O so they are unit
-testable, and `internal/` for everything not meant as a public API. A `game`/domain
package will be introduced in the change that first needs it, with its own ADR.

Speculative empty layers are themselves a code smell; Clean Architecture's real
requirements — dependency direction pointing inward, and testable boundaries — are met
by the `Screen` seam without them.
