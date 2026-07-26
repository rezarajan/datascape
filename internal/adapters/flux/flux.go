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

	// postgresPort and cnpgStatusPort are CNPG's fixed container ports
	// (postgresql, status) — used to scope AuthorizationPolicy rules so
	// the database allow-list never accidentally covers the operator's
	// own control-plane traffic, or vice versa.
	postgresPort   = "5432"
	cnpgStatusPort = "8000"
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
// one component must not leave a partial compile behind. Components are
// then split by placement (week-two plan, slices 2+3): self-hosted
// compiles to CNPG, managed to a Terraform CR wrapping a Neon provider
// config — domain validation already guarantees no other placement value
// and no mtls/allowedConsumers/rpo on a managed component ever reaches
// here (internal/domain/postgres.go, internal/domain/guarantees.go), so
// the default branch below is a defensive, not a live, path.
func (e *Emitter) Emit(stack domain.Stack) (ports.Manifests, error) {
	var selfHosted, managed []domain.Postgres
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
		switch pg.Placement {
		case domain.PlacementSelfHosted:
			selfHosted = append(selfHosted, pg)
		case domain.PlacementManaged:
			managed = append(managed, pg)
		default:
			errs = append(errs, fmt.Errorf(
				"flux emitter: postgres component %q: placement %q reached the emitter without "+
					"domain validation catching it — this is a defect", pg.Name, pg.Placement))
		}
	}
	if len(errs) > 0 {
		return ports.Manifests{}, errors.Join(errs...)
	}
	total := len(selfHosted) + len(managed)

	meshEnabled := false
	for _, pg := range selfHosted {
		if pg.Guarantees.MTLS != nil {
			meshEnabled = true
		}
	}

	files := map[string][]byte{}
	if len(selfHosted) > 0 {
		if err := emitCNPGOperator(files, meshEnabled); err != nil {
			return ports.Manifests{}, err
		}
	}
	if total > 0 {
		if err := emitAppNamespace(files, stack.Name, meshEnabled); err != nil {
			return ports.Manifests{}, err
		}
	}
	if len(managed) > 0 {
		if err := emitManagedRunnerRBAC(files, stack.Name); err != nil {
			return ports.Manifests{}, err
		}
	}
	for _, pg := range selfHosted {
		if err := emitSelfHostedPostgres(files, stack.Name, pg); err != nil {
			return ports.Manifests{}, err
		}
	}
	for _, pg := range managed {
		if err := emitManagedPostgres(files, stack.Name, pg); err != nil {
			return ports.Manifests{}, err
		}
	}
	if total > 0 {
		if err := emitAppKustomization(files, stack.Name, len(selfHosted) > 0); err != nil {
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

// emitSelfHostedPostgres compiles the self-hosted (CNPG) artifact for pg:
// the Cluster CR plus the zero-trust and durability guarantee triples'
// emitted-infra elements. Named explicitly (rather than plain
// "emitPostgres") since placement: managed now compiles through a
// different function, emitManagedPostgres (week-two plan, slices 2+3).
func emitSelfHostedPostgres(files map[string][]byte, stackName string, pg domain.Postgres) error {
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

	rules := make([]AuthorizationPolicyRule, 0, len(pg.AllowedConsumers)+1)
	for _, consumer := range pg.AllowedConsumers {
		rules = append(rules, AuthorizationPolicyRule{
			From: []AuthorizationPolicyFrom{
				{Source: AuthorizationPolicySource{Principals: []string{principal(stackName, consumer)}}},
			},
			To: []AuthorizationPolicyTo{
				{Operation: AuthorizationPolicyOperation{Ports: []string{postgresPort}}},
			},
		})
	}
	// The CNPG operator polls each instance's status endpoint to manage
	// the cluster it created — not a declared consumer of the database,
	// but an operational necessity of the durability guarantee's own
	// emitted infra. Scoped to the status port only, never the database
	// port, so it grants no data access.
	rules = append(rules, AuthorizationPolicyRule{
		From: []AuthorizationPolicyFrom{
			{Source: AuthorizationPolicySource{Namespaces: []string{cnpgSystemNamespace}}},
		},
		To: []AuthorizationPolicyTo{
			{Operation: AuthorizationPolicyOperation{Ports: []string{cnpgStatusPort}}},
		},
	})

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
// meshEnabled adds the operator's namespace to the Istio ambient mesh.
// Found necessary by running the acceptance harness live: STRICT mTLS
// on the app namespace rejects the operator's own status-polling
// connection unless the operator's namespace can itself originate an
// ambient (HBONE) connection — PeerAuthentication enforces at the
// transport layer, before AuthorizationPolicy is ever evaluated, so
// scoping the policy alone (see emitZeroTrust) was not sufficient
// (golden rule 40: only a real workload on real infrastructure proved
// this; rule 42: composition gets its own acceptance tests).
func emitCNPGOperator(files map[string][]byte, meshEnabled bool) error {
	labels := ownershipLabels("", "")
	if meshEnabled {
		labels["istio.io/dataplane-mode"] = "ambient"
	}
	ns := Namespace{
		APIVersion: "v1",
		Kind:       "Namespace",
		Metadata: ObjectMeta{
			Name:   cnpgSystemNamespace,
			Labels: labels,
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

// emitAppKustomization emits the per-stack app-layer Kustomization.
// dependsOnCNPGOperator is true only when the stack has at least one
// self-hosted component: that Cluster CR needs the CNPG CRDs the infra
// layer installs — a real ordering dependency, compiled into Flux's own
// dependency mechanism (golden rule 24). A managed-only stack has no
// such edge: tofu-controller's own CRDs are a declared environment
// prerequisite this week, not something d7s compiles (week-two plan,
// "explicitly NOT this week"), so adding a dependsOn to a Kustomization
// this compile never emits would be a dangling, hidden edge — exactly
// what rule 24 warns against.
func emitAppKustomization(files map[string][]byte, stackName string, dependsOnCNPGOperator bool) error {
	spec := KustomizationSpec{
		Interval:  reconcileInterval,
		Path:      fmt.Sprintf("./out/apps/%s", stackName),
		Prune:     true,
		SourceRef: gitSourceRef,
	}
	if dependsOnCNPGOperator {
		spec.DependsOn = []DependsOn{
			{Name: infraKustomizationName, Namespace: fluxSystemNamespace},
		}
	}
	k := Kustomization{
		APIVersion: "kustomize.toolkit.fluxcd.io/v1",
		Kind:       "Kustomization",
		Metadata: ObjectMeta{
			Name:      stackName,
			Namespace: fluxSystemNamespace,
			Labels:    ownershipLabels(stackName, ""),
		},
		Spec: spec,
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
