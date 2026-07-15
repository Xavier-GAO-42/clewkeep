package adapters

// Symlink escapes are an in-scope threat (SECURITY.md): a link planted in a
// native root must never pull outside content into the catalog or search.
// Symlink creation is unprivileged on Linux/macOS CI; on Windows it usually
// requires developer mode, so the test skips when links cannot be created.

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestScanIgnoresSymlinkedFiles(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()

	secret := filepath.Join(outside, "secret-notes.txt")
	if err := os.WriteFile(secret, []byte("{\"sessionId\":\"leaked\",\"cwd\":\"C:/synthetic/x\"}\nSYNTHETIC-SECRET\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	projectDir := filepath.Join(root, "demo")
	if err := os.MkdirAll(projectDir, 0o700); err != nil {
		t.Fatal(err)
	}
	real := filepath.Join(projectDir, "real.jsonl")
	if err := os.WriteFile(real, []byte("{\"sessionId\":\"real\",\"cwd\":\"C:/synthetic/x\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secret, filepath.Join(projectDir, "planted.jsonl")); err != nil {
		t.Skipf("cannot create symlinks on this platform: %v", err)
	}

	threads, err := (ClaudeCode{}).Scan(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(threads) != 1 || threads[0].ID != "codex/real" {
		ids := make([]string, 0, len(threads))
		for _, thread := range threads {
			ids = append(ids, thread.ID)
		}
		t.Fatalf("symlinked file escaped into the catalog: %v", ids)
	}
}

func TestScanIgnoresSymlinkedDirectories(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "outside.jsonl"), []byte("{\"sessionId\":\"dir-leak\",\"cwd\":\"C:/synthetic/x\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "linked-project")); err != nil {
		t.Skipf("cannot create symlinks on this platform: %v", err)
	}
	threads, err := (ClaudeCode{}).Scan(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(threads) != 0 {
		t.Fatalf("symlinked directory escaped into the catalog: %#v", threads)
	}
}
