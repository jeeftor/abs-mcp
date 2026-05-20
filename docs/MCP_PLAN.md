# Audiobookshelf MCP Server Plan

## Purpose

Create a high-quality MCP server that lets agents inspect and safely operate an Audiobookshelf instance without relying on stale public API docs or manual UI workflows.

The first deliverable should be a small but well-tested server that can:

- Authenticate with Audiobookshelf using an API key or compatible bearer token.
- List libraries and read library details.
- List, filter, and inspect library items.
- Trigger and observe library scans.
- Report server/auth health without leaking secrets.
- Run deterministic round-trip tests against a local Audiobookshelf Docker fixture.

## Source Reality

The old API docs are useful for vocabulary and examples, but they explicitly say they are out of date and no longer maintained:

- `https://api.audiobookshelf.org/`

Current planning should use these as stronger signals:

- Audiobookshelf source: `https://github.com/advplyr/audiobookshelf`
- Router and controller source, especially:
  - `server/routers/ApiRouter.js`
  - `server/controllers/LibraryController.js`
  - `server/controllers/LibraryItemController.js`
  - `server/controllers/ApiKeyController.js`
  - `server/controllers/MeController.js`
  - `server/controllers/SearchController.js`
- Official user docs for current auth concepts:
  - `https://www.audiobookshelf.org/guides/api-keys/`
- Official MCP SDK status:
  - `https://modelcontextprotocol.io/docs/sdk`
- Local Docker fixture:
  - `test/abs`

## Implementation Direction

Use Go first unless a later spike disproves it.

Reasons:

- The official MCP SDK list currently includes Go as a Tier 1 SDK.
- The nearby Audiobook Organizer project already has a focused Go ABS client, models, websocket client, and e2e harness patterns.
- Go produces a single local binary, which is practical for stdio MCP and corporate environments.
- Existing ABS fixture tests are Go-based and can be adapted into this repo.

Initial module layout:

```text
cmd/abo-mcp/
  main.go
internal/abs/
  client.go
  models.go
  errors.go
internal/mcpserver/
  server.go
  tools.go
  resources.go
  prompts.go
internal/config/
  config.go
internal/apicatalog/
  inventory.go
  diff.go
scripts/
  abs-fixture-reset.sh
  abs-fixture-scan.sh
test/abs/
  README.md
```

Do not copy the nearby Audiobook Organizer client blindly. Use it as a reference, then keep this repo's client scoped to MCP workflows.

## Configuration

Support environment variables first because MCP clients and container launchers
can inject them without placing secrets in process arguments:

- `ABS_BASE_URL`
- `ABS_API_KEY`
- `ABS_TIMEOUT`
- `ABS_EXTRA_HEADERS_FILE`
- `ABS_READ_ONLY`
- `ABS_FIXTURE_DIR`

Also support matching Cobra/Viper flags for explicit local runs and debugging:

- `--base-url`
- `--api-key`
- `--timeout`
- `--extra-headers-file`
- `--read-only`
- `--fixture-dir`

Use explicit flag values as higher precedence than environment variables.

Default to read-only mode unless a mutating tool is explicitly enabled through configuration. A server used by an agent should make destructive or library-mutating actions visible at the tool schema level.

Prefer Audiobookshelf API keys for automation when available. The current Audiobookshelf API key guide describes them as server-to-server automation credentials that use the `Authorization: Bearer <key>` header.

## Initial MCP Surface

The candidate tool analysis lives in `docs/API_TOOL_ANALYSIS.md`. Use that document to decide what should become an MCP tool, resource, or prompt after source and fixture verification.

Read-only tools:

- `abs_health_check`: Validate URL/auth and return server/user summary.
- `abs_list_libraries`: Return library IDs, names, media types, folders, and basic stats when available.
- `abs_get_library`: Return one library by ID.
- `abs_list_library_items`: Paginated list with filters that match ABS query semantics.
- `abs_get_library_item`: Return one item by ID with optional expanded media/files.
- `abs_search`: Search ABS using the current source-backed endpoint once verified.

Mutating tools:

- `abs_scan_library`: Trigger a scan for a library ID. Require `force` as an explicit boolean.
- `abs_remove_library_items_with_issues`: Only add after a fixture test proves the endpoint and response semantics.

Resources:

- `abs://server/info`
- `abs://libraries`
- `abs://libraries/{library_id}`
- `abs://libraries/{library_id}/items`
- `abs://api-inventory/current`
- `abs://fixture/status`

Prompts:

- `abs_library_audit`: Guide an agent through checking library health, missing items, and scan status.
- `abs_api_update_review`: Guide an agent through source inventory diff, client updates, tool schema updates, and fixture verification.

## API Inventory Strategy

Create a generated inventory file from Audiobookshelf source instead of hand-maintaining endpoint guesses.

Inventory fields:

- HTTP method.
- Path.
- Router/controller source file.
- Handler method.
- Auth/admin requirements when inferable.
- Query parameters observed in controller code.
- Request body fields observed in controller validation.
- Response shape model or sample fixture output.
- MCP tool/resource mapping, if any.

Suggested generated output:

```text
docs/api-inventory/generated/abs-api-inventory.json
docs/api-inventory/generated/abs-api-inventory.md
docs/api-inventory/baseline/abs-api-inventory.json
```

The update workflow should fail when source inventory changes but mapped MCP schemas/tests are not reviewed.

## Fixture Strategy

Use the repo-local `test/abs` fixture for discovery and integration testing. It provides:

- Compose project name `abs-mcp`.
- Container names `abs-mcp-abs-plain` and `abs-mcp-abs-metadata`.
- `abs-plain` on `http://localhost:13388`.
- `abs-metadata` on `http://localhost:13389`.
- Reset scripts.
- Scan scripts.
- Plain and sidecar metadata modes.
- Known libraries: `Audiobooks` at `/audiobooks` and `Ebooks` at `/books`.

The fixture was derived from the Audiobook Organizer ABS fixture, but should evolve here independently for MCP testing.

Round-trip test matrix:

- Health check against both ABS instances.
- List libraries and assert both expected libraries exist.
- List items before and after scan.
- Trigger forced scan and observe completion/status.
- Read an audiobook item and an ebook item.
- Verify read-only mode blocks mutating tools.
- Verify auth failure produces a safe error with no token leakage.

## Testing Without Codex MCP Registration

Because this corporate Codex instance cannot be configured with MCP servers, verification should use protocol-level tests:

- Unit tests for ABS client with `httptest`.
- Unit tests for MCP tool handlers with fake ABS clients.
- Stdio integration tests that spawn the server binary and call MCP JSON-RPC methods.
- Optional later testing with external MCP-capable tools where the user has access.

This makes the server testable before it is wired into any agent host.

## Security Constraints

- Never expose raw tokens through tools, resources, logs, panic messages, or test snapshots.
- Prefer API keys for automation.
- Make mutating tools visibly mutating in descriptions and tests.
- Keep read-only mode easy to enable and covered by tests.
- Validate URL configuration to avoid accidental requests to unintended hosts.
- Bound pagination and output sizes so an agent cannot accidentally dump a large library.
- Set request timeouts.

## Milestones

1. Repository foundation:
   - Go module.
   - Config loading.
   - Minimal ABS client.
   - MCP server skeleton over stdio.
   - Unit test harness.

2. Read-only ABS tools:
   - Health check.
   - Library list/get.
   - Library item list/get.
   - MCP protocol tests.

3. Docker fixture integration:
   - Script wrappers for reset/scan.
   - E2E tests using the fixture path.
   - Read-only mode tests.

4. Scan workflow:
   - `abs_scan_library`.
   - Scan status or websocket/polling support.
   - Round-trip fixture test.

5. API source synchronization:
   - Source inventory generator.
   - Baseline/diff command.
   - Agent skill configs for API update review.

6. Broader MCP UX:
   - Resources.
   - Prompts.
   - Pagination/output polish.
   - Optional HTTP transport if needed.

## Open Decisions

- Whether to keep the copied fixture as-is or reduce it further once MCP-specific tests exist.
- Whether scan completion should be websocket-based, polling-based, or both.
- Whether the first release should include mutating item cleanup tools or defer them until the read-only surface is stable.
- Whether API inventory generation should parse source statically, run ABS and observe endpoints dynamically, or combine both.
