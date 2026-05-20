# Agent Instructions

## Project Goal

Build a high-quality Model Context Protocol server for Audiobookshelf. The server should expose safe, typed MCP tools and resources for inspecting and managing an Audiobookshelf instance, with repeatable tests against a local Docker Audiobookshelf fixture.

## Working Assumptions

- Prefer Go for the first implementation unless a later decision record changes that. This matches the nearby Audiobook Organizer ABS client and the official Go MCP SDK.
- Treat the public Audiobookshelf API docs as useful but not authoritative. They currently state that they are out of date and no longer maintained.
- Treat Audiobookshelf source code and verified Docker round-trip tests as the source of truth for behavior.
- The user cannot configure this corporate Codex instance with MCP servers, so local verification must not depend on registering this server in Codex.

## Tooling Preferences

- Use `rg` for content searches.
- Use `fd` or `find` for file discovery.
- Use `rtk` as the command prefix for shell commands in this workspace.
- Use `uv` for Python helper scripts and virtual environments if Python is added.
- Use `go test ./...` for Go verification once Go code exists.
- Use `docker compose` only through repo scripts or documented make targets so the ABS fixture remains resettable.

## Coding Discipline

- For non-trivial changes, state the working assumption and the smallest verifiable goal before editing.
- Keep edits surgical and traceable to the current task.
- Prefer simple, typed request/response structures over generic maps at MCP boundaries.
- Do not add speculative MCP tools. Add tools only when backed by an ABS endpoint, a fixture scenario, or a documented user workflow.
- Do not log bearer tokens, API keys, cookies, or raw Authorization headers.
- Do not persist user credentials in committed files.
- Do not mutate files in `/Users/Shared/Docker/audiobook-organizer` unless explicitly asked. That fixture can be read and invoked as an external dependency while planning.

## MCP Design Rules

- Tools perform actions or bounded queries.
- Resources expose readable state snapshots such as server info, libraries, item summaries, generated API inventory, and fixture status.
- Prompts should be limited to repeatable operator workflows such as library audit, scan troubleshooting, or API update review.
- Tool names should be stable, explicit, and namespaced with `abs_`.
- Each tool schema must define required inputs, optional inputs, output shape, error cases, and whether it can mutate ABS state.
- Mutating tools must be opt-in and should require IDs rather than fuzzy names where possible.
- Long operations such as scans should return job/status information and provide a separate status/read tool instead of blocking indefinitely.

## Audiobookshelf API Update Workflow

- Maintain an API inventory generated from Audiobookshelf source, especially `server/routers/ApiRouter.js` and controller methods under `server/controllers/`.
- Compare generated inventory changes before updating MCP tools.
- For changed endpoints, update typed client methods, MCP tool schemas, docs, and round-trip tests together.
- Validate against the Docker ABS fixture before marking an API update complete.

## Test Fixture

The repo-local fixture at `test/abs` provides:

- `docker-compose.yml` with plain and metadata-enabled Audiobookshelf instances.
- Reset and scan scripts.
- Compose project name `abs-mcp`.
- Unique container names `abs-mcp-abs-plain` and `abs-mcp-abs-metadata`.
- Known ports: `13388` for plain and `13389` for metadata-enabled.
- Test credentials and tokens through its `.env.testing`/baseline setup.

## Verification

- Prefer repo-native verification first.
- For docs-only changes, run a quick file review and `git status`.
- For API client changes, add or update unit tests with `httptest`.
- For MCP protocol behavior, add tests that spawn the MCP server over stdio and issue JSON-RPC/MCP requests directly.
- For ABS behavior, run the resettable Docker fixture with `make abs-ci-smoke` or `make abs-dev-reset-scan`.
