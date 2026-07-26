package flux

// CNPGBackup is Cluster.spec.backup: present only when the RPO
// guarantee is declared for the component.
//
// Known gap (golden rule 7): v1 has no object-storage component or
// external declaration to resolve a backup destination from, so no
// barmanObjectStore is emitted here — only the retention policy, fixed
// at defaultRetentionPolicy until a destination can be declared and
// wired. Whether CNPG's own admission webhook accepts a retentionPolicy
// with no configured destination is verified live in the acceptance
// harness, not assumed.
type CNPGBackup struct {
	RetentionPolicy string `yaml:"retentionPolicy"`
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
