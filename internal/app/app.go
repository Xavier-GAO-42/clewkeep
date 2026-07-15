package app

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/Xavier-GAO-42/clewkeep/internal/adapters"
	"github.com/Xavier-GAO-42/clewkeep/internal/core"
)

const maxSearchLine = 16 * 1024 * 1024

type App struct {
	UserHome  string
	StoreHome string
	Adapters  []adapters.Adapter
}

type Status struct {
	CatalogPath string         `json:"catalog_path"`
	GeneratedAt string         `json:"generated_at"`
	Threads     int            `json:"threads"`
	Providers   map[string]int `json:"providers"`
	Projects    int            `json:"projects"`
	Warnings    int            `json:"warnings"`
}

type DoctorCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

func New() (*App, error) {
	userHome, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	storeHome, err := core.StoreHome()
	if err != nil {
		return nil, err
	}
	return NewWith(userHome, storeHome, adapters.Builtins()), nil
}

func NewWith(userHome, storeHome string, providerAdapters []adapters.Adapter) *App {
	return &App{UserHome: userHome, StoreHome: storeHome, Adapters: providerAdapters}
}

func (a *App) Scan(ctx context.Context) (*core.Catalog, error) {
	return a.scan(ctx, false)
}

// ScanFull reparses every discovered native session instead of reusing
// scan-cache.json. It still replaces the catalog and scan cache normally.
func (a *App) ScanFull(ctx context.Context) (*core.Catalog, error) {
	return a.scan(ctx, true)
}

func (a *App) scan(ctx context.Context, full bool) (*core.Catalog, error) {
	scanStarted := time.Now().UTC()
	cacheEntries := map[string]core.ScanCacheEntry{}
	if !full {
		cacheEntries = loadReusableScanCache(a.StoreHome, scanStarted)
	}
	nextCacheEntries := make([]core.ScanCacheEntry, 0, len(cacheEntries))
	threads := make([]core.Thread, 0)
	warnings := make([]string, 0)
	seenPaths := map[string]bool{}
	for _, adapter := range a.Adapters {
		for _, root := range adapter.Roots(a.UserHome) {
			info, err := os.Stat(root)
			if err != nil || !info.IsDir() {
				continue
			}
			found := []core.Thread(nil)
			if incremental, ok := adapter.(adapters.IncrementalAdapter); ok {
				var entries []core.ScanCacheEntry
				found, entries, err = scanIncrementalRoot(ctx, incremental, root, cacheEntries, scanStarted)
				nextCacheEntries = append(nextCacheEntries, entries...)
			} else {
				found, err = adapter.Scan(ctx, root)
			}
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("%s root %s: %v", adapter.Name(), root, err))
				continue
			}
			for _, thread := range found {
				pathKey := filepath.Clean(thread.NativePath)
				if runtime.GOOS == "windows" {
					// Only fold case where the filesystem does; folding on
					// case-sensitive systems drops distinct native files.
					pathKey = strings.ToLower(pathKey)
				}
				if !seenPaths[pathKey] {
					seenPaths[pathKey] = true
					threads = append(threads, thread)
				}
			}
		}
	}
	sort.Slice(threads, func(i, j int) bool {
		if threads[i].Provider != threads[j].Provider {
			return threads[i].Provider < threads[j].Provider
		}
		if threads[i].ProjectRoot != threads[j].ProjectRoot {
			return threads[i].ProjectRoot < threads[j].ProjectRoot
		}
		if threads[i].ID != threads[j].ID {
			return threads[i].ID < threads[j].ID
		}
		return threads[i].NativePath < threads[j].NativePath
	})
	catalog := &core.Catalog{
		Format:        "CtxCatalog",
		SchemaVersion: core.CatalogSchemaVersion,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339Nano),
		Threads:       threads,
		Warnings:      warnings,
	}
	if err := core.WriteJSONAtomic(core.CatalogPath(a.StoreHome), catalog); err != nil {
		return nil, err
	}
	sortScanCacheEntries(nextCacheEntries)
	cache := &core.ScanCache{
		Format:        "CtxScanCache",
		SchemaVersion: core.ScanCacheSchemaVersion,
		GeneratedAt:   catalog.GeneratedAt,
		Entries:       nextCacheEntries,
	}
	if err := core.WriteJSONAtomic(core.ScanCachePath(a.StoreHome), cache); err != nil {
		return nil, err
	}
	return catalog, nil
}

func (a *App) Status() (*Status, error) {
	catalog, err := core.LoadCatalog(a.StoreHome)
	if err != nil {
		return nil, err
	}
	providers := map[string]int{}
	projects := map[string]bool{}
	for _, thread := range catalog.Threads {
		providers[thread.Provider]++
		projects[thread.Provider+"\x00"+thread.ProjectRoot] = true
	}
	return &Status{
		CatalogPath: core.CatalogPath(a.StoreHome),
		GeneratedAt: catalog.GeneratedAt,
		Threads:     len(catalog.Threads),
		Providers:   providers,
		Projects:    len(projects),
		Warnings:    len(catalog.Warnings),
	}, nil
}

func (a *App) List(provider, project string) ([]core.Thread, error) {
	catalog, err := core.LoadCatalog(a.StoreHome)
	if err != nil {
		return nil, err
	}
	provider = strings.ToLower(strings.TrimSpace(provider))
	project = strings.ToLower(strings.TrimSpace(project))
	result := make([]core.Thread, 0)
	for _, thread := range catalog.Threads {
		if provider != "" && strings.ToLower(thread.Provider) != provider && strings.ToLower(thread.Environment) != provider {
			continue
		}
		if project != "" && !strings.Contains(strings.ToLower(thread.ProjectRoot), project) {
			continue
		}
		result = append(result, thread)
	}
	return result, nil
}

func (a *App) Show(ref string) (*core.Thread, string, error) {
	catalog, err := core.LoadCatalog(a.StoreHome)
	if err != nil {
		return nil, "", err
	}
	names, err := core.LoadNames(a.StoreHome)
	if err != nil {
		return nil, "", err
	}
	return findThread(catalog.Threads, names.Names, ref)
}

func (a *App) Name(ref, name string) (*core.Thread, error) {
	thread, _, err := a.Show(ref)
	if err != nil {
		return nil, err
	}
	name = strings.TrimSpace(name)
	if err := validateName(name); err != nil {
		return nil, err
	}
	names, err := core.LoadNames(a.StoreHome)
	if err != nil {
		return nil, err
	}
	if current, exists := names.Names[name]; exists && current != thread.ID {
		return nil, fmt.Errorf("name %q already refers to another thread", name)
	}
	names.Names[name] = thread.ID
	if err := core.WriteJSONAtomic(core.NamesPath(a.StoreHome), names); err != nil {
		return nil, err
	}
	return thread, nil
}

func (a *App) Search(query string, limit int) ([]core.SearchHit, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}
	if limit <= 0 {
		limit = 20
	}
	catalog, err := core.LoadCatalog(a.StoreHome)
	if err != nil {
		return nil, err
	}
	names, err := core.LoadNames(a.StoreHome)
	if err != nil {
		return nil, err
	}
	aliases := reverseNames(names.Names)
	lowerQuery := strings.ToLower(query)
	hits := make([]core.SearchHit, 0, limit)
	for _, thread := range catalog.Threads {
		if len(hits) >= limit {
			break
		}
		metadata := strings.Join([]string{thread.ID, thread.Provider, thread.Environment, thread.ProjectRoot, aliases[thread.ID]}, " ")
		if strings.Contains(strings.ToLower(metadata), lowerQuery) {
			hits = append(hits, hitFor(thread, aliases[thread.ID], 0, clipAround(metadata, lowerQuery)))
			if len(hits) >= limit {
				break
			}
		}
		file, err := os.Open(thread.NativePath)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 64*1024), maxSearchLine)
		line := 0
		for scanner.Scan() && len(hits) < limit {
			line++
			text := searchableText(scanner.Bytes())
			lower := strings.ToLower(text)
			if strings.Contains(lower, lowerQuery) {
				hits = append(hits, hitFor(thread, aliases[thread.ID], line, clipAround(text, lowerQuery)))
			}
		}
		_ = file.Close()
	}
	return hits, nil
}

func (a *App) Snapshot(ctx context.Context, name string) (string, *core.Snapshot, error) {
	catalog, err := a.Scan(ctx)
	if err != nil {
		return "", nil, err
	}
	name = safeSnapshotName(name)
	now := time.Now().UTC()
	snapshot := &core.Snapshot{
		Format:        "CtxSnapshot",
		SchemaVersion: core.SnapshotSchemaVersion,
		Name:          name,
		CreatedAt:     now.Format(time.RFC3339Nano),
		Threads:       catalog.Threads,
	}
	base := now.Format("20060102T150405Z")
	if name != "" {
		base += "-" + name
	}
	path := filepath.Join(core.SnapshotsDir(a.StoreHome), base+".json")
	if err := core.WriteJSONAtomic(path, snapshot); err != nil {
		return "", nil, err
	}
	if err := core.WriteJSONAtomic(filepath.Join(core.SnapshotsDir(a.StoreHome), "latest.json"), snapshot); err != nil {
		return "", nil, err
	}
	return path, snapshot, nil
}

func (a *App) DiffSince(ctx context.Context, selector string) (*core.TemporalDiff, error) {
	path, err := resolveSnapshot(a.StoreHome, selector)
	if err != nil {
		return nil, err
	}
	var before core.Snapshot
	if err := core.ReadJSON(path, &before); err != nil {
		return nil, err
	}
	current, err := a.Scan(ctx)
	if err != nil {
		return nil, err
	}
	diff := compareSnapshots(before, current.Threads)
	diff.Before = path
	return diff, nil
}

func (a *App) Doctor() []DoctorCheck {
	checks := make([]DoctorCheck, 0)
	for _, adapter := range a.Adapters {
		found := 0
		for _, root := range adapter.Roots(a.UserHome) {
			if info, err := os.Stat(root); err == nil && info.IsDir() {
				found++
			}
		}
		status := "ok"
		if found == 0 {
			status = "not-found"
		}
		checks = append(checks, DoctorCheck{Name: "adapter:" + adapter.Name(), Status: status, Detail: strconv.Itoa(found) + " native root(s)"})
	}
	if _, err := core.LoadCatalog(a.StoreHome); err != nil {
		checks = append(checks, DoctorCheck{Name: "catalog", Status: "missing", Detail: err.Error()})
	} else {
		checks = append(checks, DoctorCheck{Name: "catalog", Status: "ok", Detail: core.CatalogPath(a.StoreHome)})
	}
	checks = append(checks, DoctorCheck{Name: "permission-model", Status: "ok", Detail: "provider roots are read-only; ctx metadata writes use CTX_HOME"})
	return checks
}

func findThread(threads []core.Thread, names map[string]string, ref string) (*core.Thread, string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, "", fmt.Errorf("thread reference cannot be empty")
	}
	target := ref
	alias := ""
	if id, ok := names[ref]; ok {
		target = id
		alias = ref
	}
	exact := make([]core.Thread, 0)
	prefix := make([]core.Thread, 0)
	for _, thread := range threads {
		if thread.ID == target {
			exact = append(exact, thread)
		} else if strings.HasPrefix(thread.ID, target) {
			prefix = append(prefix, thread)
		}
	}
	candidates := exact
	if len(candidates) == 0 {
		candidates = prefix
	}
	if len(candidates) == 0 {
		return nil, "", fmt.Errorf("thread not found: %s", ref)
	}
	if len(candidates) > 1 {
		return nil, "", fmt.Errorf("thread reference is ambiguous: %s", ref)
	}
	thread := candidates[0]
	return &thread, alias, nil
}

func validateName(name string) error {
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if utf8.RuneCountInString(name) > 128 {
		return fmt.Errorf("name cannot exceed 128 characters")
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return fmt.Errorf("name cannot contain control characters")
		}
	}
	return nil
}

func reverseNames(names map[string]string) map[string]string {
	reverse := map[string]string{}
	keys := make([]string, 0, len(names))
	for name := range names {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	for _, name := range keys {
		if _, exists := reverse[names[name]]; !exists {
			reverse[names[name]] = name
		}
	}
	return reverse
}

func hitFor(thread core.Thread, name string, line int, snippet string) core.SearchHit {
	return core.SearchHit{
		Name:        name,
		ThreadID:    thread.ID,
		Provider:    thread.Provider,
		Environment: thread.Environment,
		ProjectRoot: thread.ProjectRoot,
		NativePath:  thread.NativePath,
		Line:        line,
		Snippet:     snippet,
	}
}

func searchableText(raw []byte) string {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return string(raw)
	}
	parts := make([]string, 0)
	collectStrings(value, &parts)
	return strings.Join(parts, " ")
}

func collectStrings(value any, parts *[]string) {
	switch typed := value.(type) {
	case string:
		*parts = append(*parts, typed)
	case []any:
		for _, item := range typed {
			collectStrings(item, parts)
		}
	case map[string]any:
		for _, item := range typed {
			collectStrings(item, parts)
		}
	}
}

func clipAround(text, lowerQuery string) string {
	text = strings.Join(strings.Fields(text), " ")
	if len(text) <= 280 {
		return text
	}
	index := strings.Index(strings.ToLower(text), lowerQuery)
	if index < 0 {
		index = 0
	}
	start := index - 100
	if start < 0 {
		start = 0
	}
	end := start + 280
	if end > len(text) {
		end = len(text)
		start = end - 280
		if start < 0 {
			start = 0
		}
	}
	for start > 0 && !utf8.RuneStart(text[start]) {
		start--
	}
	for end < len(text) && !utf8.RuneStart(text[end]) {
		end++
	}
	prefix, suffix := "", ""
	if start > 0 {
		prefix = "…"
	}
	if end < len(text) {
		suffix = "…"
	}
	return prefix + text[start:end] + suffix
}

func safeSnapshotName(name string) string {
	name = strings.TrimSpace(name)
	var builder strings.Builder
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.' {
			builder.WriteRune(r)
		} else if builder.Len() > 0 {
			builder.WriteRune('-')
		}
		if builder.Len() >= 48 {
			break
		}
	}
	return strings.Trim(builder.String(), "-")
}

func resolveSnapshot(storeHome, selector string) (string, error) {
	dir := core.SnapshotsDir(storeHome)
	selector = strings.TrimSpace(selector)
	if selector == "" || selector == "latest" {
		path := filepath.Join(dir, "latest.json")
		if _, err := os.Stat(path); err != nil {
			return "", fmt.Errorf("no latest snapshot exists; run ctx snapshot")
		}
		return path, nil
	}
	if info, err := os.Stat(selector); err == nil && !info.IsDir() {
		return filepath.Abs(selector)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	matches := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "latest.json" {
			continue
		}
		if strings.HasPrefix(entry.Name(), selector) {
			matches = append(matches, filepath.Join(dir, entry.Name()))
		}
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("snapshot not found: %s", selector)
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("snapshot selector is ambiguous: %s", selector)
	}
	return matches[0], nil
}

func compareSnapshots(before core.Snapshot, after []core.Thread) *core.TemporalDiff {
	beforeMap := map[string]core.Thread{}
	afterMap := map[string]core.Thread{}
	for _, thread := range before.Threads {
		beforeMap[threadKey(thread)] = thread
	}
	for _, thread := range after {
		afterMap[threadKey(thread)] = thread
	}
	diff := &core.TemporalDiff{
		Format:        "CtxTemporalDiff",
		SchemaVersion: core.DiffSchemaVersion,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339Nano),
		Added:         []core.Thread{},
		Updated:       []core.ThreadChange{},
		Removed:       []core.Thread{},
	}
	keys := make([]string, 0, len(afterMap))
	for key := range afterMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		current := afterMap[key]
		old, exists := beforeMap[key]
		if !exists {
			diff.Added = append(diff.Added, current)
			continue
		}
		if threadChanged(old, current) {
			diff.Updated = append(diff.Updated, core.ThreadChange{Before: old, After: current})
		} else {
			diff.Unchanged++
		}
	}
	keys = keys[:0]
	for key := range beforeMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if _, exists := afterMap[key]; !exists {
			diff.Removed = append(diff.Removed, beforeMap[key])
		}
	}
	return diff
}

func threadKey(thread core.Thread) string {
	return thread.ID + "\x00" + filepath.Clean(thread.NativePath)
}

func threadChanged(before, after core.Thread) bool {
	return before.LineCount != after.LineCount ||
		before.UpdatedAt != after.UpdatedAt ||
		before.Provider != after.Provider ||
		before.Environment != after.Environment ||
		before.ProjectRoot != after.ProjectRoot ||
		before.NativePath != after.NativePath ||
		before.Model != after.Model ||
		before.HarnessVersion != after.HarnessVersion
}
