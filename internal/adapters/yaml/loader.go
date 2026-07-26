// Package yaml is the declaration-loader adapter: it parses a YAML
// declaration into a domain.Stack. It implements ports.Loader and imports
// only domain and ports (golden rule 8) plus the YAML library — kept out
// of the domain package so domain stays free of third-party imports.
package yaml

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/rezarajan/datascape/internal/domain"
	"github.com/rezarajan/datascape/internal/ports"
)

var _ ports.Loader = (*Loader)(nil)

// SupportedAPIVersion and SupportedKind are the only declaration document
// header this loader accepts. A mismatch refuses loudly rather than
// guessing (golden rule 34).
const (
	SupportedAPIVersion = "d7s.dev/v1alpha1"
	SupportedKind       = "Stack"
)

type rawStack struct {
	APIVersion string         `yaml:"apiVersion"`
	Kind       string         `yaml:"kind"`
	Name       string         `yaml:"name"`
	Components []rawComponent `yaml:"components"`
}

type rawComponent struct {
	Kind        string          `yaml:"kind"`
	Name        string          `yaml:"name"`
	Placement   string          `yaml:"placement"`
	Credentials *rawCredentials `yaml:"credentials"`
	Guarantees  *rawGuarantees  `yaml:"guarantees"`
}

type rawCredentials struct {
	SecretRef *rawSecretRef `yaml:"secretRef"`
}

type rawSecretRef struct {
	Name string `yaml:"name"`
}

type rawGuarantees struct {
	MTLS *rawMTLS `yaml:"mtls"`
	RPO  *string  `yaml:"rpo"`
}

// rawMTLS is an empty marker: it decodes only from an empty mapping
// (`mtls: {}`). Any other YAML node kind (e.g. `mtls: false`) fails to
// decode — the schema has no representable way to declare mTLS "off"
// (golden rule 50).
type rawMTLS struct{}

// Loader parses a YAML declaration document.
type Loader struct{}

// New builds a Loader.
func New() *Loader { return &Loader{} }

// Load implements ports.Loader. Unknown fields anywhere in the document
// refuse loudly (golden rule 34) rather than being silently dropped.
func (l *Loader) Load(raw []byte) (domain.Stack, error) {
	dec := yaml.NewDecoder(bytes.NewReader(raw))
	dec.KnownFields(true)
	var doc rawStack
	if err := dec.Decode(&doc); err != nil {
		return domain.Stack{}, fmt.Errorf("parse declaration: %w", err)
	}

	var errs []error
	if doc.APIVersion != SupportedAPIVersion {
		errs = append(errs, fmt.Errorf(
			"declaration: unsupported apiVersion %q — expected %q", doc.APIVersion, SupportedAPIVersion))
	}
	if doc.Kind != SupportedKind {
		errs = append(errs, fmt.Errorf(
			"declaration: unsupported kind %q — expected %q", doc.Kind, SupportedKind))
	}

	stack := domain.Stack{Name: doc.Name}
	for i, rc := range doc.Components {
		switch domain.ComponentKind(rc.Kind) {
		case domain.KindPostgres:
			pg, cErrs := toPostgres(rc)
			errs = append(errs, cErrs...)
			stack.Components = append(stack.Components, pg)
		case "":
			errs = append(errs, fmt.Errorf("declaration: components[%d]: kind is required", i))
		default:
			errs = append(errs, fmt.Errorf(
				"declaration: components[%d] %q: unknown kind %q — planned kinds beyond postgres are not yet available",
				i, rc.Name, rc.Kind))
		}
	}

	if len(errs) > 0 {
		return domain.Stack{}, errors.Join(errs...)
	}
	return stack, nil
}

func toPostgres(rc rawComponent) (domain.Postgres, []error) {
	var errs []error
	pg := domain.Postgres{
		Name:      rc.Name,
		Placement: domain.Placement(rc.Placement),
	}
	if rc.Credentials != nil && rc.Credentials.SecretRef != nil {
		pg.Credentials = domain.SecretRef{Name: rc.Credentials.SecretRef.Name}
	}
	if rc.Guarantees != nil {
		if rc.Guarantees.MTLS != nil {
			pg.Guarantees.MTLS = &domain.MTLSGuarantee{}
		}
		if rc.Guarantees.RPO != nil {
			d, err := time.ParseDuration(*rc.Guarantees.RPO)
			if err != nil {
				errs = append(errs, fmt.Errorf(
					"postgres component %q: guarantees.rpo %q is not a valid duration (e.g. \"1h\", \"15m\"): %w",
					rc.Name, *rc.Guarantees.RPO, err))
			} else {
				pg.Guarantees.RPO = &domain.RPOGuarantee{Target: d}
			}
		}
	}
	return pg, errs
}
