package leetcode

// Question mirrors the key fields vleet needs from the LeetCode GraphQL questionData query.
// See docs/design.md ("Fields we need") and docs/architecture.md ("Question contains").
type Question struct {
	// QuestionID is LeetCode's internal numeric ID (often returned as a string in GraphQL).
	QuestionID string

	// FrontendID is the user-facing problem number (e.g. "1" for Two Sum).
	FrontendID string

	Title      string
	TitleSlug  string
	Difficulty string

	// ContentHTML is the HTML body returned by LeetCode (field name: content).
	ContentHTML string

	ExampleTestcases string
	SampleTestCase   string

	Hints []string

	TopicTags    []TopicTag
	CodeSnippets []CodeSnippet
}

type TopicTag struct {
	Name string
	Slug string
}

// CodeSnippet is the per-language starter snippet returned by LeetCode.
// In GraphQL, the usual fields are: lang, langSlug, code.
type CodeSnippet struct {
	// Lang is a human-friendly name (e.g. "C++", "Python3") as returned by LeetCode.
	Lang string

	// LangSlug is the slug used in submissions (e.g. "cpp", "python3").
	LangSlug string

	// Code is the starter snippet (verbatim).
	Code string
}
