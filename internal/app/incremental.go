package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Xavier-GAO-42/clewkeep/internal/adapters"
	"github.com/Xavier-GAO-42/clewkeep/internal/core"
)

func loadReusableScanCache(storeHome string, scanStarted time.Time) map[string]core.ScanCacheEntry {
	var cache core.ScanCache
	if err := core.ReadJSON(core.ScanCachePath(storeHome), &cache); err != nil {
		return map[string]core.ScanCacheEntry{}
	}
	if cache.Format != "CtxScanCache" || cache.SchemaVersion != core.ScanCacheSchemaVersion {
		return map[string]core.ScanCacheEntry{}
	}
	generatedAt, err := time.Parse(time.RFC3339Nano, cache.GeneratedAt)
	if err != nil || generatedAt.After(scanStarted) {
		return map[string]core.ScanCacheEntry{}
	}

	entries := make(map[string]core.ScanCacheEntry, len(cache.Entries))
	for _, entry := range cache.Entries {
		if !validScanCacheEntry(entry, generatedAt) {
			return map[string]core.ScanCacheEntry{}
		}
		key := scanCacheKey(entry.Adapter, entry.Root, entry.NativePath)
		if _, duplicate := entries[key]; duplicate {
			return map[string]core.ScanCacheEntry{}
		}
		entries[key] = entry
	}
	return entries
}

func validScanCacheEntry(entry core.ScanCacheEntry, generatedAt time.Time) bool {
	if strings.TrimSpace(entry.Adapter) == "" || strings.TrimSpace(entry.Root) == "" || strings.TrimSpace(entry.NativePath) == "" {
		return false
	}
	if entry.Size < 0 || entry.ModTimeUnixNano <= 0 {
		return false
	}
	if time.Unix(0, entry.ModTimeUnixNano).After(generatedAt) {
		return false
	}
	if entry.Thread.ID == "" || entry.Thread.Provider != entry.Adapter || entry.Thread.UpdatedAt == "" || entry.Thread.LineCount < 0 {
		return false
	}
	if err := core.ValidateThreadIdentity(entry.Thread); err != nil {
		return false
	}
	updatedAt, err := time.Parse(time.RFC3339Nano, entry.Thread.UpdatedAt)
	if err != nil || updatedAt.UnixNano() != entry.ModTimeUnixNano {
		return false
	}
	if filepath.Clean(entry.Thread.NativePath) != filepath.Clean(entry.NativePath) {
		return false
	}
	relative, err := filepath.Rel(filepath.Clean(entry.Root), filepath.Clean(entry.NativePath))
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func scanIncrementalRoot(
	ctx context.Context,
	adapter adapters.IncrementalAdapter,
	root string,
	cache map[string]core.ScanCacheEntry,
	scanStarted time.Time,
) ([]core.Thread, []core.ScanCacheEntry, []string, error) {
	files, err := adapter.Discover(ctx, root)
	if err != nil {
		return nil, nil, nil, err
	}
	threads := make([]core.Thread, 0, len(files))
	entries := make([]core.ScanCacheEntry, 0, len(files))
	warnings := make([]string, 0)
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return nil, nil, warnings, err
		}
		key := scanCacheKey(adapter.Name(), root, file.Path)
		if cached, ok := cache[key]; ok && reusableNativeFile(cached, file, scanStarted) {
			threads = append(threads, cached.Thread)
			entries = append(entries, cached)
			continue
		}

		thread, current, stable, err := parseStableNativeFile(ctx, adapter, root, file)
		if err != nil || thread == nil {
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("%s file %s: %v", adapter.Name(), file.Path, err))
			}
			continue
		}
		if err := core.ValidateThreadIdentity(*thread); err != nil {
			warnings = append(warnings, fmt.Sprintf("%s file %s: invalid identity: %v", adapter.Name(), file.Path, err))
			continue
		}
		threads = append(threads, *thread)
		if stable && cacheableNativeFile(current, scanStarted) {
			entries = append(entries, core.ScanCacheEntry{
				Adapter:         adapter.Name(),
				Root:            root,
				NativePath:      current.Path,
				Size:            current.Size,
				ModTimeUnixNano: current.ModTimeUnixNano,
				Thread:          *thread,
			})
		}
	}
	return threads, entries, warnings, nil
}

func reusableNativeFile(entry core.ScanCacheEntry, file adapters.NativeFile, scanStarted time.Time) bool {
	return cacheableNativeFile(file, scanStarted) &&
		entry.Size == file.Size &&
		entry.ModTimeUnixNano == file.ModTimeUnixNano &&
		filepath.Clean(entry.NativePath) == filepath.Clean(file.Path)
}

func cacheableNativeFile(file adapters.NativeFile, scanStarted time.Time) bool {
	return file.Size >= 0 &&
		file.ModTimeUnixNano > 0 &&
		!time.Unix(0, file.ModTimeUnixNano).After(scanStarted)
}

func parseStableNativeFile(
	ctx context.Context,
	adapter adapters.IncrementalAdapter,
	root string,
	file adapters.NativeFile,
) (*core.Thread, adapters.NativeFile, bool, error) {
	current := file
	var thread *core.Thread
	for attempt := 0; attempt < 2; attempt++ {
		parsed, err := adapter.Parse(ctx, root, current)
		if err != nil {
			return nil, current, false, err
		}
		thread = parsed
		after, err := statNativeFile(current.Path)
		if err != nil {
			return nil, current, false, err
		}
		if sameNativeFingerprint(current, after) {
			return thread, after, true, nil
		}
		current = after
	}
	return thread, current, false, nil
}

func statNativeFile(path string) (adapters.NativeFile, error) {
	info, err := os.Stat(path)
	if err != nil {
		return adapters.NativeFile{}, err
	}
	return adapters.NativeFile{
		Path:            path,
		Size:            info.Size(),
		ModTimeUnixNano: info.ModTime().UnixNano(),
	}, nil
}

func sameNativeFingerprint(left, right adapters.NativeFile) bool {
	return filepath.Clean(left.Path) == filepath.Clean(right.Path) &&
		left.Size == right.Size &&
		left.ModTimeUnixNano == right.ModTimeUnixNano
}

func scanCacheKey(adapter, root, nativePath string) string {
	return adapter + "\x00" + filepath.Clean(root) + "\x00" + filepath.Clean(nativePath)
}

func sortScanCacheEntries(entries []core.ScanCacheEntry) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Adapter != entries[j].Adapter {
			return entries[i].Adapter < entries[j].Adapter
		}
		if entries[i].Root != entries[j].Root {
			return entries[i].Root < entries[j].Root
		}
		return entries[i].NativePath < entries[j].NativePath
	})
}
