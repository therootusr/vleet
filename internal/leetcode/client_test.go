package leetcode

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHttpClient_FetchQuestion_Sanity(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %q, want %q", r.Method, http.MethodPost)
		}
		if r.URL.Path != kGraphQLPath {
			t.Fatalf("path = %q, want %q", r.URL.Path, kGraphQLPath)
		}
		if got := r.Header.Get(kHeaderContentType); got != kContentTypeApplicationJSON {
			t.Fatalf("%s = %q, want %q", kHeaderContentType, got, kContentTypeApplicationJSON)
		}
		if got := r.Header.Get(kHeaderAccept); got != kContentTypeApplicationJSON {
			t.Fatalf("%s = %q, want %q", kHeaderAccept, got, kContentTypeApplicationJSON)
		}

		var req struct {
			Query     string `json:"query"`
			Variables struct {
				TitleSlug string `json:"titleSlug"`
			} `json:"variables"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Variables.TitleSlug != "two-sum" {
			t.Fatalf("variables.titleSlug = %q, want %q", req.Variables.TitleSlug, "two-sum")
		}
		if !strings.Contains(req.Query, "questionData") {
			t.Fatalf("query does not contain %q", "questionData")
		}

		resp := map[string]any{
			"data": map[string]any{
				"question": map[string]any{
					"questionId":         "1",
					"questionFrontendId": "1",
					"title":              "Two Sum",
					"titleSlug":          "two-sum",
					"difficulty":         "Easy",
					"content":            "<p>desc</p>",
					"exampleTestcases":   "1\n2\n",
					"sampleTestCase":     "1\n2\n",
					"hints":              []string{"hint"},
					"topicTags": []map[string]any{
						{"name": "Array", "slug": "array"},
					},
					"codeSnippets": []map[string]any{
						{"lang": "C++", "langSlug": "cpp", "code": "class Solution {};"},
					},
				},
			},
		}

		w.Header().Set(kHeaderContentType, kContentTypeApplicationJSON)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(ts.Close)

	lc := NewHttpClient(HttpClientOptions{
		BaseURL: ts.URL,
		Http:    ts.Client(),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	got, err := lc.FetchQuestion(ctx, "two-sum")
	if err != nil {
		t.Fatalf("FetchQuestion() error = %v", err)
	}

	if got.Title != "Two Sum" {
		t.Fatalf("Title = %q, want %q", got.Title, "Two Sum")
	}
	if got.TitleSlug != "two-sum" {
		t.Fatalf("TitleSlug = %q, want %q", got.TitleSlug, "two-sum")
	}
	if got.QuestionID != "1" || got.FrontendID != "1" {
		t.Fatalf("ids = (qid=%q, frontend=%q), want (1, 1)", got.QuestionID, got.FrontendID)
	}
	if len(got.CodeSnippets) != 1 || got.CodeSnippets[0].LangSlug != "cpp" {
		t.Fatalf("codeSnippets = %+v, want one cpp snippet", got.CodeSnippets)
	}
	if len(got.TopicTags) != 1 || got.TopicTags[0].Slug != "array" {
		t.Fatalf("topicTags = %+v, want one array tag", got.TopicTags)
	}
}
