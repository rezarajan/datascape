package flux_test

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/rezarajan/datascape/internal/adapters/flux"
	"github.com/rezarajan/datascape/internal/domain"
)

// exampleStack mirrors examples/week-one/stack.yaml: the week-one
// artifact composes two guarantee families on the same component — mTLS
// and RPO-backed durability (week-three plan, slice 3) — proving the
// composition that week one's live acceptance run first caught as a bug
// (a mesh-only AuthorizationPolicy rule set that also had to admit the
// CNPG operator's own status polling; see emitZeroTrust's doc comment).
// The durability leg wires guarantees.rpo to weekOneBackupsExternal, the
// declared external the acceptance harness's own MinIO stands up
// (scripts/actions/minio-install.sh) — external by provenance
// (Amendment 2), never d7s-compiled.
func exampleStack() domain.Stack {
	return domain.Stack{
		Name: "week-one",
		Components: []domain.Component{
			domain.Postgres{
				Name:        "orders-db",
				Placement:   domain.PlacementSelfHosted,
				Credentials: domain.SecretRef{Name: "orders-db-app"},
				Guarantees: domain.Guarantees{
					MTLS: &domain.MTLSGuarantee{},
					RPO:  &domain.RPOGuarantee{Target: time.Hour, BackupTo: "backups"},
				},
				AllowedConsumers: []domain.AllowedConsumer{
					{ServiceAccount: "probe-client"},
				},
			},
		},
		Externals: []domain.External{weekOneBackupsExternal()},
	}
}

// weekOneBackupsExternal mirrors examples/week-one/stack.yaml's declared
// external exactly: the endpoint names the harness's in-cluster MinIO
// service DNS (scripts/actions/minio-install.sh stands it up in the
// d7s-harness-minio namespace, plain HTTP — an in-cluster-only service,
// no TLS cert to manage for a throwaway harness store).
func weekOneBackupsExternal() domain.External {
	return domain.External{
		Name: "backups",
		ObjectStore: &domain.ObjectStoreExternal{
			Endpoint:    "http://minio.d7s-harness-minio.svc:9000",
			Bucket:      "d7s-backups",
			Credentials: domain.SecretRef{Name: "backups-credentials"},
		},
	}
}

// TestEmitGoldenFiles pins compiled output against the checked-in golden
// tree (golden rule 45): the acceptance example, compiled once here and
// committed under testdata/golden, must match byte for byte.
func TestEmitGoldenFiles(t *testing.T) {
	manifests, err := flux.New().Emit(exampleStack())
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}

	want := readGoldenDir(t, filepath.Join("testdata", "golden", "week-one"))

	if len(manifests.Files) != len(want) {
		t.Fatalf("emitted %d files, golden has %d\nemitted: %v\ngolden: %v",
			len(manifests.Files), len(want), sortedKeys(manifests.Files), sortedKeys(want))
	}
	for path, wantBytes := range want {
		gotBytes, ok := manifests.Files[path]
		if !ok {
			t.Errorf("golden file %s was not emitted", path)
			continue
		}
		if !bytes.Equal(gotBytes, wantBytes) {
			t.Errorf("emitted %s differs from golden:\n--- got ---\n%s\n--- want ---\n%s", path, gotBytes, wantBytes)
		}
	}
}

// TestEmitDeterministic compiles the same declaration twice and requires
// byte-identical output (golden rules 22, 45).
func TestEmitDeterministic(t *testing.T) {
	a, err := flux.New().Emit(exampleStack())
	if err != nil {
		t.Fatalf("Emit (first): %v", err)
	}
	b, err := flux.New().Emit(exampleStack())
	if err != nil {
		t.Fatalf("Emit (second): %v", err)
	}
	if len(a.Files) != len(b.Files) {
		t.Fatalf("file count differs across compiles: %d vs %d", len(a.Files), len(b.Files))
	}
	for path, want := range a.Files {
		got, ok := b.Files[path]
		if !ok {
			t.Fatalf("second compile is missing file %s", path)
		}
		if !bytes.Equal(want, got) {
			t.Errorf("file %s is not byte-identical across compiles", path)
		}
	}
}

// TestEmitWithoutMTLSGuaranteeEmitsNoZeroTrustObjects proves presence is
// the only signal (golden rule 50): a component that doesn't declare
// the mtls guarantee gets no PeerAuthentication or AuthorizationPolicy
// at all, rather than a permissive or disabled variant of either.
func TestEmitWithoutMTLSGuaranteeEmitsNoZeroTrustObjects(t *testing.T) {
	stack := domain.Stack{
		Name: "week-one",
		Components: []domain.Component{
			domain.Postgres{
				Name:        "orders-db",
				Placement:   domain.PlacementSelfHosted,
				Credentials: domain.SecretRef{Name: "orders-db-app"},
			},
		},
	}
	manifests, err := flux.New().Emit(stack)
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}
	for path := range manifests.Files {
		if filepath.Base(path) == "peerauthentication.yaml" || filepath.Base(path) == "orders-db-authorizationpolicy.yaml" {
			t.Errorf("emitted %s without a declared mtls guarantee", path)
		}
	}
	if bytes.Contains(manifests.Files["apps/week-one/namespace.yaml"], []byte("dataplane-mode")) {
		t.Error("namespace carries the ambient dataplane label without a declared mtls guarantee")
	}
	if bytes.Contains(manifests.Files["infra/cnpg-operator/namespace.yaml"], []byte("dataplane-mode")) {
		t.Error("cnpg-system namespace carries the ambient dataplane label without a declared mtls guarantee")
	}
}

// TestEmitMTLSGuaranteeLabelsNamespaceAmbient proves both the app
// namespace and the CNPG operator's own namespace join the Istio
// ambient mesh when mtls is declared. Both are required: without the
// app namespace's label, ztunnel never intercepts its traffic and
// PeerAuthentication/AuthorizationPolicy would exist but never be
// enforced; without the operator namespace's label, the operator itself
// cannot originate an mTLS connection into the STRICT app namespace to
// manage the cluster it created — PeerAuthentication enforces at the
// transport layer, before AuthorizationPolicy is ever evaluated (found
// by running the acceptance harness against a live ambient mesh —
// golden rule 40; rule 42: composition gets its own acceptance tests).
func TestEmitMTLSGuaranteeLabelsNamespacesAmbient(t *testing.T) {
	manifests, err := flux.New().Emit(exampleStack())
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}
	if !bytes.Contains(manifests.Files["apps/week-one/namespace.yaml"], []byte("istio.io/dataplane-mode: ambient")) {
		t.Error("app namespace does not carry the ambient dataplane label despite a declared mtls guarantee")
	}
	if !bytes.Contains(manifests.Files["infra/cnpg-operator/namespace.yaml"], []byte("istio.io/dataplane-mode: ambient")) {
		t.Error("cnpg-system namespace does not carry the ambient dataplane label despite a declared mtls guarantee")
	}
}

// TestEmitAuthorizationPolicyScopesRulesByPort proves the AuthorizationPolicy
// restricts declared consumers to the Postgres port and separately allows
// the CNPG operator's own namespace to reach only the status port —
// found necessary live: an unscoped allow-list also gated the operator's
// own instance-status polling, which broke cluster management entirely
// (golden rule 40).
func TestEmitAuthorizationPolicyScopesRulesByPort(t *testing.T) {
	manifests, err := flux.New().Emit(exampleStack())
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}
	policy := string(manifests.Files["apps/week-one/orders-db-authorizationpolicy.yaml"])
	if !strings.Contains(policy, `- "5432"`) {
		t.Errorf("consumer rule is not scoped to the postgres port:\n%s", policy)
	}
	if !strings.Contains(policy, `- "8000"`) {
		t.Errorf("operator rule is not scoped to the status port:\n%s", policy)
	}
	if !strings.Contains(policy, "namespaces:") || !strings.Contains(policy, "cnpg-system") {
		t.Errorf("no operator-namespace allow rule present:\n%s", policy)
	}
}

// backupsExternal is the declared external object store durabilityExampleStack
// and the direct-emitter RPO tests below wire guarantees.rpo.backupTo to.
func backupsExternal() domain.External {
	return domain.External{
		Name: "backups",
		ObjectStore: &domain.ObjectStoreExternal{
			Endpoint:    "https://minio.d7s-harness.svc:9000",
			Bucket:      "d7s-backups",
			Credentials: domain.SecretRef{Name: "backups-credentials"},
		},
	}
}

// TestEmitWithoutRPOGuaranteeEmitsNoBackupObjects mirrors the mTLS case
// for durability: no ScheduledBackup and no Cluster.spec.backup unless
// the RPO guarantee is declared — and, since presence is the only signal
// (golden rule 50), no CONDITIONAL annotation either (golden rule 49:
// the label must be testable-absent).
func TestEmitWithoutRPOGuaranteeEmitsNoBackupObjects(t *testing.T) {
	stack := domain.Stack{
		Name: "week-one",
		Components: []domain.Component{
			domain.Postgres{
				Name:        "orders-db",
				Placement:   domain.PlacementSelfHosted,
				Credentials: domain.SecretRef{Name: "orders-db-app"},
			},
		},
	}
	manifests, err := flux.New().Emit(stack)
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}
	for path := range manifests.Files {
		if filepath.Base(path) == "orders-db-scheduledbackup.yaml" {
			t.Errorf("emitted %s without a declared rpo guarantee", path)
		}
	}
	if bytes.Contains(manifests.Files["apps/week-one/orders-db-cluster.yaml"], []byte("backup")) {
		t.Error("Cluster CR mentions backup without a declared rpo guarantee")
	}
	if bytes.Contains(manifests.Files["apps/week-one/orders-db-cluster.yaml"], []byte("d7s.dev/guarantee-durability")) {
		t.Error("Cluster CR carries the CONDITIONAL durability label without a declared rpo guarantee")
	}
	if len(manifests.Conditionals) != 0 {
		t.Errorf("expected no conditional-guarantee notices, got %+v", manifests.Conditionals)
	}
}

// TestEmitWithRPOGuaranteeEmitsBackupObjects exercises the durability
// guarantee's emitted-infra element directly against this emitter
// (emitCluster, emitDurability): guarantees.rpo now compiles once wired
// to a declared external (week-three plan, slices 1+2), and every object
// it emits carries the CONDITIONAL annotation (Amendment 2, B3) since the
// guarantee crosses the trust boundary to that external.
func TestEmitWithRPOGuaranteeEmitsBackupObjects(t *testing.T) {
	stack := domain.Stack{
		Name: "week-one",
		Components: []domain.Component{
			domain.Postgres{
				Name:        "orders-db",
				Placement:   domain.PlacementSelfHosted,
				Credentials: domain.SecretRef{Name: "orders-db-app"},
				Guarantees: domain.Guarantees{
					RPO: &domain.RPOGuarantee{Target: time.Hour, BackupTo: "backups"},
				},
			},
		},
		Externals: []domain.External{backupsExternal()},
	}
	manifests, err := flux.New().Emit(stack)
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}
	sb, ok := manifests.Files["apps/week-one/orders-db-scheduledbackup.yaml"]
	if !ok {
		t.Fatalf("expected a ScheduledBackup to be emitted, got %v", sortedKeys(manifests.Files))
	}
	if !bytes.Contains(sb, []byte("@every 1h0m0s")) {
		t.Errorf("ScheduledBackup schedule does not derive from the declared RPO:\n%s", sb)
	}
	if !bytes.Contains(sb, []byte("d7s.dev/guarantee-durability: conditional-on-external")) {
		t.Errorf("ScheduledBackup does not carry the CONDITIONAL durability label:\n%s", sb)
	}
	cluster := manifests.Files["apps/week-one/orders-db-cluster.yaml"]
	if !bytes.Contains(cluster, []byte("backup:")) {
		t.Error("Cluster CR does not carry spec.backup despite a declared rpo guarantee")
	}
	if !bytes.Contains(cluster, []byte("barmanObjectStore:")) {
		t.Errorf("Cluster CR does not carry spec.backup.barmanObjectStore:\n%s", cluster)
	}
	if !bytes.Contains(cluster, []byte("destinationPath: s3://d7s-backups/")) ||
		!bytes.Contains(cluster, []byte("endpointURL: https://minio.d7s-harness.svc:9000")) {
		t.Errorf("Cluster CR's barmanObjectStore is not wired from the declared external:\n%s", cluster)
	}
	if !bytes.Contains(cluster, []byte("name: backups-credentials")) {
		t.Errorf("Cluster CR's s3Credentials do not reference the declared external's Secret:\n%s", cluster)
	}
	if !bytes.Contains(cluster, []byte("d7s.dev/guarantee-durability: conditional-on-external")) {
		t.Errorf("Cluster CR does not carry the CONDITIONAL durability label:\n%s", cluster)
	}
	if len(manifests.Conditionals) != 1 {
		t.Fatalf("expected exactly 1 conditional-guarantee notice, got %+v", manifests.Conditionals)
	}
	c := manifests.Conditionals[0]
	if c.Component != "orders-db" {
		t.Errorf("conditional.Component = %q, want orders-db", c.Component)
	}
	if !strings.Contains(c.Label, "d7s.dev/guarantee-durability") || !strings.Contains(c.Label, "conditional-on-external") {
		t.Errorf("conditional.Label = %q, missing the annotation key/value", c.Label)
	}
	if !strings.Contains(c.Reason, `external store "backups"`) {
		t.Errorf("conditional.Reason = %q, does not name the external store", c.Reason)
	}
}

// TestEmitRPOResolvesNeverInlinesCredentialValue proves the s3Credentials
// CNPG reads are always secretKeyRef-shaped references, never a literal
// value (golden rule 51) — the declared external only ever names a
// Kubernetes Secret, so there is nothing else for the emitter to inline.
func TestEmitRPOResolvesNeverInlinesCredentialValue(t *testing.T) {
	stack := domain.Stack{
		Name: "week-one",
		Components: []domain.Component{
			domain.Postgres{
				Name:        "orders-db",
				Placement:   domain.PlacementSelfHosted,
				Credentials: domain.SecretRef{Name: "orders-db-app"},
				Guarantees: domain.Guarantees{
					RPO: &domain.RPOGuarantee{Target: time.Hour, BackupTo: "backups"},
				},
			},
		},
		Externals: []domain.External{backupsExternal()},
	}
	manifests, err := flux.New().Emit(stack)
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}
	cluster := string(manifests.Files["apps/week-one/orders-db-cluster.yaml"])
	if !strings.Contains(cluster, "key: ACCESS_KEY_ID") || !strings.Contains(cluster, "key: ACCESS_SECRET_KEY") {
		t.Errorf("Cluster CR's s3Credentials are not keyed by the fixed convention:\n%s", cluster)
	}
}

// TestEmitUnsatisfiableRPORefused proves the durability guarantee's
// compile-time check can fail (golden rule 49's "proven by the negative
// probe" spirit, exercised here at compile time): an RPO below what the
// emitter can honor refuses compilation with the remedy in the error
// (golden rules 34, 35), and nothing is written for that call.
func TestEmitUnsatisfiableRPORefused(t *testing.T) {
	stack := domain.Stack{
		Name: "week-one",
		Components: []domain.Component{
			domain.Postgres{
				Name:        "orders-db",
				Placement:   domain.PlacementSelfHosted,
				Credentials: domain.SecretRef{Name: "orders-db-app"},
				Guarantees: domain.Guarantees{
					RPO: &domain.RPOGuarantee{Target: 2 * time.Minute, BackupTo: "backups"},
				},
			},
		},
		Externals: []domain.External{backupsExternal()},
	}
	manifests, err := flux.New().Emit(stack)
	if err == nil {
		t.Fatal("expected an error for an unsatisfiable RPO, got nil")
	}
	if !strings.Contains(err.Error(), "cannot be honored") {
		t.Errorf("error %q does not explain why the RPO can't be honored", err.Error())
	}
	if !strings.Contains(err.Error(), "declare a larger value") {
		t.Errorf("error %q does not carry a remedy (golden rule 35)", err.Error())
	}
	if len(manifests.Files) != 0 {
		t.Errorf("expected no files written on refusal, got %v", sortedKeys(manifests.Files))
	}
}

// durabilityExampleStack mirrors examples/week-three/durability-stack.yaml:
// a self-hosted postgres declaring guarantees.rpo wired to a declared
// external object store — the durability triple reconnected (week-three
// plan, slices 1+2).
func durabilityExampleStack() domain.Stack {
	return domain.Stack{
		Name: "week-three",
		Components: []domain.Component{
			domain.Postgres{
				Name:        "orders-db",
				Placement:   domain.PlacementSelfHosted,
				Credentials: domain.SecretRef{Name: "orders-db-app"},
				Guarantees: domain.Guarantees{
					RPO: &domain.RPOGuarantee{Target: time.Hour, BackupTo: "backups"},
				},
			},
		},
		Externals: []domain.External{backupsExternal()},
	}
}

// TestEmitDurabilityGoldenFiles pins the durability-triple artifact
// against the checked-in golden tree (golden rule 45): the CNPG Cluster's
// barmanObjectStore, the ScheduledBackup, and the CONDITIONAL annotation
// on both, wired from examples/week-three/durability-stack.yaml's
// declared external.
func TestEmitDurabilityGoldenFiles(t *testing.T) {
	manifests, err := flux.New().Emit(durabilityExampleStack())
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}

	want := readGoldenDir(t, filepath.Join("testdata", "golden", "durability"))

	if len(manifests.Files) != len(want) {
		t.Fatalf("emitted %d files, golden has %d\nemitted: %v\ngolden: %v",
			len(manifests.Files), len(want), sortedKeys(manifests.Files), sortedKeys(want))
	}
	for path, wantBytes := range want {
		gotBytes, ok := manifests.Files[path]
		if !ok {
			t.Errorf("golden file %s was not emitted", path)
			continue
		}
		if !bytes.Equal(gotBytes, wantBytes) {
			t.Errorf("emitted %s differs from golden:\n--- got ---\n%s\n--- want ---\n%s", path, gotBytes, wantBytes)
		}
	}
}

// TestEmitDurabilityDeterministic compiles the durability example twice
// and requires byte-identical output (golden rules 22, 45).
func TestEmitDurabilityDeterministic(t *testing.T) {
	a, err := flux.New().Emit(durabilityExampleStack())
	if err != nil {
		t.Fatalf("Emit (first): %v", err)
	}
	b, err := flux.New().Emit(durabilityExampleStack())
	if err != nil {
		t.Fatalf("Emit (second): %v", err)
	}
	if len(a.Files) != len(b.Files) {
		t.Fatalf("file count differs across compiles: %d vs %d", len(a.Files), len(b.Files))
	}
	for path, want := range a.Files {
		got, ok := b.Files[path]
		if !ok {
			t.Fatalf("second compile is missing file %s", path)
		}
		if !bytes.Equal(want, got) {
			t.Errorf("file %s is not byte-identical across compiles", path)
		}
	}
}

// TestEmitManagedExampleUnaffectedByWeekOneDurability proves composing
// mTLS+RPO onto the self-hosted week-one example (week-three plan,
// slice 3) changed no byte of the managed-placement golden output — the
// managed example declares no guarantees.rpo (placement: managed still
// refuses it, internal/domain/postgres.go), so it must stay fully inert.
// (The self-hosted, no-regression-from-durability claim this test used
// to also make for week-one no longer holds — week-one's own golden
// output now legitimately includes the durability triple; that shape is
// pinned instead by TestEmitGoldenFiles.)
func TestEmitManagedExampleUnaffectedByWeekOneDurability(t *testing.T) {
	managed, err := flux.New().Emit(managedExampleStack())
	if err != nil {
		t.Fatalf("Emit (managed): %v", err)
	}
	wantManaged := readGoldenDir(t, filepath.Join("testdata", "golden", "managed"))
	if len(managed.Files) != len(wantManaged) {
		t.Fatalf("managed: emitted %d files, golden has %d", len(managed.Files), len(wantManaged))
	}
	for path, wantBytes := range wantManaged {
		gotBytes, ok := managed.Files[path]
		if !ok || !bytes.Equal(gotBytes, wantBytes) {
			t.Errorf("managed golden file %s regressed", path)
		}
	}
}

// TestEmitExternalAloneEmitsNoBytes proves the emitter-level half of
// problem definition Amendment 2's claim (asserted in doc comments at
// internal/domain/external.go:11-13, internal/domain/stack_test.go's
// TestStackValidateExternalAlone, and
// internal/adapters/yaml/loader_test.go's
// TestLoadExternalAloneEmitsNoComponentSideEffect, none of which reach
// this package): an external declaration with no component referencing
// it changes zero emitted bytes. Compares the exact same stack with and
// without the declared external and requires byte-identical file sets —
// not merely "no crash". Built on a plain, guarantee-free stack rather
// than exampleStack() (whose guarantees.rpo now references "backups" —
// week-three plan, slice 3): the external declared here must stay
// genuinely unreferenced by anything.
func TestEmitExternalAloneEmitsNoBytes(t *testing.T) {
	plainStack := func() domain.Stack {
		return domain.Stack{
			Name: "week-one",
			Components: []domain.Component{
				domain.Postgres{
					Name:        "orders-db",
					Placement:   domain.PlacementSelfHosted,
					Credentials: domain.SecretRef{Name: "orders-db-app"},
				},
			},
		}
	}

	without := plainStack()

	withExternal := plainStack()
	withExternal.Externals = []domain.External{backupsExternal()}

	gotWithout, err := flux.New().Emit(without)
	if err != nil {
		t.Fatalf("Emit (without external): %v", err)
	}
	gotWith, err := flux.New().Emit(withExternal)
	if err != nil {
		t.Fatalf("Emit (with unreferenced external): %v", err)
	}

	if len(gotWith.Files) != len(gotWithout.Files) {
		t.Fatalf("declaring an unreferenced external changed the emitted file count: %d vs %d",
			len(gotWith.Files), len(gotWithout.Files))
	}
	for path, wantBytes := range gotWithout.Files {
		gotBytes, ok := gotWith.Files[path]
		if !ok {
			t.Errorf("file %s present without the external is missing with it declared", path)
			continue
		}
		if !bytes.Equal(gotBytes, wantBytes) {
			t.Errorf("file %s changed bytes when an unreferenced external was declared:\n--- without ---\n%s\n--- with ---\n%s",
				path, wantBytes, gotBytes)
		}
	}
	if len(gotWith.Conditionals) != 0 {
		t.Errorf("expected no conditional-guarantee notices from an unreferenced external, got %+v", gotWith.Conditionals)
	}
}

// managedExampleEndpointHost mirrors examples/week-two/managed-stack.yaml's
// own pinned endpointHost value: a plausible EXAMPLE Neon endpoint host
// shape (Neon's own naming convention, ep-<name>-<id>.<region>.aws.neon.tech),
// not one this repo has actually provisioned (week-four plan, 2026-07-27
// finding → Revision 2).
const managedExampleEndpointHost = "ep-cool-glade-12345678.us-east-2.aws.neon.tech"

// managedExampleStack mirrors examples/week-two/managed-stack.yaml, which
// in turn mirrors examples/week-one/stack.yaml with placement flipped to
// managed and no guarantees.mtls/rpo declared (both still refuse on
// managed placement) — the seam proof shape (week-two plan, slices 2+3):
// the same declaration, only placement flipped, compiles to a different
// artifact. AllowedConsumers is declared (week-four plan, slice 2's
// un-refusal): egress compilation now gives it an enforcement point on
// managed placement too (internal/adapters/flux/egress.go). This is the
// SECOND phase of the pin ceremony (2026-07-27 finding → Revision 2) —
// managedUnpinnedExampleStack below is the first; see its own doc
// comment for the full two-step story examples/week-two demonstrates.
func managedExampleStack() domain.Stack {
	return domain.Stack{
		Name: "week-one",
		Components: []domain.Component{
			domain.Postgres{
				Name:         "orders-db",
				Placement:    domain.PlacementManaged,
				Credentials:  domain.SecretRef{Name: "orders-db-app"},
				EndpointHost: managedExampleEndpointHost,
				AllowedConsumers: []domain.AllowedConsumer{
					{ServiceAccount: "probe-client"},
				},
			},
		},
	}
}

// managedUnpinnedExampleStack mirrors
// examples/week-two/managed-stack-unpinned.yaml: the FIRST phase of the
// exact-host pin ceremony (2026-07-27 finding → Revision 2) — the same
// component, before any consumer is declared and before its endpoint is
// known. Compiling and delivering this is how an operator obtains the
// "host" value written-outputs secret credentials.secretRef.name
// (orders-db-app) carries once tofu-controller reconciles, which then
// becomes managedExampleStack's own EndpointHost pin.
func managedUnpinnedExampleStack() domain.Stack {
	return domain.Stack{
		Name: "week-one",
		Components: []domain.Component{
			domain.Postgres{
				Name:        "orders-db",
				Placement:   domain.PlacementManaged,
				Credentials: domain.SecretRef{Name: "orders-db-app"},
			},
		},
	}
}

// TestEmitManagedGoldenFiles pins the managed-placement artifact against
// the checked-in golden tree (golden rule 45): a namespace, the OpenTofu
// config (project/database/role), and the wrapping Terraform CR — no
// CNPG operator install and no dependsOn edge, since nothing self-hosted
// exists in this stack (golden rule 24: no dangling/hidden edges).
func TestEmitManagedGoldenFiles(t *testing.T) {
	manifests, err := flux.New().Emit(managedExampleStack())
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}

	want := readGoldenDir(t, filepath.Join("testdata", "golden", "managed"))

	if len(manifests.Files) != len(want) {
		t.Fatalf("emitted %d files, golden has %d\nemitted: %v\ngolden: %v",
			len(manifests.Files), len(want), sortedKeys(manifests.Files), sortedKeys(want))
	}
	for path, wantBytes := range want {
		gotBytes, ok := manifests.Files[path]
		if !ok {
			t.Errorf("golden file %s was not emitted", path)
			continue
		}
		if !bytes.Equal(gotBytes, wantBytes) {
			t.Errorf("emitted %s differs from golden:\n--- got ---\n%s\n--- want ---\n%s", path, gotBytes, wantBytes)
		}
	}
}

// TestEmitManagedDeterministic compiles the managed-placement declaration
// twice and requires byte-identical output (golden rules 22, 45).
func TestEmitManagedDeterministic(t *testing.T) {
	a, err := flux.New().Emit(managedExampleStack())
	if err != nil {
		t.Fatalf("Emit (first): %v", err)
	}
	b, err := flux.New().Emit(managedExampleStack())
	if err != nil {
		t.Fatalf("Emit (second): %v", err)
	}
	if len(a.Files) != len(b.Files) {
		t.Fatalf("file count differs across compiles: %d vs %d", len(a.Files), len(b.Files))
	}
	for path, want := range a.Files {
		got, ok := b.Files[path]
		if !ok {
			t.Fatalf("second compile is missing file %s", path)
		}
		if !bytes.Equal(want, got) {
			t.Errorf("file %s is not byte-identical across compiles", path)
		}
	}
}

// TestEmitManagedUnpinnedGoldenFiles pins the FIRST phase of the
// exact-host pin ceremony (week-four plan, 2026-07-27 finding → Revision
// 2): managedUnpinnedExampleStack declares no allowedConsumers and no
// endpointHost, so it compiles cleanly with no data-plane Neon
// ServiceEntry/AuthorizationPolicy at all — only the namespace,
// waypoint, Terraform CR/OpenTofu config, tf-runner RBAC, and the
// provisioner's own control-plane egress (implied by placement: managed
// itself, unconditionally — see neonControlPlaneHost's doc comment).
func TestEmitManagedUnpinnedGoldenFiles(t *testing.T) {
	manifests, err := flux.New().Emit(managedUnpinnedExampleStack())
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}

	want := readGoldenDir(t, filepath.Join("testdata", "golden", "managed-unpinned"))

	if len(manifests.Files) != len(want) {
		t.Fatalf("emitted %d files, golden has %d\nemitted: %v\ngolden: %v",
			len(manifests.Files), len(want), sortedKeys(manifests.Files), sortedKeys(want))
	}
	for path, wantBytes := range want {
		gotBytes, ok := manifests.Files[path]
		if !ok {
			t.Errorf("golden file %s was not emitted", path)
			continue
		}
		if !bytes.Equal(gotBytes, wantBytes) {
			t.Errorf("emitted %s differs from golden:\n--- got ---\n%s\n--- want ---\n%s", path, gotBytes, wantBytes)
		}
	}
	for path := range manifests.Files {
		if strings.Contains(filepath.Base(path), "-neon-") {
			t.Errorf("emitted the pinned data-plane object %s with no declared consumers and no pin", path)
		}
	}
}

// TestEmitManagedUnpinnedDeterministic compiles the unpinned-phase
// declaration twice and requires byte-identical output (golden rules 22,
// 45).
func TestEmitManagedUnpinnedDeterministic(t *testing.T) {
	a, err := flux.New().Emit(managedUnpinnedExampleStack())
	if err != nil {
		t.Fatalf("Emit (first): %v", err)
	}
	b, err := flux.New().Emit(managedUnpinnedExampleStack())
	if err != nil {
		t.Fatalf("Emit (second): %v", err)
	}
	if len(a.Files) != len(b.Files) {
		t.Fatalf("file count differs across compiles: %d vs %d", len(a.Files), len(b.Files))
	}
	for path, want := range a.Files {
		got, ok := b.Files[path]
		if !ok {
			t.Fatalf("second compile is missing file %s", path)
		}
		if !bytes.Equal(want, got) {
			t.Errorf("file %s is not byte-identical across compiles", path)
		}
	}
}

// TestEmitManagedPlacementEmitsNoCNPGOperatorOrDependsOn proves a
// managed-only stack never emits the CNPG operator install or a
// dependsOn edge pointing at it — that edge would be dangling (golden
// rule 24) since nothing in this stack needs CNPG's CRDs.
func TestEmitManagedPlacementEmitsNoCNPGOperatorOrDependsOn(t *testing.T) {
	manifests, err := flux.New().Emit(managedExampleStack())
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}
	for path := range manifests.Files {
		if strings.HasPrefix(path, "infra/cnpg-operator/") || path == "flux/infra-cnpg-operator.yaml" {
			t.Errorf("emitted %s for a managed-only stack with no self-hosted component", path)
		}
	}
	if bytes.Contains(manifests.Files["flux/apps-week-one.yaml"], []byte("dependsOn")) {
		t.Error("apps Kustomization carries a dependsOn edge with no self-hosted component to justify it")
	}
}

// TestEmitManagedPlacementCarriesOwnershipLabels proves the Terraform CR
// and its namespace carry the d7s.dev/* ownership labels (golden rule 27)
// exactly like every other emitted object.
func TestEmitManagedPlacementCarriesOwnershipLabels(t *testing.T) {
	manifests, err := flux.New().Emit(managedExampleStack())
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}
	tf := string(manifests.Files["apps/week-one/orders-db-terraform.yaml"])
	if !strings.Contains(tf, "d7s.dev/managed-by: d7s") ||
		!strings.Contains(tf, "d7s.dev/stack: week-one") ||
		!strings.Contains(tf, "d7s.dev/component: orders-db") {
		t.Errorf("Terraform CR does not carry the full ownership-label set:\n%s", tf)
	}
}

// TestEmitManagedPlacementCompilesRunnerServiceAccountAndRoleBinding
// proves the tf-runner ServiceAccount and its RoleBinding are compiled
// alongside the Terraform CR (found live, 2026-07-26: without them, the
// runner pod fails outright - "serviceaccount \"tf-runner\" not found" -
// and tofu-controller's own release RBAC only creates that ServiceAccount
// in flux-system, not the stack's own namespace). The RoleBinding must
// reference the tf-runner-role ClusterRole (an environment prerequisite,
// scripts/actions/tofu-install.sh) by name, scoped to the stack's own
// namespace, and carry the full ownership-label set like every other
// emitted object (golden rule 27).
func TestEmitManagedPlacementCompilesRunnerServiceAccountAndRoleBinding(t *testing.T) {
	manifests, err := flux.New().Emit(managedExampleStack())
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}
	sa := string(manifests.Files["apps/week-one/tf-runner-serviceaccount.yaml"])
	if !strings.Contains(sa, "kind: ServiceAccount") ||
		!strings.Contains(sa, "name: tf-runner") ||
		!strings.Contains(sa, "namespace: week-one") ||
		!strings.Contains(sa, "d7s.dev/managed-by: d7s") ||
		!strings.Contains(sa, "d7s.dev/stack: week-one") {
		t.Errorf("tf-runner ServiceAccount not compiled as expected:\n%s", sa)
	}
	rb := string(manifests.Files["apps/week-one/tf-runner-rolebinding.yaml"])
	if !strings.Contains(rb, "kind: RoleBinding") ||
		!strings.Contains(rb, "kind: ClusterRole") ||
		!strings.Contains(rb, "name: tf-runner-role") ||
		!strings.Contains(rb, "name: tf-runner") ||
		!strings.Contains(rb, "namespace: week-one") ||
		!strings.Contains(rb, "d7s.dev/managed-by: d7s") {
		t.Errorf("tf-runner RoleBinding not compiled as expected:\n%s", rb)
	}
}

// TestEmitSelfHostedPlacementNeverCompilesRunnerRBAC proves the tf-runner
// wiring is managed-only: a self-hosted-only stack has no Terraform CR
// and no runner pod to need it, so compiling it would be a dangling
// object nothing consumes (golden rule 24's spirit).
func TestEmitSelfHostedPlacementNeverCompilesRunnerRBAC(t *testing.T) {
	manifests, err := flux.New().Emit(exampleStack())
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}
	for path := range manifests.Files {
		if strings.Contains(path, "tf-runner") {
			t.Errorf("emitted %s for a self-hosted-only stack with no managed component", path)
		}
	}
}

// TestEmitManagedPlacementNeverInlinesAPIKey proves the emitted Terraform
// CR and OpenTofu config never carry a literal Neon API key value
// (golden rule 51): the key is always a Secret reference, resolved at
// runner-pod runtime.
func TestEmitManagedPlacementNeverInlinesAPIKey(t *testing.T) {
	manifests, err := flux.New().Emit(managedExampleStack())
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}
	tf := string(manifests.Files["apps/week-one/orders-db-terraform.yaml"])
	if !strings.Contains(tf, "secretKeyRef") || !strings.Contains(tf, "name: neon-api-key") {
		t.Errorf("Terraform CR does not inject the Neon API key from a Secret reference:\n%s", tf)
	}
	tfConfig := string(manifests.Files["apps/week-one/orders-db-managed/main.tf"])
	if strings.Contains(tfConfig, "api_key") {
		t.Errorf("OpenTofu config carries an api_key argument — the key must come only from the runner pod's environment:\n%s", tfConfig)
	}
}

// TestEmitManagedPlacementNeverInlinesProjectID proves the emitted
// Terraform CR and OpenTofu config never carry a literal Neon project id
// (week-two plan Revision 4): the project is a declared environment
// prerequisite, exactly like the Kubernetes cluster itself, and baking
// its id into compiled output would break determinism across
// environments (golden rules 22, 45). It is always a Terraform variable,
// supplied at runtime via the Terraform CR's varsFrom from the same
// neon-api-key Secret the API key comes from.
func TestEmitManagedPlacementNeverInlinesProjectID(t *testing.T) {
	manifests, err := flux.New().Emit(managedExampleStack())
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}
	tf := string(manifests.Files["apps/week-one/orders-db-terraform.yaml"])
	if !strings.Contains(tf, "varsFrom") ||
		!strings.Contains(tf, "name: neon-api-key") ||
		!strings.Contains(tf, "projectId:project_id") {
		t.Errorf("Terraform CR does not surface the Neon project id from the neon-api-key Secret via varsFrom:\n%s", tf)
	}
	tfConfig := string(manifests.Files["apps/week-one/orders-db-managed/main.tf"])
	if !strings.Contains(tfConfig, `variable "project_id"`) {
		t.Errorf("OpenTofu config does not declare a project_id variable:\n%s", tfConfig)
	}
	if !strings.Contains(tfConfig, "var.project_id") {
		t.Errorf("OpenTofu config does not reference var.project_id:\n%s", tfConfig)
	}
	// No project id ever compiled as a literal (the domain layer has no
	// project id to leak, so this also guards against a future field
	// being wired in as a literal by mistake).
	if strings.Contains(tfConfig, `resource "neon_project"`) {
		t.Errorf("OpenTofu config still declares a neon_project resource — Revision 4 supersedes project-per-stack with branch-per-stack:\n%s", tfConfig)
	}
}

// TestEmitManagedPlacementWritesOutputsToDeclaredCredentialsSecret proves
// the credentials-wiring fix (week-two plan slice 5, 2026-07-26):
// writeOutputsToSecret targets the component's DECLARED
// credentials.secretRef.name, not some other fixed or invented name —
// closing the gap that left Credentials.Name unconsumed for managed
// placement.
func TestEmitManagedPlacementWritesOutputsToDeclaredCredentialsSecret(t *testing.T) {
	manifests, err := flux.New().Emit(managedExampleStack())
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}
	tf := string(manifests.Files["apps/week-one/orders-db-terraform.yaml"])
	if !strings.Contains(tf, "writeOutputsToSecret:") || !strings.Contains(tf, "name: orders-db-app") {
		t.Errorf("Terraform CR does not write outputs to the declared credentials secret (orders-db-app):\n%s", tf)
	}
}

// TestEmitManagedPlacementNeverSetsDestroyResourcesOnDeletion proves
// retain-by-default (golden rule 28): the compiled CR must never set
// destroyResourcesOnDeletion — deleting compiled output must not destroy
// a data-bearing managed database. Only the acceptance harness, as an
// explicit operator act at teardown, may enable it on a running CR.
func TestEmitManagedPlacementNeverSetsDestroyResourcesOnDeletion(t *testing.T) {
	manifests, err := flux.New().Emit(managedExampleStack())
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}
	tf := string(manifests.Files["apps/week-one/orders-db-terraform.yaml"])
	if strings.Contains(tf, "destroyResourcesOnDeletion") {
		t.Errorf("compiled Terraform CR must never set destroyResourcesOnDeletion (golden rule 28):\n%s", tf)
	}
}

// TestEmitManagedPlacementDeclaresConnectionOutputs proves the OpenTofu
// config declares exactly the outputs a consumer needs to connect (host,
// port, database, username, password), with the password marked
// sensitive — the shape tofu-controller's writeOutputsToSecret turns
// into the credentials secret's data keys.
func TestEmitManagedPlacementDeclaresConnectionOutputs(t *testing.T) {
	manifests, err := flux.New().Emit(managedExampleStack())
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}
	tfConfig := string(manifests.Files["apps/week-one/orders-db-managed/main.tf"])
	for _, want := range []string{
		`output "host"`,
		`output "port"`,
		`output "database"`,
		`output "username"`,
		`output "password"`,
	} {
		if !strings.Contains(tfConfig, want) {
			t.Errorf("OpenTofu config is missing %s:\n%s", want, tfConfig)
		}
	}
	if !strings.Contains(tfConfig, "sensitive = true") {
		t.Errorf("OpenTofu config does not mark the password output sensitive:\n%s", tfConfig)
	}
}

// TestEmitManagedPlacementOrdersRoleAfterEndpoint proves neon_role
// declares an explicit depends_on against neon_endpoint (found live,
// 2026-07-26): the two resources share no attribute reference, so
// without this edge OpenTofu creates and destroys them in parallel -
// reproducibly breaking both directions against the real Neon API
// (create: "no read-write endpoint for branch"; destroy: "unexpected end
// of JSON input" from the provider's SDK). Confirmed by removing the edge
// and reproducing both failures twice, then restoring it and succeeding
// twice, against the real API - not a guess.
func TestEmitManagedPlacementOrdersRoleAfterEndpoint(t *testing.T) {
	manifests, err := flux.New().Emit(managedExampleStack())
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}
	tfConfig := string(manifests.Files["apps/week-one/orders-db-managed/main.tf"])
	roleIdx := strings.Index(tfConfig, `resource "neon_role"`)
	if roleIdx < 0 {
		t.Fatalf("OpenTofu config has no neon_role resource:\n%s", tfConfig)
	}
	roleBlock := tfConfig[roleIdx:]
	if !strings.Contains(roleBlock, "depends_on") || !strings.Contains(roleBlock, "neon_endpoint.orders-db") {
		t.Errorf("neon_role does not depend_on neon_endpoint:\n%s", roleBlock)
	}
}

// TestEmitSelfHostedExampleUnaffectedByManagedPlacement proves adding the
// managed-placement path did not change one byte of the existing
// self-hosted golden output (no regression) — the same assertion as
// TestEmitGoldenFiles, run again here to make the "no regression" claim
// explicit at the point managed placement was introduced.
func TestEmitSelfHostedExampleUnaffectedByManagedPlacement(t *testing.T) {
	manifests, err := flux.New().Emit(exampleStack())
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}
	want := readGoldenDir(t, filepath.Join("testdata", "golden", "week-one"))
	if len(manifests.Files) != len(want) {
		t.Fatalf("emitted %d files, golden has %d", len(manifests.Files), len(want))
	}
	for path, wantBytes := range want {
		gotBytes, ok := manifests.Files[path]
		if !ok {
			t.Errorf("golden file %s was not emitted", path)
			continue
		}
		if !bytes.Equal(gotBytes, wantBytes) {
			t.Errorf("emitted %s differs from golden after adding managed placement:\n--- got ---\n%s\n--- want ---\n%s", path, gotBytes, wantBytes)
		}
	}
}

// TestEmitBackupEgressAuthorizesOnlyDeclaredConsumers proves the declared
// backup wiring IS the allow-list (week-four plan, slice 1): the
// ServiceEntry names exactly the declared external's resolved host/port,
// and its AuthorizationPolicy's principals name exactly the consuming
// component's CNPG-created ServiceAccount — nothing wider, and the
// attachment is via targetRefs (a ServiceEntry has no workload labels a
// selector could match), enforced by the waypoint the ServiceEntry
// declares istio.io/use-waypoint for.
func TestEmitBackupEgressAuthorizesOnlyDeclaredConsumers(t *testing.T) {
	manifests, err := flux.New().Emit(exampleStack())
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}
	se := string(manifests.Files["apps/week-one/backups-serviceentry.yaml"])
	for _, want := range []string{
		"kind: ServiceEntry",
		"name: backups",
		"istio.io/use-waypoint: waypoint",
		"hosts:",
		"- minio.d7s-harness-minio.svc",
		"location: MESH_EXTERNAL",
		`number: 9000`,
		"protocol: TCP",
		"resolution: DNS",
	} {
		if !strings.Contains(se, want) {
			t.Errorf("backups ServiceEntry missing %q:\n%s", want, se)
		}
	}
	authz := string(manifests.Files["apps/week-one/backups-egress-authorizationpolicy.yaml"])
	if !strings.Contains(authz, "targetRefs:") ||
		!strings.Contains(authz, "kind: ServiceEntry") ||
		!strings.Contains(authz, "name: backups") {
		t.Errorf("backups AuthorizationPolicy does not attach via targetRefs to the ServiceEntry:\n%s", authz)
	}
	if !strings.Contains(authz, "cluster.local/ns/week-one/sa/orders-db") {
		t.Errorf("backups AuthorizationPolicy does not authorize the consuming component's own ServiceAccount:\n%s", authz)
	}
	if strings.Contains(authz, "selector:") {
		t.Errorf("backups AuthorizationPolicy uses a workload selector — a ServiceEntry has no workload to select:\n%s", authz)
	}
}

// TestEmitWaypointSharedAcrossBackupAndNeonEgress proves exactly one
// waypoint compiles per namespace even when both a self-hosted
// component's backup egress and a managed component's Neon egress are
// declared in the same stack — not one per egress target (week-four
// plan, slice 1's doc comment on emitWaypoint).
func TestEmitWaypointSharedAcrossBackupAndNeonEgress(t *testing.T) {
	stack := domain.Stack{
		Name: "week-one",
		Components: []domain.Component{
			domain.Postgres{
				Name:        "orders-db",
				Placement:   domain.PlacementSelfHosted,
				Credentials: domain.SecretRef{Name: "orders-db-app"},
				Guarantees: domain.Guarantees{
					RPO: &domain.RPOGuarantee{Target: time.Hour, BackupTo: "backups"},
				},
			},
			domain.Postgres{
				Name:         "widgets-db",
				Placement:    domain.PlacementManaged,
				Credentials:  domain.SecretRef{Name: "widgets-db-app"},
				EndpointHost: "ep-widgets-87654321.us-east-2.aws.neon.tech",
				AllowedConsumers: []domain.AllowedConsumer{
					{ServiceAccount: "probe-client"},
				},
			},
		},
		Externals: []domain.External{backupsExternal()},
	}
	manifests, err := flux.New().Emit(stack)
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}
	waypointCount := 0
	for path := range manifests.Files {
		if filepath.Base(path) == "waypoint.yaml" {
			waypointCount++
		}
	}
	if waypointCount != 1 {
		t.Errorf("expected exactly 1 waypoint for the namespace, got %d: %v", waypointCount, sortedKeys(manifests.Files))
	}
	if _, ok := manifests.Files["apps/week-one/backups-serviceentry.yaml"]; !ok {
		t.Error("expected the backup ServiceEntry to be compiled alongside the Neon one")
	}
	if _, ok := manifests.Files["apps/week-one/widgets-db-neon-serviceentry.yaml"]; !ok {
		t.Error("expected the Neon ServiceEntry to be compiled alongside the backup one")
	}
}

// TestEmitManagedWithoutAllowedConsumersEmitsControlPlaneEgressOnlyNoDataPlane
// proves presence is still the only signal for the DATA-PLANE edge
// specifically (golden rule 50): a managed component that declares no
// consumers compiles no per-component Neon ServiceEntry/
// AuthorizationPolicy (its branch endpoint stays reachable by nothing).
// It DOES still compile the shared waypoint and the control-plane
// ServiceEntry/AuthorizationPolicy scoped to tf-runner (egress.go's
// neonControlPlaneHost doc comment: a live-caught bug, 2026-07-26 —
// placement: managed always needs its own Terraform CR to reach Neon's
// API, regardless of declared consumers), so the namespace does join the
// ambient dataplane even here.
func TestEmitManagedWithoutAllowedConsumersEmitsControlPlaneEgressOnlyNoDataPlane(t *testing.T) {
	stack := domain.Stack{
		Name: "week-one",
		Components: []domain.Component{
			domain.Postgres{
				Name:        "orders-db",
				Placement:   domain.PlacementManaged,
				Credentials: domain.SecretRef{Name: "orders-db-app"},
			},
		},
	}
	manifests, err := flux.New().Emit(stack)
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}
	for path := range manifests.Files {
		base := filepath.Base(path)
		if strings.HasPrefix(base, "orders-db-neon-") {
			t.Errorf("emitted the data-plane object %s without any declared consumers", path)
		}
	}
	if _, ok := manifests.Files["apps/week-one/waypoint.yaml"]; !ok {
		t.Error("expected the shared waypoint to be compiled for the provisioner's control-plane edge even with no declared consumers")
	}
	if _, ok := manifests.Files["apps/week-one/neon-control-plane-serviceentry.yaml"]; !ok {
		t.Error("expected the control-plane ServiceEntry to be compiled even with no declared consumers")
	}
	if _, ok := manifests.Files["apps/week-one/neon-control-plane-egress-authorizationpolicy.yaml"]; !ok {
		t.Error("expected the control-plane AuthorizationPolicy to be compiled even with no declared consumers")
	}
	if !bytes.Contains(manifests.Files["apps/week-one/namespace.yaml"], []byte("dataplane-mode")) {
		t.Error("namespace does not carry the ambient dataplane label despite the provisioner's own control-plane edge needing it")
	}
}

// TestEmitNeonEgressUsesExactHostPin proves the exact-host pinning
// design (week-four plan, 2026-07-27 finding → Revision 2, supersedes
// the domain-pattern design — see egress.go's neonPostgresPort doc
// comment): the compiled ServiceEntry names pg.EndpointHost exactly,
// resolved via plain DNS — never a wildcard, never DYNAMIC_DNS.
func TestEmitNeonEgressUsesExactHostPin(t *testing.T) {
	manifests, err := flux.New().Emit(managedExampleStack())
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}
	se := string(manifests.Files["apps/week-one/orders-db-neon-serviceentry.yaml"])
	for _, want := range []string{
		managedExampleEndpointHost,
		"resolution: DNS",
		"protocol: TLS",
		`number: 5432`,
		"istio.io/use-waypoint: waypoint",
	} {
		if !strings.Contains(se, want) {
			t.Errorf("Neon ServiceEntry missing %q:\n%s", want, se)
		}
	}
	if strings.Contains(se, "*.neon.tech") || strings.Contains(se, "DYNAMIC_DNS") {
		t.Errorf("Neon ServiceEntry must not carry the retired wildcard host or DYNAMIC_DNS resolution:\n%s", se)
	}
	authz := string(manifests.Files["apps/week-one/orders-db-neon-egress-authorizationpolicy.yaml"])
	if !strings.Contains(authz, "cluster.local/ns/week-one/sa/probe-client") {
		t.Errorf("Neon AuthorizationPolicy does not authorize the declared consumer:\n%s", authz)
	}
}

// TestEmitNeonControlPlaneEgressGrantsOnlyTfRunnerOn443 is the
// composition-class test for the live-caught bug documented at
// egress.go's neonControlPlaneHost doc comment (2026-07-26): tf-runner
// (the Terraform CR's own runner pod, emitManagedRunnerRBAC) must be
// allowed the provisioner's control-plane edge (console.neon.tech:443)
// and NOT the data-plane port (5432); a declared consumer
// (probe-client) must be allowed the data-plane port (5432) and NOT the
// control-plane one (443) — the two edges never share a principal or a
// port, each enforced by its own ServiceEntry/AuthorizationPolicy pair
// rather than one shared, wider policy (golden rule 53). Uses
// managedExampleStack(), which declares both allowedConsumers and (via
// placement: managed) the always-present provisioner edge, so both
// pairs compile in the same call.
func TestEmitNeonControlPlaneEgressGrantsOnlyTfRunnerOn443(t *testing.T) {
	manifests, err := flux.New().Emit(managedExampleStack())
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}

	controlPlaneSE := string(manifests.Files["apps/week-one/neon-control-plane-serviceentry.yaml"])
	if !strings.Contains(controlPlaneSE, "console.neon.tech") {
		t.Errorf("control-plane ServiceEntry does not name console.neon.tech:\n%s", controlPlaneSE)
	}
	if !strings.Contains(controlPlaneSE, "number: 443") {
		t.Errorf("control-plane ServiceEntry does not carry port 443:\n%s", controlPlaneSE)
	}
	if strings.Contains(controlPlaneSE, "5432") {
		t.Errorf("control-plane ServiceEntry must not carry the data-plane port 5432:\n%s", controlPlaneSE)
	}

	controlPlaneAuthz := string(manifests.Files["apps/week-one/neon-control-plane-egress-authorizationpolicy.yaml"])
	if !strings.Contains(controlPlaneAuthz, "cluster.local/ns/week-one/sa/tf-runner") {
		t.Errorf("control-plane AuthorizationPolicy does not authorize tf-runner:\n%s", controlPlaneAuthz)
	}
	if strings.Contains(controlPlaneAuthz, "probe-client") {
		t.Errorf("control-plane AuthorizationPolicy must not authorize the declared consumer:\n%s", controlPlaneAuthz)
	}

	dataPlaneSE := string(manifests.Files["apps/week-one/orders-db-neon-serviceentry.yaml"])
	if !strings.Contains(dataPlaneSE, "number: 5432") {
		t.Errorf("data-plane ServiceEntry does not carry port 5432:\n%s", dataPlaneSE)
	}
	if strings.Contains(dataPlaneSE, "443") {
		t.Errorf("data-plane ServiceEntry must not carry the control-plane port 443:\n%s", dataPlaneSE)
	}

	dataPlaneAuthz := string(manifests.Files["apps/week-one/orders-db-neon-egress-authorizationpolicy.yaml"])
	if !strings.Contains(dataPlaneAuthz, "cluster.local/ns/week-one/sa/probe-client") {
		t.Errorf("data-plane AuthorizationPolicy does not authorize the declared consumer:\n%s", dataPlaneAuthz)
	}
	if strings.Contains(dataPlaneAuthz, "tf-runner") {
		t.Errorf("data-plane AuthorizationPolicy must not authorize tf-runner:\n%s", dataPlaneAuthz)
	}
}

// TestEmitManagedPlacementAlwaysCompilesControlPlaneEgress proves the
// provisioner's control-plane edge is implied by placement: managed
// itself, not by allowedConsumers (the live-caught bug's root cause,
// egress.go's neonControlPlaneHost doc comment): even the harness's own
// managed-without-consumers shape (mirrored from
// examples/week-three/harness-variant/stack.yaml) still compiles the
// waypoint and the control-plane ServiceEntry/AuthorizationPolicy.
func TestEmitManagedPlacementAlwaysCompilesControlPlaneEgress(t *testing.T) {
	stack := domain.Stack{
		Name: "harness-variant",
		Components: []domain.Component{
			domain.Postgres{
				Name:        "widgets-db",
				Placement:   domain.PlacementManaged,
				Credentials: domain.SecretRef{Name: "widgets-db-app"},
			},
		},
	}
	manifests, err := flux.New().Emit(stack)
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}
	for _, want := range []string{
		"apps/harness-variant/waypoint.yaml",
		"apps/harness-variant/neon-control-plane-serviceentry.yaml",
		"apps/harness-variant/neon-control-plane-egress-authorizationpolicy.yaml",
	} {
		if _, ok := manifests.Files[want]; !ok {
			t.Errorf("expected %s to be compiled for a managed component with no declared consumers, got %v", want, sortedKeys(manifests.Files))
		}
	}
}

// TestEmitEgressRefusesObjectStoreEndpointWithoutExplicitPort proves
// egress compilation's own compile-time check (rule 49's "shown able to
// fail and pass" spirit): a declared external whose endpoint has no
// explicit port cannot honestly compile a scoped ServiceEntry, so it
// refuses with a remedy (golden rules 34, 35) rather than guessing a
// default port, and nothing is written for that call. TestEmitGoldenFiles
// is this check's "pass" side, exercised against the well-formed
// examples/week-one/stack.yaml endpoint.
func TestEmitEgressRefusesObjectStoreEndpointWithoutExplicitPort(t *testing.T) {
	stack := domain.Stack{
		Name: "week-one",
		Components: []domain.Component{
			domain.Postgres{
				Name:        "orders-db",
				Placement:   domain.PlacementSelfHosted,
				Credentials: domain.SecretRef{Name: "orders-db-app"},
				Guarantees: domain.Guarantees{
					RPO: &domain.RPOGuarantee{Target: time.Hour, BackupTo: "backups"},
				},
			},
		},
		Externals: []domain.External{
			{
				Name: "backups",
				ObjectStore: &domain.ObjectStoreExternal{
					Endpoint:    "https://minio.d7s-harness.svc",
					Bucket:      "d7s-backups",
					Credentials: domain.SecretRef{Name: "backups-credentials"},
				},
			},
		},
	}
	manifests, err := flux.New().Emit(stack)
	if err == nil {
		t.Fatal("expected an error for an external endpoint with no explicit port, got nil")
	}
	if !strings.Contains(err.Error(), "no explicit port") {
		t.Errorf("error %q does not explain why the endpoint cannot compile", err.Error())
	}
	if !strings.Contains(err.Error(), "add a port to the declared endpoint") {
		t.Errorf("error %q does not carry a remedy (golden rule 35)", err.Error())
	}
	if len(manifests.Files) != 0 {
		t.Errorf("expected no files written on refusal, got %v", sortedKeys(manifests.Files))
	}
}

// TestEmitEgressAggregatesMalformedExternalErrors proves
// backupEgressTargets collects every malformed external across the whole
// stack rather than stopping at the first (golden rule 33: validate-time
// completeness) — a contract-review finding: two components each backing
// up to their own malformed external must report BOTH problems in one
// call, not just the first one encountered.
func TestEmitEgressAggregatesMalformedExternalErrors(t *testing.T) {
	stack := domain.Stack{
		Name: "week-one",
		Components: []domain.Component{
			domain.Postgres{
				Name:        "orders-db",
				Placement:   domain.PlacementSelfHosted,
				Credentials: domain.SecretRef{Name: "orders-db-app"},
				Guarantees: domain.Guarantees{
					RPO: &domain.RPOGuarantee{Target: time.Hour, BackupTo: "backups"},
				},
			},
			domain.Postgres{
				Name:        "widgets-db",
				Placement:   domain.PlacementSelfHosted,
				Credentials: domain.SecretRef{Name: "widgets-db-app"},
				Guarantees: domain.Guarantees{
					RPO: &domain.RPOGuarantee{Target: time.Hour, BackupTo: "widgets-backups"},
				},
			},
		},
		Externals: []domain.External{
			{
				Name: "backups",
				ObjectStore: &domain.ObjectStoreExternal{
					// No explicit port.
					Endpoint:    "https://minio.d7s-harness.svc",
					Bucket:      "d7s-backups",
					Credentials: domain.SecretRef{Name: "backups-credentials"},
				},
			},
			{
				Name: "widgets-backups",
				ObjectStore: &domain.ObjectStoreExternal{
					// Not a parseable URL at all.
					Endpoint:    "://not-a-url",
					Bucket:      "widgets-backups",
					Credentials: domain.SecretRef{Name: "widgets-backups-credentials"},
				},
			},
		},
	}
	_, err := flux.New().Emit(stack)
	if err == nil {
		t.Fatal("expected an error for two malformed externals, got nil")
	}
	joined := err.Error()
	if !strings.Contains(joined, `external "backups"`) || !strings.Contains(joined, "no explicit port") {
		t.Errorf("aggregated error %q does not include the \"backups\" refusal", joined)
	}
	if !strings.Contains(joined, `external "widgets-backups"`) {
		t.Errorf("aggregated error %q does not include the \"widgets-backups\" refusal — only the first malformed external was reported", joined)
	}
}

// unknownComponent is a domain.Component of a kind the flux emitter does
// not implement, used to prove the unimplemented-kind path refuses
// loudly (golden rule 34) instead of silently skipping it.
type unknownComponent struct{}

func (unknownComponent) ComponentName() string      { return "mystery" }
func (unknownComponent) Kind() domain.ComponentKind { return "kafka" }

func TestEmitUnknownComponentKindRefused(t *testing.T) {
	stack := domain.Stack{
		Name:       "week-one",
		Components: []domain.Component{unknownComponent{}},
	}
	_, err := flux.New().Emit(stack)
	if err == nil {
		t.Fatal("expected an error for an unimplemented component kind, got nil")
	}
}

func readGoldenDir(t *testing.T, dir string) map[string][]byte {
	t.Helper()
	files := map[string][]byte{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = b
		return nil
	})
	if err != nil {
		t.Fatalf("read golden dir %s: %v", dir, err)
	}
	return files
}

func sortedKeys(m map[string][]byte) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
