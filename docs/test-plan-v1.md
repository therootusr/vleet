## Test plan v1: vleet (QA)

This document defines the **v1 QA strategy**, the **testing frameworks/tooling**, and the **high-signal test cases** for vleet.

vleet is a local CLI that orchestrates:
- filesystem operations (workspace + solution files)
- process execution (editor)
- HTTP calls to LeetCode (GraphQL fetch + REST submit/poll)

The goal is to be comprehensive without being pedantic: focus on correctness, security/privacy, and user-visible behavior.

---

### v1 quality bar (definition of “QA complete”)

v1 is “QA complete” when the following is true:
- **End-to-end**: a user can `fetch` → `solve` (edit) → `submit` → `poll` to a final verdict for a public problem.
- **Security/Privacy**: secrets (session cookies, CSRF tokens) are not printed in normal output or error output.
- **Reliability**: the core flows are covered by deterministic automated tests, and the CLI has a small set of smoke E2E tests suitable for gating.
- **Compatibility (baseline)**: validated at least on macOS; ideally also on Linux.

---

### Testing strategy (pyramid)

We use a layered test pyramid that matches the risk profile of a CLI with external dependencies:

- **Unit tests (fast, deterministic)**
  - Pure logic: config parsing + permissions, HTML→plain text conversion, language/path resolution, snippet selection.
- **Integration tests with local fakes (deterministic)**
  - Orchestration and HTTP protocol behavior using:
    - `net/http/httptest` servers for LeetCode endpoints
    - `t.TempDir()` for filesystem isolation
    - fakes/stubs for editor/workspace where appropriate
- **CLI E2E smoke tests (few, critical)**
  - Build the `vleet` binary and run it like a user would (via `os/exec`), but against a local stub server.
- **Live LeetCode snapshot tests (networked)**
  - Compare responses from the real LeetCode server against saved “golden” snapshots to detect upstream drift.

This keeps most tests deterministic while still providing a controlled way to detect upstream API/statement changes.

---

### Frameworks / tooling

We standardize on Go’s ecosystem and keep dependencies minimal:

- **Core test framework**: Go `testing`
- **HTTP simulation**: `net/http/httptest`
- **Filesystem isolation**: `t.TempDir()`
- **CLI execution**: `os/exec`
- **Golden/snapshot comparisons**:
  - Store snapshots under `testdata/` and compare using normalized text/JSON.
  - Keep snapshot scope small (only the fields we truly depend on).

---

### Testability requirements for v1 QA (to support deterministic E2E)

To make CLI E2E tests reliable and offline, v1 QA assumes the CLI supports:

- **Base URL override for LeetCode** (e.g. `VLEET_BASE_URL` or an internal `--base-url`)
  - Allows routing GraphQL/submit/check to a local `httptest` server.
- **Config isolation**
  - At minimum, E2E tests should be able to run the CLI with `HOME` set to a temp directory so config is not shared with the developer’s machine.

These are not user-facing UX features; they are test enablers that reduce flakiness and improve safety.

---

### Test suites and high-signal test cases

#### 1) Config & secrets (security-critical)

- **YAML roundtrip**
  - Save YAML → load YAML → values preserved for all supported keys.
- **Permissions enforcement**
  - Reject config files with permissions broader than `0600` (actionable error message).
- **Secret redaction in CLI output**
  - `vleet config show` must not print the raw values of `leetcode.session` or `leetcode.csrftoken`.
- **Secret redaction in errors**
  - Any error path that mentions auth must not embed cookie/token values in the error message.

#### 2) Renderer (HTML → plain text header)

- **HTML stripping + entity unescape**
  - Ensure the output contains no raw tags and entities are unescaped (`&nbsp;`, `&amp;`, `&lt;`).
- **Structure preservation (basic)**
  - Lists (`ul/li`) become readable bullets, `pre` blocks retain line breaks.
- **Language comment prefixing**
  - `python3` uses `# `, others default to `// `.
- **Metadata inclusion**
  - Includes title/difficulty, URL, tags, and hints (when present).

#### 3) Workspace manager (data integrity + safety)

- **Path layout and extension mapping**
  - Creates/loads `./<slug>/solution.<ext>` based on language slug.
- **No-overwrite safety**
  - Attempts to write when a solution exists must fail (prevents accidental data loss).
- **File override validation**
  - `--file` extension must match the selected language extension.

#### 4) LeetCode client (stubbed protocol correctness)

Using `httptest` servers:
- **GraphQL fetch contract**
  - Request shape is correct; response maps into `leetcode.Question` (ids, title, difficulty, content, tags, snippets).
  - Detects unexpected `text/html` response (blocked/captcha) and returns an actionable error.
- **Submit contract**
  - Correct endpoint path and payload (`lang`, `question_id`, `typed_code`)
  - Cookies + CSRF header are sent when configured
  - Handles non-2xx responses and unexpected HTML bodies.
- **Poll/check contract**
  - Polls until terminal state; returns status/runtime/memory and error fields.
  - Respects timeout and cancellation.

#### 5) App orchestration (integration of modules)

- **Fetch flow**
  - Fetch → snippet selection by `langSlug` → render header → create workspace → write solution file.
  - If the file already exists, it must not be overwritten.
- **Solve flow**
  - Opens the editor with the configured command and correct file path.
  - For automated tests, editor should be stubbed/non-interactive (e.g. `true`).
- **Submit flow**
  - Loads config, requires session, reads solution, submits, polls, prints a summary result.
  - Output must not include secrets even on failures.

#### 6) CLI E2E smoke tests (offline, deterministic)

These are the small set of happy-path tests intended to be automated as the v1 QA gate.

- **E2E-1: fetch creates workspace and solution file**
  - Command: `vleet fetch two-sum --lang cpp`
  - Assert: `./two-sum/solution.cpp` exists and contains a header + snippet.
- **E2E-2: submit prints final verdict**
  - Command: `vleet submit two-sum --lang cpp`
  - Assert: output includes `Verdict: Accepted` (and runtime/memory if present).
- **E2E-3: solve --submit full flow (optional but high value)**
  - Command: `vleet solve two-sum --lang cpp --submit`
  - Assert: exits successfully and prints final verdict.

All E2E smoke tests should:
- run with isolated config (`HOME` set to temp dir)
- use a local stub server for LeetCode endpoints (no real network)
- avoid real secrets (use fake session/csrf)

#### 7) Live LeetCode snapshot tests (golden tests against the real server)

These tests compare **responses from the real LeetCode server** against committed snapshots to detect upstream drift.

**What these tests cover**
- Field presence and structure for the GraphQL question query (`questionData`)
- Stability/changes in:
  - `content` HTML (problem statement)
  - `codeSnippets` starter code
  - tags/difficulty/title

**Naming**
- These are best described as **“live contract snapshot tests”** (or **“live golden tests”**).
  - They are “golden tests” because they compare against saved outputs.
  - They are “contract tests” because they validate the upstream API/response contract.

**How snapshots should be stored**
- Store normalized JSON snapshots under `testdata/leetcode/` (one file per slug).
- Normalize the JSON before compare:
  - stable key ordering
  - consistent whitespace
  - optional: ignore fields vleet does not rely on

**Execution model**
- Keep these tests separate from the deterministic CI gate (they require network and can fail when LeetCode updates statements).
- Recommended: run them on a schedule (nightly) or as an explicit CI job.

---

### Manual QA checklist (real-world validation)

Manual QA is still required for real integrations that are hard to simulate fully:

- **Real fetch**: `vleet fetch two-sum --lang cpp`
- **Real solve**: `vleet solve two-sum --lang cpp` (verifies editor invocation)
- **Real submit**: `vleet submit two-sum --lang cpp` with a valid session in config
- **High-signal failures**
  - invalid slug
  - missing config / missing session
  - compile error output
  - LeetCode blocking/captcha (HTML response detection)
