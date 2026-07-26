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
	"fmt"

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

// Emit implements ports.Emitter.
func (e *Emitter) Emit(stack domain.Stack) (ports.Manifests, error) {
	files := map[string][]byte{}

	for _, c := range stack.Components {
		pg, ok := c.(domain.Postgres)
		if !ok {
			return ports.Manifests{}, fmt.Errorf(
				"flux emitter: component %q has kind %q, which is planned, not yet available",
				c.ComponentName(), c.Kind())
		}
		if err := emitPostgres(files, stack.Name, pg); err != nil {
			return ports.Manifests{}, err
		}
	}

	return ports.Manifests{Files: files}, nil
}

func emitPostgres(files map[string][]byte, stackName string, pg domain.Postgres) error {
	if err := emitCNPGOperator(files); err != nil {
		return err
	}
	if err := emitCluster(files, stackName, pg); err != nil {
		return err
	}
	return emitAppKustomization(files, stackName)
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
	appNS := Namespace{
		APIVersion: "v1",
		Kind:       "Namespace",
		Metadata: ObjectMeta{
			Name:   stackName,
			Labels: ownershipLabels(stackName, ""),
		},
	}
	if err := set(files, fmt.Sprintf("apps/%s/namespace.yaml", stackName), appNS); err != nil {
		return err
	}

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
	return set(files, fmt.Sprintf("apps/%s/%s-cluster.yaml", stackName, pg.Name), cluster)
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
