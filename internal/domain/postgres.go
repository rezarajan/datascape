package domain

import "fmt"

// Postgres declares a Postgres-class database component.
type Postgres struct {
	Name             string
	Placement        Placement
	Credentials      SecretRef
	Guarantees       Guarantees
	AllowedConsumers []AllowedConsumer
}

// ComponentName implements Component.
func (p Postgres) ComponentName() string { return p.Name }

// Kind implements Component.
func (p Postgres) Kind() ComponentKind { return KindPostgres }

// ExternalRefs implements the externalReferencer interface
// (internal/domain/stack.go): guarantees.rpo.backupTo is the only
// external reference a Postgres component declares this week. Stack.
// Validate cross-checks every name this returns against the stack's own
// declared externals, refusing before compilation ever reaches an
// emitter that would otherwise fail to resolve it.
func (p Postgres) ExternalRefs() []string {
	if p.Guarantees.RPO != nil && p.Guarantees.RPO.BackupTo != "" {
		return []string{p.Guarantees.RPO.BackupTo}
	}
	return nil
}

// Validate reports every structural problem with p, aggregated rather
// than stopping at the first (golden rule 33: validate-time
// completeness). Both placements compile (week-two plan, slices 2+3):
// self-hosted to the Flux/CNPG target, managed to the Flux/tofu-controller
// target wrapping a Neon provider config. A guarantee whose meaning
// cannot survive the placement change refuses loudly here rather than
// silently degrading (golden rules 34, 37, 50) — mesh mTLS specifically
// cannot cover a provider-terminated endpoint outside the mesh.
// allowedConsumers no longer belongs to that list (week-four plan,
// slice 2): its own enforcement point is now egress compilation's
// waypoint-bound ServiceEntry authorization, which does cover a managed
// placement — see the allowedConsumers checks below.
func (p Postgres) Validate() []error {
	var errs []error

	if p.Name == "" {
		errs = append(errs, fmt.Errorf("postgres component: name is required"))
	}

	switch p.Placement {
	case PlacementSelfHosted:
		// compiles to the Flux/CNPG target
	case PlacementManaged:
		// compiles to the Flux/tofu-controller target; guarantee-specific
		// refusals for this placement are below, not here
	case "":
		errs = append(errs, fmt.Errorf(
			"postgres component %q: placement is required (self-hosted | managed)", p.Name))
	default:
		errs = append(errs, fmt.Errorf(
			"postgres component %q: unknown placement %q — must be self-hosted or managed",
			p.Name, p.Placement))
	}

	if p.Credentials.Name == "" {
		errs = append(errs, fmt.Errorf(
			"postgres component %q: credentials.secretRef.name is required (a Kubernetes Secret name — d7s never accepts an inline credential value)",
			p.Name))
	}

	errs = append(errs, p.Guarantees.Validate(p.Name)...)

	// guarantees.mtls + placement: managed refuses (week-two plan): the
	// transport-security guarantee is mesh mTLS (PeerAuthentication
	// STRICT) plus compiled authorization, which cannot cover a
	// provider-terminated endpoint outside the mesh — no best-effort TLS
	// substitution (golden rule 50).
	if p.Placement == PlacementManaged && p.Guarantees.MTLS != nil {
		errs = append(errs, fmt.Errorf(
			"postgres component %q: guarantees.mtls + placement: managed refuses to compile — "+
				"the transport-security guarantee is mesh mTLS (PeerAuthentication STRICT) plus "+
				"compiled authorization, and cannot cover a provider-terminated endpoint outside "+
				"the mesh; choose placement: self-hosted to keep guarantees.mtls, or remove "+
				"guarantees.mtls to keep placement: managed",
			p.Name))
	}

	// guarantees.rpo + placement: managed refuses (week-three plan,
	// unchanged from week-two): the durability guarantee now compiles
	// for self-hosted placement (a declared external destination wires a
	// barmanObjectStore into the CNPG Cluster), but the managed emitter
	// has no destination wiring this week — no best-effort tier (golden
	// rules 34, 37, 50). This refuses independently of whether
	// guarantees.rpo.backupTo is itself well-formed (see
	// Guarantees.Validate), so both problems are reported together when
	// both are present (golden rule 33).
	if p.Placement == PlacementManaged && p.Guarantees.RPO != nil {
		errs = append(errs, fmt.Errorf(
			"postgres component %q: guarantees.rpo + placement: managed refuses to compile — "+
				"the managed emitter has no backup-destination wiring this week; choose "+
				"placement: self-hosted to keep guarantees.rpo, or remove guarantees.rpo to keep "+
				"placement: managed",
			p.Name))
	}

	// allowedConsumers + placement: managed now compiles (week-four plan,
	// slice 2): this refusal used to promise "enforcement arrives with
	// egress compilation" — it has arrived. A managed component's
	// declared consumers now gate a waypoint-enforced ServiceEntry
	// AuthorizationPolicy scoped to the provider's own domain (Neon),
	// egress compilation's own enforcement point — not mesh mTLS, so it
	// needs no guarantees.mtls companion (guarantees.mtls + placement:
	// managed still refuses independently, above: permissioned egress is
	// not mesh mTLS, and no claim conflates the two). For self-hosted
	// placement the enforcement point is still the mesh
	// AuthorizationPolicy that only guarantees.mtls turns on, so that
	// requirement stands unchanged.
	if len(p.AllowedConsumers) > 0 && p.Placement == PlacementSelfHosted && p.Guarantees.MTLS == nil {
		errs = append(errs, fmt.Errorf(
			"postgres component %q: allowedConsumers declared without guarantees.mtls — there is no enforcement point without it; declare guarantees.mtls too",
			p.Name))
	}
	for i, consumer := range p.AllowedConsumers {
		if consumer.ServiceAccount == "" {
			errs = append(errs, fmt.Errorf(
				"postgres component %q: allowedConsumers[%d].serviceAccount is required",
				p.Name, i))
		}
	}

	return errs
}
