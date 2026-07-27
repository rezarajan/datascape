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
	// mTLS — refuses to compile against this placement instead of
	// silently degrading (golden rules 34, 37, 50); see
	// internal/domain/postgres.go. allowedConsumers no longer refuses
	// here (week-four plan, slice 2): its enforcement point is now
	// egress compilation's waypoint-bound ServiceEntry authorization
	// (internal/adapters/flux/egress.go), a mechanism that does cover a
	// provider-terminated endpoint — permissioned egress is not mesh
	// mTLS, and no claim conflates the two.
	PlacementManaged Placement = "managed"
)
