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
// completeness). Unimplemented paths — placement: managed — refuse
// loudly here rather than being silently accepted (golden rule 34).
func (p Postgres) Validate() []error {
	var errs []error

	if p.Name == "" {
		errs = append(errs, fmt.Errorf("postgres component: name is required"))
	}

	switch p.Placement {
	case PlacementSelfHosted:
		// the only placement week one compiles
	case PlacementManaged:
		errs = append(errs, fmt.Errorf(
			"postgres component %q: placement \"managed\" is planned, not yet available — use placement: self-hosted",
			p.Name))
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

	if len(p.AllowedConsumers) > 0 && p.Guarantees.MTLS == nil {
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
