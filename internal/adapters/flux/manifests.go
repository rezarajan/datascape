package flux

// ObjectMeta is the subset of Kubernetes object metadata d7s emits.
// Labels/Annotations use plain maps: gopkg.in/yaml.v3 marshals
// map[string]string keys in sorted order, which is what keeps compiled
// output byte-identical across runs (golden rules 22, 45). Annotations
// is nil (and so omitted, never an empty mapping) for every object that
// carries no conditional-guarantee label — only a component whose
// durability guarantee compiles labeled CONDITIONAL sets it (golden
// rule 49: the label must be testable-absent).
type ObjectMeta struct {
	Name        string            `yaml:"name"`
	Namespace   string            `yaml:"namespace,omitempty"`
	Labels      map[string]string `yaml:"labels"`
	Annotations map[string]string `yaml:"annotations,omitempty"`
}

// Namespace is a Kubernetes Namespace.
type Namespace struct {
	APIVersion string     `yaml:"apiVersion"`
	Kind       string     `yaml:"kind"`
	Metadata   ObjectMeta `yaml:"metadata"`
}

// ServiceAccount is a Kubernetes ServiceAccount.
type ServiceAccount struct {
	APIVersion string     `yaml:"apiVersion"`
	Kind       string     `yaml:"kind"`
	Metadata   ObjectMeta `yaml:"metadata"`
}

// RoleBinding is a Kubernetes rbac.authorization.k8s.io RoleBinding —
// namespace-scoped, even though its RoleRef may name a cluster-scoped
// ClusterRole (exactly how emitManagedRunnerRBAC uses it: scoping a
// pre-existing, environment-provided ClusterRole's permissions to one
// namespace, rather than granting them cluster-wide).
type RoleBinding struct {
	APIVersion string               `yaml:"apiVersion"`
	Kind       string               `yaml:"kind"`
	Metadata   ObjectMeta           `yaml:"metadata"`
	RoleRef    RoleRef              `yaml:"roleRef"`
	Subjects   []RoleBindingSubject `yaml:"subjects"`
}

// RoleRef is RoleBinding.roleRef.
type RoleRef struct {
	APIGroup string `yaml:"apiGroup"`
	Kind     string `yaml:"kind"`
	Name     string `yaml:"name"`
}

// RoleBindingSubject is one entry of RoleBinding.subjects.
type RoleBindingSubject struct {
	Kind      string `yaml:"kind"`
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
}

// SourceRef points a Flux object at the Source it reads from.
type SourceRef struct {
	APIVersion string `yaml:"apiVersion,omitempty"`
	Kind       string `yaml:"kind"`
	Name       string `yaml:"name"`
	Namespace  string `yaml:"namespace,omitempty"`
}

// HelmRepository is a Flux source.toolkit.fluxcd.io HelmRepository.
type HelmRepository struct {
	APIVersion string             `yaml:"apiVersion"`
	Kind       string             `yaml:"kind"`
	Metadata   ObjectMeta         `yaml:"metadata"`
	Spec       HelmRepositorySpec `yaml:"spec"`
}

// HelmRepositorySpec is the subset of HelmRepository.spec d7s emits.
type HelmRepositorySpec struct {
	Interval string `yaml:"interval"`
	URL      string `yaml:"url"`
}

// HelmRelease is a Flux helm.toolkit.fluxcd.io HelmRelease.
type HelmRelease struct {
	APIVersion string          `yaml:"apiVersion"`
	Kind       string          `yaml:"kind"`
	Metadata   ObjectMeta      `yaml:"metadata"`
	Spec       HelmReleaseSpec `yaml:"spec"`
}

// HelmReleaseSpec is the subset of HelmRelease.spec d7s emits.
type HelmReleaseSpec struct {
	Interval        string    `yaml:"interval"`
	TargetNamespace string    `yaml:"targetNamespace"`
	Chart           HelmChart `yaml:"chart"`
}

// HelmChart is HelmReleaseSpec.chart.
type HelmChart struct {
	Spec HelmChartSpec `yaml:"spec"`
}

// HelmChartSpec is HelmReleaseSpec.chart.spec.
type HelmChartSpec struct {
	Chart     string    `yaml:"chart"`
	Version   string    `yaml:"version"`
	SourceRef SourceRef `yaml:"sourceRef"`
}

// DependsOn is one entry of a Kustomization's spec.dependsOn.
type DependsOn struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace,omitempty"`
}

// HealthCheck is one entry of a Kustomization's spec.healthChecks: a
// specific object whose readiness gates this Kustomization's own Ready
// condition (kustomize-controller polls it via kstatus after applying).
// week-three plan, slice 4 (closing the week-two slice-4 live finding
// recorded in docs/plans/01-week-one.md's dated notes): naming the CNPG
// operator's HelmRelease here is what makes a dependent Kustomization's
// existing `dependsOn` gate on the operator's *actual* readiness, not
// merely on this Kustomization's own apply having succeeded. Verified
// against the upstream Flux 2.x docs (fluxcd.io/flux/components/kustomize/
// kustomizations/), not memory: `.spec.healthChecks` accepts exactly this
// {apiVersion, kind, name, namespace} shape, and "if `.spec.healthChecks`
// is non-empty ... a Kustomization will be applied after all its
// dependencies' health checks have passed."
type HealthCheck struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Name       string `yaml:"name"`
	Namespace  string `yaml:"namespace,omitempty"`
}

// Kustomization is a Flux kustomize.toolkit.fluxcd.io Kustomization: the
// control object that tells Flux to reconcile a path. It is not the
// plain kustomize.config.k8s.io Kustomization file.
type Kustomization struct {
	APIVersion string            `yaml:"apiVersion"`
	Kind       string            `yaml:"kind"`
	Metadata   ObjectMeta        `yaml:"metadata"`
	Spec       KustomizationSpec `yaml:"spec"`
}

// KustomizationSpec is the subset of Kustomization.spec d7s emits.
type KustomizationSpec struct {
	Interval     string        `yaml:"interval"`
	Path         string        `yaml:"path"`
	Prune        bool          `yaml:"prune"`
	SourceRef    SourceRef     `yaml:"sourceRef"`
	DependsOn    []DependsOn   `yaml:"dependsOn,omitempty"`
	HealthChecks []HealthCheck `yaml:"healthChecks,omitempty"`
}

// CNPGCluster is a CloudNativePG postgresql.cnpg.io Cluster.
type CNPGCluster struct {
	APIVersion string          `yaml:"apiVersion"`
	Kind       string          `yaml:"kind"`
	Metadata   ObjectMeta      `yaml:"metadata"`
	Spec       CNPGClusterSpec `yaml:"spec"`
}

// CNPGClusterSpec is the subset of Cluster.spec d7s emits.
type CNPGClusterSpec struct {
	Instances int           `yaml:"instances"`
	Bootstrap CNPGBootstrap `yaml:"bootstrap"`
	Storage   CNPGStorage   `yaml:"storage"`
	Backup    *CNPGBackup   `yaml:"backup,omitempty"`
}

// CNPGBootstrap is Cluster.spec.bootstrap.
type CNPGBootstrap struct {
	InitDB CNPGInitDB `yaml:"initdb"`
}

// CNPGInitDB is Cluster.spec.bootstrap.initdb. Secret references a
// pre-existing Kubernetes Secret by name only — d7s never sees or
// provisions the credential value (golden rule 51).
type CNPGInitDB struct {
	Database string        `yaml:"database"`
	Owner    string        `yaml:"owner"`
	Secret   CNPGSecretRef `yaml:"secret"`
}

// CNPGSecretRef names a Secret.
type CNPGSecretRef struct {
	Name string `yaml:"name"`
}

// CNPGStorage is Cluster.spec.storage.
type CNPGStorage struct {
	Size string `yaml:"size"`
}
