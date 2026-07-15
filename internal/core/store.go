package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func StoreHome() (string, error) {
	if value := strings.TrimSpace(os.Getenv("CTX_HOME")); value != "" {
		return filepath.Abs(value)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home: %w", err)
	}
	return filepath.Join(home, ".ctx"), nil
}

func CatalogPath(storeHome string) string {
	return filepath.Join(storeHome, "catalog.json")
}

func ScanCachePath(storeHome string) string {
	return filepath.Join(storeHome, "scan-cache.json")
}

func NamesPath(storeHome string) string {
	return filepath.Join(storeHome, "names.json")
}

func SnapshotsDir(storeHome string) string {
	return filepath.Join(storeHome, "snapshots")
}

func ReadJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}

func WriteJSONAtomic(path string, value any) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create ctx directory: %w", err)
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	data = append(data, '\n')
	temp, err := os.CreateTemp(dir, ".ctx-write-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	tempPath := temp.Name()
	ok := false
	defer func() {
		_ = temp.Close()
		if !ok {
			_ = os.Remove(tempPath)
		}
	}()
	if err := temp.Chmod(0o600); err != nil {
		return err
	}
	if _, err := temp.Write(data); err != nil {
		return fmt.Errorf("write temporary file: %w", err)
	}
	if err := temp.Sync(); err != nil {
		return fmt.Errorf("sync temporary file: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close temporary file: %w", err)
	}
	if err := replaceFile(tempPath, path); err != nil {
		return fmt.Errorf("replace %s: %w", path, err)
	}
	ok = true
	return nil
}

// replaceFile renames over the destination. On Windows a concurrent scan
// replacing the same catalog or cache file makes MoveFileEx fail with a
// transient access-denied error, so retry briefly; last writer wins and
// every version is complete because writes go through a temp file.
func replaceFile(tempPath, path string) error {
	var err error
	for attempt := 0; attempt < 10; attempt++ {
		if err = os.Rename(tempPath, path); err == nil {
			return nil
		}
		if runtime.GOOS != "windows" {
			return err
		}
		time.Sleep(time.Duration(attempt+1) * 5 * time.Millisecond)
	}
	return err
}

func LoadCatalog(storeHome string) (*Catalog, error) {
	var catalog Catalog
	if err := ReadJSON(CatalogPath(storeHome), &catalog); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("no catalog exists; run ctx scan")
		}
		return nil, err
	}
	if catalog.Format != "CtxCatalog" || catalog.SchemaVersion != CatalogSchemaVersion {
		return nil, fmt.Errorf("catalog schema %q/%q is incompatible with %q/%q; run ctx scan", catalog.Format, catalog.SchemaVersion, "CtxCatalog", CatalogSchemaVersion)
	}
	if err := ValidateThreadSet(catalog.Threads); err != nil {
		return nil, fmt.Errorf("catalog contains invalid identity data: %w; run ctx scan", err)
	}
	return &catalog, nil
}

func LoadNames(storeHome string) (*NameIndex, error) {
	index := &NameIndex{
		Format:        "CtxNameIndex",
		SchemaVersion: NameIndexSchemaVersion,
		Names:         map[string]string{},
	}
	if err := ReadJSON(NamesPath(storeHome), index); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return index, nil
		}
		return nil, err
	}
	if index.Names == nil {
		index.Names = map[string]string{}
	}
	if index.Format != "CtxNameIndex" {
		return nil, fmt.Errorf("name index format %q is incompatible with CtxNameIndex", index.Format)
	}
	if index.SchemaVersion != NameIndexSchemaVersion && index.SchemaVersion != "0.1" {
		return nil, fmt.Errorf("name index schema %q is incompatible with %q", index.SchemaVersion, NameIndexSchemaVersion)
	}
	if index.SchemaVersion == NameIndexSchemaVersion {
		for name, id := range index.Names {
			if err := ValidateCanonicalID(id); err != nil {
				return nil, fmt.Errorf("name %q has invalid canonical target: %w", name, err)
			}
		}
	}
	return index, nil
}
