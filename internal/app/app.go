package app

import (
	"context"

	"vleet/internal/config"
	"vleet/internal/editor"
	"vleet/internal/errx"
	"vleet/internal/leetcode"
	"vleet/internal/output"
	"vleet/internal/render"
	"vleet/internal/workspace"
)

// App orchestrates the core components described in docs/architecture.md.
type App struct {
	ConfigStore config.Store
	LeetCode    leetcode.Client
	Workspace   workspace.Manager
	Renderer    render.Renderer
	Editor      editor.Runner
	Output      output.Printer
}

type SolveOptions struct {
	ProblemKey string // MVP: titleSlug
	Lang       string // LeetCode language slug (default: config.DefaultLang)
	Submit     bool
}

type FetchOptions struct {
	ProblemKey string
	Lang       string
}

type SubmitOptions struct {
	ProblemKey string
	Lang       string
	File       string // optional override; defaults to ./<problem-key>/solution.<ext>
}

func New(deps App) *App {
	// Intentionally shallow: dependency validation can be added once behavior is implemented.
	return &deps
}

// Solve implements the command flow described in docs/architecture.md:
// load config → fetch question → pick snippet → create workspace → write solution →
// open editor → optional submit → poll → print result.
//
// Skeleton only: not implemented yet.
func (a *App) Solve(ctx context.Context, opts SolveOptions) error {
	return errx.NotImplemented("app.App.Solve")
}

// Fetch fetches question metadata and (optionally) generates a solution file without opening an editor.
// See docs/design.md "Other commands (MVP-friendly)".
//
// Skeleton only: not implemented yet.
func (a *App) Fetch(ctx context.Context, opts FetchOptions) error {
	return errx.NotImplemented("app.App.Fetch")
}

// Submit submits a solution from an existing workspace.
// See docs/architecture.md "vleet submit" flow.
//
// Skeleton only: not implemented yet.
func (a *App) Submit(ctx context.Context, opts SubmitOptions) error {
	return errx.NotImplemented("app.App.Submit")
}
