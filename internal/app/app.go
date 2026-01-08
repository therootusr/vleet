package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"vleet/internal/config"
	"vleet/internal/editor"
	"vleet/internal/errx"
	"vleet/internal/leetcode"
	"vleet/internal/output"
	"vleet/internal/render"
	"vleet/internal/workspace"
)

const (
	kDefaultLang = "cpp"
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
func (a *App) Solve(ctx context.Context, opts SolveOptions) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(opts.ProblemKey) == "" {
		return fmt.Errorf("problem key (titleSlug) is required")
	}

	cfg, err := a.loadConfigOrDefault(ctx)
	if err != nil {
		return err
	}

	prep, err := a.prepareSolutionFile(ctx, opts.ProblemKey, opts.Lang)
	if err != nil {
		return err
	}

	// Keep output minimal; avoid breaking --json users by emitting extra lines.
	if sp, ok := a.Output.(*output.StdPrinter); ok && !sp.JSON {
		if prep.CreatedNewFile {
			_, _ = fmt.Fprintf(sp.Out, "wrote solution: %s\n", prep.Workspace.SolutionPath)
		} else {
			_, _ = fmt.Fprintf(sp.Out, "solution already exists: %s\n", prep.Workspace.SolutionPath)
		}
	}

	if a.Editor == nil {
		return fmt.Errorf("editor runner is not configured")
	}
	if err := a.Editor.OpenFile(ctx, cfg.Editor, prep.Workspace.SolutionPath); err != nil {
		return err
	}

	if opts.Submit {
		return errx.NotImplemented("app.App.Solve --submit")
	}
	return nil
}

// Fetch fetches question metadata and (optionally) generates a solution file without opening an editor.
// See docs/design.md "Other commands (MVP-friendly)".
func (a *App) Fetch(ctx context.Context, opts FetchOptions) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(opts.ProblemKey) == "" {
		return fmt.Errorf("problem key (titleSlug) is required")
	}

	prep, err := a.prepareSolutionFile(ctx, opts.ProblemKey, opts.Lang)
	if err != nil {
		return err
	}

	if a.Output != nil {
		if err := a.Output.PrintQuestion(ctx, prep.Question); err != nil {
			return err
		}
	}

	// Keep output minimal; avoid breaking --json users by emitting extra lines.
	if sp, ok := a.Output.(*output.StdPrinter); ok && !sp.JSON {
		if prep.CreatedNewFile {
			_, _ = fmt.Fprintf(sp.Out, "wrote solution: %s\n", prep.Workspace.SolutionPath)
		} else {
			_, _ = fmt.Fprintf(sp.Out, "solution already exists: %s\n", prep.Workspace.SolutionPath)
		}
	}
	return nil
}

// Submit submits a solution from an existing workspace.
// See docs/architecture.md "vleet submit" flow.
//
// Skeleton only: not implemented yet.
func (a *App) Submit(ctx context.Context, opts SubmitOptions) error {
	return errx.NotImplemented("app.App.Submit")
}

type preparedSolution struct {
	Workspace       workspace.Workspace
	Question        leetcode.Question
	Lang            string
	CreatedNewFile  bool
	SelectedSnippet leetcode.CodeSnippet
}

func (a *App) prepareSolutionFile(ctx context.Context, problemKey string, langFlag string) (preparedSolution, error) {
	if a.LeetCode == nil {
		return preparedSolution{}, fmt.Errorf("leetcode client is not configured")
	}
	if a.Renderer == nil {
		return preparedSolution{}, fmt.Errorf("renderer is not configured")
	}
	if a.Workspace == nil {
		return preparedSolution{}, fmt.Errorf("workspace manager is not configured")
	}

	cfg, err := a.loadConfigOrDefault(ctx)
	if err != nil {
		return preparedSolution{}, err
	}

	lang := strings.TrimSpace(langFlag)
	if lang == "" {
		lang = strings.TrimSpace(cfg.DefaultLang)
	}
	if lang == "" {
		lang = kDefaultLang
	}

	q, err := a.LeetCode.FetchQuestion(ctx, problemKey)
	if err != nil {
		return preparedSolution{}, err
	}

	snippet, err := selectSnippet(q.CodeSnippets, lang)
	if err != nil {
		return preparedSolution{}, err
	}

	header, err := a.Renderer.RenderHeader(ctx, lang, q)
	if err != nil {
		return preparedSolution{}, err
	}

	ws, err := a.Workspace.CreateWorkspace(ctx, ".", q, lang, workspace.CreateOptions{})
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			// Workspace already exists; load it and don't overwrite.
			ws, loadErr := a.Workspace.LoadWorkspace(ctx, ".", q.TitleSlug, lang, "")
			if loadErr != nil {
				return preparedSolution{}, err
			}
			return preparedSolution{
				Workspace:       ws,
				Question:        q,
				Lang:            lang,
				CreatedNewFile:  false,
				SelectedSnippet: snippet,
			}, nil
		}
		return preparedSolution{}, err
	}

	content := header + "\n" + snippet.Code
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	if err := a.Workspace.WriteSolution(ctx, ws, content); err != nil {
		if errors.Is(err, os.ErrExist) {
			// Race: file created by another process between CreateWorkspace and WriteSolution.
			return preparedSolution{
				Workspace:       ws,
				Question:        q,
				Lang:            lang,
				CreatedNewFile:  false,
				SelectedSnippet: snippet,
			}, nil
		}
		return preparedSolution{}, err
	}

	return preparedSolution{
		Workspace:       ws,
		Question:        q,
		Lang:            lang,
		CreatedNewFile:  true,
		SelectedSnippet: snippet,
	}, nil
}

func (a *App) loadConfigOrDefault(ctx context.Context) (config.Config, error) {
	if a.ConfigStore == nil {
		return config.Config{}, nil
	}
	cfg, err := a.ConfigStore.Load(ctx)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return config.Config{}, nil
		}
		return config.Config{}, err
	}
	return cfg, nil
}

func selectSnippet(snippets []leetcode.CodeSnippet, lang string) (leetcode.CodeSnippet, error) {
	lang = strings.TrimSpace(lang)
	if lang == "" {
		return leetcode.CodeSnippet{}, fmt.Errorf("lang is required")
	}

	var available []string
	for _, snip := range snippets {
		if strings.TrimSpace(snip.LangSlug) != "" {
			available = append(available, snip.LangSlug)
		}
		if strings.EqualFold(strings.TrimSpace(snip.LangSlug), lang) {
			return snip, nil
		}
	}

	if len(available) == 0 {
		return leetcode.CodeSnippet{}, fmt.Errorf("no code snippets available for this problem")
	}
	return leetcode.CodeSnippet{}, fmt.Errorf("no snippet for lang %q (available: %s)", lang, strings.Join(available, ", "))
}
