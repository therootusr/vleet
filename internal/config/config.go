package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config is the user-level configuration described in docs/design.md and docs/architecture.md.
//
// Encoding is intentionally left TBD (docs propose YAML). The store layer is responsible
// for serialization and file permissions (0600 for secrets).
type Config struct {
	// Editor is the editor command (e.g. "vim", "nvim", "$EDITOR").
	Editor string `yaml:"editor"`

	// DefaultLang is the default LeetCode language slug (default: "cpp").
	DefaultLang string `yaml:"default_lang"`

	LeetCode LeetCodeAuth `yaml:"leetcode"`
}

// LeetCodeAuth holds LeetCode auth secrets. Treat as sensitive.
type LeetCodeAuth struct {
	// Session is the value of the LEETCODE_SESSION cookie.
	Session string `yaml:"session"`

	// CSRFTOKEN is the value of the csrftoken cookie (optional depending on endpoint behavior).
	CSRFTOKEN string `yaml:"csrftoken"`
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
	if err := ctx.Err(); err != nil {
		return Config{}, err
	}
	if s.Path == "" {
		return Config{}, fmt.Errorf("config path is empty")
	}

	fi, err := os.Stat(s.Path)
	if err != nil {
		return Config{}, fmt.Errorf("stat config %s: %w", s.Path, err)
	}
	if fi.Mode().Perm()&0o077 != 0 {
		return Config{}, fmt.Errorf(
			"config file %s has insecure permissions (%#o); run: chmod 600 %s",
			s.Path,
			fi.Mode().Perm(),
			s.Path,
		)
	}

	b, err := os.ReadFile(s.Path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %s: %w", s.Path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse yaml config %s: %w", s.Path, err)
	}
	return cfg, nil
}

func (s *FileStore) Save(ctx context.Context, cfg Config) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.Path == "" {
		return fmt.Errorf("config path is empty")
	}

	dir := filepath.Dir(s.Path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("create config dir %s: %w", dir, err)
		}
	}

	b, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal yaml config: %w", err)
	}
	if len(b) > 0 && b[len(b)-1] != '\n' {
		b = append(b, '\n')
	}

	// Ensure file contents are written and permissions are enforced (0600).
	// Note: os.WriteFile does not change permissions on an existing file.
	if err := os.WriteFile(s.Path, b, 0o600); err != nil {
		return fmt.Errorf("write config %s: %w", s.Path, err)
	}
	if err := os.Chmod(s.Path, 0o600); err != nil {
		return fmt.Errorf("chmod config %s: %w", s.Path, err)
	}
	return nil
}
