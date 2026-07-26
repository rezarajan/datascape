package yaml_test

import (
	"strings"
	"testing"

	"github.com/rezarajan/datascape/internal/adapters/yaml"
	"github.com/rezarajan/datascape/internal/domain"
)

const validDoc = `
apiVersion: d7s.dev/v1alpha1
kind: Stack
name: week-one
components:
  - kind: postgres
    name: orders-db
    placement: self-hosted
    credentials:
      secretRef:
        name: orders-db-app
    guarantees:
      mtls: {}
    allowedConsumers:
      - serviceAccount: probe-client
`

func TestLoadValidDocument(t *testing.T) {
	stack, err := yaml.New().Load([]byte(validDoc))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stack.Name != "week-one" {
		t.Errorf("stack name = %q, want week-one", stack.Name)
	}
	if len(stack.Components) != 1 {
		t.Fatalf("components = %d, want 1", len(stack.Components))
	}
	pg, ok := stack.Components[0].(domain.Postgres)
	if !ok {
		t.Fatalf("component type = %T, want domain.Postgres", stack.Components[0])
	}
	if pg.Guarantees.MTLS == nil {
		t.Error("expected mtls guarantee to be declared")
	}
	if pg.Guarantees.RPO != nil {
		t.Errorf("expected no rpo guarantee declared, got %+v", pg.Guarantees.RPO)
	}
	if pg.Credentials.Name != "orders-db-app" {
		t.Errorf("credentials name = %q, want orders-db-app", pg.Credentials.Name)
	}
	if len(pg.AllowedConsumers) != 1 || pg.AllowedConsumers[0].ServiceAccount != "probe-client" {
		t.Errorf("allowedConsumers = %+v, want one consumer probe-client", pg.AllowedConsumers)
	}
	if errs := stack.Validate(); len(errs) != 0 {
		t.Errorf("expected the loaded stack to validate cleanly, got %v", errs)
	}
}

func TestLoadUnknownFieldRefused(t *testing.T) {
	doc := strings.Replace(validDoc, "name: week-one", "name: week-one\nbogus: field", 1)
	if _, err := yaml.New().Load([]byte(doc)); err == nil {
		t.Fatal("expected an error for an unknown top-level field, got nil")
	}
}

func TestLoadMTLSCannotBeExpressedDisabled(t *testing.T) {
	doc := strings.Replace(validDoc, "mtls: {}", "mtls: false", 1)
	if _, err := yaml.New().Load([]byte(doc)); err == nil {
		t.Fatal("expected an error decoding mtls: false — the schema must not accept an off value")
	}
}

func TestLoadUnsupportedAPIVersionRefused(t *testing.T) {
	doc := strings.Replace(validDoc, "d7s.dev/v1alpha1", "d7s.dev/v2", 1)
	if _, err := yaml.New().Load([]byte(doc)); err == nil {
		t.Fatal("expected an error for an unsupported apiVersion, got nil")
	}
}

func TestLoadUnknownComponentKindRefused(t *testing.T) {
	doc := strings.Replace(validDoc, "kind: postgres", "kind: kafka", 1)
	if _, err := yaml.New().Load([]byte(doc)); err == nil {
		t.Fatal("expected an error for an unsupported component kind, got nil")
	}
}

func TestLoadInvalidRPODurationRefused(t *testing.T) {
	doc := strings.Replace(validDoc, "mtls: {}", "mtls: {}\n      rpo: not-a-duration", 1)
	if _, err := yaml.New().Load([]byte(doc)); err == nil {
		t.Fatal("expected an error for an invalid rpo duration, got nil")
	}
}

func TestLoadManagedPlacementParsesButFailsDomainValidation(t *testing.T) {
	doc := strings.Replace(validDoc, "placement: self-hosted", "placement: managed", 1)
	stack, err := yaml.New().Load([]byte(doc))
	if err != nil {
		t.Fatalf("expected the schema to accept placement: managed structurally, got parse error: %v", err)
	}
	errs := stack.Validate()
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "planned, not yet available") {
		t.Fatalf("expected domain validation to refuse managed placement with a remedy, got %v", errs)
	}
}

// TestLoadRPOGuaranteeParsesButFailsClosed mirrors the managed-placement
// case above for guarantees.rpo (owner decision, week-one plan "Owner
// decisions — 2026-07-26"): the schema still accepts the field
// structurally, but domain validation refuses it unconditionally with
// the remedy in the error (golden rules 34, 35). TestLoadValidDocument
// above proves the same declaration, minus guarantees.rpo, compiles
// cleanly (rule 49: the check shown able to fail and pass).
func TestLoadRPOGuaranteeParsesButFailsClosed(t *testing.T) {
	doc := strings.Replace(validDoc, "mtls: {}", "mtls: {}\n      rpo: 1h", 1)
	stack, err := yaml.New().Load([]byte(doc))
	if err != nil {
		t.Fatalf("expected the schema to accept guarantees.rpo structurally, got parse error: %v", err)
	}
	errs := stack.Validate()
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "planned, not yet available") {
		t.Fatalf("expected domain validation to refuse guarantees.rpo with a remedy, got %v", errs)
	}
	if !strings.Contains(errs[0].Error(), "remove guarantees.rpo") {
		t.Fatalf("expected the error to name the concrete remedy, got %v", errs)
	}
}
