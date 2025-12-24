## Architecture: vleet

### Architecture at a glance

vleet is a single local CLI binary that orchestrates:

- **LeetCode client** (GraphQL fetch + REST submit/poll)
- **Workspace manager** (files on disk)
- **Editor runner** (launches Vim/$EDITOR)
- **Renderer** (HTML → comment header for `solution.<ext>`)

Implementation note: **vleet itself is intended to be written in Go** (great fit for a terminal-only, single-binary CLI). This is independent of the **default LeetCode solution language**, which is **C++** (`cpp`).

### Component diagram (MVP)

```text
┌─────────────────────────────────────────────────────────────┐
│                         vleet CLI                           │
│  (arg parsing, command routing, output formatting, logging) │
└───────────────┬───────────────────────────────┬─────────────┘
                │                               │
                ▼                               ▼
┌────────────────────────────┐        ┌─────────────────────────┐
│        Config Store        │        │     Workspace Manager   │
│  ~/.config or XDG paths    │        │  create/read workspace  │
│  (editor, lang, cookies)   │        │  solution.*             │
└───────────────┬────────────┘        └──────────────┬──────────┘
                │                                    │
                ▼                                    ▼
┌────────────────────────────┐        ┌──────────────────────────┐
│       LeetCode Client      │        │      Editor Runner       │
│  GraphQL: questionData     │        │  spawns vim/nvim/$EDITOR │
│  REST: submit + poll check │        │  blocks until exit       │
└───────────────┬────────────┘        └──────────────────────────┘
                │
                ▼
┌────────────────────────────┐
│           Renderer         │
│  HTML → comment header text│
│  metadata formatting       │
└────────────────────────────┘
```

### Command-level flow

#### `vleet solve <problem-key> --lang <lang> [--submit]`

```text
User
 │
 │ 1) vleet solve two-sum --lang cpp
 ▼
CLI
 │ 2) Load config (editor, session cookie, default language)
 │ 3) Fetch question via GraphQL (problem-key/titleSlug)
 │ 4) Choose language snippet
 │ 5) Create workspace dir in the current working directory: ./<problem-key>/
 │ 6) Write solution file (solution.cpp)
 │    - solution.cpp starts with the problem statement as a header comment
 │    - followed by the exact LeetCode starter snippet for the chosen language
 │ 7) Spawn editor (vim ./<problem-key>/solution.cpp)
 │ 8) After exit: optionally submit
 │ 9) Submit via REST -> submission_id
 │ 10) Poll check endpoint until done (or timeout)
 │ 11) Render result summary
 ▼
User sees verdict + runtime/memory/errors
```

#### `vleet submit <problem-key> --lang <lang> [--file <path>]`

- Infer workspace dir from `<problem-key>` (default: `./<problem-key>/`)
- Read `solution.<ext>` (default: `solution.<ext>` inside the workspace dir)
- Fetch question via GraphQL to obtain `question_id` (since we intentionally avoid per-workspace metadata files)
- Submit + poll

### Data and storage layout

#### Config (user-level)

Proposed default paths:

- Config: `~/.config/vleet/config.yaml` (or macOS equivalent)
- Cache: `~/.cache/vleet/` (problem fetch cache, optional)
- Workspaces: `./<problem-key>/` (relative to the current working directory)

Config contains:

- editor command (string)
- default language (string, default: `cpp`)
- auth secrets (session cookie)  ← MVP stores locally with `0600`

#### Workspace (per-problem)

Files:

- `solution.<ext>`: vleet-generated header comment (problem statement) + LeetCode starter snippet for chosen language

### Core modules (Go implementation)

Suggested package layout (can evolve):

- `cmd/vleet/`:
  - CLI entrypoint and command definitions
- `internal/config/`:
  - load/save config, permissions, env overrides
- `internal/leetcode/`:
  - HTTP client, cookie handling, GraphQL queries, submit/poll
- `internal/workspace/`:
  - create/read workspace dirs, resolve solution file paths
- `internal/render/`:
  - HTML → comment-header conversion + sanitization
- `internal/editor/`:
  - build editor command, spawn process, propagate stdio
- `internal/output/`:
  - human output + optional `--json` structs

### API contracts (internal)

#### LeetCode client

Key functions:

- `FetchQuestion(ctx, slug) -> Question`
- `Submit(ctx, slug, questionID, lang, code) -> submissionID`
- `PollSubmission(ctx, submissionID) -> SubmissionResult`

Where:

- `Question` contains:
  - ids (`questionId`, `frontendId`)
  - metadata (title, difficulty, tags)
  - `contentHTML`
  - `codeSnippets[]`
- `SubmissionResult` contains:
  - state, status, runtime, memory
  - compilation error / runtime error details

#### Workspace manager

- `CreateWorkspace(root, question, lang) -> Workspace`
- `LoadWorkspace(dir) -> Workspace`
- `ReadSolution(workspace) -> code`

### Concurrency / polling

Only polling needs concurrency/timeouts:

- Poll interval: start at 750ms–1s
- Backoff: exponential up to 3–5s
- Hard timeout: configurable (e.g. 2 minutes)
- Respect `context.Context` cancellation (Ctrl-C)

### Security posture

- Treat cookies as secrets:
  - ensure config file mode `0600`
  - redact in logs
- Avoid storing more than needed:
  - do not cache responses that include account details
- Make HTTP requests look like a normal browser:
  - set `User-Agent`
  - set `Referer` for submit/check
  - do not hammer endpoints (rate limit polling)

### Extensibility points

Designed to be easy to evolve:

- **TUI mode**: replace/augment `internal/output` with Bubble Tea UI.
- **Search/select problems**: add `problemsetQuestionList` GraphQL queries and local caching.
- **Editor integrations**: optional vim plugin that calls `vleet submit` and jumps to the function stub below the header comments.
- **Keychain auth**: replace config secret storage with OS keyring.

### Operational considerations

- Cross-platform: macOS/Linux first; Windows later (editor + paths need care).
- No background daemons required.
- Single-binary distribution (Go) is ideal for your “terminal-only” constraint.
