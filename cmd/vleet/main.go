package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/therootusr/go-leetcode"
	"vleet/internal/app"
	"vleet/internal/config"
	"vleet/internal/editor"
	"vleet/internal/errx"
	"vleet/internal/output"
	"vleet/internal/render"
	"vleet/internal/workspace"
)

const (
	kEnvVleetBaseURL        = "VLEET_BASE_URL"
	kEnvVleetConfigPath     = "VLEET_CONFIG_PATH"
	kDefaultLeetCodeBaseURL = "https://leetcode.com"
)

func main() {
	os.Exit(realMain(os.Args))
}

func validateArgs(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("missing command")
	}

	switch args[1] {
	case "solve":
	case "fetch":
	case "submit":
	case "config":
	case "help", "-h", "--help":
		break
	default:
		return fmt.Errorf("unknown command: %s", args[1])
	}

	return nil
}

func realMain(args []string) int {
	if err := validateArgs(args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		usage(os.Stderr)
		return 2
	}

	if args[1] == "help" || args[1] == "-h" || args[1] == "--help" {
		usage(os.Stdout)
		return 0
	}

	ctx := context.Background()

	cfgPath := strings.TrimSpace(os.Getenv(kEnvVleetConfigPath))
	if cfgPath == "" {
		var err error
		cfgPath, err = config.DefaultConfigPath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: resolve config path: %v\n", err)
			return 1
		}
	}

	cfgStore := config.NewFileStore(cfgPath)

	baseURL := os.Getenv(kEnvVleetBaseURL)
	if baseURL == "" {
		baseURL = kDefaultLeetCodeBaseURL
	}

	lc := leetcode.NewHttpClient(leetcode.HttpClientOptions{
		BaseURL:   baseURL,
		UserAgent: "vleet/0.1.0",
		Auth:      leetcode.Auth{},
	})
	ws := workspace.NewFSManager()
	rend := render.NewHTMLRenderer()
	ed := editor.NewProcessRunner()
	pr := output.NewStdPrinter(os.Stdout, os.Stderr, false)

	a := app.New(app.App{
		ConfigStore: cfgStore,
		LeetCode:    lc,
		Workspace:   ws,
		Renderer:    rend,
		Editor:      ed,
		Output:      pr,
	})

	cmd := args[1]
	var runErr error
	switch cmd {
	case "solve":
		runErr = runSolve(ctx, a, pr, args[2:])
	case "fetch":
		runErr = runFetch(ctx, a, pr, args[2:])
	case "submit":
		runErr = runSubmit(ctx, a, pr, args[2:])
	case "config":
		runErr = runConfig(ctx, cfgStore, pr, args[2:])
	case "help", "-h", "--help":
		usage(os.Stdout)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "error: unknown command %q\n\n", cmd)
		usage(os.Stderr)
		return 2
	}

	if runErr == nil {
		return 0
	}

	_ = pr.PrintError(ctx, runErr)
	if errors.Is(runErr, errx.ErrNotImplemented) {
		return 3
	}
	return 1
}

func runSolve(ctx context.Context, a *app.App, pr *output.StdPrinter, argv []string) error {
	fs := flag.NewFlagSet("solve", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var lang string
	var submit bool
	var asJSON bool
	fs.StringVar(&lang, "lang", "", "LeetCode language slug (e.g. cpp, python3)")
	fs.BoolVar(&submit, "submit", false, "submit immediately after editor exits")
	fs.BoolVar(&asJSON, "json", false, "emit JSON output")

	if err := fs.Parse(argv); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("solve: missing <problem-key> (titleSlug)")
	}
	pr.JSON = asJSON

	return a.Solve(ctx, app.SolveOptions{
		ProblemKey: fs.Arg(0),
		Lang:       lang,
		Submit:     submit,
	})
}

func runFetch(ctx context.Context, a *app.App, pr *output.StdPrinter, argv []string) error {
	fs := flag.NewFlagSet("fetch", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var lang string
	var asJSON bool
	fs.StringVar(&lang, "lang", "", "LeetCode language slug (e.g. cpp, python3)")
	fs.BoolVar(&asJSON, "json", false, "emit JSON output")

	if err := fs.Parse(argv); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("fetch: missing <problem-key> (titleSlug)")
	}
	pr.JSON = asJSON

	return a.Fetch(ctx, app.FetchOptions{
		ProblemKey: fs.Arg(0),
		Lang:       lang,
	})
}

func runSubmit(ctx context.Context, a *app.App, pr *output.StdPrinter, argv []string) error {
	fs := flag.NewFlagSet("submit", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var lang string
	var file string
	var asJSON bool
	fs.StringVar(&lang, "lang", "", "LeetCode language slug (e.g. cpp, python3)")
	fs.StringVar(&file, "file", "", "path to solution file (overrides default ./<problem-key>/solution.<ext>)")
	fs.BoolVar(&asJSON, "json", false, "emit JSON output")

	if err := fs.Parse(argv); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("submit: missing <problem-key> (titleSlug)")
	}
	pr.JSON = asJSON

	return a.Submit(ctx, app.SubmitOptions{
		ProblemKey: fs.Arg(0),
		Lang:       lang,
		File:       file,
	})
}

func runConfig(ctx context.Context, store *config.FileStore, pr *output.StdPrinter, argv []string) error {
	if len(argv) < 1 {
		return fmt.Errorf("config: missing subcommand (init|show)")
	}
	switch argv[0] {
	case "init":
		return runConfigInit(ctx, store, pr, argv[1:])
	case "show":
		return runConfigShow(ctx, store, pr, argv[1:])
	default:
		return fmt.Errorf("config: unknown subcommand %q (expected init|show)", argv[0])
	}
}

func runConfigInit(ctx context.Context, store *config.FileStore, pr *output.StdPrinter, argv []string) error {
	fs := flag.NewFlagSet("config init", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var editorCmd string
	var defaultLang string
	var force bool
	fs.StringVar(&editorCmd, "editor", "", "editor command (default: $EDITOR, else vim)")
	fs.StringVar(&defaultLang, "default-lang", "", "default LeetCode language slug (default: cpp)")
	fs.BoolVar(&force, "force", false, "overwrite existing config file")

	if err := fs.Parse(argv); err != nil {
		return err
	}

	if editorCmd == "" {
		if env, ok := os.LookupEnv("EDITOR"); ok && env != "" {
			editorCmd = env
		} else {
			editorCmd = "vim"
		}
	}
	if defaultLang == "" {
		defaultLang = "cpp"
	}

	if _, err := os.Stat(store.Path); err == nil && !force {
		return fmt.Errorf("config already exists at %s (use --force to overwrite)", store.Path)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat config %s: %w", store.Path, err)
	}

	cfg := config.Config{
		Editor:      editorCmd,
		DefaultLang: defaultLang,
		LeetCode: config.LeetCodeAuth{
			Session:   "",
			CSRFTOKEN: "",
		},
	}
	if err := store.Save(ctx, cfg); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(pr.Out, "wrote config: %s\n", store.Path)
	_, _ = fmt.Fprintln(pr.Out, "note: edit the file to set leetcode.session and leetcode.csrftoken")
	return nil
}

func runConfigShow(ctx context.Context, store *config.FileStore, pr *output.StdPrinter, argv []string) error {
	fs := flag.NewFlagSet("config show", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(argv); err != nil {
		return err
	}

	cfg, err := store.Load(ctx)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("config not found at %s (run: vleet config init)", store.Path)
		}
		return err
	}

	sessionStatus := "(not set)"
	if cfg.LeetCode.Session != "" {
		sessionStatus = "(set)"
	}
	csrfStatus := "(not set)"
	if cfg.LeetCode.CSRFTOKEN != "" {
		csrfStatus = "(set)"
	}

	_, _ = fmt.Fprintf(pr.Out, "path: %s\n", store.Path)
	_, _ = fmt.Fprintf(pr.Out, "editor: %s\n", cfg.Editor)
	_, _ = fmt.Fprintf(pr.Out, "default_lang: %s\n", cfg.DefaultLang)
	_, _ = fmt.Fprintf(pr.Out, "leetcode.session: %s\n", sessionStatus)
	_, _ = fmt.Fprintf(pr.Out, "leetcode.csrftoken: %s\n", csrfStatus)
	return nil
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "vleet - Vim + LeetCode in the terminal")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  vleet <command> [args]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  solve   <problem-key> --lang <lang> [--submit]")
	fmt.Fprintln(w, "  fetch   <problem-key> --lang <lang>")
	fmt.Fprintln(w, "  submit  <problem-key> --lang <lang> [--file <path>]")
	fmt.Fprintln(w, "  config  init|show")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Notes:")
	fmt.Fprintln(w, "  - problem-key is the LeetCode titleSlug in MVP (e.g. two-sum)")
	fmt.Fprintln(w, "  - Use --json on subcommands for JSON output")
}
