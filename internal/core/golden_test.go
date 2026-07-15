package core

// Golden tests freezing the v0.1 machine interface: format labels, schema
// version fields, JSON field names, field order, and empty-array/omitempty
// behavior. Agents consume these documents; any diff here is a breaking
// interface change and must bump the schema version instead.

import (
	"encoding/json"
	"testing"
	"time"
)

func TestSchemaVersionsAreFrozenAtV01(t *testing.T) {
	if CatalogSchemaVersion != "0.1" {
		t.Fatalf("CatalogSchemaVersion = %q, want 0.1", CatalogSchemaVersion)
	}
	if ScanCacheSchemaVersion != "0.1" {
		t.Fatalf("ScanCacheSchemaVersion = %q, want 0.1", ScanCacheSchemaVersion)
	}
	if SnapshotSchemaVersion != "0.1" {
		t.Fatalf("SnapshotSchemaVersion = %q, want 0.1", SnapshotSchemaVersion)
	}
	if DiffSchemaVersion != "0.1" {
		t.Fatalf("DiffSchemaVersion = %q, want 0.1", DiffSchemaVersion)
	}
}

func mustGolden(t *testing.T, doc any, want string) {
	t.Helper()
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != want {
		t.Fatalf("machine interface changed.\ngot:\n%s\nwant:\n%s", data, want)
	}
}

func fullSyntheticThread() Thread {
	return Thread{
		ID:             "thread-0001",
		Provider:       "codex",
		Environment:    "codex-exec",
		ProjectRoot:    "C:/synthetic/project",
		Title:          "thread-0001",
		CreatedAt:      "2026-01-01T00:00:00Z",
		UpdatedAt:      "2026-01-02T03:04:05Z",
		NativePath:     "HOME/.codex/sessions/thread-0001.jsonl",
		NativeFormat:   "codex.rollout-jsonl",
		Source:         "exec",
		Originator:     "codex",
		Model:          "gpt-5.2",
		HarnessVersion: "1.2.3",
		LineCount:      2,
		NativeRelations: []Relation{{
			Kind:           "spawn",
			ParentThreadID: "thread-0000",
			Confidence:     "exact",
			EvidenceFields: []string{"session_meta.payload.parent_thread_id"},
		}},
		RelationWarnings: []string{"synthetic warning"},
	}
}

func minimalSyntheticThread() Thread {
	return Thread{
		ID:           "thread-0002",
		Provider:     "claude-code",
		Environment:  "claude-code",
		ProjectRoot:  "demo",
		UpdatedAt:    "2026-01-02T03:04:05Z",
		NativePath:   "HOME/.claude/projects/demo/thread-0002.jsonl",
		NativeFormat: "claude-code.transcript-jsonl",
	}
}

const minimalThreadGolden = `{
      "id": "thread-0002",
      "provider": "claude-code",
      "environment": "claude-code",
      "project_root": "demo",
      "updated_at": "2026-01-02T03:04:05Z",
      "native_path": "HOME/.claude/projects/demo/thread-0002.jsonl",
      "native_format": "claude-code.transcript-jsonl",
      "line_count": 0
    }`

func TestCatalogGolden(t *testing.T) {
	catalog := Catalog{
		Format:        "CtxCatalog",
		SchemaVersion: CatalogSchemaVersion,
		GeneratedAt:   "2026-01-02T03:04:06.123456789Z",
		Threads:       []Thread{fullSyntheticThread(), minimalSyntheticThread()},
		Warnings:      []string{"codex root HOME/.codex/archived_sessions: synthetic"},
	}
	mustGolden(t, catalog, `{
  "format": "CtxCatalog",
  "schema_version": "0.1",
  "generated_at": "2026-01-02T03:04:06.123456789Z",
  "threads": [
    {
      "id": "thread-0001",
      "provider": "codex",
      "environment": "codex-exec",
      "project_root": "C:/synthetic/project",
      "title": "thread-0001",
      "created_at": "2026-01-01T00:00:00Z",
      "updated_at": "2026-01-02T03:04:05Z",
      "native_path": "HOME/.codex/sessions/thread-0001.jsonl",
      "native_format": "codex.rollout-jsonl",
      "source": "exec",
      "originator": "codex",
      "model": "gpt-5.2",
      "harness_version": "1.2.3",
      "line_count": 2,
      "native_relations": [
        {
          "kind": "spawn",
          "parent_thread_id": "thread-0000",
          "confidence": "exact",
          "evidence_fields": [
            "session_meta.payload.parent_thread_id"
          ]
        }
      ],
      "relation_warnings": [
        "synthetic warning"
      ]
    },
    `+minimalThreadGolden+`
  ],
  "warnings": [
    "codex root HOME/.codex/archived_sessions: synthetic"
  ]
}`)
}

// An empty catalog keeps "threads": [] for agents, while the omitempty
// "warnings" key disappears even when the slice is empty but non-nil.
func TestCatalogEmptyGolden(t *testing.T) {
	catalog := Catalog{
		Format:        "CtxCatalog",
		SchemaVersion: CatalogSchemaVersion,
		GeneratedAt:   "2026-01-02T03:04:06.123456789Z",
		Threads:       []Thread{},
		Warnings:      []string{},
	}
	mustGolden(t, catalog, `{
  "format": "CtxCatalog",
  "schema_version": "0.1",
  "generated_at": "2026-01-02T03:04:06.123456789Z",
  "threads": []
}`)
}

func TestScanCacheGolden(t *testing.T) {
	cache := ScanCache{
		Format:        "CtxScanCache",
		SchemaVersion: ScanCacheSchemaVersion,
		GeneratedAt:   "2026-01-02T03:04:06.123456789Z",
		Entries: []ScanCacheEntry{{
			Adapter:         "codex",
			Root:            "HOME/.codex/sessions",
			NativePath:      "HOME/.codex/sessions/thread-0001.jsonl",
			Size:            123,
			ModTimeUnixNano: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC).UnixNano(),
			Thread:          minimalSyntheticThread(),
		}},
	}
	mustGolden(t, cache, `{
  "format": "CtxScanCache",
  "schema_version": "0.1",
  "generated_at": "2026-01-02T03:04:06.123456789Z",
  "entries": [
    {
      "adapter": "codex",
      "root": "HOME/.codex/sessions",
      "native_path": "HOME/.codex/sessions/thread-0001.jsonl",
      "size": 123,
      "mod_time_unix_nano": 1767323045000000000,
      "thread": {
        "id": "thread-0002",
        "provider": "claude-code",
        "environment": "claude-code",
        "project_root": "demo",
        "updated_at": "2026-01-02T03:04:05Z",
        "native_path": "HOME/.claude/projects/demo/thread-0002.jsonl",
        "native_format": "claude-code.transcript-jsonl",
        "line_count": 0
      }
    }
  ]
}`)
	empty := ScanCache{
		Format:        "CtxScanCache",
		SchemaVersion: ScanCacheSchemaVersion,
		GeneratedAt:   "2026-01-02T03:04:06.123456789Z",
		Entries:       []ScanCacheEntry{},
	}
	mustGolden(t, empty, `{
  "format": "CtxScanCache",
  "schema_version": "0.1",
  "generated_at": "2026-01-02T03:04:06.123456789Z",
  "entries": []
}`)
}

func TestNameIndexGolden(t *testing.T) {
	index := NameIndex{
		Format:        "CtxNameIndex",
		SchemaVersion: "0.1",
		Names:         map[string]string{"demo-session": "thread-0001"},
	}
	mustGolden(t, index, `{
  "format": "CtxNameIndex",
  "schema_version": "0.1",
  "names": {
    "demo-session": "thread-0001"
  }
}`)
}

func TestSearchHitsGolden(t *testing.T) {
	hits := []SearchHit{{
		Name:        "demo-session",
		ThreadID:    "thread-0001",
		Provider:    "codex",
		Environment: "codex-exec",
		ProjectRoot: "C:/synthetic/project",
		NativePath:  "HOME/.codex/sessions/thread-0001.jsonl",
		Line:        2,
		Snippet:     "synthetic needle",
	}}
	mustGolden(t, hits, `[
  {
    "name": "demo-session",
    "thread_id": "thread-0001",
    "provider": "codex",
    "environment": "codex-exec",
    "project_root": "C:/synthetic/project",
    "native_path": "HOME/.codex/sessions/thread-0001.jsonl",
    "line": 2,
    "snippet": "synthetic needle"
  }
]`)
	mustGolden(t, []SearchHit{}, `[]`)
}

func TestSnapshotGolden(t *testing.T) {
	snapshot := Snapshot{
		Format:        "CtxSnapshot",
		SchemaVersion: SnapshotSchemaVersion,
		Name:          "baseline",
		CreatedAt:     "2026-01-02T03:04:06.123456789Z",
		Threads:       []Thread{minimalSyntheticThread()},
	}
	mustGolden(t, snapshot, `{
  "format": "CtxSnapshot",
  "schema_version": "0.1",
  "name": "baseline",
  "created_at": "2026-01-02T03:04:06.123456789Z",
  "threads": [
    `+minimalThreadGolden+`
  ]
}`)
}

// A diff with no additions or removals must keep "added": [] and
// "removed": [] so agents can index the arrays unconditionally.
func TestTemporalDiffGolden(t *testing.T) {
	diff := TemporalDiff{
		Format:        "CtxTemporalDiff",
		SchemaVersion: DiffSchemaVersion,
		GeneratedAt:   "2026-01-02T03:04:06.123456789Z",
		Before:        "HOME/.ctx/snapshots/latest.json",
		Added:         []Thread{},
		Updated:       []ThreadChange{{Before: minimalSyntheticThread(), After: minimalSyntheticThread()}},
		Removed:       []Thread{},
		Unchanged:     1,
	}
	mustGolden(t, diff, `{
  "format": "CtxTemporalDiff",
  "schema_version": "0.1",
  "generated_at": "2026-01-02T03:04:06.123456789Z",
  "before": "HOME/.ctx/snapshots/latest.json",
  "added": [],
  "updated": [
    {
      "before": {
        "id": "thread-0002",
        "provider": "claude-code",
        "environment": "claude-code",
        "project_root": "demo",
        "updated_at": "2026-01-02T03:04:05Z",
        "native_path": "HOME/.claude/projects/demo/thread-0002.jsonl",
        "native_format": "claude-code.transcript-jsonl",
        "line_count": 0
      },
      "after": {
        "id": "thread-0002",
        "provider": "claude-code",
        "environment": "claude-code",
        "project_root": "demo",
        "updated_at": "2026-01-02T03:04:05Z",
        "native_path": "HOME/.claude/projects/demo/thread-0002.jsonl",
        "native_format": "claude-code.transcript-jsonl",
        "line_count": 0
      }
    }
  ],
  "removed": [],
  "unchanged": 1
}`)
}
