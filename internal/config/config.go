package config

import (
	"context"

	"vleet/internal/errx"
)

// Config is the user-level configuration described in docs/design.md and docs/architecture.md.
//
// Encoding is intentionally left TBD (docs propose YAML). The store layer is responsible
// for serialization and file permissions (0600 for secrets).
type Config struct {
	// Editor is the editor command (e.g. "vim", "nvim", "$EDITOR").
	Editor string

	// DefaultLang is the default LeetCode language slug (default: "cpp").
	DefaultLang string

	LeetCode LeetCodeAuth
}

// LeetCodeAuth holds LeetCode auth secrets. Treat as sensitive.
type LeetCodeAuth struct {
	// Session is the value of the LEETCODE_SESSION cookie.
	Session string

	// CSRFTOKEN is the value of the csrftoken cookie (optional depending on endpoint behavior).
	CSRFTOKEN string
}

// Store loads and saves config. Implementations must protect secrets at rest
// (e.g. file mode 0600) and must not log/print secret values.
type Store interface {
	Load(ctx context.Context) (Config, error)
	Save(ctx context.Context, cfg Config) error
}

// FileStore is a filesystem-backed config store (e.g. ~/.config/vleet/config.yaml).
// Skeleton only: load/save are not implemented yet.
type FileStore struct {
	Path string
}

func NewFileStore(path string) *FileStore {
	return &FileStore{Path: path}
}

func (s *FileStore) Load(ctx context.Context) (Config, error) {
	return Config{}, errx.NotImplemented("config.FileStore.Load")
}

func (s *FileStore) Save(ctx context.Context, cfg Config) error {
	return errx.NotImplemented("config.FileStore.Save")
}
