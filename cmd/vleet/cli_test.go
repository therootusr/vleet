package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"vleet/internal/config"
)

func TestCLI_ConfigShow_RedactsSecrets(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission semantics differ on Windows")
	}

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	// Create config with secrets.
	store := config.NewFileStore(cfgPath)
	if err := store.Save(context.Background(), config.Config{
		Editor:      "vim",
		DefaultLang: "cpp",
		LeetCode: config.LeetCodeAuth{
			Session:   "sess-secret",
			CSRFTOKEN: "csrf-secret",
		},
	}); err != nil {
		t.Fatalf("save config: %v", err)
	}

	t.Setenv(kEnvVleetConfigPath, cfgPath)

	code, stdout, stderr := runRealMainCaptured(t, dir, []string{"vleet", "config", "show"})
	if code != 0 {
		t.Fatalf("exit=%d\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}

	if strings.Contains(stdout, "sess-secret") || strings.Contains(stderr, "sess-secret") {
		t.Fatalf("session secret leaked in output")
	}
	if strings.Contains(stdout, "csrf-secret") || strings.Contains(stderr, "csrf-secret") {
		t.Fatalf("csrf secret leaked in output")
	}

	// Check secrets are redacted as expected.
	if !strings.Contains(stdout, "leetcode.session: (set)") {
		t.Fatalf("expected session status to be shown as set; stdout:\n%s", stdout)
	}
	if !strings.Contains(stdout, "leetcode.csrftoken: (set)") {
		t.Fatalf("expected csrftoken status to be shown as set; stdout:\n%s", stdout)
	}
}

func TestCLI_Fetch_Smoke_CreatesSolutionFile(t *testing.T) {
	dir := t.TempDir()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/graphql" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var req struct {
			Query     string `json:"query"`
			Variables struct {
				TitleSlug string `json:"titleSlug"`
			} `json:"variables"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"question": map[string]any{
					"questionId":         "1",
					"questionFrontendId": "1",
					"title":              "Two Sum",
					"titleSlug":          "two-sum",
					"difficulty":         "Easy",
					"content":            "<p>desc</p>",
					"topicTags": []map[string]any{
						{"name": "Array", "slug": "array"},
					},
					"codeSnippets": []map[string]any{
						{"lang": "C++", "langSlug": "cpp", "code": "class Solution {};"},
					},
				},
			},
		})
	}))
	t.Cleanup(ts.Close)

	t.Setenv(kEnvVleetBaseURL, ts.URL)
	t.Setenv(kEnvVleetConfigPath, filepath.Join(dir, "config.yaml"))

	code, stdout, stderr := runRealMainCaptured(t, dir, []string{"vleet", "fetch", "two-sum", "--lang", "cpp"})
	if code != 0 {
		t.Fatalf("exit=%d\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}

	solutionPath := filepath.Join(dir, "two-sum", "solution.cpp")
	b, err := os.ReadFile(solutionPath)
	if err != nil {
		t.Fatalf("read solution: %v", err)
	}
	s := string(b)

	// Sanity: contains header text and snippet.
	if !strings.Contains(s, "Two Sum") {
		t.Fatalf("expected solution to contain title; got:\n%s", s)
	}
	if !strings.Contains(s, "class Solution") {
		t.Fatalf("expected solution to contain starter snippet; got:\n%s", s)
	}
}

func TestCLI_Submit_Smoke_PrintsVerdict_AndDoesNotLeakSecrets(t *testing.T) {
	dir := t.TempDir()

	cfgPath := filepath.Join(dir, "config.yaml")
	store := config.NewFileStore(cfgPath)
	if err := store.Save(context.Background(), config.Config{
		Editor:      "true",
		DefaultLang: "cpp",
		LeetCode: config.LeetCodeAuth{
			Session:   "sess-secret",
			CSRFTOKEN: "csrf-secret",
		},
	}); err != nil {
		t.Fatalf("save config: %v", err)
	}

	// Create an existing workspace + solution file.
	wsDir := filepath.Join(dir, "two-sum")
	if err := os.MkdirAll(wsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wsDir, "solution.cpp"), []byte("CODE\n"), 0o644); err != nil {
		t.Fatalf("write solution: %v", err)
	}

	submitOK := true
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/graphql":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"question": map[string]any{
						"questionId":         "1",
						"questionFrontendId": "1",
						"title":              "Two Sum",
						"titleSlug":          "two-sum",
						"difficulty":         "Easy",
						"codeSnippets":       []map[string]any{},
					},
				},
			})
			return
		case "/problems/two-sum/submit/":
			if !submitOK {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"forbidden"}`))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"submission_id": 123})
			return
		case "/submissions/detail/123/check/":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"state":      "SUCCESS",
				"status_msg": "Accepted",
				"runtime":    "1 ms",
				"memory":     "2 MB",
			})
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	t.Cleanup(ts.Close)

	t.Setenv(kEnvVleetBaseURL, ts.URL)
	t.Setenv(kEnvVleetConfigPath, cfgPath)

	code, stdout, stderr := runRealMainCaptured(t, dir, []string{"vleet", "submit", "two-sum", "--lang", "cpp"})
	if code != 0 {
		t.Fatalf("exit=%d\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "Verdict: Accepted") {
		t.Fatalf("expected verdict; stdout:\n%s", stdout)
	}

	// Ensure secrets do not appear in output.
	if strings.Contains(stdout, "sess-secret") || strings.Contains(stderr, "sess-secret") {
		t.Fatalf("session secret leaked in output")
	}
	if strings.Contains(stdout, "csrf-secret") || strings.Contains(stderr, "csrf-secret") {
		t.Fatalf("csrf secret leaked in output")
	}
}

func TestCLI_Submit_Error_DoesNotLeakSecrets(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission semantics differ on Windows")
	}

	dir := t.TempDir()

	cfgPath := filepath.Join(dir, "config.yaml")
	store := config.NewFileStore(cfgPath)
	if err := store.Save(context.Background(), config.Config{
		Editor:      "true",
		DefaultLang: "cpp",
		LeetCode: config.LeetCodeAuth{
			Session:   "sess-secret",
			CSRFTOKEN: "csrf-secret",
		},
	}); err != nil {
		t.Fatalf("save config: %v", err)
	}

	wsDir := filepath.Join(dir, "two-sum")
	if err := os.MkdirAll(wsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wsDir, "solution.cpp"), []byte("CODE\n"), 0o644); err != nil {
		t.Fatalf("write solution: %v", err)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/graphql":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"question": map[string]any{
						"questionId":         "1",
						"questionFrontendId": "1",
						"titleSlug":          "two-sum",
					},
				},
			})
			return
		case "/problems/two-sum/submit/":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":"forbidden"}`))
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	t.Cleanup(ts.Close)

	t.Setenv(kEnvVleetBaseURL, ts.URL)
	t.Setenv(kEnvVleetConfigPath, cfgPath)

	code, stdout, stderr := runRealMainCaptured(t, dir, []string{"vleet", "submit", "two-sum", "--lang", "cpp"})
	if code != 1 {
		t.Fatalf("exit=%d (want 1)\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	if strings.Contains(stdout, "sess-secret") || strings.Contains(stderr, "sess-secret") {
		t.Fatalf("session secret leaked in output")
	}
	if strings.Contains(stdout, "csrf-secret") || strings.Contains(stderr, "csrf-secret") {
		t.Fatalf("csrf secret leaked in output")
	}
}

func runRealMainCaptured(t *testing.T, dir string, args []string) (code int, stdout string, stderr string) {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if dir != "" {
		if err := os.Chdir(dir); err != nil {
			t.Fatalf("chdir: %v", err)
		}
		defer func() { _ = os.Chdir(wd) }()
	}

	oldStdout := os.Stdout
	oldStderr := os.Stderr
	defer func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	}()

	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stdout: %v", err)
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		_ = rOut.Close()
		_ = wOut.Close()
		t.Fatalf("pipe stderr: %v", err)
	}

	os.Stdout = wOut
	os.Stderr = wErr

	outCh := make(chan []byte, 1)
	errCh := make(chan []byte, 1)

	go func() {
		b, _ := io.ReadAll(rOut)
		outCh <- b
	}()
	go func() {
		b, _ := io.ReadAll(rErr)
		errCh <- b
	}()

	code = realMain(args)

	_ = wOut.Close()
	_ = wErr.Close()

	stdout = string(<-outCh)
	stderr = string(<-errCh)

	_ = rOut.Close()
	_ = rErr.Close()

	return code, stdout, stderr
}
