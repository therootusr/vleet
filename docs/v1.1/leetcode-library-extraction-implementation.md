# v1.1 Implementation Plan: Extract LeetCode logic into a reusable library

## Status

- **Type**: Implementation / execution plan (v1.1)
- **Owner**: vleet maintainers
- **Scope**: Extract LeetCode-specific logic (HTTP client + types + language slug helpers) into a reusable Go library.
- **Non-scope**: No behavior changes intended beyond refactoring and boundary cleanup.

## Background / current architecture

In v1, vleet is a single local CLI. LeetCode integration lives in `internal/leetcode/` and is invoked via an interface boundary (`leetcode.Client`) injected into the orchestrator (`internal/app.App`). Tests include both deterministic `httptest`-based protocol tests and optional live snapshot tests (tagged `live`).

Key current coupling to address for library extraction:

- `internal/leetcode.HttpClient` depends on `vleet/internal/config` for auth (`config.LeetCodeAuth`).
- The package lives under `internal/`, so it cannot be imported by other repos (by design in Go).

## Goals (v1.1)

- **Reusable library**: Make LeetCode client logic importable by other projects.
- **Stable boundary**: Define a clear API surface (types + client behavior) that can evolve safely.
- **Decoupled from vleet app**: The library must not depend on `vleet/internal/*`.
- **No UX regression**: `vleet` CLI behavior and output remain compatible.
- **Test parity**: Preserve existing deterministic tests; keep live snapshot tests optional.

## Non-goals (v1.1)

- **No microservice** in v1.1.
- **No MCP server** in v1.1 (can be a later adapter that uses this library).
- **No new LeetCode features** (problem list, search, caching expansions) unless required for the extraction.
- **No auth automation** (username/password login).

## Extraction approach decision (v1.1)

### Decision

**We will extract the LeetCode logic into a dedicated, separate Git repository** that provides a reusable Go module (the “LeetCode library”).

Rationale:

- You explicitly want **library-level reuse across projects**, and a separate repo is the cleanest long-term boundary for ownership, releases, and consumers.
- It avoids multi-module tagging friction and keeps the vleet repo focused on the CLI product.

### Alternatives (explicitly not chosen for v1.1)

- **Multi-module repo** (library module inside `vleet/`):
  - Valid approach, but not chosen due to workflow/release preference.
- **Exported package in the same module** (e.g. `pkg/leetcode`):
  - Simpler mechanically, but couples library lifecycle to the CLI and requires the vleet module path to be externally meaningful.

## Proposed library scope and API

### What moves into the library

- **HTTP client and protocols**
  - GraphQL `questionData` fetch
  - REST submit
  - REST poll/check
  - “blocked/captcha” HTML detection
  - backoff + timeout behavior
- **Public data types**
  - `Question`, `CodeSnippet`, `TopicTag`
  - `SubmitRequest`, `SubmissionID`, `PollOptions`, `SubmissionResult`
- **Language slug helpers**
  - `LangSlug`, known constants, `ParseLangSlug` and `Info`
- **Test assets + tests**
  - `httptest` protocol tests
  - live snapshot tests (optional build tag)
  - snapshots under `testdata/` (scoped to the library module)

### What stays in vleet (not part of the library)

- Config file formats and storage (`internal/config`)
- Workspace creation and file IO policies (`internal/workspace`)
- HTML-to-comment rendering (`internal/render`)
- Editor execution (`internal/editor`)
- CLI output formatting (`internal/output`)
- Orchestrator logic (`internal/app`)

Rationale: the library is “LeetCode API client + primitives”, not “vleet product behavior”.

### Auth type (decoupling requirement)

The library should define its own minimal auth struct, e.g.:

- `type Auth struct { Session string; Csrftoken string }`

Key requirements:

- Treat auth values as secrets (never include raw values in error strings).
- Support “read-only” usage (fetch question) without auth.
- Require session for submit/poll.

vleet will map its config (`config.LeetCodeAuth`) into the library’s `Auth`.

## New library repository layout (recommended)

Create a new repository (example): `github.com/<you>/vleet-leetcode` (name is flexible).

Inside that repo:

```text
vleet-leetcode/
  go.mod                 # module github.com/<you>/vleet-leetcode (example)
  README.md
  client.go
  types.go
  lang.go
  testdata/leetcode/
    two-sum.json
    valid-parentheses.json
  *_test.go
```

Notes:

- Pick a **stable module path** early. Changing module paths later is painful for consumers.
- Keep the public API surface small and intentional; you can expand later.

## Execution plan (step-by-step)

### Phase 0 — Repo + module decisions (no code)

- Choose:
  - new repo name
  - Go module path (must match the repo path you intend to publish)
  - license (if open-source)
  - initial version policy (recommend start at `v0.1.0`)
- Decide whether the library repo will accept PRs from vleet as the “source of truth” (recommended).

**Exit criteria**
- Repo exists and is reachable by `go get` (even before first release, for local `replace` during migration).

### Phase 1 — Create library repo skeleton and migrate code (copy first)

- In the new repo:
  - initialize `go.mod`
  - add a minimal `README.md`
  - copy the current `internal/leetcode` implementation and tests into the new repo
  - copy `internal/leetcode/testdata/leetcode/*` into `testdata/leetcode/`

**Exit criteria**
- In the library repo, `go test ./...` passes (deterministic tests).

### Phase 2 — Decouple the library from vleet internals

- Replace any references to `vleet/internal/config` with library-local equivalents (`Auth`).
- Ensure the library has **no imports** from `vleet/internal/*` (or from the vleet repo at all).
- Ensure error messages remain actionable but do not leak secrets.

**Exit criteria**
- Library builds and tests pass with no dependency on the vleet repo.

### Phase 3 — Publish an initial library version

- Tag an initial version (recommend `v0.1.0`).
- Ensure `README.md` documents:
  - supported operations
  - auth requirements
  - rate limit/polling guidance
  - security notes (secrets)

**Exit criteria**
- A separate consumer repo can `go get <module>@v0.1.0` successfully.

### Phase 4 — Migrate vleet to depend on the external library

- In vleet:
  - update imports from `vleet/internal/leetcode` to the library module path
  - update dependency wiring and config mapping (vleet config auth → library `Auth`)
  - during development, optionally use a `replace` directive to point at a local checkout of the library repo

**Exit criteria**
- `go test ./...` passes in vleet.
- CLI behavior remains unchanged for fetch/submit/poll flows.

### Phase 5 — Remove the old in-repo implementation

- Remove `internal/leetcode/` once vleet is fully migrated.
- Ensure tests/snapshots are not duplicated between repos (source-of-truth should be the library repo).

**Exit criteria**
- No remaining references to `internal/leetcode` in vleet.

## Versioning and release strategy (Go best practices)

### Library version vs vleet version

Treat them as **independent**:

- vleet can release v1.1 while the library could be v0.x (if you want to signal API churn) or v1.x (if you commit to stability).

### Tagging in a separate repo

Use standard Go module tags:

- `v0.1.0`, `v0.2.0`, ...
- Once stable, `v1.0.0` and follow semantic versioning.

### Compatibility policy (suggested)

- **v0.x**: allow breaking changes as needed while the API is settling.
- **v1.0.0**: once stable, follow semantic versioning strictly.

## Testing plan (v1.1)

### Preserve and enhance deterministic tests

- Keep `httptest` contract tests in the library module:
  - fetch request shape
  - submit payload + headers
  - poll state machine
  - HTML response rejection
- Ensure tests do not require network access.

### Live snapshot tests remain optional

- Keep build tag (e.g. `//go:build live`) and environment variable controls.
- Store snapshots under `leetcode/testdata/leetcode/*.json`.

### vleet tests

- vleet should retain its orchestration tests.
- If vleet has tests stubbing LeetCode behavior, it should now depend on the library interfaces/types.

## Security and privacy considerations

- **Secrets handling**: library errors must never include cookie/token values.
- **Logging**: avoid adding logging inside the library by default; if logging is needed, accept an interface or `*slog.Logger` passed in by the caller and ensure redaction.
- **Rate limiting**: preserve conservative polling behavior; document how callers can tune it via `PollOptions`.
- **TLS**: never disable TLS verification.

## Risks and mitigations

- **Two-repo coordination**
  - Mitigate with a “library first” workflow: LeetCode changes land in the library repo, then vleet updates the dependency.
  - Keep a short cadence for bumping the library version in vleet (reduces drift).
- **API stability expectations**
  - Mitigate by starting the library at v0.x and promoting to v1.0 when stable.
- **Accidental behavior changes**
  - Mitigate with golden tests (protocol tests + snapshot tests) and a “no UX regression” acceptance checklist.

## Acceptance criteria (definition of done for v1.1)

- **Library exists** as a separate repo with a real Go module path.
- **No imports from `vleet/internal/*`** inside the library.
- **vleet builds and tests pass** using the new library.
- **Library tests pass** (deterministic), with optional live snapshot tests preserved.
- **Docs added** for how to consume the library from other repos.

## Follow-ups (post v1.1)

- Add an MCP server adapter that depends on the library (not the CLI).
- Consider extracting a “policy layer” (rate-limit, caching) if multiple adapters emerge.
- If external adoption grows, consider moving the library module to its own repository.

