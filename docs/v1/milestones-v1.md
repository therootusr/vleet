## vleet v1 milestones + execution plan (offline-first)

This doc is the **source-of-truth execution plan** for delivering **vleet v1** from the existing Go skeleton in this repo.

v1 is “done” when a user can:
- fetch a LeetCode problem (by `titleSlug`)
- generate a workspace + solution file with a readable header and official starter snippet
- open the solution in an editor
- submit to LeetCode
- poll/check the final evaluation result and print it

The plan explicitly prefers **simple implementations** over sophisticated ones, while keeping logic **replaceable** (interfaces already exist in `internal/*`).

---

### Repo orientation (what exists today)

The codebase is currently a compile-ready skeleton. These files are the primary entrypoints for v1 implementation:

- **CLI entrypoint**: `cmd/vleet/main.go`
  - Parses commands: `solve`, `fetch`, `submit`, `config` (placeholder), `help`
  - Wires dependencies into the orchestrator
- **Orchestrator**: `internal/app/app.go`
  - `(*App).Fetch`, `Solve`, `Submit` exist but are not implemented
- **Config (YAML desired for v1)**: `internal/config/config.go`, `internal/config/paths.go`
  - `config.FileStore.Load/Save` are not implemented
- **LeetCode client**: `internal/leetcode/client.go`, `internal/leetcode/types.go`
  - Network methods are not implemented
  - Concrete implementation type is `leetcode.HttpClient` via `leetcode.NewHttpClient(...)`
- **Renderer**: `internal/render/render.go` (`render.Renderer`)
  - Implement as HTML → plain text → comment header block (simple v1)
- **Workspace manager**: `internal/workspace/workspace.go` (`workspace.Manager`)
  - Create workspace directory + resolve solution path + read/write solution file
- **Editor runner**: `internal/editor/editor.go` (`editor.Runner`)
  - Launch external editor process and block until exit
- **Output**: `internal/output/output.go` (`output.Printer`)
  - Minimal printer exists; improve for verdict display in v1
- **Not-implemented helper**: `internal/errx/errx.go`

---

### v1 CLI contract (behavior)

#### `vleet fetch <problem-key> --lang <lang> [--json]`
- `problem-key` is the LeetCode **titleSlug** (e.g. `two-sum`)
- Fetches question via GraphQL and writes:
  - workspace dir: `./<problem-key>/`
  - solution file: `./<problem-key>/solution.<ext>`
- Solution file contents:
  - **Header block**: plain text derived from LeetCode HTML, emitted as comments
  - **Starter snippet**: pasted **verbatim** from LeetCode `codeSnippets` for the chosen language
- v1 overwrite policy: **do not overwrite** an existing solution file (error with a message).

#### `vleet solve <problem-key> --lang <lang> [--submit] [--json]`
- Runs the same steps as `fetch` (creating solution file if needed)
- Opens the solution file in the configured editor
- If `--submit` is set, submits after the editor exits and prints final verdict

#### `vleet submit <problem-key> --lang <lang> [--file <path>] [--json]`
- Locates the solution file:
  - default: `./<problem-key>/solution.<ext>`
  - override: `--file <path>`
- Re-fetches question to obtain `question_id`
- Submits via REST, polls check endpoint until completion (or timeout)
- Prints verdict + runtime/memory + error details (compile/runtime)

---

### v1 config contract (YAML + local file secrets)

#### Storage location
- Use `internal/config.DefaultConfigPath()` (currently `~/.config/vleet/config.yaml` on macOS/Linux via Go’s `os.UserConfigDir`).

#### Permissions
- Config file must be stored with mode **0600**
- **Never print** cookie values to stdout/stderr (even with `--json`)

#### YAML schema (v1)

```yaml
editor: vim
default_lang: cpp
leetcode:
  session: "LEETCODE_SESSION_VALUE"
  csrftoken: "CSRFTOKEN_VALUE"
```

#### Precedence and defaults
- Flags override config
- Default editor: `$EDITOR` if set; else `vim`
- Default language: `cpp`

---

### v1 implementation milestones

Each milestone should end in a “working checkpoint” with a manual validation command.

#### M0 — Confirm decisions (no code)
- **Goal**: avoid churn.
- **Confirm**:
  - overwrite policy (v1: no overwrite)
  - supported languages list for v1 (recommend: `cpp`, `python3`, `golang`)
  - polling timeout defaults (recommend: 2m total, backoff to 5s max)
- **Exit criteria**: plan accepted.

#### M1 — Config store (YAML read/write + 0600)
- **Implement**
  - `internal/config.FileStore.Load`
  - `internal/config.FileStore.Save`
  - ensure parent dir exists
  - enforce file mode **0600**
- **Dependency choice**
  - Use a small YAML library (e.g. `gopkg.in/yaml.v3`).
- **CLI UX (v1 simple)**
  - Accept “user edits YAML manually” as sufficient.
- **Validate**
  - Missing config produces actionable error: where to create it
  - Loading config never prints secrets

#### M2 — LeetCode GraphQL fetch (`questionData`)
- **Implement**
  - `internal/leetcode.HttpClient.FetchQuestion`
  - POST `https://leetcode.com/graphql`
  - Query `questionData(titleSlug: $slug)` and map response into `leetcode.Question`:
    - `questionId`, `questionFrontendId`, `title`, `titleSlug`, `difficulty`
    - `content` (HTML)
    - `hints`, `topicTags { name slug }`
    - `codeSnippets { lang langSlug code }`
- **Hardening (simple v1)**
  - HTTP client timeout
  - non-200 errors readable
  - detect “HTML when expecting JSON” and explain likely blocking/captcha
- **Validate**
  - `FetchQuestion("two-sum")` returns populated fields

#### M3 — Renderer: HTML → plain text → comment header
- **Implement**
  - `internal/render.HTMLRenderer.RenderHeader`
- **v1 approach (simple, extendable)**
  - Convert HTML into plain text with a minimal transform:
    - preserve rough paragraph/list/code-block structure
    - strip tags and unescape entities
  - Emit as comment lines, chosen by language:
    - `python3`: `# `
    - otherwise (v1): `// `
  - Header should include:
    - title + difficulty
    - URL (`https://leetcode.com/problems/<slug>/`)
    - tags (comma-separated)
    - statement body (plain text)
- **Validate**
  - header is readable and doesn’t break compilation in `.cpp`

#### M4 — Workspace manager: create/load/read/write solution files
- **Implement**
  - `internal/workspace.FSManager.CreateWorkspace`
  - `LoadWorkspace`, `ReadSolution`, `WriteSolution`
- **Rules**
  - workspace dir: `./<problem-key>/`
  - default solution name: `solution.<ext>`
  - map `lang` slug → extension (v1):
    - `cpp`→`.cpp`, `python3`→`.py`, `golang`→`.go`, `javascript`→`.js`, `typescript`→`.ts`
  - content composition: `header + "\n\n" + starterSnippet + "\n"`
  - no overwrite by default
- **Validate**
  - `vleet fetch two-sum --lang cpp` creates `./two-sum/solution.cpp`

#### M5 — Offline slice complete: implement `fetch` and `solve` (no submit required)
- **Implement**
  - `internal/app.App.Fetch`
    - load config (for default lang/editor)
    - fetch question
    - select code snippet by `LangSlug == lang`
    - render header
    - create workspace + write solution
    - print path + summary via `output.Printer`
  - `internal/app.App.Solve`
    - run `Fetch` logic (or share an internal helper)
    - open editor via `editor.Runner.OpenFile`
    - if `--submit`, call `Submit` after editor exit
  - `internal/editor.ProcessRunner.OpenFile`
    - simplest: `strings.Fields(editorCmd)` + append filepath; `exec.CommandContext`; attach stdio
- **Validate**
  - `vleet solve two-sum --lang cpp` opens the file in editor

#### M6 — Submit + poll: complete v1 end-to-end
- **Implement**
  - `internal/app.App.Submit`
    - load config; require auth fields present
    - resolve/read solution file
    - re-fetch question for `QuestionID`
    - submit + poll + print result
  - `internal/leetcode.HttpClient.Submit`
    - POST `https://leetcode.com/problems/<slug>/submit/`
    - JSON: `{ lang, question_id, typed_code }`
    - set headers:
      - `Content-Type: application/json`
      - `Referer: https://leetcode.com/problems/<slug>/`
      - `User-Agent`
      - `x-csrftoken: <csrftoken>` (if present)
    - set cookies: `LEETCODE_SESSION`, `csrftoken`
    - parse response for `submission_id`
  - `internal/leetcode.HttpClient.PollSubmission`
    - GET `https://leetcode.com/submissions/detail/<id>/check/`
    - poll with exponential backoff:
      - start 1s, cap 5s
      - timeout 2m (configurable later)
    - stop when complete and map to `leetcode.SubmissionResult`
  - `internal/output.StdPrinter.PrintSubmissionResult`
    - human output includes: verdict, runtime/memory, and compile/runtime error blocks
    - `--json` continues to work
- **Validate**
  - With valid YAML auth:
    - `vleet submit two-sum --lang cpp` returns final verdict
    - `vleet solve two-sum --lang cpp --submit` completes full flow

---

### Security + privacy constraints (v1)
- Treat cookies as secrets:
  - never print them
  - never log HTTP request/response bodies that include them
  - store config with mode 0600
- Do not disable TLS verification.
- Poll conservatively; avoid hammering endpoints.

---

### Manual QA checklist (v1)
- **Fetch**
  - `vleet fetch two-sum --lang cpp` creates file; re-running errors instead of overwriting
- **Solve**
  - `vleet solve two-sum --lang cpp` opens editor at correct path
- **Submit**
  - With correct auth: returns `Accepted` for a known-correct solution
  - With bad solution: returns `Wrong Answer`
  - With compile error: prints compile error details
- **Output**
  - `--json` emits valid JSON

---

### Post-v1 follow-ups (explicitly not required for v1)
- Better HTML rendering (real parsing, spacing, code block fidelity)
- Cache fetched questions in `~/.cache/vleet/`
- More config UX (`config set`, keychain integration)
- Tests (only after core behavior stabilizes): renderer golden tests, polling state tests, etc.
