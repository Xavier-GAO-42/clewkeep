package core

import (
	"strings"
	"testing"
)

func TestCanonicalIDsAreProviderQualifiedAndInjective(t *testing.T) {
	tests := []struct {
		name    string
		build   func() (string, error)
		want    string
		wantErr bool
	}{
		{"codex", func() (string, error) { return CodexCanonicalID("same") }, "codex/same", false},
		{"claude main", func() (string, error) { return ClaudeCanonicalID("same", "") }, "claude/same", false},
		{"claude agent", func() (string, error) { return ClaudeCanonicalID("same", "a:1%2F") }, "claude/same/agent/a:1%2F", false},
		{"colon and percent remain distinct", func() (string, error) { return CodexCanonicalID("a:%2F") }, "codex/a:%2F", false},
		{"slash", func() (string, error) { return CodexCanonicalID("a/b") }, "", true},
		{"backslash", func() (string, error) { return ClaudeCanonicalID("a", `b\c`) }, "", true},
		{"control", func() (string, error) { return ClaudeCanonicalID("a", "b\n") }, "", true},
		{"leading whitespace", func() (string, error) { return CodexCanonicalID(" id") }, "", true},
		{"invalid UTF-8", func() (string, error) { return CodexCanonicalID(string([]byte{0xff})) }, "", true},
		{"maximum length", func() (string, error) { return CodexCanonicalID(strings.Repeat("x", 512)) }, "codex/" + strings.Repeat("x", 512), false},
		{"overlong", func() (string, error) { return CodexCanonicalID(strings.Repeat("x", 513)) }, "", true},
	}
	seen := map[string]string{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := test.build()
			if test.wantErr {
				if err == nil {
					t.Fatalf("got %q, want error", got)
				}
				return
			}
			if err != nil || got != test.want {
				t.Fatalf("got %q, %v; want %q", got, err, test.want)
			}
			if previous, duplicate := seen[got]; duplicate {
				t.Fatalf("canonical collision with %s", previous)
			}
			seen[got] = test.name
		})
	}
}

func TestCanonicalValidationRejectsDelimiterInjection(t *testing.T) {
	for _, id := range []string{
		"claude/session/agent/agent/agent/injected",
		"claude/session/agent/",
		"claude/session\\agent\\injected",
		"unknown/session",
	} {
		if err := ValidateCanonicalID(id); err == nil {
			t.Errorf("ValidateCanonicalID(%q) succeeded, want error", id)
		}
	}
}

func TestValidateThreadSetRejectsInvalidAndDuplicateIdentities(t *testing.T) {
	valid := Thread{ID: "codex/one", NativeSessionID: "one", RecordKind: RecordKindSession, Provider: "codex"}
	if err := ValidateThreadSet([]Thread{valid}); err != nil {
		t.Fatal(err)
	}
	invalid := valid
	invalid.ID = "codex/two"
	if err := ValidateThreadSet([]Thread{invalid}); err == nil || !strings.Contains(err.Error(), "invalid identity") {
		t.Fatalf("invalid set error = %v", err)
	}
	if err := ValidateThreadSet([]Thread{valid, valid}); err == nil || !strings.Contains(err.Error(), "duplicate canonical id") {
		t.Fatalf("duplicate set error = %v", err)
	}
}
