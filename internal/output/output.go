package output

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"vleet/internal/leetcode"
)

// Printer renders user-facing output (human and/or JSON).
// See docs/architecture.md "internal/output".
type Printer interface {
	PrintQuestion(ctx context.Context, q leetcode.Question) error
	PrintSubmissionResult(ctx context.Context, r leetcode.SubmissionResult) error
	PrintError(ctx context.Context, err error) error
}

// StdPrinter is a simple stdout/stderr printer.
// This is intentionally minimal; richer formatting comes later.
type StdPrinter struct {
	Out  io.Writer
	Err  io.Writer
	JSON bool
}

func NewStdPrinter(out io.Writer, err io.Writer, asJSON bool) *StdPrinter {
	return &StdPrinter{Out: out, Err: err, JSON: asJSON}
}

func (p *StdPrinter) PrintQuestion(ctx context.Context, q leetcode.Question) error {
	if p.JSON {
		return json.NewEncoder(p.Out).Encode(q)
	}
	_, err := fmt.Fprintf(p.Out, "%s (%s)\n", q.Title, q.Difficulty)
	return err
}

func (p *StdPrinter) PrintSubmissionResult(ctx context.Context, r leetcode.SubmissionResult) error {
	if p.JSON {
		return json.NewEncoder(p.Out).Encode(r)
	}
	_, err := fmt.Fprintf(p.Out, "%s\n", r.Status)
	return err
}

func (p *StdPrinter) PrintError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	_, werr := fmt.Fprintf(p.Err, "error: %v\n", err)
	return werr
}
