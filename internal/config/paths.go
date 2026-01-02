package config

import (
	"os"
	"path/filepath"
)

// DefaultConfigPath returns the default config path described in docs/architecture.md.
//
// Note: this function does not create directories or files.
func DefaultConfigPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "vleet", "config.yaml"), nil
}

// DefaultCacheDir returns the default cache directory described in docs/architecture.md.
//
// Note: this function does not create directories.
func DefaultCacheDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "vleet"), nil
}
