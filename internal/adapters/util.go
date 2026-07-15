package adapters

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Xavier-GAO-42/clewkeep/internal/core"
)

const maxJSONLLine = 16 * 1024 * 1024

func discoverJSONL(ctx context.Context, root string) ([]NativeFile, error) {
	files := make([]NativeFile, 0)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		// entry.Type() is Lstat-based: reject symlinks so discovery can
		// never follow a planted link out of the native root (SECURITY.md).
		if !entry.Type().IsRegular() || !strings.EqualFold(filepath.Ext(path), ".jsonl") {
			return nil
		}
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() {
			return nil
		}
		files = append(files, NativeFile{
			Path:            path,
			Size:            info.Size(),
			ModTimeUnixNano: info.ModTime().UnixNano(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

func scanDiscovered(ctx context.Context, root string, adapter IncrementalAdapter) ([]core.Thread, error) {
	files, err := adapter.Discover(ctx, root)
	if err != nil {
		return nil, err
	}
	threads := make([]core.Thread, 0, len(files))
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		thread, err := adapter.Parse(ctx, root, file)
		if err == nil && thread != nil {
			threads = append(threads, *thread)
		}
	}
	sort.Slice(threads, func(i, j int) bool {
		if threads[i].ID == threads[j].ID {
			return threads[i].NativePath < threads[j].NativePath
		}
		return threads[i].ID < threads[j].ID
	})
	return threads, nil
}

func scanJSONLines(path string, limit int, visit func(line int, raw []byte) bool) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), maxJSONLLine)
	count := 0
	for scanner.Scan() {
		count++
		if !visit(count, append([]byte(nil), scanner.Bytes()...)) {
			break
		}
		if limit > 0 && count >= limit {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return count, fmt.Errorf("scan %s: %w", path, err)
	}
	return count, nil
}

func lineCount(path string) (int, error) {
	return scanJSONLines(path, 0, func(_ int, _ []byte) bool { return true })
}

func decodeObject(raw []byte) (map[string]any, bool) {
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, false
	}
	return value, true
}

func stringValue(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}

func objectValue(value any) map[string]any {
	if object, ok := value.(map[string]any); ok {
		return object
	}
	return nil
}
