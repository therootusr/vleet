package workspace

import (
	"context"

	"vleet/internal/errx"
	"vleet/internal/leetcode"
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
// Skeleton only: behaviors are not implemented yet.
type FSManager struct{}

func NewFSManager() *FSManager { return &FSManager{} }

func (m *FSManager) CreateWorkspace(ctx context.Context, root string, q leetcode.Question, lang string, opts CreateOptions) (Workspace, error) {
	return Workspace{}, errx.NotImplemented("workspace.FSManager.CreateWorkspace")
}

func (m *FSManager) LoadWorkspace(ctx context.Context, dir string, problemKey string, lang string, file string) (Workspace, error) {
	return Workspace{}, errx.NotImplemented("workspace.FSManager.LoadWorkspace")
}

func (m *FSManager) ReadSolution(ctx context.Context, ws Workspace) (string, error) {
	return "", errx.NotImplemented("workspace.FSManager.ReadSolution")
}

func (m *FSManager) WriteSolution(ctx context.Context, ws Workspace, content string) error {
	return errx.NotImplemented("workspace.FSManager.WriteSolution")
}
