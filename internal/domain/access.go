package domain

// AllowedConsumer names a Kubernetes ServiceAccount permitted to reach a
// component's declared enforcement point: guarantees.mtls's mesh
// AuthorizationPolicy for a self-hosted component, or egress
// compilation's waypoint-bound ServiceEntry AuthorizationPolicy for a
// managed one (week-four plan, slice 2 — internal/adapters/flux/
// egress.go; managed placement has no mesh mTLS to depend on, since a
// provider-terminated endpoint sits outside the mesh). It is the
// smallest expression of golden rule 53's reference graph until a
// second component kind exists to wire full cross-component references
// against — the compiled allow-list comes only from this declared list,
// never a default (golden rule 53). Declaring zero consumers is legal:
// the component stays reachable by nothing, which is the correct
// default-deny state until a consumer is declared.
type AllowedConsumer struct {
	ServiceAccount string
	// Namespace defaults to the declaring component's own stack
	// namespace when empty.
	Namespace string
}
