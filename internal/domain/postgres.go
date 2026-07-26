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

// Validate reports every structural problem with p, aggregated rather
// than stopping at the first (golden rule 33: validate-time
// completeness). Both placements compile (week-two plan, slices 2+3):
// self-hosted to the Flux/CNPG target, managed to the Flux/tofu-controller
// target wrapping a Neon provider config. A guarantee whose meaning
// cannot survive the placement change refuses loudly here rather than
// silently degrading (golden rules 34, 37, 50) — mesh mTLS and the
// AuthorizationPolicy allow-list it depends on cannot cover a
// provider-terminated endpoint outside the mesh.
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

	if len(p.AllowedConsumers) > 0 {
		if p.Placement == PlacementManaged {
			// allowedConsumers + placement: managed refuses (week-two
			// plan): the allow-list compiles to a mesh AuthorizationPolicy,
			// which cannot gate a provider-terminated endpoint outside the
			// mesh — a schema-accepted field nothing consumes here would
			// be a defect otherwise (golden rule 34).
			errs = append(errs, fmt.Errorf(
				"postgres component %q: allowedConsumers + placement: managed refuses to compile — "+
					"a consumer allow-list compiles to a mesh AuthorizationPolicy, which cannot gate "+
					"a provider-terminated endpoint outside the mesh; remove allowedConsumers, or "+
					"choose placement: self-hosted (enforcement for managed placement arrives with "+
					"egress compilation, skeleton scope)",
				p.Name))
		} else if p.Guarantees.MTLS == nil {
			errs = append(errs, fmt.Errorf(
				"postgres component %q: allowedConsumers declared without guarantees.mtls — there is no enforcement point without it; declare guarantees.mtls too",
				p.Name))
		}
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
