package app

// QA robustness tests for the incremental scan: forged or corrupt cache
// input, concurrent scans, files changing mid-scan, and the privacy
// guarantees of ctx-owned artifacts. All fixtures are synthetic.

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Xavier-GAO-42/clewkeep/internal/adapters"
	"github.com/Xavier-GAO-42/clewkeep/internal/core"
)

type hookedIncrementalAdapter struct {
	adapters.IncrementalAdapter
	beforeParse func(file adapters.NativeFile)
}

func (h *hookedIncrementalAdapter) Parse(ctx context.Context, root string, file adapters.NativeFile) (*core.Thread, error) {
	if h.beforeParse != nil {
		h.beforeParse(file)
	}
	return h.IncrementalAdapter.Parse(ctx, root, file)
}

func validCacheEntryFixture(root string) core.ScanCacheEntry {
	modTime := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	path := filepath.Join(root, "alpha.jsonl")
	return core.ScanCacheEntry{
		Adapter:         "codex",
		Root:            root,
		NativePath:      path,
		Size:            10,
		ModTimeUnixNano: modTime.UnixNano(),
		Thread: core.Thread{
			ID:         "alpha",
			Provider:   "codex",
			UpdatedAt:  modTime.Format(time.RFC3339Nano),
			NativePath: path,
			LineCount:  1,
		},
	}
}

func TestValidScanCacheEntryRejectsForgeries(t *testing.T) {
	root := filepath.Join(t.TempDir(), "sessions")
	generatedAt := time.Date(2026, 1, 2, 4, 0, 0, 0, time.UTC)
	cases := []struct {
		name   string
		mutate func(entry *core.ScanCacheEntry)
		want   bool
	}{
		{"valid entry", func(entry *core.ScanCacheEntry) {}, true},
		{"path escapes root", func(entry *core.ScanCacheEntry) {
			entry.NativePath = filepath.Join(root, "..", "outside.jsonl")
			entry.Thread.NativePath = entry.NativePath
		}, false},
		{"path is the root's parent", func(entry *core.ScanCacheEntry) {
			entry.NativePath = filepath.Dir(root)
			entry.Thread.NativePath = entry.NativePath
		}, false},
		{"thread path differs from entry path", func(entry *core.ScanCacheEntry) {
			entry.Thread.NativePath = filepath.Join(root, "other.jsonl")
		}, false},
		{"provider does not match adapter", func(entry *core.ScanCacheEntry) {
			entry.Thread.Provider = "claude-code"
		}, false},
		{"negative size", func(entry *core.ScanCacheEntry) { entry.Size = -1 }, false},
		{"zero mod time", func(entry *core.ScanCacheEntry) { entry.ModTimeUnixNano = 0 }, false},
		{"negative mod time", func(entry *core.ScanCacheEntry) { entry.ModTimeUnixNano = -5 }, false},
		{"mod time after cache generation", func(entry *core.ScanCacheEntry) {
			entry.ModTimeUnixNano = generatedAt.Add(time.Hour).UnixNano()
			entry.Thread.UpdatedAt = generatedAt.Add(time.Hour).Format(time.RFC3339Nano)
		}, false},
		{"updated_at disagrees with mod time", func(entry *core.ScanCacheEntry) {
			entry.Thread.UpdatedAt = time.Unix(0, entry.ModTimeUnixNano).Add(time.Second).Format(time.RFC3339Nano)
		}, false},
		{"unparseable updated_at", func(entry *core.ScanCacheEntry) { entry.Thread.UpdatedAt = "yesterday" }, false},
		{"empty thread id", func(entry *core.ScanCacheEntry) { entry.Thread.ID = "" }, false},
		{"negative line count", func(entry *core.ScanCacheEntry) { entry.Thread.LineCount = -1 }, false},
		{"blank adapter", func(entry *core.ScanCacheEntry) { entry.Adapter = "  " }, false},
		{"blank root", func(entry *core.ScanCacheEntry) { entry.Root = "" }, false},
	}
	for _, testCase := range cases {
		entry := validCacheEntryFixture(root)
		testCase.mutate(&entry)
		if got := validScanCacheEntry(entry, generatedAt); got != testCase.want {
			t.Errorf("%s: validScanCacheEntry = %v, want %v", testCase.name, got, testCase.want)
		}
	}
}

func TestLoadReusableScanCacheRejectsDuplicateAndForgedFiles(t *testing.T) {
	scanStarted := time.Date(2026, 1, 2, 5, 0, 0, 0, time.UTC)
	root := filepath.Join(t.TempDir(), "sessions")
	writeCache := func(t *testing.T, storeHome string, cache core.ScanCache) {
		t.Helper()
		if err := core.WriteJSONAtomic(core.ScanCachePath(storeHome), &cache); err != nil {
			t.Fatal(err)
		}
	}
	base := core.ScanCache{
		Format:        "CtxScanCache",
		SchemaVersion: core.ScanCacheSchemaVersion,
		GeneratedAt:   scanStarted.Add(-time.Hour).Format(time.RFC3339Nano),
	}

	t.Run("duplicate entries poison the whole cache", func(t *testing.T) {
		storeHome := t.TempDir()
		cache := base
		cache.Entries = []core.ScanCacheEntry{validCacheEntryFixture(root), validCacheEntryFixture(root)}
		writeCache(t, storeHome, cache)
		if got := loadReusableScanCache(storeHome, scanStarted); len(got) != 0 {
			t.Fatalf("duplicate entries were accepted: %d reusable", len(got))
		}
	})
	t.Run("one escaping entry poisons the whole cache", func(t *testing.T) {
		storeHome := t.TempDir()
		escape := validCacheEntryFixture(root)
		escape.NativePath = filepath.Join(root, "..", "..", "secrets", "x.jsonl")
		escape.Thread.NativePath = escape.NativePath
		cache := base
		cache.Entries = []core.ScanCacheEntry{validCacheEntryFixture(root), escape}
		writeCache(t, storeHome, cache)
		if got := loadReusableScanCache(storeHome, scanStarted); len(got) != 0 {
			t.Fatalf("escaping entry was accepted: %d reusable", len(got))
		}
	})
	t.Run("wrong format label is ignored", func(t *testing.T) {
		storeHome := t.TempDir()
		cache := base
		cache.Format = "NotCtxScanCache"
		cache.Entries = []core.ScanCacheEntry{validCacheEntryFixture(root)}
		writeCache(t, storeHome, cache)
		if got := loadReusableScanCache(storeHome, scanStarted); len(got) != 0 {
			t.Fatalf("wrong format accepted: %d reusable", len(got))
		}
	})
	t.Run("wrong schema version is ignored", func(t *testing.T) {
		storeHome := t.TempDir()
		cache := base
		cache.SchemaVersion = "0.2"
		cache.Entries = []core.ScanCacheEntry{validCacheEntryFixture(root)}
		writeCache(t, storeHome, cache)
		if got := loadReusableScanCache(storeHome, scanStarted); len(got) != 0 {
			t.Fatalf("wrong schema version accepted: %d reusable", len(got))
		}
	})
	t.Run("valid cache is reusable", func(t *testing.T) {
		storeHome := t.TempDir()
		cache := base
		cache.Entries = []core.ScanCacheEntry{validCacheEntryFixture(root)}
		writeCache(t, storeHome, cache)
		if got := loadReusableScanCache(storeHome, scanStarted); len(got) != 1 {
			t.Fatalf("valid cache not reusable: %d entries", len(got))
		}
	})
}

func TestConcurrentScansDoNotCorruptCatalogOrCache(t *testing.T) {
	userHome := t.TempDir()
	storeHome := t.TempDir()
	sessions := filepath.Join(userHome, ".codex", "sessions")
	baseTime := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	for index := 0; index < 3; index++ {
		writeSyntheticCodexSession(t, filepath.Join(sessions, fmt.Sprintf("s%d.jsonl", index)), fmt.Sprintf("synthetic-%d", index), 1, baseTime.Add(time.Duration(index)*time.Minute))
	}
	a := NewWith(userHome, storeHome, adapters.Builtins())
	ctx := context.Background()

	const workers = 4
	const scansPerWorker = 3
	var wg sync.WaitGroup
	errs := make(chan error, workers*scansPerWorker)
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for round := 0; round < scansPerWorker; round++ {
				if _, err := a.Scan(ctx); err != nil {
					errs <- err
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("concurrent scan error: %v", err)
	}

	var catalog core.Catalog
	if err := core.ReadJSON(core.CatalogPath(storeHome), &catalog); err != nil {
		t.Fatalf("catalog corrupted by concurrent scans: %v", err)
	}
	if catalog.Format != "CtxCatalog" || len(catalog.Threads) != 3 {
		t.Fatalf("catalog inconsistent after concurrent scans: %#v", catalog)
	}
	var cache core.ScanCache
	if err := core.ReadJSON(core.ScanCachePath(storeHome), &cache); err != nil {
		t.Fatalf("scan cache corrupted by concurrent scans: %v", err)
	}
	if cache.Format != "CtxScanCache" || len(cache.Entries) != 3 {
		t.Fatalf("scan cache inconsistent after concurrent scans: %#v", cache)
	}
	generatedAt, err := time.Parse(time.RFC3339Nano, cache.GeneratedAt)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range cache.Entries {
		if !validScanCacheEntry(entry, generatedAt) {
			t.Fatalf("concurrent scans produced an invalid cache entry: %#v", entry)
		}
	}
	finalCatalog, err := a.Scan(ctx)
	if err != nil {
		t.Fatal(err)
	}
	assertThreadIDs(t, finalCatalog, "synthetic-0", "synthetic-1", "synthetic-2")
}

func TestScanDoesNotCacheFileStillBeingWritten(t *testing.T) {
	userHome := t.TempDir()
	storeHome := t.TempDir()
	sessions := filepath.Join(userHome, ".codex", "sessions")
	baseTime := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	stablePath := filepath.Join(sessions, "stable.jsonl")
	busyPath := filepath.Join(sessions, "busy.jsonl")
	writeSyntheticCodexSession(t, stablePath, "synthetic-stable", 1, baseTime)
	writeSyntheticCodexSession(t, busyPath, "synthetic-busy", 1, baseTime.Add(time.Minute))

	// Simulate an agent appending to busy.jsonl during every parse attempt,
	// like a live session that keeps streaming while ctx scans.
	appending := true
	hooked := &hookedIncrementalAdapter{IncrementalAdapter: adapters.Codex{}}
	hooked.beforeParse = func(file adapters.NativeFile) {
		if !appending || filepath.Base(file.Path) != "busy.jsonl" {
			return
		}
		handle, err := os.OpenFile(busyPath, os.O_APPEND|os.O_WRONLY, 0)
		if err != nil {
			t.Errorf("append to busy fixture: %v", err)
			return
		}
		defer handle.Close()
		if _, err := handle.WriteString("{\"type\":\"response_item\",\"payload\":{\"text\":\"streamed\"}}\n"); err != nil {
			t.Errorf("append to busy fixture: %v", err)
		}
	}
	a := NewWith(userHome, storeHome, []adapters.Adapter{hooked})
	catalog, err := a.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	assertThreadIDs(t, catalog, "synthetic-busy", "synthetic-stable")

	var cache core.ScanCache
	if err := core.ReadJSON(core.ScanCachePath(storeHome), &cache); err != nil {
		t.Fatal(err)
	}
	if len(cache.Entries) != 1 || !strings.HasSuffix(cache.Entries[0].NativePath, "stable.jsonl") {
		t.Fatalf("an unstable mid-write file was cached: %#v", cache.Entries)
	}

	// Once the writer stops, the next scan parses busy.jsonl again and can
	// then cache it.
	appending = false
	if _, err := a.Scan(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := core.ReadJSON(core.ScanCachePath(storeHome), &cache); err != nil {
		t.Fatal(err)
	}
	if len(cache.Entries) != 2 {
		t.Fatalf("settled file was not cached on the next scan: %#v", cache.Entries)
	}
}

func TestScanToleratesFileDeletedDuringScan(t *testing.T) {
	userHome := t.TempDir()
	storeHome := t.TempDir()
	sessions := filepath.Join(userHome, ".codex", "sessions")
	baseTime := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	writeSyntheticCodexSession(t, filepath.Join(sessions, "keep.jsonl"), "synthetic-keep", 1, baseTime)
	doomedPath := filepath.Join(sessions, "doomed.jsonl")
	writeSyntheticCodexSession(t, doomedPath, "synthetic-doomed", 1, baseTime.Add(time.Minute))

	hooked := &hookedIncrementalAdapter{IncrementalAdapter: adapters.Codex{}}
	hooked.beforeParse = func(file adapters.NativeFile) {
		if filepath.Base(file.Path) == "doomed.jsonl" {
			_ = os.Remove(doomedPath)
		}
	}
	a := NewWith(userHome, storeHome, []adapters.Adapter{hooked})
	catalog, err := a.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	assertThreadIDs(t, catalog, "synthetic-keep")
	var cache core.ScanCache
	if err := core.ReadJSON(core.ScanCachePath(storeHome), &cache); err != nil {
		t.Fatal(err)
	}
	if len(cache.Entries) != 1 || !strings.HasSuffix(cache.Entries[0].NativePath, "keep.jsonl") {
		t.Fatalf("unexpected cache after mid-scan deletion: %#v", cache.Entries)
	}
}

// Known, accepted v0.1 limitation: a rewrite that keeps both the byte size
// and the modification time is invisible to the fingerprint, so the catalog
// keeps the stale metadata until the file changes size or time again. Full
// content hashing was rejected for v0.1 (it would re-read every transcript
// on every scan). This test documents the trade-off; if it ever fails, the
// fingerprint has changed and SECURITY.md must be updated.
func TestSameSizeSameMtimeRewriteKeepsStaleMetadataByDesign(t *testing.T) {
	userHome := t.TempDir()
	storeHome := t.TempDir()
	sessions := filepath.Join(userHome, ".codex", "sessions")
	baseTime := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	path := filepath.Join(sessions, "victim.jsonl")
	writeSyntheticCodexSession(t, path, "synthetic-old", 1, baseTime)

	counter := &countingIncrementalAdapter{inner: adapters.Codex{}}
	a := NewWith(userHome, storeHome, []adapters.Adapter{counter})
	if _, err := a.Scan(context.Background()); err != nil {
		t.Fatal(err)
	}

	// Same id length, same content length, same mtime.
	writeSyntheticCodexSession(t, path, "synthetic-new", 1, baseTime)
	before := len(counter.parsePaths)
	catalog, err := a.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if delta := len(counter.parsePaths) - before; delta != 0 {
		t.Fatalf("same-size same-mtime rewrite was reparsed (%d) — fingerprint semantics changed, update SECURITY.md", delta)
	}
	assertThreadIDs(t, catalog, "synthetic-old")
}

func TestCtxArtifactsContainOnlyMetadataAndStayInCtxHome(t *testing.T) {
	userHome := t.TempDir()
	storeHome := filepath.Join(t.TempDir(), "ctx-home")
	sessions := filepath.Join(userHome, ".codex", "sessions")
	if err := os.MkdirAll(sessions, 0o700); err != nil {
		t.Fatal(err)
	}
	const needle = "SYNTHETIC-SECRET-NEEDLE-4242"
	path := filepath.Join(sessions, "private.jsonl")
	content := "{\"type\":\"session_meta\",\"payload\":{\"id\":\"synthetic-private\",\"cwd\":\"C:/synthetic/project\",\"timestamp\":\"2026-01-01T00:00:00Z\",\"source\":\"exec\"}}\n" +
		"{\"type\":\"response_item\",\"payload\":{\"text\":\"" + needle + "\"}}\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	a := NewWith(userHome, storeHome, adapters.Builtins())
	if _, err := a.Scan(context.Background()); err != nil {
		t.Fatal(err)
	}

	for _, artifact := range []string{core.CatalogPath(storeHome), core.ScanCachePath(storeHome)} {
		raw, err := os.ReadFile(artifact)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(raw, []byte(needle)) {
			t.Fatalf("%s contains transcript content, not just metadata", artifact)
		}
	}

	entries, err := os.ReadDir(storeHome)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	if len(names) != 2 || names[0] != "catalog.json" || names[1] != "scan-cache.json" {
		t.Fatalf("unexpected files in CTX_HOME (leftover temp files?): %v", names)
	}

	if runtime.GOOS != "windows" {
		info, err := os.Stat(storeHome)
		if err != nil {
			t.Fatal(err)
		}
		if perm := info.Mode().Perm(); perm&0o077 != 0 {
			t.Fatalf("CTX_HOME is group/world accessible: %v", perm)
		}
		for _, name := range names {
			info, err := os.Stat(filepath.Join(storeHome, name))
			if err != nil {
				t.Fatal(err)
			}
			if perm := info.Mode().Perm(); perm&0o077 != 0 {
				t.Fatalf("%s is group/world accessible: %v", name, perm)
			}
		}
	}
}
