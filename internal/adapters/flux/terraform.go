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
	// Postgres component's Terraform CR reads its Neon API key (and,
	// since week-two plan Revision 4, its Neon project id) from (golden
	// rule 51 — never a literal value in the emitted config). Fixed for
	// v1 (owner decision, week-two plan Revision 1: "the API-key secret
	// stays in the design ... default name neon-api-key stands unless
	// the owner renames it") — not yet a declared schema field, a known
	// v1 gap (golden rule 7).
	neonAPIKeySecretName = "neon-api-key"
	// neonAPIKeySecretKey is the key inside that Secret's data the API
	// key value lives under. d7s never provisions or reads this secret
	// (rule 51) — this constant is the single documented naming
	// convention the harness/operator must follow to populate it.
	neonAPIKeySecretKey = "apiKey"
	// neonProjectIDSecretKey is the key inside that same Secret's data
	// the Neon project id lives under (week-two plan Revision 4: the
	// Neon project is an environment prerequisite, exactly like the
	// Kubernetes cluster itself — d7s never creates or destroys it, and
	// its id must never appear in compiled output, an environment
	// binding that would break determinism across environments, golden
	// rules 22/45). It reaches the OpenTofu config at runtime alongside
	// the API key, via the same Secret and the same trust path
	// (TerraformSpec.VarsFrom below), never a literal value.
	neonProjectIDSecretKey = "projectId"
	// neonProjectIDVar is the Terraform input variable name
	// neonConfigTemplate declares for the project id, and the alias
	// TerraformSpec.VarsFrom renames neonProjectIDSecretKey to (tofu-
	// controller's "key:alias" varsKeys syntax, verified against the
	// pinned v0.16.4 tag's controllers/tc000075_..._rename_... test,
	// 2026-07-26).
	neonProjectIDVar = "project_id"

	// neonAPIKeyEnvVar is the environment variable the kislerdm/neon
	// provider reads api_key from when the provider block itself leaves
	// the argument unset (verified against the provider's docs,
	// 2026-07-26) — the mechanism that lets the runner pod inject the
	// key without it ever appearing in the emitted HCL.
	neonAPIKeyEnvVar = "NEON_API_KEY"

	// tfRunnerServiceAccountName is the ServiceAccount name tofu-
	// controller's Terraform CR runs its runner pod under by default
	// (Spec.ServiceAccountName, verified against the pinned v0.16.4 tag's
	// controllers/tf_controller_runner.go, 2026-07-26). The runner pod
	// lives in the CR's OWN namespace, but tofu-controller's own release
	// RBAC only creates this ServiceAccount (and its ClusterRoleBinding)
	// in flux-system — found live, 2026-07-26: a managed stack's runner
	// pod fails outright ("serviceaccount \"tf-runner\" not found")
	// without one in the stack's own namespace too.
	tfRunnerServiceAccountName = "tf-runner"
	// tfRunnerClusterRoleName is the ClusterRole tofu-controller's own
	// release RBAC defines with exactly the permissions a runner pod
	// needs (verified against the pinned v0.16.4 tag's
	// tofu-controller.rbac.yaml, 2026-07-26) — an environment
	// prerequisite (scripts/actions/tofu-install.sh), never compiled
	// here. emitManagedRunnerRBAC only references it by name, scoping
	// its existing permissions to the stack's namespace via a
	// RoleBinding, rather than duplicating its rules.
	tfRunnerClusterRoleName = "tf-runner-role"
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
//
// WriteOutputsToSecret is deliberately always populated (never omitted)
// for a managed Postgres component: it targets the component's DECLARED
// credentials.secretRef.name, the mechanism that closes the credentials
// gap disclosed on emitManagedPostgres below (week-two plan slice 5).
// VarsFrom is how the Neon project id (an environment binding, never
// compiled — week-two plan Revision 4) and the API key both reach the
// OpenTofu config, from the same neon-api-key Secret the runner pod's
// own environment already trusts.
// DestroyResourcesOnDeletion is deliberately never set here (golden
// rule 28, retain-by-default): deleting this compiled CR must not
// destroy a data-bearing managed branch. Only the acceptance harness,
// as an explicit operator act at teardown, patches a running CR to
// enable it before deleting — never the compiler.
type TerraformSpec struct {
	Interval             string                         `yaml:"interval"`
	Path                 string                         `yaml:"path"`
	ApprovePlan          string                         `yaml:"approvePlan"`
	SourceRef            SourceRef                      `yaml:"sourceRef"`
	RunnerPodTemplate    TerraformRunnerPodTemplate     `yaml:"runnerPodTemplate"`
	VarsFrom             []TerraformVarsReference       `yaml:"varsFrom,omitempty"`
	WriteOutputsToSecret *TerraformWriteOutputsToSecret `yaml:"writeOutputsToSecret"`
}

// TerraformVarsReference is Terraform.spec.varsFrom's element type
// (tofu-controller's own CRD field, verified against the pinned v0.16.4
// tag's api/v1alpha2/reference_types.go, 2026-07-26): generates Terraform
// input variables from a Secret or ConfigMap's data. VarsKeys entries use
// tofu-controller's "sourceKey:variableName" syntax to rename a Secret
// data key to the Terraform variable name the config actually declares —
// here, the neon-api-key Secret's "projectId" key becomes the
// "project_id" variable neonConfigTemplate's OpenTofu config reads.
type TerraformVarsReference struct {
	Kind     string   `yaml:"kind"`
	Name     string   `yaml:"name"`
	VarsKeys []string `yaml:"varsKeys"`
}

// TerraformWriteOutputsToSecret is Terraform.spec.writeOutputsToSecret
// (tofu-controller's own CRD field, verified against the pinned v0.16.4
// tag's api/v1alpha2/terraform_types.go, 2026-07-26): the mechanism that
// lands every output the wrapped OpenTofu config declares (host, port,
// database, username, password — see neonConfigTemplate) into the named
// Kubernetes Secret's data, one key per output name, verbatim. Leaving
// Outputs unset writes all of them — deliberate, since neonConfigTemplate
// declares exactly the outputs a consumer needs to connect, nothing more.
type TerraformWriteOutputsToSecret struct {
	Name string `yaml:"name"`
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
// component: a branch inside the prerequisite Neon project (parented on
// the project's own default branch), a database on that branch, a role
// owning it, and a read-write compute endpoint so the branch is actually
// reachable — the resource set week-two plan Revision 4 calls for
// (branch-per-stack, superseding the earlier project-per-stack design:
// the owner's `.env` key is project-scoped and Neon forbids project-
// scoped keys from creating projects, confirmed live 2026-07-26). The
// project itself is an environment prerequisite, exactly like the
// Kubernetes cluster — d7s never creates or destroys it, and its id
// never appears here as a literal (see the project_id variable below).
// Executed with fixed, non-map data, so output is deterministic across
// compiles (golden rules 22, 45) the same way the Kubernetes-object
// emitters are via gopkg.in/yaml.v3's map-key sorting — here there is no
// map to sort in the first place.
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

# The Neon project is an environment prerequisite (week-two plan
# Revision 4) - d7s never provisions or destroys it, and its id is an
# environment binding that must never appear here as a literal (golden
# rules 22/45: baking it in would make compiled output depend on which
# environment compiled it). The wrapping Terraform CR's spec.varsFrom
# supplies this variable from the same Secret the API key comes from.
variable "{{.ProjectIDVar}}" {
  type = string
}

data "neon_project" "{{.Name}}" {
  id = var.{{.ProjectIDVar}}
}

resource "neon_branch" "{{.Name}}" {
  project_id = var.{{.ProjectIDVar}}
  parent_id  = data.neon_project.{{.Name}}.default_branch_id
  name       = "{{.Name}}"
}

resource "neon_endpoint" "{{.Name}}" {
  project_id = var.{{.ProjectIDVar}}
  branch_id  = neon_branch.{{.Name}}.id
  type       = "read_write"
}

# depends_on is required, not stylistic (found live, 2026-07-26): neon_role
# and neon_endpoint have no attribute reference between them, so without
# an explicit edge OpenTofu creates/destroys them in parallel - which
# reproducibly broke both directions against the real API. On create,
# neon_role's password generation raced neon_endpoint's own creation
# ("no read-write endpoint for branch"). On destroy, deleting the role
# concurrently with the endpoint reproducibly broke the provider's SDK
# ("unexpected end of JSON input" - the SDK decodes an empty response
# body it never checks for, a real upstream fragility, but only reachable
# because of the missing dependency edge). Confirmed by removing this
# line and reproducing both failures twice, then adding it back and
# succeeding twice - not a guess.
resource "neon_role" "{{.Name}}" {
  project_id = var.{{.ProjectIDVar}}
  branch_id  = neon_branch.{{.Name}}.id
  name       = "{{.Name}}"
  depends_on = [neon_endpoint.{{.Name}}]
}

resource "neon_database" "{{.Name}}" {
  project_id = var.{{.ProjectIDVar}}
  branch_id  = neon_branch.{{.Name}}.id
  name       = "{{.Name}}"
  owner_name = neon_role.{{.Name}}.name
}

# Connection outputs: tofu-controller's spec.writeOutputsToSecret (see
# TerraformWriteOutputsToSecret) writes each of these into the Kubernetes
# Secret named by the component's declared credentials.secretRef.name —
# the output name becomes the Secret's data key, verbatim, so these five
# names ARE the consumer-facing contract (mirrors CNPG's own "username"/
# "password" convention for the self-hosted placement, plus host/port/
# database for a placement where there is no in-cluster Service to
# default those from).
output "host" {
  value = neon_endpoint.{{.Name}}.host
}

output "port" {
  # Neon's connection proxy always listens on 5432 - the provider
  # exposes no separate port attribute because there is only ever one.
  value = "5432"
}

output "database" {
  value = neon_database.{{.Name}}.name
}

output "username" {
  value = neon_role.{{.Name}}.name
}

output "password" {
  value     = neon_role.{{.Name}}.password
  sensitive = true
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
	ProjectIDVar     string
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
		ProjectIDVar:     neonProjectIDVar,
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
// plan, slices 2+3, resource set superseded by Revision 4's branch-per-
// stack redesign): the OpenTofu config plus the Terraform CR that
// reconciles it. Everything this emitter cannot honor for managed
// placement (guarantees.mtls, allowedConsumers) refuses at domain
// validation before compilation ever reaches this function
// (internal/domain/postgres.go) — so pg here never carries either.
//
// Gap closed (week-two plan slice 5, 2026-07-26): pg.Credentials.Name
// (credentials.secretRef.name) is schema-required for every Postgres
// component, and is now consumed here — WriteOutputsToSecret targets it
// directly, so the declared name is where connection credentials land on
// both placements, the same as CNPG's pre-existing bootstrap secret does
// for self-hosted (internal/domain/secret.go). The neon_role's
// Terraform-managed password (a computed, sensitive attribute in the
// provider's state), plus host/port/database, are written into that
// Secret's data by tofu-controller itself once the CR reconciles — see
// TerraformWriteOutputsToSecret and neonConfigTemplate's output blocks.
//
// Revision 4 (2026-07-26): the live proof found the owner's Neon API key
// is project-scoped, and Neon forbids project-scoped keys from creating
// projects — so the Neon project is now a declared environment
// prerequisite (like the Kubernetes cluster itself), and this function
// compiles a branch + database + role + endpoint inside it, never a
// project. VarsFrom is how the project id reaches the OpenTofu config
// without ever appearing in compiled output (golden rules 22, 45).
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
			VarsFrom: []TerraformVarsReference{
				{
					Kind: "Secret",
					Name: neonAPIKeySecretName,
					VarsKeys: []string{
						neonProjectIDSecretKey + ":" + neonProjectIDVar,
					},
				},
			},
			WriteOutputsToSecret: &TerraformWriteOutputsToSecret{
				Name: pg.Credentials.Name,
			},
		},
	}
	return set(files, fmt.Sprintf("apps/%s/%s-terraform.yaml", stackName, pg.Name), terraform)
}

// emitManagedRunnerRBAC compiles the tf-runner ServiceAccount and its
// RoleBinding every managed Postgres component's Terraform CR needs to
// actually reconcile — called once per stack (namespace-scoped, not
// per-component), the same way emitAppNamespace and emitCNPGOperator
// are. This is static, deterministic, environment-independent wiring: no
// credential material (golden rule 51 doesn't apply — it carries no
// secret), no per-tenant policy choice, just what makes the compiled
// Terraform CR functional, the same class as the namespace object itself
// — so it is compiled here rather than left as an undocumented per-stack
// operator step. The RoleBinding only references the pre-existing
// tf-runner-role ClusterRole by name (an environment prerequisite,
// scripts/actions/tofu-install.sh) — it does not duplicate that
// ClusterRole's rules, the same reference-not-duplicate relationship the
// Terraform CR itself has with the GitRepository source.
func emitManagedRunnerRBAC(files map[string][]byte, stackName string) error {
	sa := ServiceAccount{
		APIVersion: "v1",
		Kind:       "ServiceAccount",
		Metadata: ObjectMeta{
			Name:      tfRunnerServiceAccountName,
			Namespace: stackName,
			Labels:    ownershipLabels(stackName, ""),
		},
	}
	if err := set(files, fmt.Sprintf("apps/%s/tf-runner-serviceaccount.yaml", stackName), sa); err != nil {
		return err
	}

	rb := RoleBinding{
		APIVersion: "rbac.authorization.k8s.io/v1",
		Kind:       "RoleBinding",
		Metadata: ObjectMeta{
			Name:      tfRunnerServiceAccountName,
			Namespace: stackName,
			Labels:    ownershipLabels(stackName, ""),
		},
		RoleRef: RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     tfRunnerClusterRoleName,
		},
		Subjects: []RoleBindingSubject{
			{
				Kind:      "ServiceAccount",
				Name:      tfRunnerServiceAccountName,
				Namespace: stackName,
			},
		},
	}
	return set(files, fmt.Sprintf("apps/%s/tf-runner-rolebinding.yaml", stackName), rb)
}
