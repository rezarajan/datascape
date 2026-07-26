package flux

import (
	"fmt"

	"github.com/rezarajan/datascape/internal/domain"
)

const (
	// durabilityConditionalAnnotationKey and
	// durabilityConditionalAnnotationValue are the annotation every
	// object emitted for a conditionally-satisfied guarantee carries
	// (problem definition Amendment 2, B3): a durability guarantee that
	// crosses the trust boundary to a declared external compiles, but
	// only labeled CONDITIONAL — never silently. Present on at least the
	// Cluster and the ScheduledBackup this week; testable-absent (golden
	// rule 49) via the golden-file byte comparison plus a dedicated test
	// proving a fixture missing it fails that comparison
	// (internal/adapters/flux/flux_test.go).
	durabilityConditionalAnnotationKey   = "d7s.dev/guarantee-durability"
	durabilityConditionalAnnotationValue = "conditional-on-external"

	// objectStoreAccessKeyIDSecretKey and objectStoreSecretAccessKeySecretKey
	// are the fixed data-key convention every declared external object
	// store's credentials Secret must carry (mirrors neonAPIKeySecretKey's
	// role in terraform.go): CNPG's barmanObjectStore.s3Credentials wants
	// two separate secret keys, but rule 51 gives d7s only one declared
	// Secret name per external — both keys are read from that same
	// Secret, under these two fixed names, the one documented convention
	// the harness/operator must populate it with.
	objectStoreAccessKeyIDSecretKey     = "ACCESS_KEY_ID"
	objectStoreSecretAccessKeySecretKey = "ACCESS_SECRET_KEY"
	// objectStoreRegionSecretKey is the same convention extended to the
	// optional region field: CNPG's s3Credentials.region is itself a
	// secret-key reference (not a literal), so a declared region is
	// wired the same way — never inlined (golden rule 51) — under this
	// fixed key in the same credentials Secret. Only wired when the
	// external declares a non-empty region.
	objectStoreRegionSecretKey = "REGION"
)

// CNPGBackup is Cluster.spec.backup: present only when the RPO
// guarantee is declared for the component. BarmanObjectStore is present
// only alongside it (week-three plan, slices 1+2): the durability
// guarantee's destination, wired from the declared external the
// component's guarantees.rpo.backupTo names. A declared RPO always
// resolves to an external by the time compilation reaches here — Stack.
// Validate (internal/domain/stack.go) refuses before compilation ever
// reaches the emitter otherwise.
type CNPGBackup struct {
	RetentionPolicy   string                 `yaml:"retentionPolicy"`
	BarmanObjectStore *CNPGBarmanObjectStore `yaml:"barmanObjectStore,omitempty"`
}

// CNPGBarmanObjectStore is Cluster.spec.backup.barmanObjectStore (CNPG's
// own CRD field, verified against the project's documented MinIO/S3
// examples, 2026-07-26): the destination base backups are shipped to.
// DestinationPath is the S3 URI form ("s3://<bucket>/"); EndpointURL is
// the external's own endpoint, letting this same shape address any
// S3-compatible store (MinIO included), not only AWS S3.
type CNPGBarmanObjectStore struct {
	DestinationPath string                  `yaml:"destinationPath"`
	EndpointURL     string                  `yaml:"endpointURL"`
	S3Credentials   CNPGBarmanS3Credentials `yaml:"s3Credentials"`
}

// CNPGBarmanS3Credentials is barmanObjectStore.s3Credentials. Every field
// is a secretKeyRef-shaped reference, never a literal (golden rule 51) —
// CNPG's own CRD requires this even for region, which is otherwise a
// plain declared string in domain.ObjectStoreExternal.
type CNPGBarmanS3Credentials struct {
	AccessKeyID     CNPGSecretKeySelector  `yaml:"accessKeyId"`
	SecretAccessKey CNPGSecretKeySelector  `yaml:"secretAccessKey"`
	Region          *CNPGSecretKeySelector `yaml:"region,omitempty"`
}

// CNPGSecretKeySelector names a Secret and a key within it (CNPG's own
// selector shape — a flat {name, key}, not Kubernetes core's nested
// valueFrom.secretKeyRef convention).
type CNPGSecretKeySelector struct {
	Name string `yaml:"name"`
	Key  string `yaml:"key"`
}

// ScheduledBackup is a CloudNativePG postgresql.cnpg.io ScheduledBackup.
type ScheduledBackup struct {
	APIVersion string              `yaml:"apiVersion"`
	Kind       string              `yaml:"kind"`
	Metadata   ObjectMeta          `yaml:"metadata"`
	Spec       ScheduledBackupSpec `yaml:"spec"`
}

// ScheduledBackupSpec is the subset of ScheduledBackup.spec d7s emits.
// Schedule uses the "@every <duration>" cron shorthand so it derives
// directly and deterministically from the declared RPO, with no cron
// arithmetic to get wrong.
type ScheduledBackupSpec struct {
	Schedule  string             `yaml:"schedule"`
	Cluster   ScheduledBackupRef `yaml:"cluster"`
	Immediate bool               `yaml:"immediate"`
}

// ScheduledBackupRef names the Cluster a ScheduledBackup targets.
type ScheduledBackupRef struct {
	Name string `yaml:"name"`
}

// conditionalDurabilityAnnotation returns the fixed one-entry annotation
// map every object emitted for a conditionally-satisfied durability
// guarantee carries.
func conditionalDurabilityAnnotation() map[string]string {
	return map[string]string{durabilityConditionalAnnotationKey: durabilityConditionalAnnotationValue}
}

// barmanObjectStore builds Cluster.spec.backup.barmanObjectStore from a
// declared external. The caller (emitCluster) only invokes this once
// guarantees.rpo is declared and its backupTo has already resolved to
// ext — ext.ObjectStore is nil only if domain validation was bypassed
// (e.g. a test exercising this emitter directly against a hand-built,
// invalid Stack), which errors here rather than panicking or silently
// emitting a malformed Cluster.
func barmanObjectStore(ext domain.External) (*CNPGBarmanObjectStore, error) {
	if ext.ObjectStore == nil {
		return nil, fmt.Errorf(
			"flux emitter: external %q has no objectStore declared — this is a defect (domain validation should have caught it)",
			ext.Name)
	}
	os := ext.ObjectStore
	bos := &CNPGBarmanObjectStore{
		DestinationPath: fmt.Sprintf("s3://%s/", os.Bucket),
		EndpointURL:     os.Endpoint,
		S3Credentials: CNPGBarmanS3Credentials{
			AccessKeyID: CNPGSecretKeySelector{
				Name: os.Credentials.Name,
				Key:  objectStoreAccessKeyIDSecretKey,
			},
			SecretAccessKey: CNPGSecretKeySelector{
				Name: os.Credentials.Name,
				Key:  objectStoreSecretAccessKeySecretKey,
			},
		},
	}
	if os.Region != "" {
		bos.S3Credentials.Region = &CNPGSecretKeySelector{
			Name: os.Credentials.Name,
			Key:  objectStoreRegionSecretKey,
		}
	}
	return bos, nil
}
