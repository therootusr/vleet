package config

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestFileStore_SaveLoad_RoundTripAndPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission semantics differ on Windows")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	store := NewFileStore(path)

	want := Config{
		Editor:      "vim",
		DefaultLang: "cpp",
		LeetCode: LeetCodeAuth{
			Session:   "session-value",
			CSRFTOKEN: "csrf-value",
		},
	}

	if err := store.Save(context.Background(), want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat written config: %v", err)
	}
	if got := fi.Mode().Perm(); got != 0o600 {
		t.Fatalf("config perms = %#o, want %#o", got, 0o600)
	}

	got, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got.Editor != want.Editor {
		t.Fatalf("Editor = %q, want %q", got.Editor, want.Editor)
	}
	if got.DefaultLang != want.DefaultLang {
		t.Fatalf("DefaultLang = %q, want %q", got.DefaultLang, want.DefaultLang)
	}
	if got.LeetCode.Session != want.LeetCode.Session {
		t.Fatalf("LeetCode.Session = %q, want %q", got.LeetCode.Session, want.LeetCode.Session)
	}
	if got.LeetCode.CSRFTOKEN != want.LeetCode.CSRFTOKEN {
		t.Fatalf("LeetCode.CSRFTOKEN = %q, want %q", got.LeetCode.CSRFTOKEN, want.LeetCode.CSRFTOKEN)
	}
}

func TestFileStore_Load_RejectsInsecurePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission semantics differ on Windows")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	// Write a minimal YAML config with insecure permissions.
	const yamlConfig = "editor: vim\ndefault_lang: cpp\nleetcode:\n  session: s\n  csrftoken: c\n"
	if err := os.WriteFile(path, []byte(yamlConfig), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatalf("chmod config: %v", err)
	}

	store := NewFileStore(path)
	_, err := store.Load(context.Background())
	if err == nil {
		t.Fatalf("Load() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "insecure permissions") {
		t.Fatalf("Load() error = %q, want message to include %q", err.Error(), "insecure permissions")
	}
}
