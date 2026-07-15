package app

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/Xavier-GAO-42/clewkeep/internal/adapters"
	"github.com/Xavier-GAO-42/clewkeep/internal/core"
)

type countingIncrementalAdapter struct {
	inner      adapters.IncrementalAdapter
	parsePaths []string
}

func (a *countingIncrementalAdapter) Name() string {
	return a.inner.Name()
}

func (a *countingIncrementalAdapter) Roots(userHome string) []string {
	return a.inner.Roots(userHome)
}

func (a *countingIncrementalAdapter) Scan(ctx context.Context, root string) ([]core.Thread, error) {
	return a.inner.Scan(ctx, root)
}

func (a *countingIncrementalAdapter) Discover(ctx context.Context, root string) ([]adapters.NativeFile, error) {
	return a.inner.Discover(ctx, root)
}

func (a *countingIncrementalAdapter) Parse(ctx context.Context, root string, file adapters.NativeFile) (*core.Thread, error) {
	a.parsePaths = append(a.parsePaths, file.Path)
	return a.inner.Parse(ctx, root, file)
}

func TestIncrementalScanReusesMetadataAndTracksFileChanges(t *testing.T) {
	userHome := t.TempDir()
	storeHome := t.TempDir()
	sessions := filepath.Join(userHome, ".codex", "sessions")
	baseTime := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	alphaPath := filepath.Join(sessions, "alpha.jsonl")
	betaPath := filepath.Join(sessions, "beta.jsonl")
	alphaData := writeSyntheticCodexSession(t, alphaPath, "synthetic-alpha", 1, baseTime)
	writeSyntheticCodexSession(t, betaPath, "synthetic-beta", 1, baseTime.Add(time.Minute))
	alphaInfo := mustStat(t, alphaPath)
	alphaHash := sha256.Sum256(alphaData)

	counter := &countingIncrementalAdapter{inner: adapters.Codex{}}
	a := NewWith(userHome, storeHome, []adapters.Adapter{counter})
	ctx := context.Background()
	parseDeltas := make([]int, 0, 5)

	before := len(counter.parsePaths)
	catalog, err := a.Scan(ctx)
	if err != nil {
		t.Fatal(err)
	}
	parseDeltas = append(parseDeltas, len(counter.parsePaths)-before)
	assertThreadIDs(t, catalog, "synthetic-alpha", "synthetic-beta")
	var cache core.ScanCache
	if err := core.ReadJSON(core.ScanCachePath(storeHome), &cache); err != nil {
		t.Fatalf("first scan did not create a usable scan cache: %v", err)
	}
	if len(cache.Entries) != 2 {
		t.Fatalf("first scan cached %d files, want 2", len(cache.Entries))
	}

	before = len(counter.parsePaths)
	catalog, err = a.Scan(ctx)
	if err != nil {
		t.Fatal(err)
	}
	parseDeltas = append(parseDeltas, len(counter.parsePaths)-before)
	assertThreadIDs(t, catalog, "synthetic-alpha", "synthetic-beta")

	gammaPath := filepath.Join(sessions, "gamma.jsonl")
	writeSyntheticCodexSession(t, gammaPath, "synthetic-gamma", 1, baseTime.Add(2*time.Minute))
	before = len(counter.parsePaths)
	catalog, err = a.Scan(ctx)
	if err != nil {
		t.Fatal(err)
	}
	parseDeltas = append(parseDeltas, len(counter.parsePaths)-before)
	assertThreadIDs(t, catalog, "synthetic-alpha", "synthetic-beta", "synthetic-gamma")

	writeSyntheticCodexSession(t, betaPath, "synthetic-beta", 3, baseTime.Add(3*time.Minute))
	before = len(counter.parsePaths)
	catalog, err = a.Scan(ctx)
	if err != nil {
		t.Fatal(err)
	}
	parseDeltas = append(parseDeltas, len(counter.parsePaths)-before)
	assertThreadIDs(t, catalog, "synthetic-alpha", "synthetic-beta", "synthetic-gamma")
	if got := threadByID(t, catalog, "synthetic-beta").LineCount; got != 4 {
		t.Fatalf("updated thread has %d lines, want 4", got)
	}

	if err := os.Remove(gammaPath); err != nil {
		t.Fatal(err)
	}
	before = len(counter.parsePaths)
	catalog, err = a.Scan(ctx)
	if err != nil {
		t.Fatal(err)
	}
	parseDeltas = append(parseDeltas, len(counter.parsePaths)-before)
	assertThreadIDs(t, catalog, "synthetic-alpha", "synthetic-beta")

	wantDeltas := []int{2, 0, 1, 1, 0}
	if !reflect.DeepEqual(parseDeltas, wantDeltas) {
		t.Fatalf("parse work per scan = %v, want %v", parseDeltas, wantDeltas)
	}
	t.Logf("JSONL parse counts: first=%d unchanged=%d added=%d updated=%d deleted=%d", parseDeltas[0], parseDeltas[1], parseDeltas[2], parseDeltas[3], parseDeltas[4])

	afterData, err := os.ReadFile(alphaPath)
	if err != nil {
		t.Fatal(err)
	}
	if afterHash := sha256.Sum256(afterData); afterHash != alphaHash {
		t.Fatal("incremental scans modified an unchanged native transcript")
	}
	afterInfo := mustStat(t, alphaPath)
	if !afterInfo.ModTime().Equal(alphaInfo.ModTime()) || afterInfo.Mode() != alphaInfo.Mode() || afterInfo.Size() != alphaInfo.Size() {
		t.Fatalf("native transcript metadata changed: before=%v after=%v", alphaInfo, afterInfo)
	}

	files := regularFilesUnder(t, userHome)
	wantFiles := []string{filepath.Clean(alphaPath), filepath.Clean(betaPath)}
	if !reflect.DeepEqual(files, wantFiles) {
		t.Fatalf("writes escaped CTX_HOME; native tree files = %v, want %v", files, wantFiles)
	}
	if err := core.ReadJSON(core.ScanCachePath(storeHome), &cache); err != nil {
		t.Fatal(err)
	}
	if !sort.SliceIsSorted(cache.Entries, func(i, j int) bool {
		return cache.Entries[i].NativePath < cache.Entries[j].NativePath
	}) {
		t.Fatalf("scan cache entries are not deterministic: %#v", cache.Entries)
	}
}

func TestFullScanReparsesAllFilesAndRepairsSameFingerprintRewrite(t *testing.T) {
	userHome := t.TempDir()
	storeHome := t.TempDir()
	sessions := filepath.Join(userHome, ".codex", "sessions")
	baseTime := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	alphaPath := filepath.Join(sessions, "alpha.jsonl")
	betaPath := filepath.Join(sessions, "beta.jsonl")
	writeSyntheticCodexSession(t, alphaPath, "synthetic-alpha", 1, baseTime)
	writeSyntheticCodexSession(t, betaPath, "synthetic-beta", 1, baseTime.Add(time.Minute))

	counter := &countingIncrementalAdapter{inner: adapters.Codex{}}
	a := NewWith(userHome, storeHome, []adapters.Adapter{counter})
	ctx := context.Background()

	before := len(counter.parsePaths)
	first, err := a.Scan(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if delta := len(counter.parsePaths) - before; delta != 2 {
		t.Fatalf("first scan parsed %d files, want 2", delta)
	}
	assertThreadIDs(t, first, "synthetic-alpha", "synthetic-beta")

	before = len(counter.parsePaths)
	second, err := a.Scan(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if delta := len(counter.parsePaths) - before; delta != 0 {
		t.Fatalf("second incremental scan parsed %d files, want 0", delta)
	}
	assertThreadIDs(t, second, "synthetic-alpha", "synthetic-beta")

	alphaInfo := mustStat(t, alphaPath)
	rewritten := writeSyntheticCodexSession(t, alphaPath, "synthetic-omega", 1, alphaInfo.ModTime())
	rewrittenInfo := mustStat(t, alphaPath)
	if rewrittenInfo.Size() != alphaInfo.Size() || !rewrittenInfo.ModTime().Equal(alphaInfo.ModTime()) {
		t.Fatalf("test rewrite did not preserve fingerprint: before=%v after=%v", alphaInfo, rewrittenInfo)
	}

	before = len(counter.parsePaths)
	stale, err := a.Scan(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if delta := len(counter.parsePaths) - before; delta != 0 {
		t.Fatalf("same-fingerprint incremental scan parsed %d files, want 0", delta)
	}
	assertThreadIDs(t, stale, "synthetic-alpha", "synthetic-beta")

	nativeBefore := snapshotNativeFiles(t, alphaPath, betaPath)
	before = len(counter.parsePaths)
	repaired, err := a.ScanFull(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if delta := len(counter.parsePaths) - before; delta != 2 {
		t.Fatalf("full scan parsed %d files, want 2", delta)
	}
	assertThreadIDs(t, repaired, "synthetic-beta", "synthetic-omega")
	assertNativeFilesUnchanged(t, nativeBefore)

	alphaAfter, err := os.ReadFile(alphaPath)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(alphaAfter, rewritten) {
		t.Fatal("full scan modified rewritten native transcript content")
	}
	var cache core.ScanCache
	if err := core.ReadJSON(core.ScanCachePath(storeHome), &cache); err != nil {
		t.Fatal(err)
	}
	if got := cacheThreadByPath(t, cache, alphaPath).ID; got != "synthetic-omega" {
		t.Fatalf("full scan cache retained thread ID %q, want synthetic-omega", got)
	}
}

func TestIncrementalScanFallsBackForAbnormalTimestamp(t *testing.T) {
	userHome := t.TempDir()
	storeHome := t.TempDir()
	sessions := filepath.Join(userHome, ".codex", "sessions")
	writeSyntheticCodexSession(t, filepath.Join(sessions, "stable.jsonl"), "synthetic-stable", 1, time.Now().UTC().Add(-time.Hour))
	writeSyntheticCodexSession(t, filepath.Join(sessions, "future.jsonl"), "synthetic-future", 1, time.Now().UTC().Add(24*time.Hour))

	counter := &countingIncrementalAdapter{inner: adapters.Codex{}}
	a := NewWith(userHome, storeHome, []adapters.Adapter{counter})
	if _, err := a.Scan(context.Background()); err != nil {
		t.Fatal(err)
	}
	before := len(counter.parsePaths)
	catalog, err := a.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	assertThreadIDs(t, catalog, "synthetic-future", "synthetic-stable")
	if delta := len(counter.parsePaths) - before; delta != 1 {
		t.Fatalf("abnormal timestamp reparsed %d files, want only the unsafe file", delta)
	}
	if !strings.HasSuffix(counter.parsePaths[len(counter.parsePaths)-1], "future.jsonl") {
		t.Fatalf("reparsed %q, want the future-dated file", counter.parsePaths[len(counter.parsePaths)-1])
	}
}

func TestIncrementalScanFallsBackWhenCacheClockMovesBackward(t *testing.T) {
	userHome := t.TempDir()
	storeHome := t.TempDir()
	sessions := filepath.Join(userHome, ".codex", "sessions")
	baseTime := time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)
	writeSyntheticCodexSession(t, filepath.Join(sessions, "one.jsonl"), "synthetic-one", 1, baseTime)
	writeSyntheticCodexSession(t, filepath.Join(sessions, "two.jsonl"), "synthetic-two", 1, baseTime.Add(time.Minute))

	counter := &countingIncrementalAdapter{inner: adapters.Codex{}}
	a := NewWith(userHome, storeHome, []adapters.Adapter{counter})
	if _, err := a.Scan(context.Background()); err != nil {
		t.Fatal(err)
	}
	var cache core.ScanCache
	if err := core.ReadJSON(core.ScanCachePath(storeHome), &cache); err != nil {
		t.Fatal(err)
	}
	cache.GeneratedAt = time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339Nano)
	if err := core.WriteJSONAtomic(core.ScanCachePath(storeHome), &cache); err != nil {
		t.Fatal(err)
	}

	before := len(counter.parsePaths)
	catalog, err := a.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	assertThreadIDs(t, catalog, "synthetic-one", "synthetic-two")
	if delta := len(counter.parsePaths) - before; delta != 2 {
		t.Fatalf("future cache timestamp triggered %d parses, want full fallback of 2", delta)
	}
}

func TestIncrementalScanFallsBackForCorruptCache(t *testing.T) {
	userHome := t.TempDir()
	storeHome := t.TempDir()
	sessions := filepath.Join(userHome, ".codex", "sessions")
	baseTime := time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)
	writeSyntheticCodexSession(t, filepath.Join(sessions, "one.jsonl"), "synthetic-one", 1, baseTime)
	writeSyntheticCodexSession(t, filepath.Join(sessions, "two.jsonl"), "synthetic-two", 1, baseTime.Add(time.Minute))

	counter := &countingIncrementalAdapter{inner: adapters.Codex{}}
	a := NewWith(userHome, storeHome, []adapters.Adapter{counter})
	if _, err := a.Scan(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(core.ScanCachePath(storeHome), []byte("{not-json\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	before := len(counter.parsePaths)
	catalog, err := a.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	assertThreadIDs(t, catalog, "synthetic-one", "synthetic-two")
	if delta := len(counter.parsePaths) - before; delta != 2 {
		t.Fatalf("corrupt cache triggered %d parses, want full fallback of 2", delta)
	}
	var repaired core.ScanCache
	if err := core.ReadJSON(core.ScanCachePath(storeHome), &repaired); err != nil {
		t.Fatalf("scan did not replace corrupt cache: %v", err)
	}
	if len(repaired.Entries) != 2 {
		t.Fatalf("repaired cache has %d entries, want 2", len(repaired.Entries))
	}
}

func writeSyntheticCodexSession(t *testing.T, path, id string, responseLines int, modTime time.Time) []byte {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	var builder strings.Builder
	fmt.Fprintf(&builder, "{\"type\":\"session_meta\",\"payload\":{\"id\":%q,\"cwd\":\"C:/synthetic/project\",\"timestamp\":\"2026-01-01T00:00:00Z\",\"source\":\"exec\"}}\n", id)
	for index := 0; index < responseLines; index++ {
		fmt.Fprintf(&builder, "{\"type\":\"response_item\",\"payload\":{\"text\":\"synthetic line %d\"}}\n", index+1)
	}
	data := []byte(builder.String())
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatal(err)
	}
	return data
}

func mustStat(t *testing.T, path string) os.FileInfo {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info
}

type nativeFileSnapshot struct {
	path    string
	data    []byte
	size    int64
	modTime time.Time
	mode    os.FileMode
}

func snapshotNativeFiles(t *testing.T, paths ...string) []nativeFileSnapshot {
	t.Helper()
	snapshots := make([]nativeFileSnapshot, 0, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		info := mustStat(t, path)
		snapshots = append(snapshots, nativeFileSnapshot{
			path: path, data: data, size: info.Size(), modTime: info.ModTime(), mode: info.Mode(),
		})
	}
	return snapshots
}

func assertNativeFilesUnchanged(t *testing.T, snapshots []nativeFileSnapshot) {
	t.Helper()
	for _, before := range snapshots {
		data, err := os.ReadFile(before.path)
		if err != nil {
			t.Fatal(err)
		}
		info := mustStat(t, before.path)
		if !reflect.DeepEqual(data, before.data) || info.Size() != before.size ||
			!info.ModTime().Equal(before.modTime) || info.Mode() != before.mode {
			t.Fatalf("full scan changed native file %s", before.path)
		}
	}
}

func cacheThreadByPath(t *testing.T, cache core.ScanCache, path string) core.Thread {
	t.Helper()
	for _, entry := range cache.Entries {
		if filepath.Clean(entry.NativePath) == filepath.Clean(path) {
			return entry.Thread
		}
	}
	t.Fatalf("cache entry not found: %s", path)
	return core.Thread{}
}

func assertThreadIDs(t *testing.T, catalog *core.Catalog, want ...string) {
	t.Helper()
	got := make([]string, 0, len(catalog.Threads))
	for _, thread := range catalog.Threads {
		got = append(got, thread.ID)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("thread IDs = %v, want %v", got, want)
	}
}

func threadByID(t *testing.T, catalog *core.Catalog, id string) core.Thread {
	t.Helper()
	for _, thread := range catalog.Threads {
		if thread.ID == id {
			return thread
		}
	}
	t.Fatalf("thread not found: %s", id)
	return core.Thread{}
}

func regularFilesUnder(t *testing.T, root string) []string {
	t.Helper()
	files := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() {
			files = append(files, filepath.Clean(path))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(files)
	return files
}
