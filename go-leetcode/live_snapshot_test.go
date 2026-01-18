//go:build live

// Live LeetCode snapshot (golden) tests.
//
// These tests contact the real LeetCode server and compare normalized responses
// against committed snapshots under:
//
//	testdata/leetcode/*.json
//
// They are excluded from the default test run. To run them explicitly:
//
//	go test -tags=live ./... -run TestLiveSnapshot_QuestionData -count=1
//
// To (re)generate snapshots from the current live output:
//
//	GO_LEETCODE_UPDATE_SNAPSHOTS=1 go test -tags=live ./... -run TestLiveSnapshot_QuestionData -count=1
//
// Optional:
//   - Override the live base URL (useful for regional endpoints):
//     GO_LEETCODE_LIVE_BASE_URL=https://leetcode.com ...
package leetcode

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

const (
	kEnvUpdateLiveSnapshots = "GO_LEETCODE_UPDATE_SNAPSHOTS"
	kEnvLiveBaseURL         = "GO_LEETCODE_LIVE_BASE_URL"

	kLiveSnapshotDir = "testdata/leetcode"

	kSnapshotLangCPP        = "cpp"
	kSnapshotLangPython3    = "python3"
	kSnapshotLangGolang     = "golang"
	kSnapshotLangJavaScript = "javascript"
	kSnapshotLangTypeScript = "typescript"
)

type liveQuestionSnapshot struct {
	TitleSlug   string `json:"titleSlug"`
	QuestionID  string `json:"questionId"`
	FrontendID  string `json:"questionFrontendId"`
	Title       string `json:"title"`
	Difficulty  string `json:"difficulty"`
	ContentHTML string `json:"content"`

	ExampleTestcases string `json:"exampleTestcases,omitempty"`
	SampleTestCase   string `json:"sampleTestCase,omitempty"`

	Hints []string `json:"hints,omitempty"`

	TopicTags    []TopicTag    `json:"topicTags,omitempty"`
	CodeSnippets []CodeSnippet `json:"codeSnippets,omitempty"`
}

func TestLiveSnapshot_QuestionData(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	baseURL := strings.TrimSpace(os.Getenv(kEnvLiveBaseURL))
	if baseURL == "" {
		baseURL = kDefaultBaseURL
	}

	httpClient := &http.Client{
		Timeout: 20 * time.Second,
	}
	lc := NewHttpClient(HttpClientOptions{
		BaseURL:   baseURL,
		UserAgent: "Mozilla/5.0 (go-leetcode live snapshot test)",
		Http:      httpClient,
	})

	// Keep the set small and stable; add more slugs as coverage grows.
	slugs := []string{
		"two-sum",
		"valid-parentheses",
	}

	update := strings.TrimSpace(os.Getenv(kEnvUpdateLiveSnapshots)) == "1"

	for _, slug := range slugs {
		t.Run(slug, func(t *testing.T) {
			q, err := lc.FetchQuestion(ctx, slug)
			if err != nil {
				t.Fatalf("FetchQuestion(%q) error = %v", slug, err)
			}

			got := buildLiveSnapshot(q)
			path := filepath.Join(kLiveSnapshotDir, slug+".json")

			if update {
				if err := writeSnapshot(path, got); err != nil {
					t.Fatalf("write snapshot %s: %v", path, err)
				}
				t.Logf("updated snapshot: %s", path)
				return
			}

			want, err := readSnapshot(path)
			if err != nil {
				t.Fatalf("read snapshot %s: %v", path, err)
			}

			wantJSON, _ := json.MarshalIndent(want, "", "  ")
			gotJSON, _ := json.MarshalIndent(got, "", "  ")
			if string(wantJSON) != string(gotJSON) {
				t.Fatalf(
					"snapshot mismatch for %s\n\nTo update snapshots:\n  %s=1 go test -tags=live ./... -run TestLiveSnapshot_QuestionData -count=1\n",
					slug,
					kEnvUpdateLiveSnapshots,
				)
			}
		})
	}
}

func buildLiveSnapshot(q Question) liveQuestionSnapshot {
	s := liveQuestionSnapshot{
		TitleSlug:   strings.TrimSpace(q.TitleSlug),
		QuestionID:  strings.TrimSpace(q.QuestionID),
		FrontendID:  strings.TrimSpace(q.FrontendID),
		Title:       strings.TrimSpace(q.Title),
		Difficulty:  strings.TrimSpace(q.Difficulty),
		ContentHTML: q.ContentHTML,

		ExampleTestcases: q.ExampleTestcases,
		SampleTestCase:   q.SampleTestCase,

		Hints: q.Hints,
	}

	// Normalize tags (order can drift).
	s.TopicTags = append([]TopicTag(nil), q.TopicTags...)
	sort.Slice(s.TopicTags, func(i, j int) bool {
		if s.TopicTags[i].Slug == s.TopicTags[j].Slug {
			return s.TopicTags[i].Name < s.TopicTags[j].Name
		}
		return s.TopicTags[i].Slug < s.TopicTags[j].Slug
	})

	// Filter and normalize code snippets to languages we intentionally snapshot.
	for _, cs := range q.CodeSnippets {
		if !isSnapshotLang(strings.TrimSpace(cs.LangSlug)) {
			continue
		}
		s.CodeSnippets = append(s.CodeSnippets, cs)
	}
	sort.Slice(s.CodeSnippets, func(i, j int) bool {
		if s.CodeSnippets[i].LangSlug == s.CodeSnippets[j].LangSlug {
			return s.CodeSnippets[i].Lang < s.CodeSnippets[j].Lang
		}
		return s.CodeSnippets[i].LangSlug < s.CodeSnippets[j].LangSlug
	})

	return s
}

func isSnapshotLang(langSlug string) bool {
	switch langSlug {
	case kSnapshotLangCPP,
		kSnapshotLangPython3,
		kSnapshotLangGolang,
		kSnapshotLangJavaScript,
		kSnapshotLangTypeScript:
		return true
	default:
		return false
	}
}

func readSnapshot(path string) (liveQuestionSnapshot, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return liveQuestionSnapshot{}, err
	}
	var s liveQuestionSnapshot
	if err := json.Unmarshal(b, &s); err != nil {
		return liveQuestionSnapshot{}, err
	}
	return s, nil
}

func writeSnapshot(path string, s liveQuestionSnapshot) error {
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')

	// Write atomically to reduce partial snapshot corruption.
	tmp := fmt.Sprintf("%s.tmp", path)
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
