package render

import (
	"context"

	"vleet/internal/errx"
	"vleet/internal/leetcode"
)

// Renderer converts LeetCode content (HTML) into a comment-friendly header block.
// See docs/design.md "Rendering the problem statement".
type Renderer interface {
	// RenderHeader returns a header comment block appropriate for the selected language.
	// The returned string is intended to be placed at the top of solution.<ext>.
	RenderHeader(ctx context.Context, lang string, q leetcode.Question) (string, error)
}

// HTMLRenderer is a renderer for LeetCode HTML content.
// Skeleton only: HTML parsing/sanitization is not implemented yet.
type HTMLRenderer struct{}

func NewHTMLRenderer() *HTMLRenderer { return &HTMLRenderer{} }

func (r *HTMLRenderer) RenderHeader(ctx context.Context, lang string, q leetcode.Question) (string, error) {
	return "", errx.NotImplemented("render.HTMLRenderer.RenderHeader")
}
