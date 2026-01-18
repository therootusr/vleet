package workspace

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/therootusr/go-leetcode"
)

func TestFSManager_CreateWriteRead_Sanity(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	m := NewFSManager()
	q := leetcode.Question{TitleSlug: "two-sum"}

	ws, err := m.CreateWorkspace(context.Background(), root, q, "cpp", CreateOptions{})
	if err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
	}

	wantDir := filepath.Join(root, "two-sum")
	wantPath := filepath.Join(wantDir, "solution.cpp")
	if ws.Dir != wantDir {
		t.Fatalf("ws.Dir = %q, want %q", ws.Dir, wantDir)
	}
	if ws.SolutionPath != wantPath {
		t.Fatalf("ws.SolutionPath = %q, want %q", ws.SolutionPath, wantPath)
	}

	if fi, err := os.Stat(ws.Dir); err != nil {
		t.Fatalf("workspace dir stat error = %v", err)
	} else if !fi.IsDir() {
		t.Fatalf("workspace dir is not a directory: %s", ws.Dir)
	}

	// CreateWorkspace should not create the solution file; it only reserves the path.
	if _, err := os.Stat(ws.SolutionPath); err == nil {
		t.Fatalf("expected solution file to not exist yet: %s", ws.SolutionPath)
	}

	const content = "// header\n\nint main() {}\n"
	if err := m.WriteSolution(context.Background(), ws, content); err != nil {
		t.Fatalf("WriteSolution() error = %v", err)
	}
	got, err := m.ReadSolution(context.Background(), ws)
	if err != nil {
		t.Fatalf("ReadSolution() error = %v", err)
	}
	if got != content {
		t.Fatalf("ReadSolution() = %q, want %q", got, content)
	}

	// v1 policy: never overwrite.
	if err := m.WriteSolution(context.Background(), ws, "overwrite"); err == nil {
		t.Fatalf("WriteSolution() expected error on overwrite, got nil")
	}
}

func TestFSManager_LoadWorkspace_Sanity(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	dir := filepath.Join(root, "two-sum")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir workspace dir: %v", err)
	}

	m := NewFSManager()
	ws, err := m.LoadWorkspace(context.Background(), root, "two-sum", "cpp", "")
	if err != nil {
		t.Fatalf("LoadWorkspace() error = %v", err)
	}

	if ws.Dir != dir {
		t.Fatalf("ws.Dir = %q, want %q", ws.Dir, dir)
	}
	if ws.ProblemKey != "two-sum" {
		t.Fatalf("ws.ProblemKey = %q, want %q", ws.ProblemKey, "two-sum")
	}
	if want := filepath.Join(dir, "solution.cpp"); ws.SolutionPath != want {
		t.Fatalf("ws.SolutionPath = %q, want %q", ws.SolutionPath, want)
	}
}

func TestFSManager_CreateWorkspace_FileOverride_ExtensionMismatch(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	m := NewFSManager()
	q := leetcode.Question{TitleSlug: "two-sum"}

	_, err := m.CreateWorkspace(context.Background(), root, q, "cpp", CreateOptions{File: "custom.py"})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "does not match language extension") {
		t.Fatalf("error = %q, want message to include %q", err.Error(), "does not match language extension")
	}
}
