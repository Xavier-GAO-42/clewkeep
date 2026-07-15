package app

import (
	"context"
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"

	"github.com/Xavier-GAO-42/clewkeep/internal/adapters"
)

func TestLocalWorkflow(t *testing.T) {
	userHome := t.TempDir()
	storeHome := t.TempDir()
	codexDir := filepath.Join(userHome, ".codex", "sessions")
	claudeDir := filepath.Join(userHome, ".claude", "projects", "demo")
	if err := os.MkdirAll(codexDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(claudeDir, 0o700); err != nil {
		t.Fatal(err)
	}
	codexPath := filepath.Join(codexDir, "codex-1.jsonl")
	codexData := []byte("{\"type\":\"session_meta\",\"payload\":{\"id\":\"codex-1\",\"cwd\":\"D:/work/demo\",\"timestamp\":\"2026-07-13T01:00:00Z\",\"source\":\"exec\"}}\n{\"type\":\"response_item\",\"payload\":{\"text\":\"rare-needle\"}}\n")
	if err := os.WriteFile(codexPath, codexData, 0o600); err != nil {
		t.Fatal(err)
	}
	claudePath := filepath.Join(claudeDir, "claude-1.jsonl")
	if err := os.WriteFile(claudePath, []byte("{\"sessionId\":\"claude-1\",\"cwd\":\"D:/work/other\",\"message\":{\"content\":\"hello\"}}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	before := sha256.Sum256(codexData)
	a := NewWith(userHome, storeHome, adapters.Builtins())
	ctx := context.Background()

	catalog, err := a.Scan(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Threads) != 2 {
		t.Fatalf("got %d threads, want 2", len(catalog.Threads))
	}
	status, err := a.Status()
	if err != nil || status.Threads != 2 {
		t.Fatalf("unexpected status: %#v, %v", status, err)
	}
	hits, err := a.Search("rare-needle", 10)
	if err != nil || len(hits) != 1 || hits[0].ThreadID != "codex/codex-1" {
		t.Fatalf("unexpected hits: %#v, %v", hits, err)
	}
	if _, err := a.Name("codex-1", "demo-session"); err != nil {
		t.Fatal(err)
	}
	thread, alias, err := a.Show("demo-session")
	if err != nil || alias != "demo-session" || thread.ID != "codex/codex-1" {
		t.Fatalf("unexpected show result: %#v, %q, %v", thread, alias, err)
	}
	if _, _, err := a.Snapshot(ctx, "baseline"); err != nil {
		t.Fatal(err)
	}
	afterData, err := os.ReadFile(codexPath)
	if err != nil {
		t.Fatal(err)
	}
	if after := sha256.Sum256(afterData); after != before {
		t.Fatal("ctx workflow modified the native transcript")
	}
	file, err := os.OpenFile(codexPath, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString("{\"type\":\"response_item\",\"payload\":{\"text\":\"new work\"}}\n"); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	diff, err := a.DiffSince(ctx, "latest")
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Updated) != 1 || diff.Updated[0].After.ID != "codex/codex-1" {
		t.Fatalf("unexpected diff: %#v", diff)
	}
}
