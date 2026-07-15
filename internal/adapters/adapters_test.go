package adapters

import (
	"context"
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"
)

func TestCodexScanPreservesNativeFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "session.jsonl")
	data := []byte("{\"type\":\"session_meta\",\"payload\":{\"id\":\"codex-1\",\"cwd\":\"D:/work/demo\",\"timestamp\":\"2026-07-13T01:00:00Z\",\"cli_version\":\"1.2.3\",\"source\":{\"subagent\":{\"thread_spawn\":{\"parent_thread_id\":\"parent-1\"}}}}}\n{\"type\":\"response_item\",\"payload\":{\"text\":\"needle\"}}\n")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	before := sha256.Sum256(data)

	threads, err := (Codex{}).Scan(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(threads) != 1 {
		t.Fatalf("got %d threads, want 1", len(threads))
	}
	thread := threads[0]
	if thread.ID != "codex/codex-1" || thread.NativeSessionID != "codex-1" || thread.RecordKind != "session" || thread.Environment != "codex-subagent" || thread.ProjectRoot != "D:/work/demo" {
		t.Fatalf("unexpected thread: %#v", thread)
	}
	if len(thread.NativeRelations) != 1 || thread.NativeRelations[0].ParentThreadID != "parent-1" {
		t.Fatalf("unexpected relations: %#v", thread.NativeRelations)
	}
	afterData, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if after := sha256.Sum256(afterData); after != before {
		t.Fatal("scan modified the native Codex transcript")
	}
}

func TestClaudeCodeScan(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "project-a")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "claude-1.jsonl")
	data := []byte("{\"sessionId\":\"claude-1\",\"cwd\":\"D:/work/claude-demo\",\"timestamp\":\"2026-07-13T02:00:00Z\",\"message\":{\"content\":\"hello\"}}\n")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	threads, err := (ClaudeCode{}).Scan(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(threads) != 1 {
		t.Fatalf("got %d threads, want 1", len(threads))
	}
	thread := threads[0]
	if thread.ID != "claude/claude-1" || thread.NativeSessionID != "claude-1" || thread.RecordKind != "session" || thread.Provider != "claude-code" || thread.ProjectRoot != "D:/work/claude-demo" {
		t.Fatalf("unexpected thread: %#v", thread)
	}
}
