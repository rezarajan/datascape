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
					RPO:  &domain.RPOGuarantee{Target: time.Hour},
				},
				AllowedConsumers: []domain.AllowedConsumer{
					{ServiceAccount: "probe-client"},
				},
			},
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

// TestEmitWithoutRPOGuaranteeEmitsNoBackupObjects mirrors the mTLS case
// for durability: no ScheduledBackup and no Cluster.spec.backup unless
// the RPO guarantee is declared.
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
					RPO: &domain.RPOGuarantee{Target: 2 * time.Minute},
				},
			},
		},
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
