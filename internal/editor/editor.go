package editor

import (
	"context"

	"vleet/internal/errx"
)

// Runner launches an external editor process and blocks until it exits.
// See docs/architecture.md "Editor runner".
type Runner interface {
	OpenFile(ctx context.Context, editorCmd string, filePath string) error
}

// ProcessRunner is a Runner implemented via os/exec.
// Skeleton only: process execution is not implemented yet.
type ProcessRunner struct{}

func NewProcessRunner() *ProcessRunner { return &ProcessRunner{} }

func (r *ProcessRunner) OpenFile(ctx context.Context, editorCmd string, filePath string) error {
	return errx.NotImplemented("editor.ProcessRunner.OpenFile")
}
