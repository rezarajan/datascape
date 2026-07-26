// Package flux is the Flux target emitter adapter: it compiles a Stack
// into Flux-consumable Kubernetes manifests. It implements ports.Emitter
// and imports only domain and ports (golden rule 8).
//
// v1 convention (declared, not resolved — golden rule 15 does not apply
// to a repo-layout convention Flux itself requires at compile time):
// compiled output is always committed at the git repo root as ./out,
// exactly as the acceptance scenario invokes it
// (docs/plans/01-week-one.md). The emitted Kustomization spec.path
// fields assume this; a different output location needs the paths
// adjusted by hand until v2 makes the convention configurable.
package flux

import (
	"errors"
	"fmt"
	"time"

	"github.com/rezarajan/datascape/internal/domain"
	"github.com/rezarajan/datascape/internal/ports"
)

const (
	cnpgSystemNamespace = "cnpg-system"
	cnpgHelmRepoName    = "cloudnative-pg"
	cnpgHelmRepoURL     = "https://cloudnative-pg.github.io/charts"
	cnpgHelmReleaseName = "cnpg-operator"
	cnpgChartName       = "cloudnative-pg"
	// cnpgChartVersion is pinned for determinism; bump deliberately via a
	// dated commit, never silently.
	cnpgChartVersion = "0.22.1"

	infraKustomizationName = "cnpg-operator"
	fluxSystemNamespace    = "flux-system"
	reconcileInterval      = "10m"

	// minSupportedRPO is the smallest RPO this emitter can honor with
	// scheduled base backups (no continuous WAL-archiving support this
	// week — a known, deliberate gap, golden rule 7). A smaller declared
	// RPO fails compilation rather than silently under-delivering
	// (golden rules 34, 37, 50: no best-effort tier for a guarantee).
	minSupportedRPO = 5 * time.Minute

	// defaultRetentionPolicy is fixed for v1: retention isn't yet a
	// declared field (golden rule 7).
	defaultRetentionPolicy = "30d"
)

// gitSourceRef is the Flux Source every Kustomization/HelmRelease this
// emitter produces reads from. Flux's own bootstrap (the GitRepository
// object itself, and Flux's controllers) is a declared environment
// prerequisite this week, not something d7s compiles (week-one plan,
// "explicitly NOT this week").
var gitSourceRef = SourceRef{
	Kind:      "GitRepository",
	Name:      "d7s",
	Namespace: fluxSystemNamespace,
}

// Emitter compiles a Stack to Flux manifests.
type Emitter struct{}

// New builds a Flux Emitter.
func New() *Emitter {
	return &Emitter{}
}

// Emit implements ports.Emitter. It checks every component against what
// this target can actually honor before writing anything (golden
// rule 33's spirit carried into the emitter): an unsatisfiable RPO on
// one component must not leave a partial compile behind.
func (e *Emitter) Emit(stack domain.Stack) (ports.Manifests, error) {
	var pgs []domain.Postgres
	var errs []error
	for _, c := range stack.Components {
		pg, ok := c.(domain.Postgres)
		if !ok {
			errs = append(errs, fmt.Errorf(
				"flux emitter: component %q has kind %q, which is planned, not yet available",
				c.ComponentName(), c.Kind()))
			continue
		}
		if err := checkRPOSatisfiable(pg); err != nil {
			errs = append(errs, err)
			continue
		}
		pgs = append(pgs, pg)
	}
	if len(errs) > 0 {
		return ports.Manifests{}, errors.Join(errs...)
	}

	meshEnabled := false
	for _, pg := range pgs {
		if pg.Guarantees.MTLS != nil {
			meshEnabled = true
		}
	}

	files := map[string][]byte{}
	if err := emitAppNamespace(files, stack.Name, meshEnabled); err != nil {
		return ports.Manifests{}, err
	}
	for _, pg := range pgs {
		if err := emitPostgres(files, stack.Name, pg); err != nil {
			return ports.Manifests{}, err
		}
	}
	if len(pgs) > 0 {
		if err := emitAppKustomization(files, stack.Name); err != nil {
			return ports.Manifests{}, err
		}
	}
	return ports.Manifests{Files: files}, nil
}

// emitAppNamespace emits the stack's application namespace once.
// meshEnabled adds the Istio ambient dataplane label: without it,
// ztunnel never intercepts the namespace's traffic and a declared mtls
// guarantee would compile PeerAuthentication/AuthorizationPolicy objects
// that exist but are never enforced (found by running the acceptance
// harness against a live ambient mesh, not assumed — golden rule 40).
func emitAppNamespace(files map[string][]byte, stackName string, meshEnabled bool) error {
	labels := ownershipLabels(stackName, "")
	if meshEnabled {
		labels["istio.io/dataplane-mode"] = "ambient"
	}
	ns := Namespace{
		APIVersion: "v1",
		Kind:       "Namespace",
		Metadata: ObjectMeta{
			Name:   stackName,
			Labels: labels,
		},
	}
	return set(files, fmt.Sprintf("apps/%s/namespace.yaml", stackName), ns)
}

// checkRPOSatisfiable is the durability guarantee's compile-time check
// (problem definition Amendment 1, "the guarantee primitive"): a
// declared RPO this emitter cannot honor fails compilation with the
// remedy in the error, rather than compiling a schedule that quietly
// can't meet it.
func checkRPOSatisfiable(pg domain.Postgres) error {
	if pg.Guarantees.RPO == nil {
		return nil
	}
	if pg.Guarantees.RPO.Target < minSupportedRPO {
		return fmt.Errorf(
			"flux emitter: postgres component %q: guarantees.rpo of %s cannot be honored — "+
				"the minimum RPO this emitter supports is %s (scheduled base backups only; "+
				"continuous WAL archiving is planned, not yet available); declare a larger value",
			pg.Name, pg.Guarantees.RPO.Target, minSupportedRPO)
	}
	return nil
}

func emitPostgres(files map[string][]byte, stackName string, pg domain.Postgres) error {
	if err := emitCNPGOperator(files); err != nil {
		return err
	}
	if err := emitCluster(files, stackName, pg); err != nil {
		return err
	}
	if err := emitZeroTrust(files, stackName, pg); err != nil {
		return err
	}
	return emitDurability(files, stackName, pg)
}

// emitZeroTrust compiles the transport-security guarantee triple's
// emitted-infra element: STRICT PeerAuthentication plus an
// AuthorizationPolicy whose allow rules come only from declared wiring
// (golden rule 53) — an empty AllowedConsumers list compiles to a
// default-deny AuthorizationPolicy (empty rules), never an implicit
// allow. Emits nothing if the mtls guarantee isn't declared for pg.
//
// Known gap (golden rule 7): PeerAuthentication is namespace-wide by
// Istio's own model, but the mtls guarantee is declared per component.
// With one component per namespace this week that distinction doesn't
// yet bite; it will need resolving before a second component can share
// a namespace with mixed mtls declarations.
func emitZeroTrust(files map[string][]byte, stackName string, pg domain.Postgres) error {
	if pg.Guarantees.MTLS == nil {
		return nil
	}

	peerAuth := PeerAuthentication{
		APIVersion: "security.istio.io/v1",
		Kind:       "PeerAuthentication",
		Metadata: ObjectMeta{
			Name:      "default",
			Namespace: stackName,
			Labels:    ownershipLabels(stackName, ""),
		},
		Spec: PeerAuthenticationSpec{
			MTLS: PeerAuthenticationMTLS{Mode: "STRICT"},
		},
	}
	if err := set(files, fmt.Sprintf("apps/%s/peerauthentication.yaml", stackName), peerAuth); err != nil {
		return err
	}

	rules := make([]AuthorizationPolicyRule, 0, len(pg.AllowedConsumers))
	for _, consumer := range pg.AllowedConsumers {
		rules = append(rules, AuthorizationPolicyRule{
			From: []AuthorizationPolicyFrom{
				{Source: AuthorizationPolicySource{Principals: []string{principal(stackName, consumer)}}},
			},
		})
	}

	authzPolicy := AuthorizationPolicy{
		APIVersion: "security.istio.io/v1",
		Kind:       "AuthorizationPolicy",
		Metadata: ObjectMeta{
			Name:      pg.Name,
			Namespace: stackName,
			Labels:    ownershipLabels(stackName, pg.Name),
		},
		Spec: AuthorizationPolicySpec{
			Selector: AuthorizationPolicySelector{
				MatchLabels: map[string]string{"cnpg.io/cluster": pg.Name},
			},
			Rules: rules,
		},
	}
	return set(files, fmt.Sprintf("apps/%s/%s-authorizationpolicy.yaml", stackName, pg.Name), authzPolicy)
}

// principal resolves a declared AllowedConsumer to its mesh identity
// (SPIFFE principal). Namespace defaults to the component's own stack
// namespace when the consumer doesn't declare one.
func principal(stackName string, consumer domain.AllowedConsumer) string {
	ns := consumer.Namespace
	if ns == "" {
		ns = stackName
	}
	return fmt.Sprintf("cluster.local/ns/%s/sa/%s", ns, consumer.ServiceAccount)
}

// emitCNPGOperator emits everything an empty cluster needs to run the
// CNPG operator: its namespace, the Helm chart source and release, and
// the Flux Kustomization that reconciles them. Safe to call once per
// stack even with multiple postgres components — later calls overwrite
// identical bytes.
func emitCNPGOperator(files map[string][]byte) error {
	ns := Namespace{
		APIVersion: "v1",
		Kind:       "Namespace",
		Metadata: ObjectMeta{
			Name:   cnpgSystemNamespace,
			Labels: ownershipLabels("", ""),
		},
	}
	if err := set(files, "infra/cnpg-operator/namespace.yaml", ns); err != nil {
		return err
	}

	repo := HelmRepository{
		APIVersion: "source.toolkit.fluxcd.io/v1",
		Kind:       "HelmRepository",
		Metadata: ObjectMeta{
			Name:      cnpgHelmRepoName,
			Namespace: cnpgSystemNamespace,
			Labels:    ownershipLabels("", ""),
		},
		Spec: HelmRepositorySpec{
			Interval: reconcileInterval,
			URL:      cnpgHelmRepoURL,
		},
	}
	if err := set(files, "infra/cnpg-operator/helmrepository.yaml", repo); err != nil {
		return err
	}

	release := HelmRelease{
		APIVersion: "helm.toolkit.fluxcd.io/v2",
		Kind:       "HelmRelease",
		Metadata: ObjectMeta{
			Name:      cnpgHelmReleaseName,
			Namespace: cnpgSystemNamespace,
			Labels:    ownershipLabels("", ""),
		},
		Spec: HelmReleaseSpec{
			Interval:        reconcileInterval,
			TargetNamespace: cnpgSystemNamespace,
			Chart: HelmChart{
				Spec: HelmChartSpec{
					Chart:   cnpgChartName,
					Version: cnpgChartVersion,
					SourceRef: SourceRef{
						Kind:      "HelmRepository",
						Name:      cnpgHelmRepoName,
						Namespace: cnpgSystemNamespace,
					},
				},
			},
		},
	}
	if err := set(files, "infra/cnpg-operator/helmrelease.yaml", release); err != nil {
		return err
	}

	infra := Kustomization{
		APIVersion: "kustomize.toolkit.fluxcd.io/v1",
		Kind:       "Kustomization",
		Metadata: ObjectMeta{
			Name:      infraKustomizationName,
			Namespace: fluxSystemNamespace,
			Labels:    ownershipLabels("", ""),
		},
		Spec: KustomizationSpec{
			Interval:  reconcileInterval,
			Path:      "./out/infra/cnpg-operator",
			Prune:     true,
			SourceRef: gitSourceRef,
		},
	}
	return set(files, "flux/infra-cnpg-operator.yaml", infra)
}

func emitCluster(files map[string][]byte, stackName string, pg domain.Postgres) error {
	cluster := CNPGCluster{
		APIVersion: "postgresql.cnpg.io/v1",
		Kind:       "Cluster",
		Metadata: ObjectMeta{
			Name:      pg.Name,
			Namespace: stackName,
			Labels:    ownershipLabels(stackName, pg.Name),
		},
		Spec: CNPGClusterSpec{
			Instances: 1,
			Bootstrap: CNPGBootstrap{
				InitDB: CNPGInitDB{
					Database: pg.Name,
					Owner:    pg.Name,
					Secret:   CNPGSecretRef{Name: pg.Credentials.Name},
				},
			},
			Storage: CNPGStorage{Size: "1Gi"},
		},
	}
	if pg.Guarantees.RPO != nil {
		cluster.Spec.Backup = &CNPGBackup{RetentionPolicy: defaultRetentionPolicy}
	}
	return set(files, fmt.Sprintf("apps/%s/%s-cluster.yaml", stackName, pg.Name), cluster)
}

// emitDurability compiles the durability guarantee triple's emitted-infra
// element: a ScheduledBackup whose cadence is derived directly from the
// declared RPO. Emits nothing if the RPO guarantee isn't declared.
func emitDurability(files map[string][]byte, stackName string, pg domain.Postgres) error {
	if pg.Guarantees.RPO == nil {
		return nil
	}
	sb := ScheduledBackup{
		APIVersion: "postgresql.cnpg.io/v1",
		Kind:       "ScheduledBackup",
		Metadata: ObjectMeta{
			Name:      pg.Name,
			Namespace: stackName,
			Labels:    ownershipLabels(stackName, pg.Name),
		},
		Spec: ScheduledBackupSpec{
			Schedule:  "@every " + pg.Guarantees.RPO.Target.String(),
			Cluster:   ScheduledBackupRef{Name: pg.Name},
			Immediate: true,
		},
	}
	return set(files, fmt.Sprintf("apps/%s/%s-scheduledbackup.yaml", stackName, pg.Name), sb)
}

func emitAppKustomization(files map[string][]byte, stackName string) error {
	k := Kustomization{
		APIVersion: "kustomize.toolkit.fluxcd.io/v1",
		Kind:       "Kustomization",
		Metadata: ObjectMeta{
			Name:      stackName,
			Namespace: fluxSystemNamespace,
			Labels:    ownershipLabels(stackName, ""),
		},
		Spec: KustomizationSpec{
			Interval:  reconcileInterval,
			Path:      fmt.Sprintf("./out/apps/%s", stackName),
			Prune:     true,
			SourceRef: gitSourceRef,
			// The app layer's Cluster CR needs the CNPG CRDs the infra
			// layer installs — a real ordering dependency, compiled into
			// Flux's own dependency mechanism (golden rule 24).
			DependsOn: []DependsOn{
				{Name: infraKustomizationName, Namespace: fluxSystemNamespace},
			},
		},
	}
	return set(files, fmt.Sprintf("flux/apps-%s.yaml", stackName), k)
}

func set(files map[string][]byte, path string, v any) error {
	b, err := marshalYAML(v)
	if err != nil {
		return fmt.Errorf("flux emitter: marshal %s: %w", path, err)
	}
	files[path] = b
	return nil
}
