package editor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	kDefaultEditor = "vim"
)

// Runner launches an external editor process and blocks until it exits.
// See docs/architecture.md "Editor runner".
type Runner interface {
	OpenFile(ctx context.Context, editorCmd string, filePath string) error
}

// ProcessRunner is a Runner implemented via os/exec.
type ProcessRunner struct{}

func NewProcessRunner() *ProcessRunner { return &ProcessRunner{} }

func (r *ProcessRunner) OpenFile(ctx context.Context, editorCmd string, filePath string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return fmt.Errorf("filePath is required")
	}

	editorCmd = strings.TrimSpace(editorCmd)
	if editorCmd == "" {
		if env, ok := os.LookupEnv("EDITOR"); ok && env != "" {
			editorCmd = env
		} else {
			editorCmd = kDefaultEditor
		}
	}

	parts := strings.Fields(editorCmd)
	if len(parts) == 0 {
		return fmt.Errorf("editor command is empty")
	}
	name := parts[0]
	args := append(parts[1:], filePath)

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run editor %q: %w", editorCmd, err)
	}
	return nil
}
