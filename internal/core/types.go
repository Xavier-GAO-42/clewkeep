package core

const (
	CatalogSchemaVersion   = "0.1"
	ScanCacheSchemaVersion = "0.1"
	SnapshotSchemaVersion  = "0.1"
	DiffSchemaVersion      = "0.1"
)

type Relation struct {
	Kind           string   `json:"kind"`
	ParentThreadID string   `json:"parent_thread_id"`
	Confidence     string   `json:"confidence"`
	EvidenceFields []string `json:"evidence_fields"`
}

type Thread struct {
	ID               string     `json:"id"`
	Provider         string     `json:"provider"`
	Environment      string     `json:"environment"`
	ProjectRoot      string     `json:"project_root"`
	Title            string     `json:"title,omitempty"`
	CreatedAt        string     `json:"created_at,omitempty"`
	UpdatedAt        string     `json:"updated_at"`
	NativePath       string     `json:"native_path"`
	NativeFormat     string     `json:"native_format"`
	Source           string     `json:"source,omitempty"`
	Originator       string     `json:"originator,omitempty"`
	Model            string     `json:"model,omitempty"`
	HarnessVersion   string     `json:"harness_version,omitempty"`
	LineCount        int        `json:"line_count"`
	NativeRelations  []Relation `json:"native_relations,omitempty"`
	RelationWarnings []string   `json:"relation_warnings,omitempty"`
}

type Catalog struct {
	Format        string   `json:"format"`
	SchemaVersion string   `json:"schema_version"`
	GeneratedAt   string   `json:"generated_at"`
	Threads       []Thread `json:"threads"`
	Warnings      []string `json:"warnings,omitempty"`
}

type ScanCacheEntry struct {
	Adapter         string `json:"adapter"`
	Root            string `json:"root"`
	NativePath      string `json:"native_path"`
	Size            int64  `json:"size"`
	ModTimeUnixNano int64  `json:"mod_time_unix_nano"`
	Thread          Thread `json:"thread"`
}

type ScanCache struct {
	Format        string           `json:"format"`
	SchemaVersion string           `json:"schema_version"`
	GeneratedAt   string           `json:"generated_at"`
	Entries       []ScanCacheEntry `json:"entries"`
}

type NameIndex struct {
	Format        string            `json:"format"`
	SchemaVersion string            `json:"schema_version"`
	Names         map[string]string `json:"names"`
}

type SearchHit struct {
	Name        string `json:"name,omitempty"`
	ThreadID    string `json:"thread_id"`
	Provider    string `json:"provider"`
	Environment string `json:"environment"`
	ProjectRoot string `json:"project_root"`
	NativePath  string `json:"native_path"`
	Line        int    `json:"line"`
	Snippet     string `json:"snippet"`
}

type Snapshot struct {
	Format        string   `json:"format"`
	SchemaVersion string   `json:"schema_version"`
	Name          string   `json:"name,omitempty"`
	CreatedAt     string   `json:"created_at"`
	Threads       []Thread `json:"threads"`
}

type ThreadChange struct {
	Before Thread `json:"before"`
	After  Thread `json:"after"`
}

type TemporalDiff struct {
	Format        string         `json:"format"`
	SchemaVersion string         `json:"schema_version"`
	GeneratedAt   string         `json:"generated_at"`
	Before        string         `json:"before"`
	Added         []Thread       `json:"added"`
	Updated       []ThreadChange `json:"updated"`
	Removed       []Thread       `json:"removed"`
	Unchanged     int            `json:"unchanged"`
}
