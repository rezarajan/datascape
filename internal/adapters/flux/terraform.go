package flux

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/rezarajan/datascape/internal/domain"
)

const (
	// terraformAPIVersion is tofu-controller's Terraform custom resource
	// group/version. Verified against the tag recorded as pinnable by
	// the week-two slice-1 feasibility spike (v0.16.4, 2026-06-08 —
	// TASK_PROGRESS, 2026-07-26): current for the whole v0.16.x line, not
	// assumed.
	terraformAPIVersion = "infra.contrib.fluxcd.io/v1alpha2"

	// neonProviderSource and neonProviderVersion pin the community
	// kislerdm/neon Terraform provider (golden rule 61 — no first-party
	// Neon provider exists; this is the ~584k-download community one
	// recorded in the slice-1 feasibility spike). Bumped deliberately via
	// a dated commit, never silently — mirrors cnpgChartVersion's
	// discipline in flux.go.
	neonProviderSource  = "kislerdm/neon"
	neonProviderVersion = "0.14.0"

	// neonAPIKeySecretName is the Kubernetes Secret every managed
	// Postgres component's Terraform CR reads its Neon API key from
	// (golden rule 51 — never a literal value in the emitted config).
	// Fixed for v1 (owner decision, week-two plan Revision 1: "the
	// API-key secret stays in the design ... default name neon-api-key
	// stands unless the owner renames it") — not yet a declared schema
	// field, a known v1 gap (golden rule 7).
	neonAPIKeySecretName = "neon-api-key"
	// neonAPIKeySecretKey is the key inside that Secret's data the API
	// key value lives under. d7s never provisions or reads this secret
	// (rule 51) — this constant is the single documented naming
	// convention the harness/operator must follow to populate it.
	neonAPIKeySecretKey = "apiKey"

	// neonAPIKeyEnvVar is the environment variable the kislerdm/neon
	// provider reads api_key from when the provider block itself leaves
	// the argument unset (verified against the provider's docs,
	// 2026-07-26) — the mechanism that lets the runner pod inject the
	// key without it ever appearing in the emitted HCL.
	neonAPIKeyEnvVar = "NEON_API_KEY"
)

// Terraform is a tofu-controller infra.contrib.fluxcd.io Terraform custom
// resource: the delivery object that reconciles an OpenTofu configuration
// through the same Flux git source every other emitted object reads from
// (golden rule 24 — one delivery plane, no second apply path; week-two
// plan).
type Terraform struct {
	APIVersion string        `yaml:"apiVersion"`
	Kind       string        `yaml:"kind"`
	Metadata   ObjectMeta    `yaml:"metadata"`
	Spec       TerraformSpec `yaml:"spec"`
}

// TerraformSpec is the subset of Terraform.spec d7s emits. ApprovePlan is
// fixed to "auto" (week-two plan build order item 2): there is no
// interactive plan-review step in a GitOps compiler with no mutating
// verbs of its own (problem definition Q3.2) — the reviewable artifact is
// the git diff of this compiled CR and its .tf files, same as every other
// emitted object (golden rule 20's binding interpretation).
type TerraformSpec struct {
	Interval          string                     `yaml:"interval"`
	Path              string                     `yaml:"path"`
	ApprovePlan       string                     `yaml:"approvePlan"`
	SourceRef         SourceRef                  `yaml:"sourceRef"`
	RunnerPodTemplate TerraformRunnerPodTemplate `yaml:"runnerPodTemplate"`
}

// TerraformRunnerPodTemplate is Terraform.spec.runnerPodTemplate — used
// here only to inject the Neon API key into the runner pod's environment
// from a referenced Secret (golden rule 51: never a value in the emitted
// config itself).
type TerraformRunnerPodTemplate struct {
	Spec TerraformRunnerPodSpec `yaml:"spec"`
}

// TerraformRunnerPodSpec is TerraformRunnerPodTemplate.spec.
type TerraformRunnerPodSpec struct {
	Env []TerraformEnvVar `yaml:"env"`
}

// TerraformEnvVar is one runner-pod environment variable.
type TerraformEnvVar struct {
	Name      string                `yaml:"name"`
	ValueFrom TerraformEnvVarSource `yaml:"valueFrom"`
}

// TerraformEnvVarSource sources an env var's value from elsewhere in the
// cluster — here, always a Secret key, never an inline value.
type TerraformEnvVarSource struct {
	SecretKeyRef TerraformSecretKeyRef `yaml:"secretKeyRef"`
}

// TerraformSecretKeyRef names a Secret and a key within it.
type TerraformSecretKeyRef struct {
	Name string `yaml:"name"`
	Key  string `yaml:"key"`
}

// neonConfigTemplate renders the OpenTofu config for one managed Postgres
// component: a Neon project, its default branch's database, and a role
// owning it — the minimal resource set the week-two plan's managed
// emitter calls for. Executed with fixed, non-map data, so output is
// deterministic across compiles (golden rules 22, 45) the same way the
// Kubernetes-object emitters are via gopkg.in/yaml.v3's map-key sorting —
// here there is no map to sort in the first place.
var neonConfigTemplate = template.Must(template.New("neon.tf").Parse(`terraform {
  required_providers {
    neon = {
      source  = "{{.ProviderSource}}"
      version = "{{.ProviderVersion}}"
    }
  }
}

# The Neon API key is never a literal value here (golden rule 51): the
# kislerdm/neon provider reads it from the {{.APIKeyEnvVar}} environment
# variable, which the wrapping Terraform CR injects from the
# "{{.APIKeySecretName}}" Kubernetes Secret at runner-pod runtime.
provider "neon" {}

resource "neon_project" "{{.Name}}" {
  name = "{{.Name}}"
}

resource "neon_role" "{{.Name}}" {
  project_id = neon_project.{{.Name}}.id
  branch_id  = neon_project.{{.Name}}.default_branch_id
  name       = "{{.Name}}"
}

resource "neon_database" "{{.Name}}" {
  project_id = neon_project.{{.Name}}.id
  branch_id  = neon_project.{{.Name}}.default_branch_id
  name       = "{{.Name}}"
  owner_name = neon_role.{{.Name}}.name
}
`))

// neonConfigData is neonConfigTemplate's fixed input — a plain struct,
// never a map, so template execution has nothing to sort and needs
// nothing to guarantee determinism beyond Go's own deterministic struct
// field access.
type neonConfigData struct {
	Name             string
	ProviderSource   string
	ProviderVersion  string
	APIKeyEnvVar     string
	APIKeySecretName string
}

// neonTerraformConfig renders the .tf file content for pg.
func neonTerraformConfig(pg domain.Postgres) ([]byte, error) {
	var buf bytes.Buffer
	if err := neonConfigTemplate.Execute(&buf, neonConfigData{
		Name:             pg.Name,
		ProviderSource:   neonProviderSource,
		ProviderVersion:  neonProviderVersion,
		APIKeyEnvVar:     neonAPIKeyEnvVar,
		APIKeySecretName: neonAPIKeySecretName,
	}); err != nil {
		return nil, fmt.Errorf("flux emitter: render neon terraform config for %q: %w", pg.Name, err)
	}
	return buf.Bytes(), nil
}

// managedDir is the path (relative to the compile output root) holding
// one managed Postgres component's OpenTofu config — read directly by
// tofu-controller via the Terraform CR's own sourceRef+path, not by
// Flux's kustomize-controller: the .tf file has no YAML/JSON extension,
// so kustomize-controller's auto-generated kustomization.yaml (built from
// *.yaml/*.yml files found under the path) never touches it. It still
// lives under the app Kustomization's prune path for one-tree-per-stack
// hygiene, the same convention as every other per-component file.
func managedDir(stackName, component string) string {
	return fmt.Sprintf("apps/%s/%s-managed", stackName, component)
}

// emitManagedPostgres compiles the managed-placement artifact (week-two
// plan, slices 2+3): the OpenTofu config plus the Terraform CR that
// reconciles it. Everything this emitter cannot honor for managed
// placement (guarantees.mtls, allowedConsumers) refuses at domain
// validation before compilation ever reaches this function
// (internal/domain/postgres.go) — so pg here never carries either.
//
// Known gap, named rather than silently left implicit (golden rule 34;
// contract-review finding, 2026-07-26): pg.Credentials.Name
// (credentials.secretRef.name) is schema-required for every Postgres
// component, but this function does not consume it. The neon_role this
// config creates gets a Terraform-managed password (a computed, sensitive
// attribute in the provider's state) — no Kubernetes Secret carrying it
// is emitted here, so nothing currently lets an application consumer
// obtain the managed database's credentials the way CNPG's pre-existing
// bootstrap secret does for self-hosted (internal/domain/secret.go). The
// natural mechanism is tofu-controller's spec.writeOutputsToSecret,
// writing the role's password output into the Secret named by
// pg.Credentials.Name — deliberately not wired here (out of this slice's
// authorized scope). Home: week-two plan slice 5 (acceptance extension),
// where it must be wired or explicitly re-deferred, not left implicit.
func emitManagedPostgres(files map[string][]byte, stackName string, pg domain.Postgres) error {
	dir := managedDir(stackName, pg.Name)
	tf, err := neonTerraformConfig(pg)
	if err != nil {
		return err
	}
	files[dir+"/main.tf"] = tf

	terraform := Terraform{
		APIVersion: terraformAPIVersion,
		Kind:       "Terraform",
		Metadata: ObjectMeta{
			Name:      pg.Name,
			Namespace: stackName,
			Labels:    ownershipLabels(stackName, pg.Name),
		},
		Spec: TerraformSpec{
			Interval:    reconcileInterval,
			Path:        "./out/" + dir,
			ApprovePlan: "auto",
			SourceRef:   gitSourceRef,
			RunnerPodTemplate: TerraformRunnerPodTemplate{
				Spec: TerraformRunnerPodSpec{
					Env: []TerraformEnvVar{
						{
							Name: neonAPIKeyEnvVar,
							ValueFrom: TerraformEnvVarSource{
								SecretKeyRef: TerraformSecretKeyRef{
									Name: neonAPIKeySecretName,
									Key:  neonAPIKeySecretKey,
								},
							},
						},
					},
				},
			},
		},
	}
	return set(files, fmt.Sprintf("apps/%s/%s-terraform.yaml", stackName, pg.Name), terraform)
}
