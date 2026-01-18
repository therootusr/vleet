package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/therootusr/go-leetcode"
	"vleet/internal/config"
	"vleet/internal/output"
	"vleet/internal/workspace"
)

func TestApp_Submit_Sanity_SubmitAndPoll(t *testing.T) {
	t.Parallel()

	var baseURL string
	submitCalled := false
	checkCalled := false
	graphqlCalled := false

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/graphql":
			graphqlCalled = true
			var req struct {
				Query     string `json:"query"`
				Variables struct {
					TitleSlug string `json:"titleSlug"`
				} `json:"variables"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("decode graphql request: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if req.Variables.TitleSlug != "two-sum" {
				t.Errorf("graphql titleSlug = %q, want %q", req.Variables.TitleSlug, "two-sum")
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"question": map[string]any{
						"questionId":         "1",
						"questionFrontendId": "1",
						"title":              "Two Sum",
						"titleSlug":          "two-sum",
						"difficulty":         "Easy",
					},
				},
			})
			return

		case "/problems/two-sum/submit/":
			submitCalled = true
			if r.Method != http.MethodPost {
				t.Errorf("submit method = %q, want %q", r.Method, http.MethodPost)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if got := r.Header.Get("Referer"); got != baseURL+"/problems/two-sum/" {
				t.Errorf("submit Referer = %q, want %q", got, baseURL+"/problems/two-sum/")
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if got := r.Header.Get("Origin"); got != baseURL {
				t.Errorf("submit Origin = %q, want %q", got, baseURL)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if got := r.Header.Get("x-csrftoken"); got != "csrf" {
				t.Errorf("submit x-csrftoken = %q, want %q", got, "csrf")
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if c, err := r.Cookie("LEETCODE_SESSION"); err != nil || c.Value != "sess" {
				t.Errorf("submit cookie LEETCODE_SESSION missing/invalid: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if c, err := r.Cookie("csrftoken"); err != nil || c.Value != "csrf" {
				t.Errorf("submit cookie csrftoken missing/invalid: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			var body struct {
				Lang       string `json:"lang"`
				QuestionID string `json:"question_id"`
				TypedCode  string `json:"typed_code"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode submit body: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if body.Lang != "cpp" {
				t.Errorf("submit lang = %q, want %q", body.Lang, "cpp")
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if body.QuestionID != "1" {
				t.Errorf("submit question_id = %q, want %q", body.QuestionID, "1")
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if body.TypedCode != "CODE\n" {
				t.Errorf("submit typed_code = %q, want %q", body.TypedCode, "CODE\\n")
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"submission_id": 123})
			return

		case "/submissions/detail/123/check/":
			checkCalled = true
			if r.Method != http.MethodGet {
				t.Errorf("check method = %q, want %q", r.Method, http.MethodGet)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if got := r.Header.Get("Referer"); got != baseURL+"/submissions/detail/123/" {
				t.Errorf("check Referer = %q, want %q", got, baseURL+"/submissions/detail/123/")
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if got := r.Header.Get("x-csrftoken"); got != "csrf" {
				t.Errorf("check x-csrftoken = %q, want %q", got, "csrf")
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if c, err := r.Cookie("LEETCODE_SESSION"); err != nil || c.Value != "sess" {
				t.Errorf("check cookie LEETCODE_SESSION missing/invalid: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"state":         "SUCCESS",
				"status_msg":    "Accepted",
				"runtime":       "1 ms",
				"memory":        "2 MB",
				"compile_error": "",
				"runtime_error": "",
			})
			return
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	t.Cleanup(ts.Close)
	baseURL = ts.URL

	lc := leetcode.NewHttpClient(leetcode.HttpClientOptions{
		BaseURL:   ts.URL,
		UserAgent: "ua",
		Http:      ts.Client(),
	})

	wm := &fakeWorkspaceManager{
		ws: workspace.Workspace{
			Dir:          "/tmp/two-sum",
			ProblemKey:   "two-sum",
			Lang:         "cpp",
			SolutionPath: "/tmp/two-sum/solution.cpp",
		},
		readSolution: "CODE\n",
	}

	cs := &fakeConfigStore{
		cfg: config.Config{
			DefaultLang: "cpp",
			LeetCode: config.LeetCodeAuth{
				Session:   "sess",
				CSRFTOKEN: "csrf",
			},
		},
	}

	var out bytes.Buffer
	var errBuf bytes.Buffer
	pr := output.NewStdPrinter(&out, &errBuf, false)

	a := New(App{
		ConfigStore: cs,
		LeetCode:    lc,
		Workspace:   wm,
		Output:      pr,
	})

	if err := a.Submit(context.Background(), SubmitOptions{ProblemKey: "two-sum", Lang: "cpp"}); err != nil {
		t.Fatalf("Submit() error = %v", err)
	}

	if !graphqlCalled {
		t.Fatalf("expected graphql to be called")
	}
	if !submitCalled {
		t.Fatalf("expected submit to be called")
	}
	if !checkCalled {
		t.Fatalf("expected submission check to be called")
	}
	if !wm.loadCalled || !wm.readCalled {
		t.Fatalf("expected workspace LoadWorkspace + ReadSolution to be called")
	}

	s := out.String()
	if !bytes.Contains([]byte(s), []byte("Verdict: Accepted")) {
		t.Fatalf("expected output to contain verdict, got:\n%s", s)
	}
	if !bytes.Contains([]byte(s), []byte("Runtime: 1 ms")) {
		t.Fatalf("expected output to contain runtime, got:\n%s", s)
	}
	if !bytes.Contains([]byte(s), []byte("Memory: 2 MB")) {
		t.Fatalf("expected output to contain memory, got:\n%s", s)
	}
}
