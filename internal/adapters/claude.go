package adapters

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Xavier-GAO-42/clewkeep/internal/core"
)

type ClaudeCode struct{}

func (ClaudeCode) Name() string { return "claude-code" }

func (ClaudeCode) Roots(userHome string) []string {
	return []string{filepath.Join(userHome, ".claude", "projects")}
}

func (ClaudeCode) Scan(ctx context.Context, root string) ([]core.Thread, error) {
	return scanDiscovered(ctx, root, ClaudeCode{})
}

func (ClaudeCode) Discover(ctx context.Context, root string) ([]NativeFile, error) {
	return discoverJSONL(ctx, root)
}

func (ClaudeCode) Parse(ctx context.Context, root string, file NativeFile) (*core.Thread, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return readClaudeThread(file.Path, root)
}

func readClaudeThread(path, root string) (*core.Thread, error) {
	sessionID := ""
	agentID := ""
	cwd := ""
	createdAt := ""
	var sidechain *bool
	var parseErr error
	_, err := scanJSONLines(path, 25, func(_ int, raw []byte) bool {
		object, ok := decodeObject(raw)
		if !ok {
			return true
		}
		if value, exists := object["sessionId"]; exists {
			text, ok := value.(string)
			if !ok || text == "" {
				parseErr = fmt.Errorf("invalid Claude sessionId field")
				return false
			}
			if sessionID != "" && sessionID != text {
				parseErr = fmt.Errorf("conflicting Claude sessionId fields")
				return false
			}
			sessionID = text
		}
		if value, exists := object["agentId"]; exists {
			text, ok := value.(string)
			if !ok || text == "" {
				parseErr = fmt.Errorf("invalid Claude agentId field")
				return false
			}
			if agentID != "" && agentID != text {
				parseErr = fmt.Errorf("conflicting Claude agentId fields")
				return false
			}
			agentID = text
		}
		if value, exists := object["isSidechain"]; exists {
			flag, ok := value.(bool)
			if !ok {
				parseErr = fmt.Errorf("invalid Claude isSidechain field")
				return false
			}
			if sidechain != nil && *sidechain != flag {
				parseErr = fmt.Errorf("conflicting Claude isSidechain fields")
				return false
			}
			sidechain = &flag
		}
		if cwd == "" {
			cwd = stringValue(object["cwd"])
		}
		if createdAt == "" {
			createdAt = stringValue(object["timestamp"])
		}
		return true
	})
	if err != nil {
		return nil, err
	}
	if parseErr != nil {
		return nil, parseErr
	}
	if sessionID == "" {
		return nil, fmt.Errorf("missing Claude sessionId; file-name fallback is not allowed")
	}
	if sidechain != nil && !*sidechain && agentID != "" {
		return nil, fmt.Errorf("Claude record has agentId but isSidechain is false")
	}
	isSubagent := agentID != "" || (sidechain != nil && *sidechain)
	if isSubagent && agentID == "" {
		return nil, fmt.Errorf("Claude subagent is missing agentId")
	}
	canonicalID, err := core.ClaudeCanonicalID(sessionID, agentID)
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
	project := cwd
	if project == "" {
		projectDir := filepath.Dir(path)
		if relative, err := filepath.Rel(root, projectDir); err == nil {
			project = relative
		} else {
			project = projectDir
		}
	}
	thread := &core.Thread{
		ID:              canonicalID,
		NativeSessionID: sessionID,
		NativeAgentID:   agentID,
		RecordKind:      core.RecordKindSession,
		Provider:        "claude-code",
		Environment:     "claude-code",
		ProjectRoot:     project,
		Title:           sessionID,
		CreatedAt:       createdAt,
		UpdatedAt:       info.ModTime().UTC().Format(timeFormat),
		NativePath:      path,
		NativeFormat:    "claude-code.transcript-jsonl",
		Source:          "jsonl",
		Originator:      "claude-code",
		LineCount:       count,
		NativeRelations: nil,
	}
	if isSubagent {
		thread.RecordKind = core.RecordKindSubagent
	}
	return thread, nil
}
