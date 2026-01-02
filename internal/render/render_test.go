package render

import (
	"context"
	"strings"
	"testing"

	"vleet/internal/leetcode"
)

func TestHTMLRenderer_RenderHeader_Sanity_CPP(t *testing.T) {
	t.Parallel()

	r := NewHTMLRenderer()

	q := leetcode.Question{
		Title:      "Two Sum",
		Difficulty: "Easy",
		TitleSlug:  "two-sum",
		TopicTags: []leetcode.TopicTag{
			{Name: "Array"},
			{Name: "Hash Table"},
		},
		ContentHTML: "<p>Given <code>nums</code>&nbsp;and <em>target</em> &amp; return indices.</p>" +
			"<ul><li>One</li><li>Two</li></ul>" +
			"<pre><strong>Input:</strong> x\n<strong>Output:</strong> y\n</pre>" +
			"<p>Constraints: &lt;int&gt;</p>",
		Hints: []string{
			"<p>Use a hash map.</p>",
		},
	}

	out, err := r.RenderHeader(context.Background(), "cpp", q)
	if err != nil {
		t.Fatalf("RenderHeader() error = %v", err)
	}

	// Basic prefixing.
	if !strings.HasPrefix(out, "// ") {
		t.Fatalf("expected cpp header to start with %q, got: %q", "// ", out[:min(10, len(out))])
	}

	// Metadata.
	assertContains(t, out, "// Two Sum (Easy)")
	assertContains(t, out, "// URL: https://leetcode.com/problems/two-sum/")
	assertContains(t, out, "// Tags: Array, Hash Table")

	// HTML -> plain text conversion sanity.
	assertNotContains(t, out, "<p>")
	assertNotContains(t, out, "&nbsp;")
	assertContains(t, out, "// Given nums and target & return indices.")
	assertContains(t, out, "// - One")
	assertContains(t, out, "// - Two")
	assertContains(t, out, "// Input:")
	assertContains(t, out, "// Output:")
	// Ensure entities are unescaped after tag stripping (angle brackets should remain).
	assertContains(t, out, "// Constraints: <int>")

	// Hints block.
	assertContains(t, out, "// Hints:")
	assertContains(t, out, "// - Use a hash map.")
}

func TestHTMLRenderer_RenderHeader_Sanity_Python3Prefix(t *testing.T) {
	t.Parallel()

	r := NewHTMLRenderer()
	q := leetcode.Question{Title: "T", ContentHTML: "<p>hi</p>"}

	out, err := r.RenderHeader(context.Background(), "python3", q)
	if err != nil {
		t.Fatalf("RenderHeader() error = %v", err)
	}
	if !strings.HasPrefix(out, "# ") {
		t.Fatalf("expected python3 header to start with %q, got: %q", "# ", out[:min(10, len(out))])
	}
	assertContains(t, out, "# T")
	assertContains(t, out, "# hi")
}

func assertContains(t *testing.T, s, sub string) {
	t.Helper()
	if !strings.Contains(s, sub) {
		t.Fatalf("expected output to contain %q\n--- output ---\n%s\n--- end ---", sub, s)
	}
}

func assertNotContains(t *testing.T, s, sub string) {
	t.Helper()
	if strings.Contains(s, sub) {
		t.Fatalf("expected output to NOT contain %q\n--- output ---\n%s\n--- end ---", sub, s)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
