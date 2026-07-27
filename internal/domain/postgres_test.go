package domain_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/rezarajan/datascape/internal/domain"
)

func validPostgres() domain.Postgres {
	return domain.Postgres{
		Name:        "orders-db",
		Placement:   domain.PlacementSelfHosted,
		Credentials: domain.SecretRef{Name: "orders-db-app"},
		Guarantees: domain.Guarantees{
			MTLS: &domain.MTLSGuarantee{},
		},
	}
}

func TestPostgresValidateAcceptsValid(t *testing.T) {
	if errs := validPostgres().Validate(); len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
}

// TestPostgresValidateManagedPlacementCompiles proves placement: managed
// now compiles (week-two plan, slices 2+3): the seam proof requires that
// the same declaration, with only placement flipped and no
// mtls/allowedConsumers/rpo declared, validates cleanly.
func TestPostgresValidateManagedPlacementCompiles(t *testing.T) {
	p := validPostgres()
	p.Placement = domain.PlacementManaged
	p.Guarantees.MTLS = nil
	if errs := p.Validate(); len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
}

// TestPostgresValidateMTLSRefusedOnManagedPlacement proves the
// transport-security guarantee refuses to compile against managed
// placement rather than silently degrading (golden rules 34, 37, 50):
// mesh mTLS and its compiled authorization cannot cover a
// provider-terminated endpoint outside the mesh.
func TestPostgresValidateMTLSRefusedOnManagedPlacement(t *testing.T) {
	p := validPostgres()
	p.Placement = domain.PlacementManaged
	errs := p.Validate()
	if len(errs) != 1 {
		t.Fatalf("expected exactly 1 error, got %v", errs)
	}
	got := errs[0].Error()
	if !strings.Contains(got, "guarantees.mtls + placement: managed refuses to compile") {
		t.Errorf("error %q does not name the boundary", got)
	}
	if !strings.Contains(got, "placement: self-hosted") || !strings.Contains(got, "remove guarantees.mtls") {
		t.Errorf("error %q does not carry the remedy (golden rule 35)", got)
	}
}

// TestPostgresValidateMTLSRefusalAggregatesWithOtherErrors pins the
// mtls+managed refusal's aggregation with other validation errors
// (rule 33), mirroring the existing rpo aggregation test.
func TestPostgresValidateMTLSRefusalAggregatesWithOtherErrors(t *testing.T) {
	p := validPostgres()
	p.Name = ""
	p.Placement = domain.PlacementManaged
	errs := p.Validate()
	joined := errors.Join(errs...).Error()
	if !strings.Contains(joined, "name is required") {
		t.Errorf("aggregated errors %v do not include the name error", errs)
	}
	if !strings.Contains(joined, "guarantees.mtls + placement: managed refuses to compile") {
		t.Errorf("aggregated errors %v do not include the mtls refusal", errs)
	}
}

// TestPostgresValidateAllowedConsumersCompilesOnManagedPlacement proves
// the un-refusal (week-four plan, slice 2): allowedConsumers +
// placement: managed no longer refuses at domain validation — its
// enforcement point is now egress compilation's waypoint-bound
// ServiceEntry AuthorizationPolicy (internal/adapters/flux/egress.go),
// which does cover a managed placement, so it needs no guarantees.mtls
// companion (mtls itself still refuses independently on managed
// placement — see TestPostgresValidateMTLSRefusedOnManagedPlacement).
func TestPostgresValidateAllowedConsumersCompilesOnManagedPlacement(t *testing.T) {
	p := validPostgres()
	p.Placement = domain.PlacementManaged
	p.Guarantees.MTLS = nil
	p.AllowedConsumers = []domain.AllowedConsumer{{ServiceAccount: "probe-client"}}
	if errs := p.Validate(); len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
}

// TestPostgresValidateWithoutAllowedConsumersOrMTLSManagedCompiles proves
// the same declaration, minus allowedConsumers and guarantees.mtls,
// compiles cleanly on managed placement (rule 49: the refusal shown able
// to fail and pass).
func TestPostgresValidateWithoutAllowedConsumersOrMTLSManagedCompiles(t *testing.T) {
	p := validPostgres()
	p.Placement = domain.PlacementManaged
	p.Guarantees.MTLS = nil
	p.AllowedConsumers = nil
	if errs := p.Validate(); len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
}

// TestPostgresValidateRPORefusedOnManagedPlacementToo verifies
// guarantees.rpo continues to refuse on managed placement too (week-three
// plan: the durability guarantee now compiles for self-hosted placement,
// but the managed emitter still has no destination wiring) — not assumed
// from the self-hosted case, since the two placements are validated by
// separate switch arms above it. BackupTo is well-formed here (a non-empty
// name) so this pins the managed-placement refusal specifically, not the
// missing-backupTo refusal (whether that name resolves to a declared
// external is a stack-level question Postgres.Validate cannot see).
func TestPostgresValidateRPORefusedOnManagedPlacementToo(t *testing.T) {
	p := validPostgres()
	p.Placement = domain.PlacementManaged
	p.Guarantees.MTLS = nil
	p.Guarantees.RPO = &domain.RPOGuarantee{Target: time.Hour, BackupTo: "backups"}
	errs := p.Validate()
	if len(errs) != 1 {
		t.Fatalf("expected exactly 1 error, got %v", errs)
	}
	got := errs[0].Error()
	if !strings.Contains(got, "guarantees.rpo + placement: managed refuses to compile") {
		t.Errorf("error %q does not name the boundary", got)
	}
	if !strings.Contains(got, "placement: self-hosted") || !strings.Contains(got, "remove guarantees.rpo") {
		t.Errorf("error %q does not carry the remedy (golden rule 35)", got)
	}
}

func TestPostgresValidateAggregatesAllErrors(t *testing.T) {
	// Every field wrong at once: validate-time completeness (rule 33)
	// means every problem is reported together, not one at a time.
	p := domain.Postgres{}
	errs := p.Validate()
	// name, placement, credentials => at least 3 distinct problems.
	if len(errs) < 3 {
		t.Fatalf("expected aggregated errors for name/placement/credentials, got %v", errs)
	}
}

func TestPostgresValidateUnknownPlacementRefused(t *testing.T) {
	p := validPostgres()
	p.Placement = "on-prem"
	errs := p.Validate()
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "unknown placement") {
		t.Fatalf("expected an unknown-placement error, got %v", errs)
	}
}

// TestPostgresValidateRPOWithoutBackupToFailsClosed proves guarantees.rpo
// refuses to compile in the same fail-closed style as placement: managed
// (golden rules 34, 35) when no destination is named — week-three plan,
// slices 1+2: v1 has exactly one declarable destination shape (a declared
// external object store) and no default, so a bare RPO with no backupTo
// still cannot compile, though the remedy now names the external block
// rather than claiming the guarantee is unimplemented outright.
func TestPostgresValidateRPOWithoutBackupToFailsClosed(t *testing.T) {
	p := validPostgres()
	p.Guarantees.RPO = &domain.RPOGuarantee{Target: time.Hour}
	errs := p.Validate()
	if len(errs) != 1 {
		t.Fatalf("expected exactly 1 error, got %v", errs)
	}
	got := errs[0].Error()
	if !strings.Contains(got, "requires backupTo naming a declared external") {
		t.Errorf("error %q does not name the missing destination", got)
	}
	if !strings.Contains(got, "declare an external block") || !strings.Contains(got, "remove guarantees.rpo") {
		t.Errorf("error %q does not carry the remedy (golden rule 35)", got)
	}
}

// TestPostgresValidateRPOWithBackupToCompiles proves guarantees.rpo now
// compiles once a destination is named (week-three plan, slices 1+2):
// the seam that used to fail unconditionally. Whether "backups" actually
// resolves to a declared external is a stack-level question
// (Stack.Validate) Postgres.Validate has no visibility into, so this is
// silent on that — only proving the postgres-level check passes.
func TestPostgresValidateRPOWithBackupToCompiles(t *testing.T) {
	p := validPostgres()
	p.Guarantees.RPO = &domain.RPOGuarantee{Target: time.Hour, BackupTo: "backups"}
	if errs := p.Validate(); len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
}

// TestPostgresValidateWithoutRPOCompiles proves the same declaration,
// minus guarantees.rpo, compiles cleanly (rule 49: the check shown able
// to fail and pass).
func TestPostgresValidateWithoutRPOCompiles(t *testing.T) {
	p := validPostgres()
	if errs := p.Validate(); len(errs) != 0 {
		t.Fatalf("expected no errors without guarantees.rpo, got %v", errs)
	}
}

// TestPostgresValidateRPORefusalAggregatesWithOtherErrors pins the
// "aggregates with other validation errors" claim (rules 33, 39) for
// guarantees.rpo specifically: a declaration with both an unknown
// placement and a declared rpo with no backupTo must report both
// problems together in one Validate() call, not just one alone.
func TestPostgresValidateRPORefusalAggregatesWithOtherErrors(t *testing.T) {
	p := validPostgres()
	p.Placement = "on-prem"
	p.Guarantees.RPO = &domain.RPOGuarantee{Target: time.Hour}
	errs := p.Validate()
	if len(errs) != 2 {
		t.Fatalf("expected exactly 2 aggregated errors, got %v", errs)
	}
	joined := errors.Join(errs...).Error()
	if !strings.Contains(joined, "unknown placement") {
		t.Errorf("aggregated errors %v do not include the unknown-placement error", errs)
	}
	if !strings.Contains(joined, "requires backupTo naming a declared external") {
		t.Errorf("aggregated errors %v do not include the rpo refusal", errs)
	}
}

func TestPostgresValidateAllowedConsumersRequireMTLS(t *testing.T) {
	p := validPostgres()
	p.Guarantees.MTLS = nil
	p.AllowedConsumers = []domain.AllowedConsumer{{ServiceAccount: "probe-client"}}
	errs := p.Validate()
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "allowedConsumers declared without guarantees.mtls") {
		t.Fatalf("expected an allowedConsumers-without-mtls error, got %v", errs)
	}
}

func TestPostgresValidateAllowedConsumerRequiresServiceAccount(t *testing.T) {
	p := validPostgres()
	p.AllowedConsumers = []domain.AllowedConsumer{{}}
	errs := p.Validate()
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "serviceAccount is required") {
		t.Fatalf("expected a serviceAccount-required error, got %v", errs)
	}
}

func TestPostgresValidateAcceptsDeclaredConsumer(t *testing.T) {
	p := validPostgres()
	p.AllowedConsumers = []domain.AllowedConsumer{{ServiceAccount: "probe-client"}}
	if errs := p.Validate(); len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
}
