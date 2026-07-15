package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Xavier-GAO-42/clewkeep/internal/core"
)

func TestParseSearchArgsStrict(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    searchArgs
		wantErr string
	}{
		{"query only", []string{"two", "words"}, searchArgs{query: "two words", limit: 20}, ""},
		{"separate flags", []string{"needle", "--provider", "CoDeX", "--project", "Demo", "--limit", "3", "--json"}, searchArgs{query: "needle", provider: "CoDeX", project: "Demo", limit: 3, json: true}, ""},
		{"inline flags", []string{"--provider=claude-code", "--project=work", "--limit=7", "needle"}, searchArgs{query: "needle", provider: "claude-code", project: "work", limit: 7}, ""},
		{"end flags", []string{"--provider", "codex", "--", "--literal", "needle"}, searchArgs{query: "--literal needle", provider: "codex", limit: 20}, ""},
		{"all literal after end flags", []string{"--", "--provider", "codex"}, searchArgs{query: "--provider codex", limit: 20}, ""},
		{"unknown", []string{"needle", "--wat"}, searchArgs{}, "unknown"},
		{"unknown inline", []string{"needle", "--wat=value"}, searchArgs{}, "unknown"},
		{"missing", []string{"needle", "--project"}, searchArgs{}, "missing"},
		{"missing before another flag", []string{"needle", "--project", "--json"}, searchArgs{}, "missing"},
		{"empty", []string{"needle", "--provider="}, searchArgs{}, "empty"},
		{"empty project", []string{"needle", "--project=   "}, searchArgs{}, "empty"},
		{"empty limit", []string{"needle", "--limit="}, searchArgs{}, "empty"},
		{"duplicate mixed", []string{"needle", "--limit", "1", "--limit=2"}, searchArgs{}, "duplicate"},
		{"duplicate provider", []string{"needle", "--provider", "codex", "--provider=claude-code"}, searchArgs{}, "duplicate"},
		{"duplicate project", []string{"needle", "--project", "a", "--project=b"}, searchArgs{}, "duplicate"},
		{"duplicate json", []string{"needle", "--json", "--json"}, searchArgs{}, "duplicate"},
		{"json value", []string{"needle", "--json=true"}, searchArgs{}, "does not accept"},
		{"zero limit", []string{"needle", "--limit", "0"}, searchArgs{}, "between 1 and 500"},
		{"negative limit", []string{"needle", "--limit", "-1"}, searchArgs{}, "between 1 and 500"},
		{"large limit", []string{"needle", "--limit", "501"}, searchArgs{}, "between 1 and 500"},
		{"non-number limit", []string{"needle", "--limit", "many"}, searchArgs{}, "between 1 and 500"},
		{"no query", []string{"--json"}, searchArgs{}, "usage"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseSearchArgs(test.args)
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("got %#v, %v; want error containing %q", got, err, test.wantErr)
				}
				return
			}
			if err != nil || !reflect.DeepEqual(got, test.want) {
				t.Fatalf("got %#v, %v; want %#v", got, err, test.want)
			}
		})
	}
}

func TestRunSearchEmptyJSONAndVersion(t *testing.T) {
	storeHome := t.TempDir()
	t.Setenv("CTX_HOME", storeHome)
	catalog := core.Catalog{
		Format:        "CtxCatalog",
		SchemaVersion: core.CatalogSchemaVersion,
		GeneratedAt:   "2026-01-01T00:00:00Z",
		Threads:       []core.Thread{},
	}
	if err := core.WriteJSONAtomic(core.CatalogPath(storeHome), catalog); err != nil {
		t.Fatal(err)
	}
	if got := string(captureRunOutput(t, []string{"search", "absent", "--json"})); got != "[]\n" {
		t.Fatalf("empty search JSON = %q, want []\\n", got)
	}
	if got := string(captureRunOutput(t, []string{"version"})); got != "ctx 0.1.0-rc.2\n" {
		t.Fatalf("version output = %q", got)
	}
}
