package workspace

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/therootusr/go-leetcode"
)

const (
	kDefaultSolutionBaseName = "solution"
)

// Workspace represents the per-problem directory described in docs/design.md.
type Workspace struct {
	// Dir is the workspace directory (typically ./<problem-key>/).
	Dir string

	// ProblemKey is the problem key (MVP: titleSlug) used for the workspace dir name.
	ProblemKey string

	// Lang is the LeetCode language slug (e.g. "cpp").
	Lang string

	// SolutionPath is the resolved path to solution.<ext> inside Dir (unless overridden).
	SolutionPath string
}

type CreateOptions struct {
	// File overrides the default solution file name/path (must match language extension).
	File string
}

// Manager is the internal API contract described in docs/architecture.md.
type Manager interface {
	CreateWorkspace(ctx context.Context, root string, q leetcode.Question, lang string, opts CreateOptions) (Workspace, error)
	LoadWorkspace(ctx context.Context, dir string, problemKey string, lang string, file string) (Workspace, error)
	ReadSolution(ctx context.Context, ws Workspace) (string, error)
	WriteSolution(ctx context.Context, ws Workspace, content string) error
}

// FSManager manages workspaces on disk.
type FSManager struct{}

func NewFSManager() *FSManager { return &FSManager{} }

func (m *FSManager) CreateWorkspace(ctx context.Context, root string, q leetcode.Question, lang string, opts CreateOptions) (Workspace, error) {
	if err := ctx.Err(); err != nil {
		return Workspace{}, err
	}

	lang = strings.TrimSpace(lang)
	if lang == "" {
		return Workspace{}, fmt.Errorf("lang is required")
	}

	problemKey := strings.TrimSpace(q.TitleSlug)
	if problemKey == "" {
		return Workspace{}, fmt.Errorf("question titleSlug is required")
	}

	workspaceDir := strings.TrimSpace(root)
	if workspaceDir == "" {
		workspaceDir = "."
	}
	workspaceDir = filepath.Join(workspaceDir, problemKey)

	ext, err := extensionForLang(lang)
	if err != nil {
		return Workspace{}, err
	}

	solutionPath, err := resolveSolutionPath(workspaceDir, ext, opts.File)
	if err != nil {
		return Workspace{}, err
	}

	// Create workspace directory if it doesn't exist.
	// Use 0755 so the directory is traversable by the user and common tools.
	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
		return Workspace{}, fmt.Errorf("create workspace dir %s: %w", workspaceDir, err)
	}

	// Ensure solution doesn't already exist (v1 policy: never overwrite).
	if _, err := os.Stat(solutionPath); err == nil {
		return Workspace{}, fmt.Errorf("solution already exists at %s: %w", solutionPath, os.ErrExist)
	} else if !errorsIsNotExist(err) {
		return Workspace{}, fmt.Errorf("stat solution %s: %w", solutionPath, err)
	}

	return Workspace{
		Dir:          workspaceDir,
		ProblemKey:   problemKey,
		Lang:         lang,
		SolutionPath: solutionPath,
	}, nil
}

func (m *FSManager) LoadWorkspace(ctx context.Context, dir string, problemKey string, lang string, file string) (Workspace, error) {
	if err := ctx.Err(); err != nil {
		return Workspace{}, err
	}

	lang = strings.TrimSpace(lang)
	if lang == "" {
		return Workspace{}, fmt.Errorf("lang is required")
	}

	dir = strings.TrimSpace(dir)
	problemKey = strings.TrimSpace(problemKey)

	var workspaceDir string
	switch {
	case dir != "" && problemKey != "":
		workspaceDir = filepath.Join(dir, problemKey)
	case dir != "" && problemKey == "":
		workspaceDir = dir
		problemKey = filepath.Base(strings.TrimRight(workspaceDir, string(filepath.Separator)))
	case dir == "" && problemKey != "":
		workspaceDir = filepath.Join(".", problemKey)
	default:
		return Workspace{}, fmt.Errorf("dir or problemKey is required")
	}

	ext, err := extensionForLang(lang)
	if err != nil {
		return Workspace{}, err
	}

	solutionPath, err := resolveSolutionPath(workspaceDir, ext, file)
	if err != nil {
		return Workspace{}, err
	}

	fi, err := os.Stat(workspaceDir)
	if err != nil {
		return Workspace{}, fmt.Errorf("stat workspace dir %s: %w", workspaceDir, err)
	}
	if !fi.IsDir() {
		return Workspace{}, fmt.Errorf("workspace path is not a directory: %s", workspaceDir)
	}

	return Workspace{
		Dir:          workspaceDir,
		ProblemKey:   problemKey,
		Lang:         lang,
		SolutionPath: solutionPath,
	}, nil
}

func (m *FSManager) ReadSolution(ctx context.Context, ws Workspace) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if strings.TrimSpace(ws.SolutionPath) == "" {
		return "", fmt.Errorf("workspace solution path is empty")
	}
	b, err := os.ReadFile(ws.SolutionPath)
	if err != nil {
		return "", fmt.Errorf("read solution %s: %w", ws.SolutionPath, err)
	}
	return string(b), nil
}

func (m *FSManager) WriteSolution(ctx context.Context, ws Workspace, content string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(ws.SolutionPath) == "" {
		return fmt.Errorf("workspace solution path is empty")
	}

	if err := os.MkdirAll(filepath.Dir(ws.SolutionPath), 0o755); err != nil {
		return fmt.Errorf("create solution dir %s: %w", filepath.Dir(ws.SolutionPath), err)
	}

	// v1 policy: never overwrite. Create file exclusively.
	f, err := os.OpenFile(ws.SolutionPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if errorsIsExist(err) {
			return fmt.Errorf("solution already exists at %s: %w", ws.SolutionPath, os.ErrExist)
		}
		return fmt.Errorf("create solution %s: %w", ws.SolutionPath, err)
	}
	defer f.Close()

	if _, err := f.WriteString(content); err != nil {
		return fmt.Errorf("write solution %s: %w", ws.SolutionPath, err)
	}
	return nil
}

func extensionForLang(lang string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "cpp":
		return ".cpp", nil
	case "golang":
		return ".go", nil
	case "python3":
		return ".py", nil
	case "javascript":
		return ".js", nil
	case "typescript":
		return ".ts", nil
	default:
		return "", fmt.Errorf("unsupported language slug: %q", lang)
	}
}

func resolveSolutionPath(workspaceDir string, expectedExt string, fileOverride string) (string, error) {
	if strings.TrimSpace(workspaceDir) == "" {
		return "", fmt.Errorf("workspaceDir is required")
	}
	if strings.TrimSpace(expectedExt) == "" {
		return "", fmt.Errorf("expectedExt is required")
	}

	if strings.TrimSpace(fileOverride) == "" {
		return filepath.Join(workspaceDir, kDefaultSolutionBaseName+expectedExt), nil
	}

	p := strings.TrimSpace(fileOverride)
	if !filepath.IsAbs(p) {
		p = filepath.Join(workspaceDir, p)
	}

	if ext := filepath.Ext(p); ext != expectedExt {
		return "", fmt.Errorf("solution file extension %q does not match language extension %q", ext, expectedExt)
	}
	return p, nil
}

func errorsIsNotExist(err error) bool { return errors.Is(err, os.ErrNotExist) }
func errorsIsExist(err error) bool    { return errors.Is(err, os.ErrExist) }
