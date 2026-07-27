package domain

import (
	"fmt"
	"strings"
)

// endpointHostSuffix is the Neon provider's own domain (verified against
// Neon's connection docs, 2026-07-26/27: every real endpoint host is a
// subdomain of it, e.g. ep-<name>.<region>.aws.neon.tech). EndpointHost
// validates against this suffix so a pin cannot be an arbitrary hostname
// masquerading as a Neon endpoint (golden rules 34, 50).
const endpointHostSuffix = "neon.tech"

// Postgres declares a Postgres-class database component.
type Postgres struct {
	Name             string
	Placement        Placement
	Credentials      SecretRef
	Guarantees       Guarantees
	AllowedConsumers []AllowedConsumer
	// EndpointHost pins the exact Neon endpoint host a managed
	// component's declared allowedConsumers are authorized to reach
	// through the mesh (week-four plan, 2026-07-27 finding → Revision
	// 2 — internal/adapters/flux/egress.go's doc comment on
	// neonPostgresPort). It is only knowable after tofu-controller
	// first provisions the component and writes it to the "host" key
	// of the Secret named by Credentials.Name (terraform.go's
	// WriteOutputsToSecret) — d7s never reads that Secret itself (rule
	// 51); the operator reads it and pins the value here. Declaring
	// allowedConsumers on a managed component without this pin refuses
	// (see Validate below): Istio ambient waypoints cannot actually
	// program a wildcard TLS ServiceEntry (the capability is alpha,
	// upstream-gated, and was evaluated and declined — no best-effort
	// security tier), so exact-host pinning is a two-step ceremony
	// (provision unpinned, read the host, pin, recompile) rather than a
	// wildcard domain-pattern compile. Only meaningful for placement:
	// managed — declaring it on placement: self-hosted refuses, since
	// nothing would consume it there (rule 37: no schema-accepted field
	// nothing consumes).
	EndpointHost string
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
// placement — see the allowedConsumers checks below. It does, however,
// need endpointHost pinned first on managed placement (2026-07-27
// finding → Revision 2): the compiled ServiceEntry is exact-host, not
// wildcard, and the exact host is runtime-bound.
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

	// endpointHost structural validation (week-four plan, 2026-07-27
	// finding → Revision 2): only meaningful for placement: managed
	// (rule 37 — a schema-accepted field nothing consumes is a defect),
	// and must be a bare hostname (no scheme/port/path) within the
	// provider's own domain — a pin outside it cannot honestly compile
	// an exact-host ServiceEntry (golden rules 34, 50).
	if p.EndpointHost != "" {
		switch {
		case p.Placement != PlacementManaged:
			errs = append(errs, fmt.Errorf(
				"postgres component %q: endpointHost only applies to placement: managed — remove it, or set placement: managed",
				p.Name))
		case strings.Contains(p.EndpointHost, "://"):
			errs = append(errs, fmt.Errorf(
				"postgres component %q: endpointHost %q must be a bare hostname, not a URL — drop the scheme "+
					"(e.g. \"ep-xxx.us-east-2.aws.neon.tech\", not \"https://ep-xxx...\")",
				p.Name, p.EndpointHost))
		case strings.ContainsAny(p.EndpointHost, "/:"):
			errs = append(errs, fmt.Errorf(
				"postgres component %q: endpointHost %q must be a bare hostname with no port or path",
				p.Name, p.EndpointHost))
		case p.EndpointHost != endpointHostSuffix && !strings.HasSuffix(p.EndpointHost, "."+endpointHostSuffix):
			errs = append(errs, fmt.Errorf(
				"postgres component %q: endpointHost %q is outside %s — pin the exact host read from the "+
					"written-outputs secret (credentials.secretRef.name %q, key \"host\"), not an arbitrary hostname",
				p.Name, p.EndpointHost, endpointHostSuffix, p.Credentials.Name))
		}
	}

	// allowedConsumers + placement: managed WITHOUT the pin refuses
	// (week-four plan, 2026-07-27 finding → Revision 2, supersedes the
	// wildcard domain-pattern design): the compiled exact-host
	// ServiceEntry needs the exact host, which is runtime-bound (only
	// known after first provisioning) — so the first compile of a new
	// managed component with declared consumers cannot honestly emit
	// one. The remedy is the full two-step ceremony (rule 35), not just
	// a pointer to endpointHost's own doc comment.
	if len(p.AllowedConsumers) > 0 && p.Placement == PlacementManaged && p.EndpointHost == "" {
		errs = append(errs, fmt.Errorf(
			"postgres component %q: allowedConsumers declared without endpointHost — pin the exact endpoint "+
				"first: compile this component without allowedConsumers, deliver it, read the \"host\" key from "+
				"the Secret named by credentials.secretRef.name (%q) once tofu-controller writes it, add "+
				"endpointHost: <that value> to the declaration, then recompile and redeliver",
			p.Name, p.Credentials.Name))
	}

	return errs
}
