package app

import (
	"context"
	"errors"
	"os"
	"testing"

	"vleet/internal/config"
	"vleet/internal/leetcode"
	"vleet/internal/workspace"
)

type fakeConfigStore struct {
	cfg config.Config
	err error
}

func (s *fakeConfigStore) Load(ctx context.Context) (config.Config, error) { return s.cfg, s.err }
func (s *fakeConfigStore) Save(ctx context.Context, cfg config.Config) error {
	return errors.New("not needed in tests")
}

type fakeLeetCodeClient struct {
	gotSlug string
	q       leetcode.Question
	err     error
}

func (c *fakeLeetCodeClient) FetchQuestion(ctx context.Context, titleSlug string) (leetcode.Question, error) {
	c.gotSlug = titleSlug
	return c.q, c.err
}

func (c *fakeLeetCodeClient) Submit(ctx context.Context, req leetcode.SubmitRequest) (leetcode.SubmissionID, error) {
	return 0, errors.New("not needed in tests")
}

func (c *fakeLeetCodeClient) PollSubmission(ctx context.Context, submissionID leetcode.SubmissionID, opts leetcode.PollOptions) (leetcode.SubmissionResult, error) {
	return leetcode.SubmissionResult{}, errors.New("not needed in tests")
}

type fakeRenderer struct {
	gotLang string
	gotSlug string
	header  string
	err     error
}

func (r *fakeRenderer) RenderHeader(ctx context.Context, lang string, q leetcode.Question) (string, error) {
	r.gotLang = lang
	r.gotSlug = q.TitleSlug
	return r.header, r.err
}

type fakeWorkspaceManager struct {
	ws        workspace.Workspace
	createErr error

	gotCreateRoot string
	gotCreateLang string
	gotCreateSlug string
	createCalled  bool

	wrotePath    string
	wroteContent string
	writeErr     error

	loadCalled bool
	loadErr    error
}

func (m *fakeWorkspaceManager) CreateWorkspace(ctx context.Context, root string, q leetcode.Question, lang string, opts workspace.CreateOptions) (workspace.Workspace, error) {
	m.createCalled = true
	m.gotCreateRoot = root
	m.gotCreateLang = lang
	m.gotCreateSlug = q.TitleSlug
	if m.createErr != nil {
		return workspace.Workspace{}, m.createErr
	}
	return m.ws, nil
}

func (m *fakeWorkspaceManager) LoadWorkspace(ctx context.Context, dir string, problemKey string, lang string, file string) (workspace.Workspace, error) {
	m.loadCalled = true
	if m.loadErr != nil {
		return workspace.Workspace{}, m.loadErr
	}
	return m.ws, nil
}

func (m *fakeWorkspaceManager) ReadSolution(ctx context.Context, ws workspace.Workspace) (string, error) {
	return "", errors.New("not needed in tests")
}

func (m *fakeWorkspaceManager) WriteSolution(ctx context.Context, ws workspace.Workspace, content string) error {
	m.wrotePath = ws.SolutionPath
	m.wroteContent = content
	return m.writeErr
}

type fakeEditor struct {
	gotEditorCmd string
	gotFilePath  string
	err          error
}

func (e *fakeEditor) OpenFile(ctx context.Context, editorCmd string, filePath string) error {
	e.gotEditorCmd = editorCmd
	e.gotFilePath = filePath
	return e.err
}

type fakeOutput struct {
	printQuestionCalled bool
	gotQuestionSlug     string
	err                 error
}

func (o *fakeOutput) PrintQuestion(ctx context.Context, q leetcode.Question) error {
	o.printQuestionCalled = true
	o.gotQuestionSlug = q.TitleSlug
	return o.err
}

func (o *fakeOutput) PrintSubmissionResult(ctx context.Context, r leetcode.SubmissionResult) error {
	return errors.New("not needed in tests")
}

func (o *fakeOutput) PrintError(ctx context.Context, err error) error { return nil }

func TestApp_Fetch_Sanity_WritesSolution(t *testing.T) {
	a := New(App{
		ConfigStore: &fakeConfigStore{cfg: config.Config{DefaultLang: "cpp"}},
		LeetCode: &fakeLeetCodeClient{
			q: leetcode.Question{
				TitleSlug:  "two-sum",
				Title:      "Two Sum",
				Difficulty: "Easy",
				CodeSnippets: []leetcode.CodeSnippet{
					{Lang: "C++", LangSlug: "cpp", Code: "CODE"},
				},
			},
		},
		Workspace: &fakeWorkspaceManager{
			ws: workspace.Workspace{
				Dir:          "/tmp/two-sum",
				ProblemKey:   "two-sum",
				Lang:         "cpp",
				SolutionPath: "/tmp/two-sum/solution.cpp",
			},
		},
		Renderer: &fakeRenderer{header: "HEADER"},
		Output:   &fakeOutput{},
	})

	if err := a.Fetch(context.Background(), FetchOptions{ProblemKey: "two-sum", Lang: "cpp"}); err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	lc := a.LeetCode.(*fakeLeetCodeClient)
	if lc.gotSlug != "two-sum" {
		t.Fatalf("FetchQuestion called with slug = %q, want %q", lc.gotSlug, "two-sum")
	}

	rm := a.Renderer.(*fakeRenderer)
	if rm.gotLang != "cpp" {
		t.Fatalf("RenderHeader lang = %q, want %q", rm.gotLang, "cpp")
	}
	if rm.gotSlug != "two-sum" {
		t.Fatalf("RenderHeader question slug = %q, want %q", rm.gotSlug, "two-sum")
	}

	wm := a.Workspace.(*fakeWorkspaceManager)
	if wm.gotCreateRoot != "." {
		t.Fatalf("CreateWorkspace root = %q, want %q", wm.gotCreateRoot, ".")
	}
	if wm.wrotePath != "/tmp/two-sum/solution.cpp" {
		t.Fatalf("WriteSolution path = %q, want %q", wm.wrotePath, "/tmp/two-sum/solution.cpp")
	}
	if want := "HEADER\nCODE\n"; wm.wroteContent != want {
		t.Fatalf("WriteSolution content = %q, want %q", wm.wroteContent, want)
	}

	out := a.Output.(*fakeOutput)
	if !out.printQuestionCalled {
		t.Fatalf("expected PrintQuestion to be called")
	}
	if out.gotQuestionSlug != "two-sum" {
		t.Fatalf("PrintQuestion slug = %q, want %q", out.gotQuestionSlug, "two-sum")
	}
}

func TestApp_Solve_Sanity_OpensEditor(t *testing.T) {
	ed := &fakeEditor{}
	cs := &fakeConfigStore{cfg: config.Config{Editor: "nvim", DefaultLang: "cpp"}}

	wm := &fakeWorkspaceManager{
		ws: workspace.Workspace{
			Dir:          "/tmp/two-sum",
			ProblemKey:   "two-sum",
			Lang:         "cpp",
			SolutionPath: "/tmp/two-sum/solution.cpp",
		},
	}

	a := New(App{
		ConfigStore: cs,
		LeetCode: &fakeLeetCodeClient{
			q: leetcode.Question{
				TitleSlug: "two-sum",
				CodeSnippets: []leetcode.CodeSnippet{
					{LangSlug: "cpp", Code: "CODE"},
				},
			},
		},
		Workspace: wm,
		Renderer:  &fakeRenderer{header: "HEADER"},
		Editor:    ed,
	})

	if err := a.Solve(context.Background(), SolveOptions{ProblemKey: "two-sum", Lang: "cpp"}); err != nil {
		t.Fatalf("Solve() error = %v", err)
	}

	if ed.gotEditorCmd != "nvim" {
		t.Fatalf("OpenFile editorCmd = %q, want %q", ed.gotEditorCmd, "nvim")
	}
	if ed.gotFilePath != "/tmp/two-sum/solution.cpp" {
		t.Fatalf("OpenFile filePath = %q, want %q", ed.gotFilePath, "/tmp/two-sum/solution.cpp")
	}
}

func TestApp_Fetch_WhenWorkspaceExists_DoesNotOverwrite(t *testing.T) {
	// This asserts the "no overwrite" control flow at the app layer: if the workspace
	// manager indicates existence via os.ErrExist, App should fall back to LoadWorkspace
	// and avoid writing.
	wm := &fakeWorkspaceManager{
		ws: workspace.Workspace{
			Dir:          "/tmp/two-sum",
			ProblemKey:   "two-sum",
			Lang:         "cpp",
			SolutionPath: "/tmp/two-sum/solution.cpp",
		},
	}
	wm.createErr = os.ErrExist
	wm.writeErr = errors.New("should not be called")
	wm.loadErr = nil

	a := New(App{
		ConfigStore: &fakeConfigStore{cfg: config.Config{DefaultLang: "cpp"}},
		LeetCode: &fakeLeetCodeClient{
			q: leetcode.Question{
				TitleSlug: "two-sum",
				CodeSnippets: []leetcode.CodeSnippet{
					{LangSlug: "cpp", Code: "CODE"},
				},
			},
		},
		Workspace: wm,
		Renderer:  &fakeRenderer{header: "HEADER"},
		Output:    &fakeOutput{},
	})

	if err := a.Fetch(context.Background(), FetchOptions{ProblemKey: "two-sum", Lang: "cpp"}); err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if !wm.createCalled {
		t.Fatalf("expected CreateWorkspace to be called")
	}
	if !wm.loadCalled {
		t.Fatalf("expected LoadWorkspace to be called when workspace exists")
	}
	if wm.wroteContent != "" {
		t.Fatalf("expected WriteSolution not to be called when workspace exists")
	}
}
