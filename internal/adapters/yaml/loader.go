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
	// External is d7s's source()-style declaration (problem definition
	// Amendment 2): a named resource d7s never provisions or mutates.
	// Named "external" (singular), matching Amendment 2's own naming
	// ("the external declaration", "the stack's external block").
	External []rawExternal `yaml:"external"`
}

// rawExternal is the only external shape v1 declares (an S3-compatible
// object store) — see domain.External's doc comment for how a future
// external kind would extend this.
type rawExternal struct {
	Name        string          `yaml:"name"`
	ObjectStore *rawObjectStore `yaml:"objectStore"`
}

type rawObjectStore struct {
	Endpoint    string          `yaml:"endpoint"`
	Bucket      string          `yaml:"bucket"`
	Region      string          `yaml:"region"`
	Credentials *rawCredentials `yaml:"credentials"`
}

type rawComponent struct {
	Kind             string               `yaml:"kind"`
	Name             string               `yaml:"name"`
	Placement        string               `yaml:"placement"`
	Credentials      *rawCredentials      `yaml:"credentials"`
	Guarantees       *rawGuarantees       `yaml:"guarantees"`
	AllowedConsumers []rawAllowedConsumer `yaml:"allowedConsumers"`
}

type rawAllowedConsumer struct {
	ServiceAccount string `yaml:"serviceAccount"`
	Namespace      string `yaml:"namespace"`
}

type rawCredentials struct {
	SecretRef *rawSecretRef `yaml:"secretRef"`
}

type rawSecretRef struct {
	Name string `yaml:"name"`
}

type rawGuarantees struct {
	MTLS *rawMTLS `yaml:"mtls"`
	RPO  *rawRPO  `yaml:"rpo"`
}

// rawRPO is guarantees.rpo's nested shape (week-three plan, slices
// 1+2): a bare duration string is no longer enough once the guarantee
// also needs a destination reference, so rpo becomes an object —
// target (the recovery point objective) and backupTo (the declared
// external's name).
type rawRPO struct {
	Target   string `yaml:"target"`
	BackupTo string `yaml:"backupTo"`
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
	for _, re := range doc.External {
		stack.Externals = append(stack.Externals, toExternal(re))
	}
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
			d, err := time.ParseDuration(rc.Guarantees.RPO.Target)
			if err != nil {
				errs = append(errs, fmt.Errorf(
					"postgres component %q: guarantees.rpo.target %q is not a valid duration (e.g. \"1h\", \"15m\"): %w",
					rc.Name, rc.Guarantees.RPO.Target, err))
			} else {
				pg.Guarantees.RPO = &domain.RPOGuarantee{
					Target:   d,
					BackupTo: rc.Guarantees.RPO.BackupTo,
				}
			}
		}
	}
	for _, rac := range rc.AllowedConsumers {
		pg.AllowedConsumers = append(pg.AllowedConsumers, domain.AllowedConsumer{
			ServiceAccount: rac.ServiceAccount,
			Namespace:      rac.Namespace,
		})
	}
	return pg, errs
}

// toExternal converts a parsed rawExternal into its domain form. No
// parse-time errors can arise here (every field is a plain string) —
// structural completeness (required fields, an unrecognized/absent
// store shape) is domain.External.Validate's job, called via
// Stack.Validate, exactly like every other component's structural
// validation.
func toExternal(re rawExternal) domain.External {
	ext := domain.External{Name: re.Name}
	if re.ObjectStore != nil {
		os := &domain.ObjectStoreExternal{
			Endpoint: re.ObjectStore.Endpoint,
			Bucket:   re.ObjectStore.Bucket,
			Region:   re.ObjectStore.Region,
		}
		if re.ObjectStore.Credentials != nil && re.ObjectStore.Credentials.SecretRef != nil {
			os.Credentials = domain.SecretRef{Name: re.ObjectStore.Credentials.SecretRef.Name}
		}
		ext.ObjectStore = os
	}
	return ext
}
