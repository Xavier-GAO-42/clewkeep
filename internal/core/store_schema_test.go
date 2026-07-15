package core

import (
	"strings"
	"testing"
)

func TestLoadNamesValidatesSchemaAndCanonicalTargets(t *testing.T) {
	storeHome := t.TempDir()
	missing, err := LoadNames(storeHome)
	if err != nil || missing.SchemaVersion != NameIndexSchemaVersion {
		t.Fatalf("missing names = %#v, %v", missing, err)
	}
	legacy := NameIndex{Format: "CtxNameIndex", SchemaVersion: "0.1", Names: map[string]string{"legacy": "native-id"}}
	if err := WriteJSONAtomic(NamesPath(storeHome), legacy); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadNames(storeHome); err != nil {
		t.Fatalf("legacy name index must remain readable for deterministic migration: %v", err)
	}
	invalid := NameIndex{Format: "CtxNameIndex", SchemaVersion: NameIndexSchemaVersion, Names: map[string]string{"bad": "native-id"}}
	if err := WriteJSONAtomic(NamesPath(storeHome), invalid); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadNames(storeHome); err == nil || !strings.Contains(err.Error(), "invalid canonical target") {
		t.Fatalf("invalid target error = %v", err)
	}
	invalid.SchemaVersion = "0.3"
	if err := WriteJSONAtomic(NamesPath(storeHome), invalid); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadNames(storeHome); err == nil || !strings.Contains(err.Error(), "incompatible") {
		t.Fatalf("future schema error = %v", err)
	}
}

func TestLoadCatalogRejectsDuplicateCanonicalIdentity(t *testing.T) {
	storeHome := t.TempDir()
	thread := Thread{ID: "codex/same", NativeSessionID: "same", RecordKind: RecordKindSession, Provider: "codex"}
	catalog := Catalog{Format: "CtxCatalog", SchemaVersion: CatalogSchemaVersion, Threads: []Thread{thread, thread}}
	if err := WriteJSONAtomic(CatalogPath(storeHome), catalog); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadCatalog(storeHome); err == nil || !strings.Contains(err.Error(), "duplicate canonical id") {
		t.Fatalf("duplicate catalog error = %v", err)
	}
}
