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

// TestLoadInvalidRPODurationRefused pins guarantees.rpo's nested shape
// (week-three plan, slices 1+2: target + backupTo, since a bare duration
// string can no longer carry both) — an invalid target duration still
// refuses at parse time (time.ParseDuration inside toPostgres).
func TestLoadInvalidRPODurationRefused(t *testing.T) {
	doc := strings.Replace(validDoc, "mtls: {}",
		"mtls: {}\n      rpo:\n        target: not-a-duration\n        backupTo: backups", 1)
	if _, err := yaml.New().Load([]byte(doc)); err == nil {
		t.Fatal("expected an error for an invalid rpo duration, got nil")
	}
}

// managedDoc mirrors validDoc with placement flipped to managed and no
// guarantees.mtls/allowedConsumers declared — the seam proof shape
// (week-two plan): the same declaration, only placement flipped,
// compiles cleanly.
const managedDoc = `
apiVersion: d7s.dev/v1alpha1
kind: Stack
name: week-one
components:
  - kind: postgres
    name: orders-db
    placement: managed
    credentials:
      secretRef:
        name: orders-db-app
`

// TestLoadManagedPlacementWithoutGuaranteesCompiles proves placement:
// managed now compiles (week-two plan, slices 2+3) when no guarantee
// whose meaning cannot survive the placement change is declared.
func TestLoadManagedPlacementWithoutGuaranteesCompiles(t *testing.T) {
	stack, err := yaml.New().Load([]byte(managedDoc))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if errs := stack.Validate(); len(errs) != 0 {
		t.Fatalf("expected the loaded managed-placement stack to validate cleanly, got %v", errs)
	}
}

// TestLoadManagedPlacementWithMTLSFailsDomainValidationButAllowedConsumersCompiles
// proves the schema still structurally accepts guarantees.mtls and
// allowedConsumers alongside placement: managed (golden rule 34's
// "schema-accepted, refused loudly" shape, not a parse-time rejection),
// and that domain validation still refuses guarantees.mtls there — but,
// since the week-four plan's un-refusal, allowedConsumers alongside it no
// longer contributes a second error: its own enforcement point is now
// egress compilation (internal/adapters/flux/egress.go), which does
// cover managed placement.
func TestLoadManagedPlacementWithMTLSFailsDomainValidationButAllowedConsumersCompiles(t *testing.T) {
	doc := strings.Replace(validDoc, "placement: self-hosted", "placement: managed", 1)
	stack, err := yaml.New().Load([]byte(doc))
	if err != nil {
		t.Fatalf("expected the schema to accept placement: managed structurally, got parse error: %v", err)
	}
	errs := stack.Validate()
	if len(errs) != 1 {
		t.Fatalf("expected exactly 1 aggregated error (mtls only — allowedConsumers now compiles on managed placement), got %v", errs)
	}
	if !strings.Contains(errs[0].Error(), "guarantees.mtls + placement: managed refuses to compile") {
		t.Errorf("aggregated errors %v do not include the mtls refusal", errs)
	}
}

// TestLoadRPOWithoutBackupToParsesButFailsClosed mirrors the
// managed-placement case above for guarantees.rpo (week-three plan,
// slices 1+2): the schema still accepts a bare rpo (no backupTo)
// structurally, but domain validation refuses it with a remedy naming
// the external block (golden rules 34, 35). TestLoadValidDocument above
// proves the same declaration, minus guarantees.rpo entirely, compiles
// cleanly (rule 49: the check shown able to fail and pass).
func TestLoadRPOWithoutBackupToParsesButFailsClosed(t *testing.T) {
	doc := strings.Replace(validDoc, "mtls: {}", "mtls: {}\n      rpo:\n        target: 1h", 1)
	stack, err := yaml.New().Load([]byte(doc))
	if err != nil {
		t.Fatalf("expected the schema to accept guarantees.rpo structurally, got parse error: %v", err)
	}
	errs := stack.Validate()
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "requires backupTo naming a declared external") {
		t.Fatalf("expected domain validation to refuse guarantees.rpo with a remedy, got %v", errs)
	}
	if !strings.Contains(errs[0].Error(), "declare an external block") {
		t.Fatalf("expected the error to name the concrete remedy, got %v", errs)
	}
}

// externalDoc declares an external S3-compatible object store alongside
// a component wiring guarantees.rpo.backupTo to it — the whole
// week-three durability shape, end to end through the loader.
const externalDoc = `
apiVersion: d7s.dev/v1alpha1
kind: Stack
name: week-three
external:
  - name: backups
    objectStore:
      endpoint: https://minio.d7s-harness.svc:9000
      bucket: d7s-backups
      credentials:
        secretRef:
          name: backups-credentials
components:
  - kind: postgres
    name: orders-db
    placement: self-hosted
    credentials:
      secretRef:
        name: orders-db-app
    guarantees:
      rpo:
        target: 1h
        backupTo: backups
`

// TestLoadExternalAndRPOBackupToCompiles proves the durability triple's
// declaration shape parses and validates cleanly end to end: an external
// object store plus a component's guarantees.rpo.backupTo naming it
// (week-three plan, slices 1+2).
func TestLoadExternalAndRPOBackupToCompiles(t *testing.T) {
	stack, err := yaml.New().Load([]byte(externalDoc))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stack.Externals) != 1 {
		t.Fatalf("expected 1 external, got %d", len(stack.Externals))
	}
	ext := stack.Externals[0]
	if ext.Name != "backups" {
		t.Errorf("external name = %q, want backups", ext.Name)
	}
	if ext.ObjectStore == nil {
		t.Fatal("expected an objectStore declared")
	}
	if ext.ObjectStore.Endpoint != "https://minio.d7s-harness.svc:9000" {
		t.Errorf("objectStore.endpoint = %q, unexpected", ext.ObjectStore.Endpoint)
	}
	if ext.ObjectStore.Bucket != "d7s-backups" {
		t.Errorf("objectStore.bucket = %q, unexpected", ext.ObjectStore.Bucket)
	}
	if ext.ObjectStore.Credentials.Name != "backups-credentials" {
		t.Errorf("objectStore.credentials.secretRef.name = %q, unexpected", ext.ObjectStore.Credentials.Name)
	}
	pg, ok := stack.Components[0].(domain.Postgres)
	if !ok {
		t.Fatalf("component type = %T, want domain.Postgres", stack.Components[0])
	}
	if pg.Guarantees.RPO == nil || pg.Guarantees.RPO.BackupTo != "backups" {
		t.Fatalf("expected guarantees.rpo.backupTo = backups, got %+v", pg.Guarantees.RPO)
	}
	if errs := stack.Validate(); len(errs) != 0 {
		t.Errorf("expected the loaded stack to validate cleanly, got %v", errs)
	}
}

// TestLoadExternalAloneEmitsNoComponentSideEffect proves an external
// declaration with no component referencing it still parses and
// validates cleanly on its own (problem definition Amendment 2: an
// external alone emits nothing — the emitter-level half of this claim is
// pinned in internal/adapters/flux/flux_test.go).
func TestLoadExternalAloneEmitsNoComponentSideEffect(t *testing.T) {
	doc := strings.Replace(externalDoc, "      rpo:\n        target: 1h\n        backupTo: backups\n", "", 1)
	stack, err := yaml.New().Load([]byte(doc))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stack.Externals) != 1 {
		t.Fatalf("expected the external to still parse, got %d", len(stack.Externals))
	}
	if errs := stack.Validate(); len(errs) != 0 {
		t.Errorf("expected the loaded stack to validate cleanly, got %v", errs)
	}
}

// TestLoadRPOReferencingUndeclaredExternalRefused proves the
// cross-declaration reference check reaches all the way through the
// loader: a backupTo naming an external that was never declared refuses
// at Stack.Validate, with a remedy (golden rules 34, 35).
func TestLoadRPOReferencingUndeclaredExternalRefused(t *testing.T) {
	doc := strings.Replace(externalDoc, "backupTo: backups", "backupTo: nonexistent", 1)
	stack, err := yaml.New().Load([]byte(doc))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	errs := stack.Validate()
	found := false
	for _, e := range errs {
		if strings.Contains(e.Error(), `references undeclared external "nonexistent"`) {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected an undeclared-external error, got %v", errs)
	}
}

// TestLoadExternalUnknownFieldRefused proves the external block is
// covered by the same document-wide strict unknown-field refusal as
// every other declaration surface (golden rule 34) — it is not a special
// case exempted from KnownFields(true).
func TestLoadExternalUnknownFieldRefused(t *testing.T) {
	doc := strings.Replace(externalDoc, "bucket: d7s-backups", "bucket: d7s-backups\n      bogus: field", 1)
	if _, err := yaml.New().Load([]byte(doc)); err == nil {
		t.Fatal("expected an error for an unknown field in the external block, got nil")
	}
}
