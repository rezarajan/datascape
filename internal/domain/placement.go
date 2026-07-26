package domain

// Placement declares where a component's infrastructure is provisioned.
// It is a declared, compiler-validated binding (problem definition
// Amendment 1, A2) — not an architecture rewrite.
type Placement string

const (
	// PlacementSelfHosted compiles to the Flux/Kubernetes target — the
	// only placement week one can honor.
	PlacementSelfHosted Placement = "self-hosted"
	// PlacementManaged is a reserved value: week one accepts it in the
	// schema but always refuses to compile it (golden rule 34 — a
	// schema-accepted field nothing consumes is a defect, so this one
	// consumes it by refusing loudly, not by silently ignoring it).
	PlacementManaged Placement = "managed"
)
