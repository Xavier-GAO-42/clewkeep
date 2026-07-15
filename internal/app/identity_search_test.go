package app

import (
	"context"
	"crypto/sha256"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Xavier-GAO-42/clewkeep/internal/adapters"
	"github.com/Xavier-GAO-42/clewkeep/internal/core"
)

type fixtureFingerprint struct {
	hash    [32]byte
	size    int64
	mode    os.FileMode
	modTime time.Time
}

func writeClaudeFixture(t *testing.T, path, body string) fixtureFingerprint {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	data := []byte(body + "\n")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return fixtureFingerprint{sha256.Sum256(data), info.Size(), info.Mode(), info.ModTime()}
}

func assertFingerprint(t *testing.T, path string, before fixtureFingerprint) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if sha256.Sum256(data) != before.hash || info.Size() != before.size || info.Mode() != before.mode || !info.ModTime().Equal(before.modTime) {
		t.Fatalf("native fixture changed: %s", path)
	}
}

func fingerprintFile(t *testing.T, path string) fixtureFingerprint {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return fixtureFingerprint{sha256.Sum256(data), info.Size(), info.Mode(), info.ModTime()}
}

func TestClaudeMainAndSubagentsHaveStableAddressableIDs(t *testing.T) {
	userHome := t.TempDir()
	storeHome := t.TempDir()
	root := filepath.Join(userHome, ".claude", "projects", "demo")
	files := map[string]fixtureFingerprint{}
	mainPath := filepath.Join(root, "main.jsonl")
	files[mainPath] = writeClaudeFixture(t, mainPath, `{"sessionId":"shared:session%25","cwd":"C:/synthetic/demo","message":{"content":"identity-needle"}}`)
	for index, agentID := range []string{"agent-a", "agent:b", "agent%25c"} {
		path := filepath.Join(root, "sub-"+string(rune('a'+index))+".jsonl")
		body := `{"sessionId":"shared:session%25","agentId":"` + agentID + `","isSidechain":true,"cwd":"C:/synthetic/demo","message":{"content":"identity-needle"}}`
		files[path] = writeClaudeFixture(t, path, body)
	}
	a := NewWith(userHome, storeHome, adapters.Builtins())
	catalog, err := a.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"claude/shared:session%25",
		"claude/shared:session%25/agent/agent%25c",
		"claude/shared:session%25/agent/agent-a",
		"claude/shared:session%25/agent/agent:b",
	}
	got := make([]string, 0, len(catalog.Threads))
	for _, thread := range catalog.Threads {
		got = append(got, thread.ID)
		if shown, _, err := a.Show(thread.ID); err != nil || shown.ID != thread.ID {
			t.Fatalf("show(%q) = %#v, %v", thread.ID, shown, err)
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("canonical IDs = %v, want %v", got, want)
	}
	for _, scan := range []func(context.Context) (*core.Catalog, error){a.Scan, a.ScanFull} {
		repeated, err := scan(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		repeatedIDs := make([]string, 0, len(repeated.Threads))
		for _, thread := range repeated.Threads {
			repeatedIDs = append(repeatedIDs, thread.ID)
		}
		if !reflect.DeepEqual(repeatedIDs, want) {
			t.Fatalf("IDs changed across cache/full scan: %v", repeatedIDs)
		}
	}
	main, _, err := a.Show("shared:session%25")
	if err != nil || main.RecordKind != core.RecordKindSession || main.ID != want[0] {
		t.Fatalf("bare Claude session did not prefer main: %#v, %v", main, err)
	}
	hits, err := a.Search("identity-needle", 20)
	if err != nil || len(hits) != 4 {
		t.Fatalf("search hits = %#v, %v", hits, err)
	}
	for _, hit := range hits {
		if _, _, err := a.Show(hit.ThreadID); err != nil {
			t.Fatalf("search hit %q is not show-addressable: %v", hit.ThreadID, err)
		}
	}
	for path, fingerprint := range files {
		assertFingerprint(t, path, fingerprint)
	}
}

func TestBareClaudeSessionWithoutMainListsCandidates(t *testing.T) {
	userHome := t.TempDir()
	storeHome := t.TempDir()
	root := filepath.Join(userHome, ".claude", "projects", "demo")
	writeClaudeFixture(t, filepath.Join(root, "a.jsonl"), `{"sessionId":"only-subs","agentId":"a","isSidechain":true}`)
	writeClaudeFixture(t, filepath.Join(root, "b.jsonl"), `{"sessionId":"only-subs","agentId":"b","isSidechain":true}`)
	a := NewWith(userHome, storeHome, adapters.Builtins())
	if _, err := a.Scan(context.Background()); err != nil {
		t.Fatal(err)
	}
	_, _, err := a.Show("only-subs")
	if err == nil || !strings.Contains(err.Error(), "claude/only-subs/agent/a") || !strings.Contains(err.Error(), "claude/only-subs/agent/b") {
		t.Fatalf("got %v, want candidate-list ambiguity", err)
	}
}

func TestProviderQualificationAndCanonicalAliasPriority(t *testing.T) {
	userHome := t.TempDir()
	storeHome := t.TempDir()
	writeSyntheticCodexSession(t, filepath.Join(userHome, ".codex", "sessions", "same.jsonl"), "same", 1, time.Now().Add(-time.Minute))
	writeClaudeFixture(t, filepath.Join(userHome, ".claude", "projects", "demo", "same.jsonl"), `{"sessionId":"same","cwd":"C:/synthetic/claude"}`)
	a := NewWith(userHome, storeHome, adapters.Builtins())
	catalog, err := a.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Threads) != 2 || catalog.Threads[0].ID != "claude/same" || catalog.Threads[1].ID != "codex/same" {
		t.Fatalf("cross-provider IDs = %#v", catalog.Threads)
	}
	index := core.NameIndex{Format: "CtxNameIndex", SchemaVersion: core.NameIndexSchemaVersion, Names: map[string]string{"codex/same": "claude/same"}}
	if err := core.WriteJSONAtomic(core.NamesPath(storeHome), index); err != nil {
		t.Fatal(err)
	}
	thread, alias, err := a.Show("codex/same")
	if err != nil || alias != "" || thread.ID != "codex/same" {
		t.Fatalf("canonical exact was hijacked by alias: %#v, %q, %v", thread, alias, err)
	}
	if _, _, err := a.Show("same"); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("cross-provider bare ID must be ambiguous, got %v", err)
	}
}

func TestLegacyNameMigratesOnlyToDeterministicMain(t *testing.T) {
	userHome := t.TempDir()
	storeHome := t.TempDir()
	root := filepath.Join(userHome, ".claude", "projects", "demo")
	writeClaudeFixture(t, filepath.Join(root, "main.jsonl"), `{"sessionId":"legacy","cwd":"C:/synthetic/demo"}`)
	writeClaudeFixture(t, filepath.Join(root, "sub.jsonl"), `{"sessionId":"legacy","agentId":"sub","isSidechain":true,"cwd":"C:/synthetic/demo"}`)
	a := NewWith(userHome, storeHome, adapters.Builtins())
	if _, err := a.Scan(context.Background()); err != nil {
		t.Fatal(err)
	}
	legacy := core.NameIndex{Format: "CtxNameIndex", SchemaVersion: "0.1", Names: map[string]string{"old-name": "legacy"}}
	if err := core.WriteJSONAtomic(core.NamesPath(storeHome), legacy); err != nil {
		t.Fatal(err)
	}
	thread, alias, err := a.Show("old-name")
	if err != nil || alias != "old-name" || thread.ID != "claude/legacy" {
		t.Fatalf("legacy name resolution = %#v, %q, %v", thread, alias, err)
	}
	if _, err := a.Name("old-name", "new-name"); err != nil {
		t.Fatal(err)
	}
	var persisted core.NameIndex
	if err := core.ReadJSON(core.NamesPath(storeHome), &persisted); err != nil {
		t.Fatal(err)
	}
	if persisted.SchemaVersion != core.NameIndexSchemaVersion || persisted.Names["old-name"] != "claude/legacy" || persisted.Names["new-name"] != "claude/legacy" {
		t.Fatalf("persisted migrated names = %#v", persisted)
	}
}

func TestDuplicateCanonicalIDFailsWithoutReplacingCatalog(t *testing.T) {
	userHome := t.TempDir()
	storeHome := t.TempDir()
	root := filepath.Join(userHome, ".claude", "projects", "demo")
	writeClaudeFixture(t, filepath.Join(root, "one.jsonl"), `{"sessionId":"duplicate","agentId":"same-agent","isSidechain":true}`)
	writeClaudeFixture(t, filepath.Join(root, "two.jsonl"), `{"sessionId":"duplicate","agentId":"same-agent","isSidechain":true}`)
	marker := []byte("previous-catalog\n")
	if err := os.WriteFile(core.CatalogPath(storeHome), marker, 0o600); err != nil {
		t.Fatal(err)
	}
	a := NewWith(userHome, storeHome, adapters.Builtins())
	_, err := a.ScanFull(context.Background())
	if err == nil || !strings.Contains(err.Error(), "duplicate canonical id") {
		t.Fatalf("got %v, want duplicate canonical ID error", err)
	}
	after, readErr := os.ReadFile(core.CatalogPath(storeHome))
	if readErr != nil || !reflect.DeepEqual(after, marker) {
		t.Fatalf("catalog changed on collision: %q, %v", after, readErr)
	}
	if _, err := os.Stat(core.ScanCachePath(storeHome)); !os.IsNotExist(err) {
		t.Fatalf("cache was written on collision: %v", err)
	}
}

func TestInvalidClaudeIdentityIsSkippedWithVisibleWarnings(t *testing.T) {
	userHome := t.TempDir()
	storeHome := t.TempDir()
	root := filepath.Join(userHome, ".claude", "projects", "demo")
	files := map[string]fixtureFingerprint{}
	for name, body := range map[string]string{
		"missing-session.jsonl":     `{"cwd":"C:/synthetic"}`,
		"missing-agent.jsonl":       `{"sessionId":"s","isSidechain":true}`,
		"illegal-agent.jsonl":       `{"sessionId":"s2","agentId":"bad/agent","isSidechain":true}`,
		"changed-session.jsonl":     "{\"sessionId\":\"first\"}\n{\"sessionId\":\"second\"}",
		"malformed-sidechain.jsonl": `{"sessionId":"s3","isSidechain":"yes"}`,
	} {
		path := filepath.Join(root, name)
		files[path] = writeClaudeFixture(t, path, body)
	}
	a := NewWith(userHome, storeHome, adapters.Builtins())
	catalog, err := a.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Threads) != 0 || len(catalog.Warnings) != 5 {
		t.Fatalf("invalid records = %d, warnings = %#v", len(catalog.Threads), catalog.Warnings)
	}
	joined := strings.Join(catalog.Warnings, "\n")
	for _, needle := range []string{"file-name fallback is not allowed", "missing agentId", "path separator", "conflicting Claude sessionId", "invalid Claude isSidechain"} {
		if !strings.Contains(joined, needle) {
			t.Fatalf("warnings missing %q: %s", needle, joined)
		}
	}
	for path, fingerprint := range files {
		assertFingerprint(t, path, fingerprint)
	}
}

func TestLegacyNameRefusesCrossProviderAndSubagentOnlyAmbiguity(t *testing.T) {
	t.Run("cross provider", func(t *testing.T) {
		userHome := t.TempDir()
		storeHome := t.TempDir()
		writeSyntheticCodexSession(t, filepath.Join(userHome, ".codex", "sessions", "same.jsonl"), "same", 1, time.Now().Add(-time.Minute))
		writeClaudeFixture(t, filepath.Join(userHome, ".claude", "projects", "demo", "same.jsonl"), `{"sessionId":"same"}`)
		a := NewWith(userHome, storeHome, adapters.Builtins())
		if _, err := a.Scan(context.Background()); err != nil {
			t.Fatal(err)
		}
		legacy := core.NameIndex{Format: "CtxNameIndex", SchemaVersion: "0.1", Names: map[string]string{"legacy": "same"}}
		if err := core.WriteJSONAtomic(core.NamesPath(storeHome), legacy); err != nil {
			t.Fatal(err)
		}
		if _, _, err := a.Show("legacy"); err == nil || !strings.Contains(err.Error(), "cannot deterministically migrate") {
			t.Fatalf("cross-provider legacy name error = %v", err)
		}
	})

	t.Run("subagents only", func(t *testing.T) {
		userHome := t.TempDir()
		storeHome := t.TempDir()
		root := filepath.Join(userHome, ".claude", "projects", "demo")
		writeClaudeFixture(t, filepath.Join(root, "a.jsonl"), `{"sessionId":"same","agentId":"a","isSidechain":true}`)
		writeClaudeFixture(t, filepath.Join(root, "b.jsonl"), `{"sessionId":"same","agentId":"b","isSidechain":true}`)
		a := NewWith(userHome, storeHome, adapters.Builtins())
		if _, err := a.Scan(context.Background()); err != nil {
			t.Fatal(err)
		}
		legacy := core.NameIndex{Format: "CtxNameIndex", SchemaVersion: "0.1", Names: map[string]string{"legacy": "same"}}
		if err := core.WriteJSONAtomic(core.NamesPath(storeHome), legacy); err != nil {
			t.Fatal(err)
		}
		if _, _, err := a.Show("legacy"); err == nil || !strings.Contains(err.Error(), "cannot deterministically migrate") {
			t.Fatalf("subagent-only legacy name error = %v", err)
		}
	})
}

func TestNameCannotCreateDeadCanonicalAlias(t *testing.T) {
	userHome := t.TempDir()
	storeHome := t.TempDir()
	writeSyntheticCodexSession(t, filepath.Join(userHome, ".codex", "sessions", "one.jsonl"), "one", 1, time.Now().Add(-time.Minute))
	writeClaudeFixture(t, filepath.Join(userHome, ".claude", "projects", "demo", "two.jsonl"), `{"sessionId":"two"}`)
	a := NewWith(userHome, storeHome, adapters.Builtins())
	if _, err := a.Scan(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Name("claude/two", "codex/one"); err == nil || !strings.Contains(err.Error(), "conflicts with canonical") {
		t.Fatalf("canonical alias conflict error = %v", err)
	}
}

func TestInvalidCurrentSnapshotRejectedBeforeScanWrites(t *testing.T) {
	valid := core.Thread{ID: "codex/same", NativeSessionID: "same", RecordKind: core.RecordKindSession, Provider: "codex"}
	invalid := valid
	invalid.ID = "codex/different"
	for name, threads := range map[string][]core.Thread{
		"invalid identity":   {invalid},
		"duplicate identity": {valid, valid},
	} {
		t.Run(name, func(t *testing.T) {
			userHome := t.TempDir()
			storeHome := t.TempDir()
			writeSyntheticCodexSession(t, filepath.Join(userHome, ".codex", "sessions", "new.jsonl"), "new", 1, time.Now().Add(-time.Minute))
			marker := core.Catalog{Format: "CtxCatalog", SchemaVersion: core.CatalogSchemaVersion, GeneratedAt: "2026-01-01T00:00:00Z", Threads: []core.Thread{}}
			if err := core.WriteJSONAtomic(core.CatalogPath(storeHome), marker); err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(core.CatalogPath(storeHome))
			if err != nil {
				t.Fatal(err)
			}
			snapshot := core.Snapshot{Format: "CtxSnapshot", SchemaVersion: core.SnapshotSchemaVersion, CreatedAt: "2026-01-01T00:00:00Z", Threads: threads}
			if err := core.WriteJSONAtomic(filepath.Join(core.SnapshotsDir(storeHome), "latest.json"), snapshot); err != nil {
				t.Fatal(err)
			}
			a := NewWith(userHome, storeHome, adapters.Builtins())
			if _, err := a.DiffSince(context.Background(), "latest"); err == nil || !strings.Contains(err.Error(), "invalid identity data") {
				t.Fatalf("invalid snapshot error = %v", err)
			}
			after, err := os.ReadFile(core.CatalogPath(storeHome))
			if err != nil || !reflect.DeepEqual(before, after) {
				t.Fatalf("catalog changed before rejecting snapshot: %v", err)
			}
			if _, err := os.Stat(core.ScanCachePath(storeHome)); !os.IsNotExist(err) {
				t.Fatalf("cache written before rejecting snapshot: %v", err)
			}
		})
	}
}

func TestCompleteReadWorkflowPreservesNativeFileMetadata(t *testing.T) {
	userHome := t.TempDir()
	storeHome := t.TempDir()
	codexPath := filepath.Join(userHome, ".codex", "sessions", "codex.jsonl")
	writeSyntheticCodexSession(t, codexPath, "codex-native", 1, time.Now().Add(-time.Minute))
	claudeMain := filepath.Join(userHome, ".claude", "projects", "demo", "main.jsonl")
	claudeSub := filepath.Join(userHome, ".claude", "projects", "demo", "sub.jsonl")
	fingerprints := map[string]fixtureFingerprint{
		codexPath:  fingerprintFile(t, codexPath),
		claudeMain: writeClaudeFixture(t, claudeMain, `{"sessionId":"claude-native","cwd":"C:/synthetic/demo","message":{"content":"workflow-needle"}}`),
		claudeSub:  writeClaudeFixture(t, claudeSub, `{"sessionId":"claude-native","agentId":"sub","isSidechain":true,"cwd":"C:/synthetic/demo","message":{"content":"workflow-needle"}}`),
	}
	a := NewWith(userHome, storeHome, adapters.Builtins())
	if _, err := a.Scan(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := a.List("", "synthetic"); err != nil {
		t.Fatal(err)
	}
	hits, err := a.Search("workflow-needle", 20)
	if err != nil {
		t.Fatal(err)
	}
	for _, hit := range hits {
		if _, _, err := a.Show(hit.ThreadID); err != nil {
			t.Fatalf("show search hit %q: %v", hit.ThreadID, err)
		}
	}
	if _, err := a.Name("claude/claude-native", "workflow-main"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := a.Snapshot(context.Background(), "workflow"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.DiffSince(context.Background(), "latest"); err != nil {
		t.Fatal(err)
	}
	for path, fingerprint := range fingerprints {
		assertFingerprint(t, path, fingerprint)
	}
}

func TestSearchFiltersBeforeLimitAndEveryHitCanShow(t *testing.T) {
	userHome := t.TempDir()
	storeHome := t.TempDir()
	writeClaudeFixture(t, filepath.Join(userHome, ".claude", "projects", "first", "one.jsonl"), `{"sessionId":"first","cwd":"C:/synthetic/first","message":{"content":"filter-needle"}}`)
	writeSyntheticCodexSession(t, filepath.Join(userHome, ".codex", "sessions", "second.jsonl"), "second", 0, time.Now().Add(-time.Minute))
	codexPath := filepath.Join(userHome, ".codex", "sessions", "second.jsonl")
	file, err := os.OpenFile(codexPath, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = file.WriteString("{\"type\":\"response_item\",\"payload\":{\"text\":\"filter-needle\"}}\n")
	_ = file.Close()
	a := NewWith(userHome, storeHome, adapters.Builtins())
	if _, err := a.Scan(context.Background()); err != nil {
		t.Fatal(err)
	}
	hits, err := a.SearchWithOptions("filter-needle", SearchOptions{Provider: "CoDeX", Project: " SYNTHETIC ", Limit: 1})
	if err != nil || len(hits) != 1 || hits[0].ThreadID != "codex/second" {
		t.Fatalf("filtered hits = %#v, %v", hits, err)
	}
	if _, _, err := a.Show(hits[0].ThreadID); err != nil {
		t.Fatalf("filtered search hit cannot show: %v", err)
	}
	environmentHits, err := a.SearchWithOptions("filter-needle", SearchOptions{Provider: "CoDeX-ExEc", Limit: 10})
	if err != nil || len(environmentHits) != 1 || environmentHits[0].ThreadID != "codex/second" {
		t.Fatalf("environment provider filter = %#v, %v", environmentHits, err)
	}
	andHits, err := a.SearchWithOptions("filter-needle", SearchOptions{Provider: "codex", Project: "first", Limit: 10})
	if err != nil || len(andHits) != 0 {
		t.Fatalf("provider/project filters are not AND: %#v, %v", andHits, err)
	}
	legacy, err := a.Search("filter-needle", 10)
	if err != nil {
		t.Fatal(err)
	}
	unfiltered, err := a.SearchWithOptions("filter-needle", SearchOptions{Limit: 10})
	legacyOrder := make([]string, 0, len(legacy))
	newOrder := make([]string, 0, len(unfiltered))
	for _, hit := range legacy {
		legacyOrder = append(legacyOrder, hit.ThreadID)
	}
	for _, hit := range unfiltered {
		newOrder = append(newOrder, hit.ThreadID)
	}
	if err != nil || !reflect.DeepEqual(legacyOrder, newOrder) {
		t.Fatalf("no-filter ordering changed: legacy=%#v new=%#v err=%v", legacy, unfiltered, err)
	}
}

func TestOldCatalogAndSnapshotAreRejectedBeforeScanWrites(t *testing.T) {
	storeHome := t.TempDir()
	userHome := t.TempDir()
	oldCatalog := core.Catalog{Format: "CtxCatalog", SchemaVersion: "0.1", GeneratedAt: "2026-01-01T00:00:00Z", Threads: []core.Thread{}}
	if err := core.WriteJSONAtomic(core.CatalogPath(storeHome), oldCatalog); err != nil {
		t.Fatal(err)
	}
	if _, err := core.LoadCatalog(storeHome); err == nil || !strings.Contains(err.Error(), "run ctx scan") {
		t.Fatalf("old catalog error = %v", err)
	}
	oldSnapshot := core.Snapshot{Format: "CtxSnapshot", SchemaVersion: "0.1", CreatedAt: "2026-01-01T00:00:00Z", Threads: []core.Thread{}}
	latest := filepath.Join(core.SnapshotsDir(storeHome), "latest.json")
	if err := core.WriteJSONAtomic(latest, oldSnapshot); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(core.CatalogPath(storeHome))
	a := NewWith(userHome, storeHome, adapters.Builtins())
	if _, err := a.DiffSince(context.Background(), "latest"); err == nil || !strings.Contains(err.Error(), "incompatible") {
		t.Fatalf("old snapshot diff error = %v", err)
	}
	after, _ := os.ReadFile(core.CatalogPath(storeHome))
	if !reflect.DeepEqual(before, after) {
		t.Fatal("DiffSince rewrote catalog before rejecting old snapshot")
	}
	if _, err := os.Stat(core.ScanCachePath(storeHome)); !os.IsNotExist(err) {
		t.Fatalf("DiffSince wrote cache before rejecting old snapshot: %v", err)
	}
}

func TestOldScanCacheForcesCompleteReparse(t *testing.T) {
	userHome := t.TempDir()
	storeHome := t.TempDir()
	path := filepath.Join(userHome, ".codex", "sessions", "cache.jsonl")
	modTime := time.Now().Add(-time.Hour).UTC()
	writeSyntheticCodexSession(t, path, "cache-native", 1, modTime)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Dir(path)
	old := core.ScanCache{
		Format:        "CtxScanCache",
		SchemaVersion: "0.1",
		GeneratedAt:   time.Now().Add(-time.Minute).UTC().Format(time.RFC3339Nano),
		Entries: []core.ScanCacheEntry{{
			Adapter:         "codex",
			Root:            root,
			NativePath:      path,
			Size:            info.Size(),
			ModTimeUnixNano: info.ModTime().UnixNano(),
			Thread:          core.Thread{ID: "cache-native", Provider: "codex", NativePath: path, UpdatedAt: info.ModTime().UTC().Format(time.RFC3339Nano)},
		}},
	}
	if err := core.WriteJSONAtomic(core.ScanCachePath(storeHome), old); err != nil {
		t.Fatal(err)
	}
	counter := &countingIncrementalAdapter{inner: adapters.Codex{}}
	a := NewWith(userHome, storeHome, []adapters.Adapter{counter})
	catalog, err := a.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(counter.parsePaths) != 1 || len(catalog.Threads) != 1 || catalog.Threads[0].ID != "codex/cache-native" {
		t.Fatalf("old cache was reused: parses=%d threads=%#v", len(counter.parsePaths), catalog.Threads)
	}
	var current core.ScanCache
	if err := core.ReadJSON(core.ScanCachePath(storeHome), &current); err != nil {
		t.Fatal(err)
	}
	if current.SchemaVersion != core.ScanCacheSchemaVersion || current.Entries[0].Thread.ID != "codex/cache-native" {
		t.Fatalf("new cache = %#v", current)
	}
}

func TestDiffIdentitySurvivesNativePathMove(t *testing.T) {
	before := core.Thread{ID: "codex/stable", NativeSessionID: "stable", RecordKind: core.RecordKindSession, Provider: "codex", NativePath: "old.jsonl", UpdatedAt: "2026-01-01T00:00:00Z"}
	after := before
	after.NativePath = "new.jsonl"
	diff := compareSnapshots(core.Snapshot{Threads: []core.Thread{before}}, []core.Thread{after})
	if len(diff.Updated) != 1 || len(diff.Added) != 0 || len(diff.Removed) != 0 {
		t.Fatalf("path move diff = %#v", diff)
	}
}

func TestDiffWorkflowTreatsNativePathMoveAsUpdate(t *testing.T) {
	userHome := t.TempDir()
	storeHome := t.TempDir()
	root := filepath.Join(userHome, ".codex", "sessions")
	oldPath := filepath.Join(root, "old.jsonl")
	newPath := filepath.Join(root, "new.jsonl")
	writeSyntheticCodexSession(t, oldPath, "stable", 1, time.Now().Add(-time.Minute))
	a := NewWith(userHome, storeHome, adapters.Builtins())
	if _, _, err := a.Snapshot(context.Background(), "before-move"); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(oldPath, newPath); err != nil {
		t.Fatal(err)
	}
	diff, err := a.DiffSince(context.Background(), "latest")
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Updated) != 1 || len(diff.Added) != 0 || len(diff.Removed) != 0 || diff.Updated[0].After.ID != "codex/stable" {
		t.Fatalf("path move workflow diff = %#v", diff)
	}
}
