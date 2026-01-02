package render

import (
	"context"
	"fmt"
	"html"
	"regexp"
	"strings"
	"unicode"

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
type HTMLRenderer struct{}

func NewHTMLRenderer() *HTMLRenderer { return &HTMLRenderer{} }

func (r *HTMLRenderer) RenderHeader(ctx context.Context, lang string, q leetcode.Question) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	prefix := commentPrefix(lang)

	var b strings.Builder

	title := strings.TrimSpace(q.Title)
	difficulty := strings.TrimSpace(q.Difficulty)
	switch {
	case title != "" && difficulty != "":
		b.WriteString(fmt.Sprintf("%s (%s)\n", title, difficulty))
	case title != "":
		b.WriteString(title + "\n")
	default:
		b.WriteString("LeetCode Problem\n")
	}

	slug := strings.TrimSpace(q.TitleSlug)
	if slug != "" {
		// Keep the URL stable and explicit; if we later support regions, this can become configurable.
		b.WriteString(fmt.Sprintf("URL: https://leetcode.com/problems/%s/\n", slug))
	}

	if tags := joinTags(q.TopicTags); tags != "" {
		b.WriteString("Tags: " + tags + "\n")
	}

	b.WriteString("\n")
	if stmt := htmlToPlainText(q.ContentHTML); stmt != "" {
		b.WriteString(stmt)
		b.WriteString("\n")
	}

	if hints := formatHints(q.Hints); hints != "" {
		b.WriteString("\n")
		b.WriteString("Hints:\n")
		b.WriteString(hints)
		b.WriteString("\n")
	}

	headerBody := strings.TrimSpace(b.String())
	return prefixLines(prefix, headerBody), nil
}

func commentPrefix(lang string) string {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "python3":
		return "# "
	default:
		return "// "
	}
}

func prefixLines(prefix string, body string) string {
	// Always return at least one comment line to clearly mark the header boundary.
	if strings.TrimSpace(body) == "" {
		return prefix + "\n"
	}

	lines := strings.Split(body, "\n")
	var b strings.Builder
	for _, line := range lines {
		b.WriteString(prefix)
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

func joinTags(tags []leetcode.TopicTag) string {
	if len(tags) == 0 {
		return ""
	}
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		name := strings.TrimSpace(t.Name)
		if name == "" {
			continue
		}
		out = append(out, name)
	}
	return strings.Join(out, ", ")
}

func formatHints(hints []string) string {
	var out []string
	for _, h := range hints {
		txt := strings.TrimSpace(htmlToPlainText(h))
		if txt == "" {
			continue
		}
		out = append(out, "- "+txt)
	}
	return strings.Join(out, "\n")
}

var (
	reBR          = regexp.MustCompile(`(?i)<br\s*/?>`)
	rePreOpen     = regexp.MustCompile(`(?i)<pre[^>]*>`)
	rePreClose    = regexp.MustCompile(`(?i)</pre>`)
	reLiOpen      = regexp.MustCompile(`(?i)<li[^>]*>`)
	reLiClose     = regexp.MustCompile(`(?i)</li>`)
	reBlockClose  = regexp.MustCompile(`(?i)</(p|div|section|h[1-6]|ul|ol|table|tr|blockquote)>`)
	reStripTags   = regexp.MustCompile(`(?s)<[^>]*>`)
	reManyNewline = regexp.MustCompile(`\n{3,}`)
)

// htmlToPlainText performs a minimal HTML → plain text conversion.
//
// This is intentionally simple for v1, but the function boundary makes it easy to swap
// in a more sophisticated renderer later (e.g. proper HTML parsing / Markdown formatting).
func htmlToPlainText(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	// Preserve basic structure by translating common tags to newlines/bullets.
	s = reBR.ReplaceAllString(s, "\n")
	s = rePreOpen.ReplaceAllString(s, "\n\n")
	s = rePreClose.ReplaceAllString(s, "\n\n")
	s = reLiOpen.ReplaceAllString(s, "\n- ")
	s = reLiClose.ReplaceAllString(s, "\n")
	s = reBlockClose.ReplaceAllString(s, "\n\n")

	// Strip remaining tags, then unescape entities. Unescaping must happen AFTER tag
	// stripping so code like "&lt;int&gt;" doesn't turn into "<int>" and get stripped.
	s = reStripTags.ReplaceAllString(s, "")
	s = html.UnescapeString(s)

	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = strings.ReplaceAll(s, "\u00a0", " ")

	// Trim trailing whitespace per line while keeping leading whitespace (helps code blocks).
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = strings.TrimRightFunc(lines[i], unicode.IsSpace)
	}
	s = strings.Join(lines, "\n")

	// Collapse excessive newlines for readability.
	s = reManyNewline.ReplaceAllString(s, "\n\n")

	return strings.TrimSpace(s)
}
