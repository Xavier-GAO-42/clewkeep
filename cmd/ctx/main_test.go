package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/Xavier-GAO-42/clewkeep/internal/core"
)

func TestParseScanFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantFull bool
		wantJSON bool
		wantErr  bool
	}{
		{name: "default"},
		{name: "json", args: []string{"--json"}, wantJSON: true},
		{name: "full", args: []string{"--full"}, wantFull: true},
		{name: "full json", args: []string{"--full", "--json"}, wantFull: true, wantJSON: true},
		{name: "json full", args: []string{"--json", "--full"}, wantFull: true, wantJSON: true},
		{name: "unknown flag", args: []string{"--force"}, wantErr: true},
		{name: "positional", args: []string{"all"}, wantErr: true},
		{name: "inline value", args: []string{"--full=true"}, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseScanFlags(test.args)
			if test.wantErr {
				if err == nil {
					t.Fatalf("parseScanFlags(%v) succeeded, want error", test.args)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.full != test.wantFull || got.json != test.wantJSON {
				t.Fatalf("parseScanFlags(%v) = %#v, want full=%v json=%v", test.args, got, test.wantFull, test.wantJSON)
			}
		})
	}
}

func TestRunScanFullJSONUsesFrozenCatalogContract(t *testing.T) {
	userHome := t.TempDir()
	storeHome := filepath.Join(t.TempDir(), ".ctx")
	t.Setenv("HOME", userHome)
	t.Setenv("USERPROFILE", userHome)
	t.Setenv("CTX_HOME", storeHome)

	sessions := filepath.Join(userHome, ".codex", "sessions")
	if err := os.MkdirAll(sessions, 0o700); err != nil {
		t.Fatal(err)
	}
	data := []byte("{\"type\":\"session_meta\",\"payload\":{\"id\":\"synthetic-cli\",\"cwd\":\"C:/synthetic/project\",\"timestamp\":\"2026-01-01T00:00:00Z\",\"source\":\"exec\"}}\n")
	if err := os.WriteFile(filepath.Join(sessions, "synthetic.jsonl"), data, 0o600); err != nil {
		t.Fatal(err)
	}

	output := captureRunOutput(t, []string{"scan", "--full", "--json"})
	var catalog core.Catalog
	if err := json.Unmarshal(output, &catalog); err != nil {
		t.Fatalf("scan --full --json returned invalid JSON: %v\n%s", err, output)
	}
	if catalog.Format != "CtxCatalog" || catalog.SchemaVersion != core.CatalogSchemaVersion {
		t.Fatalf("scan --full --json contract = %q/%q", catalog.Format, catalog.SchemaVersion)
	}
	if len(catalog.Threads) != 1 || catalog.Threads[0].ID != "codex/synthetic-cli" {
		t.Fatalf("scan --full --json threads = %#v", catalog.Threads)
	}
	for _, path := range []string{core.CatalogPath(storeHome), core.ScanCachePath(storeHome)} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("scan --full did not write %s: %v", path, err)
		}
	}
}

func captureRunOutput(t *testing.T, args []string) []byte {
	t.Helper()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	previous := os.Stdout
	os.Stdout = writer
	t.Cleanup(func() { os.Stdout = previous })

	runErr := run(args)
	closeErr := writer.Close()
	os.Stdout = previous
	output, readErr := io.ReadAll(reader)
	_ = reader.Close()
	if runErr != nil {
		t.Fatal(runErr)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
	if readErr != nil {
		t.Fatal(readErr)
	}
	return output
}
