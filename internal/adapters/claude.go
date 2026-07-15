package adapters

import (
	"context"
	"os"
	"path/filepath"
	"strings"

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
	cwd := ""
	createdAt := ""
	_, err := scanJSONLines(path, 25, func(_ int, raw []byte) bool {
		object, ok := decodeObject(raw)
		if !ok {
			return true
		}
		if sessionID == "" {
			sessionID = stringValue(object["sessionId"])
		}
		if cwd == "" {
			cwd = stringValue(object["cwd"])
		}
		if createdAt == "" {
			createdAt = stringValue(object["timestamp"])
		}
		return sessionID == "" || cwd == ""
	})
	if err != nil {
		return nil, err
	}
	if sessionID == "" {
		sessionID = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
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
	return &core.Thread{
		ID:              sessionID,
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
	}, nil
}
