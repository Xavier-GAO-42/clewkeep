package adapters

import (
	"context"

	"github.com/Xavier-GAO-42/clewkeep/internal/core"
)

type Adapter interface {
	Name() string
	Roots(userHome string) []string
	Scan(ctx context.Context, root string) ([]core.Thread, error)
}

// NativeFile is the stable, cheap-to-read identity used by incremental scans.
// It contains no transcript content.
type NativeFile struct {
	Path            string
	Size            int64
	ModTimeUnixNano int64
}

// IncrementalAdapter separates file discovery from transcript parsing. App can
// then reuse ctx-owned metadata for files whose native fingerprint is unchanged.
type IncrementalAdapter interface {
	Adapter
	Discover(ctx context.Context, root string) ([]NativeFile, error)
	Parse(ctx context.Context, root string, file NativeFile) (*core.Thread, error)
}

func Builtins() []Adapter {
	return []Adapter{Codex{}, ClaudeCode{}}
}
