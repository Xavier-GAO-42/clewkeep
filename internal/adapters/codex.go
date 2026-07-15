package adapters

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Xavier-GAO-42/clewkeep/internal/core"
)

type Codex struct{}

func (Codex) Name() string { return "codex" }

func (Codex) Roots(userHome string) []string {
	return []string{
		filepath.Join(userHome, ".codex", "sessions"),
		filepath.Join(userHome, ".codex", "archived_sessions"),
	}
}

func (Codex) Scan(ctx context.Context, root string) ([]core.Thread, error) {
	return scanDiscovered(ctx, root, Codex{})
}

func (Codex) Discover(ctx context.Context, root string) ([]NativeFile, error) {
	return discoverJSONL(ctx, root)
}

func (Codex) Parse(ctx context.Context, _ string, file NativeFile) (*core.Thread, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return readCodexThread(file.Path)
}

func readCodexThread(path string) (*core.Thread, error) {
	var meta map[string]any
	_, err := scanJSONLines(path, 1, func(_ int, raw []byte) bool {
		meta, _ = decodeObject(raw)
		return false
	})
	if err != nil {
		return nil, err
	}
	if meta == nil || stringValue(meta["type"]) != "session_meta" {
		return nil, fmt.Errorf("not a Codex session_meta JSONL")
	}
	payload := objectValue(meta["payload"])
	if payload == nil {
		return nil, fmt.Errorf("missing session_meta payload")
	}
	id := stringValue(payload["id"])
	if id == "" {
		id = stringValue(payload["session_id"])
	}
	if id == "" {
		return nil, fmt.Errorf("missing Codex thread id")
	}
	canonicalID, err := core.CodexCanonicalID(id)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	count, err := lineCount(path)
	if err != nil {
		return nil, err
	}
	source := codexSourceLabel(payload["source"])
	relations, warnings := codexRelations(payload)
	return &core.Thread{
		ID:               canonicalID,
		NativeSessionID:  id,
		RecordKind:       core.RecordKindSession,
		Provider:         "codex",
		Environment:      codexEnvironment(source),
		ProjectRoot:      stringValue(payload["cwd"]),
		Title:            id,
		CreatedAt:        stringValue(payload["timestamp"]),
		UpdatedAt:        info.ModTime().UTC().Format(timeFormat),
		NativePath:       path,
		NativeFormat:     "codex.rollout-jsonl",
		Source:           source,
		Originator:       stringValue(payload["originator"]),
		Model:            stringValue(payload["model"]),
		HarnessVersion:   stringValue(payload["cli_version"]),
		LineCount:        count,
		NativeRelations:  relations,
		RelationWarnings: warnings,
	}, nil
}

const timeFormat = "2006-01-02T15:04:05.999999999Z07:00"

func codexSourceLabel(value any) string {
	if text := stringValue(value); text != "" {
		return text
	}
	object := objectValue(value)
	if object == nil {
		return "unknown"
	}
	if objectValue(object["subagent"]) != nil {
		return "subagent"
	}
	if kind := stringValue(object["kind"]); kind != "" {
		return kind
	}
	return "object"
}

func codexEnvironment(source string) string {
	switch strings.ToLower(source) {
	case "vscode":
		return "codex-vscode"
	case "exec":
		return "codex-exec"
	case "subagent":
		return "codex-subagent"
	case "", "unknown":
		return "codex-unknown"
	default:
		return "codex-" + strings.ToLower(source)
	}
}

func codexRelations(payload map[string]any) ([]core.Relation, []string) {
	relations := make([]core.Relation, 0, 2)
	warnings := make([]string, 0)
	if parent := stringValue(payload["forked_from_id"]); parent != "" {
		relations = append(relations, core.Relation{
			Kind:           "fork",
			ParentThreadID: parent,
			Confidence:     "exact",
			EvidenceFields: []string{"session_meta.payload.forked_from_id"},
		})
	}
	topParent := stringValue(payload["parent_thread_id"])
	nestedParent := ""
	if source := objectValue(payload["source"]); source != nil {
		if subagent := objectValue(source["subagent"]); subagent != nil {
			if spawn := objectValue(subagent["thread_spawn"]); spawn != nil {
				nestedParent = stringValue(spawn["parent_thread_id"])
			}
		}
	}
	if topParent != "" && nestedParent != "" && topParent != nestedParent {
		warnings = append(warnings, "conflicting native spawn parent fields")
		return relations, warnings
	}
	parent := topParent
	fields := make([]string, 0, 2)
	if topParent != "" {
		fields = append(fields, "session_meta.payload.parent_thread_id")
	}
	if nestedParent != "" {
		parent = nestedParent
		fields = append(fields, "session_meta.payload.source.subagent.thread_spawn.parent_thread_id")
	}
	if parent != "" {
		relations = append(relations, core.Relation{
			Kind:           "spawn",
			ParentThreadID: parent,
			Confidence:     "exact",
			EvidenceFields: fields,
		})
	}
	return relations, warnings
}
