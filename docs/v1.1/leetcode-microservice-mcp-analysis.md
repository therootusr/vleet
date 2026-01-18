# LeetCode integration: library vs microservice vs MCP server (analysis)

## Current state (in this repo)

- **This is a single local CLI binary** (Go) orchestrating config + workspace + editor + rendering + LeetCode calls.
- The LeetCode boundary is already **abstracted behind an interface** (`internal/leetcode.Client`) and injected into the app (`internal/app.App`).
- The LeetCode client uses **browser-session cookies / CSRF token** (secrets), and it already anticipates **blocking / HTML responses** from LeetCode when automation is detected.

This matters because “extract to microservice” changes how secrets are handled, how requests look to LeetCode, and how you operate/deploy the system.

## Is splitting into a microservice a good idea here?

### My recommendation

**Not as the first move.** For this project as it exists today (a local CLI), a separate microservice is usually **not** a net win.

If your real goal is **reuse across multiple projects**, the most practical “industry-shaped” approach is:

- **Primary**: a **library** (stable Go package) that implements the LeetCode client + types and is easy to embed.
- **Optional**: a thin **service wrapper** around that library *only if/when* you have multiple independent consumers that truly need a network boundary.
- **Optional**: an **MCP server wrapper** (also using the same library) when your consumers are LLM toolchains that speak MCP.

The key is: **one shared core** (library) + **multiple adapters** (CLI, service, MCP).

### When a microservice *is* a good idea

A LeetCode microservice becomes reasonable when most of these are true:

- **Multiple clients**: you have several apps (web, mobile, bots, CI jobs) that need the same LeetCode operations.
- **Centralized policy**: you need shared rate limiting, caching, retries/backoff, and consistent error handling.
- **Centralized secrets**: you want a single place where LeetCode auth material is stored/rotated.
- **Operational needs**: you need observability, request auditing, quota management, and controlled rollout independent of clients.
- **Language/stack diversity**: clients are in different languages and don’t want to embed a Go lib.

### Why a microservice is often a bad idea *for this specific repo today*

- **Increases operational complexity**: deploy, version, monitor, secure, and maintain an always-on component.
- **Harder auth/secrets**: instead of “local config file with 0600”, you now need secret storage, rotation, and transport security between CLI and service.
- **New failure modes**: network failures, partial outages, timeouts, dependency chain issues, plus API versioning.
- **Latency + UX**: a local CLI calling a remote service adds latency and makes “offline-ish” workflows harder.
- **LeetCode anti-bot risk**: centralizing traffic can look *more* like automation; you may need stronger rate limiting and “human-like” behavior.
- **Potential compliance / ToS risk**: if this is shared publicly as a service, it increases the likelihood you’ll violate LeetCode terms or get blocked.

Net: if the only consumer is the CLI, keep it in-process.

## Is “microservice + MCP server for the same thing” industry standard?

### What is standard vs what is emerging

- **Microservice**: standard pattern when you truly need a network boundary (multi-consumer, different release cadence, centralized policy, scaling, isolation).
- **MCP server**: a newer, fast-growing pattern in the LLM ecosystem for tool integration. It is **not** (yet) a universal industry standard like REST/gRPC, but it’s becoming common for “AI tool” surfaces.

### Is it a best practice to build both?

**It can be a good architecture** if you treat MCP as *one adapter* and you don’t duplicate business logic.

Best-practice shape:

- **Core**: “LeetCode domain client” library (types + operations + policies)
- **Adapters**:
  - CLI adapter (your current app)
  - HTTP/gRPC microservice adapter (optional)
  - MCP adapter (optional)

Anti-pattern:

- Implement logic twice (service and MCP diverge)
- MCP server becomes a “backdoor” with different auth/limits than the service

## Library vs microservice vs MCP server: decision matrix

### Library (recommended first)

- **Pros**
  - Lowest complexity, easiest to test and version
  - Best performance (no network hop)
  - Fits CLI distribution (single binary)
  - Reusable internally and across repos (via Go module or internal package split)
- **Cons**
  - Consumers must be in Go (unless you create bindings)
  - No centralized rate limiting / caching across machines by default

### Microservice

- **Pros**
  - Language-agnostic reuse
  - Centralized caching, quotas, retries, backoff, telemetry
  - Centralized secret management and policy enforcement
- **Cons**
  - Operational burden and on-call surface
  - AuthN/AuthZ and transport security become required
  - API versioning and backward compatibility become a permanent cost

### MCP server

- **Pros**
  - First-class integration with LLM toolchains
  - Standardized “tool invocation” semantics for AI clients
  - Great for “agent does X” workflows (fetch problem, submit code, check status)
- **Cons**
  - Smaller audience than HTTP/gRPC (today)
  - Security model is often “whoever can invoke tools can act as you” unless designed carefully
  - Still needs the same rate-limit/caching/secret considerations

## Recommended path (practical, avoids rework)

### Phase 0: Keep it in-process

This repo already has the seam (`leetcode.Client`). Keep using it and stabilize:

- request/response types
- error taxonomy
- rate limiting / polling behavior

### Phase 1: Make a reusable library the “source of truth”

Extract the LeetCode client + types into a module/package meant for reuse.

Consumers:

- this CLI
- future HTTP service
- future MCP server

### Phase 2 (optional): Add an HTTP/gRPC microservice adapter

Do this only when you have:

- multiple independent consumers
- a clear auth story (per-user vs shared account)
- a need for shared caching/quotas/telemetry

### Phase 3 (optional): Add an MCP server adapter

This can exist:

- as a *separate binary* that uses the library directly, or
- as a thin MCP-to-service gateway (if the service exists and you want one canonical enforcement point)

Which is better?

- **If you already have the microservice** and you care about centralized policy: MCP should call the service.
- **If you don’t have the microservice**: MCP can use the library directly and keep things simple.

## Key architectural questions you should decide up front

- **Auth model**
  - Per-user session cookies (each user provides their own) vs a shared “service account”
- **Threat model**
  - Who is allowed to submit code? What prevents an arbitrary client from using your service to act as you?
- **Rate limits / blocking**
  - How you prevent hammering LeetCode and getting blocked (especially with polling)
- **Caching**
  - What is safe to cache (problem statements/snippets) vs what should never be cached (anything user/session specific)
- **Contract stability**
  - Do you want to promise a stable API to others? If yes, microservice/MCP imply long-term backward compatibility.

## Bottom line

- **For vleet as a single local CLI**: a microservice is likely **premature complexity**.
- **For reuse across multiple projects**: start with a **library** as the shared core.
- **“Microservice + MCP server”**: not the default industry baseline, but **a reasonable architecture** when MCP is just an adapter and policy/security are centralized (ideally in one place).

