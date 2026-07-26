package domain

// Placement declares where a component's infrastructure is provisioned.
// It is a declared, compiler-validated binding (problem definition
// Amendment 1, A2) — not an architecture rewrite.
type Placement string

const (
	// PlacementSelfHosted compiles to the Flux/CloudNativePG target.
	PlacementSelfHosted Placement = "self-hosted"
	// PlacementManaged compiles to the Flux/tofu-controller target,
	// wrapping a Neon provider config (week-two plan, slices 2+3). A
	// guarantee whose meaning cannot survive the placement change — mesh
	// mTLS and the AuthorizationPolicy allow-list it depends on — refuses
	// to compile against this placement instead of silently degrading
	// (golden rules 34, 37, 50); see internal/domain/postgres.go.
	PlacementManaged Placement = "managed"
)
