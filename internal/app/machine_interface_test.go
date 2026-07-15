package app

// End-to-end golden tests for the files agents actually read: catalog.json
// and scan-cache.json produced by a real scan over synthetic fixtures. These
// freeze field emission, thread ordering, cache ordering, and the RFC3339
// UTC generated_at format for v0.1. Temp-dir paths and generated_at are the
// only nondeterministic values, so they are replaced with placeholders
// before comparing bytes.

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Xavier-GAO-42/clewkeep/internal/adapters"
	"github.com/Xavier-GAO-42/clewkeep/internal/core"
)

var generatedAtPattern = regexp.MustCompile(`"generated_at": "[^"]+"`)

func normalizeMachineJSON(t *testing.T, raw []byte, userHome string) string {
	t.Helper()
	escapedHome, err := json.Marshal(userHome)
	if err != nil {
		t.Fatal(err)
	}
	home := strings.Trim(string(escapedHome), `"`)
	text := strings.ReplaceAll(string(raw), home, "HOME")
	text = strings.ReplaceAll(text, `\\`, "/")
	return generatedAtPattern.ReplaceAllString(text, `"generated_at": "GENERATED_AT"`)
}

func readGeneratedAt(t *testing.T, path string) string {
	t.Helper()
	var doc struct {
		GeneratedAt string `json:"generated_at"`
	}
	if err := core.ReadJSON(path, &doc); err != nil {
		t.Fatal(err)
	}
	return doc.GeneratedAt
}

func TestScanOutputsGolden(t *testing.T) {
	userHome := t.TempDir()
	storeHome := filepath.Join(userHome, ".ctx")
	t1 := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	t2 := time.Date(2026, 1, 2, 3, 4, 6, 0, time.UTC)
	t3 := time.Date(2026, 1, 2, 3, 4, 7, 0, time.UTC)

	sessions := filepath.Join(userHome, ".codex", "sessions")
	// zeta before alpha on disk name order proves catalog ordering comes
	// from provider/project/id sorting, not discovery order.
	writeSyntheticCodexSession(t, filepath.Join(sessions, "a-zeta.jsonl"), "zeta", 1, t1)
	writeSyntheticCodexSession(t, filepath.Join(sessions, "b-alpha.jsonl"), "alpha", 1, t2)

	claudeDir := filepath.Join(userHome, ".claude", "projects", "demo")
	if err := os.MkdirAll(claudeDir, 0o700); err != nil {
		t.Fatal(err)
	}
	claudePath := filepath.Join(claudeDir, "claude-1.jsonl")
	if err := os.WriteFile(claudePath, []byte("{\"sessionId\":\"claude-1\",\"cwd\":\"C:/synthetic/other\",\"timestamp\":\"2026-01-01T00:00:00Z\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(claudePath, t3, t3); err != nil {
		t.Fatal(err)
	}

	a := NewWith(userHome, storeHome, adapters.Builtins())
	if _, err := a.Scan(context.Background()); err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{core.CatalogPath(storeHome), core.ScanCachePath(storeHome)} {
		generatedAt := readGeneratedAt(t, path)
		parsed, err := time.Parse(time.RFC3339Nano, generatedAt)
		if err != nil {
			t.Fatalf("%s generated_at %q is not RFC3339Nano: %v", path, generatedAt, err)
		}
		if !strings.HasSuffix(generatedAt, "Z") || parsed.Location() != time.UTC {
			t.Fatalf("%s generated_at %q is not UTC", path, generatedAt)
		}
	}

	catalogRaw, err := os.ReadFile(core.CatalogPath(storeHome))
	if err != nil {
		t.Fatal(err)
	}
	gotCatalog := normalizeMachineJSON(t, catalogRaw, userHome)
	wantCatalog := `{
  "format": "CtxCatalog",
  "schema_version": "0.1",
  "generated_at": "GENERATED_AT",
  "threads": [
    {
      "id": "claude-1",
      "provider": "claude-code",
      "environment": "claude-code",
      "project_root": "C:/synthetic/other",
      "title": "claude-1",
      "created_at": "2026-01-01T00:00:00Z",
      "updated_at": "2026-01-02T03:04:07Z",
      "native_path": "HOME/.claude/projects/demo/claude-1.jsonl",
      "native_format": "claude-code.transcript-jsonl",
      "source": "jsonl",
      "originator": "claude-code",
      "line_count": 1
    },
    {
      "id": "alpha",
      "provider": "codex",
      "environment": "codex-exec",
      "project_root": "C:/synthetic/project",
      "title": "alpha",
      "created_at": "2026-01-01T00:00:00Z",
      "updated_at": "2026-01-02T03:04:06Z",
      "native_path": "HOME/.codex/sessions/b-alpha.jsonl",
      "native_format": "codex.rollout-jsonl",
      "source": "exec",
      "line_count": 2
    },
    {
      "id": "zeta",
      "provider": "codex",
      "environment": "codex-exec",
      "project_root": "C:/synthetic/project",
      "title": "zeta",
      "created_at": "2026-01-01T00:00:00Z",
      "updated_at": "2026-01-02T03:04:05Z",
      "native_path": "HOME/.codex/sessions/a-zeta.jsonl",
      "native_format": "codex.rollout-jsonl",
      "source": "exec",
      "line_count": 2
    }
  ]
}
`
	if gotCatalog != wantCatalog {
		t.Fatalf("catalog.json machine interface changed.\ngot:\n%s\nwant:\n%s", gotCatalog, wantCatalog)
	}

	cacheRaw, err := os.ReadFile(core.ScanCachePath(storeHome))
	if err != nil {
		t.Fatal(err)
	}
	gotCache := normalizeMachineJSON(t, cacheRaw, userHome)
	wantCache := `{
  "format": "CtxScanCache",
  "schema_version": "0.1",
  "generated_at": "GENERATED_AT",
  "entries": [
    {
      "adapter": "claude-code",
      "root": "HOME/.claude/projects",
      "native_path": "HOME/.claude/projects/demo/claude-1.jsonl",
      "size": 87,
      "mod_time_unix_nano": 1767323047000000000,
      "thread": {
        "id": "claude-1",
        "provider": "claude-code",
        "environment": "claude-code",
        "project_root": "C:/synthetic/other",
        "title": "claude-1",
        "created_at": "2026-01-01T00:00:00Z",
        "updated_at": "2026-01-02T03:04:07Z",
        "native_path": "HOME/.claude/projects/demo/claude-1.jsonl",
        "native_format": "claude-code.transcript-jsonl",
        "source": "jsonl",
        "originator": "claude-code",
        "line_count": 1
      }
    },
    {
      "adapter": "codex",
      "root": "HOME/.codex/sessions",
      "native_path": "HOME/.codex/sessions/a-zeta.jsonl",
      "size": 191,
      "mod_time_unix_nano": 1767323045000000000,
      "thread": {
        "id": "zeta",
        "provider": "codex",
        "environment": "codex-exec",
        "project_root": "C:/synthetic/project",
        "title": "zeta",
        "created_at": "2026-01-01T00:00:00Z",
        "updated_at": "2026-01-02T03:04:05Z",
        "native_path": "HOME/.codex/sessions/a-zeta.jsonl",
        "native_format": "codex.rollout-jsonl",
        "source": "exec",
        "line_count": 2
      }
    },
    {
      "adapter": "codex",
      "root": "HOME/.codex/sessions",
      "native_path": "HOME/.codex/sessions/b-alpha.jsonl",
      "size": 192,
      "mod_time_unix_nano": 1767323046000000000,
      "thread": {
        "id": "alpha",
        "provider": "codex",
        "environment": "codex-exec",
        "project_root": "C:/synthetic/project",
        "title": "alpha",
        "created_at": "2026-01-01T00:00:00Z",
        "updated_at": "2026-01-02T03:04:06Z",
        "native_path": "HOME/.codex/sessions/b-alpha.jsonl",
        "native_format": "codex.rollout-jsonl",
        "source": "exec",
        "line_count": 2
      }
    }
  ]
}
`
	if gotCache != wantCache {
		t.Fatalf("scan-cache.json machine interface changed.\ngot:\n%s\nwant:\n%s", gotCache, wantCache)
	}

	if _, err := a.ScanFull(context.Background()); err != nil {
		t.Fatal(err)
	}
	fullCatalogRaw, err := os.ReadFile(core.CatalogPath(storeHome))
	if err != nil {
		t.Fatal(err)
	}
	if gotFullCatalog := normalizeMachineJSON(t, fullCatalogRaw, userHome); gotFullCatalog != wantCatalog {
		t.Fatalf("full scan JSON contract changed.\ngot:\n%s\nwant:\n%s", gotFullCatalog, wantCatalog)
	}
}

// A machine with no native roots must still produce a valid, empty catalog
// with "threads": [] so agent consumers never see null.
func TestScanEmptyMachineGolden(t *testing.T) {
	userHome := t.TempDir()
	storeHome := filepath.Join(userHome, ".ctx")
	a := NewWith(userHome, storeHome, adapters.Builtins())
	if _, err := a.Scan(context.Background()); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(core.CatalogPath(storeHome))
	if err != nil {
		t.Fatal(err)
	}
	got := normalizeMachineJSON(t, raw, userHome)
	want := `{
  "format": "CtxCatalog",
  "schema_version": "0.1",
  "generated_at": "GENERATED_AT",
  "threads": []
}
`
	if got != want {
		t.Fatalf("empty catalog changed.\ngot:\n%s\nwant:\n%s", got, want)
	}
}

// Search and List return non-nil empty slices so their --json output is []
// rather than null.
func TestEmptyResultsMarshalAsArrays(t *testing.T) {
	userHome := t.TempDir()
	storeHome := filepath.Join(userHome, ".ctx")
	a := NewWith(userHome, storeHome, adapters.Builtins())
	if _, err := a.Scan(context.Background()); err != nil {
		t.Fatal(err)
	}
	hits, err := a.Search("no-such-needle-anywhere", 5)
	if err != nil {
		t.Fatal(err)
	}
	if data, _ := json.Marshal(hits); string(data) != "[]" {
		t.Fatalf("empty search marshals to %s, want []", data)
	}
	threads, err := a.List("", "")
	if err != nil {
		t.Fatal(err)
	}
	if data, _ := json.Marshal(threads); string(data) != "[]" {
		t.Fatalf("empty list marshals to %s, want []", data)
	}
}

// Distinct native files whose names differ only by case must both appear in
// the catalog on case-sensitive filesystems. Skipped where the filesystem
// folds case (the two writes land on one file).
func TestCaseDifferingFilesBothIndexed(t *testing.T) {
	userHome := t.TempDir()
	storeHome := t.TempDir()
	claudeDir := filepath.Join(userHome, ".claude", "projects", "demo")
	if err := os.MkdirAll(claudeDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claudeDir, "abc.jsonl"), []byte("{\"sessionId\":\"case-lower\",\"cwd\":\"C:/synthetic/x\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claudeDir, "ABC.jsonl"), []byte("{\"sessionId\":\"case-upper\",\"cwd\":\"C:/synthetic/x\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(claudeDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) < 2 {
		t.Skip("filesystem folds case; only one file exists on disk")
	}
	a := NewWith(userHome, storeHome, adapters.Builtins())
	catalog, err := a.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	assertThreadIDs(t, catalog, "case-lower", "case-upper")
}
