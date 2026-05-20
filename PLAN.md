# Audiobookshelf MCP Server Rebuild Plan

This document is an implementation blueprint for recreating this repository as a
working Model Context Protocol server for Audiobookshelf. Treat the
Audiobookshelf source code and the Docker fixture tests as authoritative over
the public API docs.

## Goal

Build a Go stdio MCP server named `abs-mcp` that exposes safe, typed tools,
resources, and prompts for inspecting and operating an Audiobookshelf instance.
The server should be useful to local AI agents without requiring Codex or any
other client to register custom MCP servers during tests.

## Technology Choices

- Language: Go.
- MCP SDK: `github.com/modelcontextprotocol/go-sdk/mcp`.
- CLI: Cobra.
- Configuration: Viper with environment variable support.
- HTTP client: standard library `net/http`.
- Tests: Go `testing`, `httptest`, MCP JSON-RPC/protocol tests, Python unittest
  for helper scripts, and optional Docker-backed Audiobookshelf integration
  tests.
- Local fixture: Docker Compose under `test/abs`.
- Distribution: GitHub Actions release artifacts and GHCR Docker images.

## Configuration Contract

Environment variables and env files are the preferred production and MCP-client
interfaces. Explicit CLI flags exist for local debugging and override
environment values.

| Environment variable | CLI flag | Required | Default | Purpose |
| --- | --- | --- | --- | --- |
| n/a | `--env-file` | no | unset | Docker-style dotenv file with ABS settings. |
| `ABS_BASE_URL` | `--base-url` | yes | none | Audiobookshelf base URL. |
| `ABS_API_KEY` | `--api-key` | yes | none | ABS API key or compatible bearer token. |
| `ABS_READ_ONLY` | `--read-only` | no | `true` | Blocks mutating MCP tools when true. |
| `ABS_TIMEOUT` | `--timeout` | no | `30s` | ABS request timeout as Go duration or seconds. |
| `ABS_FIXTURE_DIR` | `--fixture-dir` | no | `test/abs` | Fixture directory used by fixture status resources. |
| `ABS_EXTRA_HEADERS_FILE` | `--extra-headers-file` | no | unset | JSON file of extra request headers. |
| n/a | `--header NAME=VALUE` | no | unset | Repeatable local extra request header. |
| `ABS_TLS_CA_CERT_FILE` | `--tls-ca-cert-file` | no | unset | PEM CA bundle for corporate/private TLS. |
| `ABS_TLS_INSECURE_SKIP_VERIFY` | `--tls-insecure-skip-verify` | no | `false` | Temporary TLS verification bypass. |

`ABS_EXTRA_HEADERS_FILE` must contain a JSON object from header name to string
value. Reject `Authorization` in this file because auth must come from
`ABS_API_KEY` or `--api-key`.

Support repeatable `--header NAME=VALUE` flags for quick local runs. Merge them
with `ABS_EXTRA_HEADERS_FILE`, with explicit header flags overriding duplicate
file headers. Do not provide an environment variable for individual headers;
use the header file for containerized or secret-bearing configuration.

Support `--env-file PATH` for desktop MCP clients that can pass arguments but
should not embed secrets directly in JSON config. Parse simple Docker-style
dotenv lines: `KEY=value`, `KEY="value"`, `KEY='value'`, blank lines, comments,
and optional `export` prefixes. Apply env-file values as defaults so precedence
is explicit CLI flags, process environment, env file, then built-in defaults.
Ignore unknown env-file keys.

For corporate and self-signed TLS, support a CA bundle file first. The insecure
skip-verify option exists only as an explicit temporary fallback and should
default to false.

Do not implement password login or OIDC/OAuth as a first-class MCP auth mode.
Audiobookshelf supports OIDC for browser/mobile SSO, but API keys are the
appropriate server-to-server automation credential.

## Server Startup

Implement `cmd/abs-mcp` as a Cobra root command that:

1. Binds the configuration flags to a fresh Viper instance.
2. Applies `--env-file` values as Viper defaults when present.
3. Loads config from flags, process environment, env file, and defaults.
4. Constructs an authenticated ABS client.
5. Sets an HTTP timeout and extra headers.
6. Creates the MCP server and runs it over stdio.

The binary should not log bearer tokens, API keys, cookies, or raw
`Authorization` headers.

## Docker Image

Provide a Dockerfile for running the stdio MCP server in a minimal container.

Image requirements:

- Build a static Linux `abs-mcp` binary.
- Run as a non-root user.
- Include CA certificates so HTTPS ABS endpoints work.
- Include `docs/api-inventory/generated/abs-api-inventory.json` at the same
  relative path expected by the server so `abs://api-inventory/current` works.
- Accept the same environment variables and CLI flags as the native binary.
- Support mounting header files and CA bundles into the container, then pointing
  `ABS_EXTRA_HEADERS_FILE` and `ABS_TLS_CA_CERT_FILE` at those mounted paths.
- Support local corporate builds with overridable base images and an optional
  MITRE certificate installation build argument.

Example local build:

```bash
make docker-build
```

Example runtime:

```bash
docker run --rm -i \
  -e ABS_BASE_URL=http://host.docker.internal:13388 \
  -e ABS_API_KEY=... \
  -e ABS_READ_ONLY=true \
  ghcr.io/jeeftor/abs-mcp:latest
```

Example runtime with Cloudflare Access headers and a private CA:

```bash
docker run --rm -i \
  -e ABS_BASE_URL=https://abs.example.com \
  -e ABS_API_KEY=... \
  -e ABS_EXTRA_HEADERS_FILE=/run/secrets/abs-headers.json \
  -e ABS_TLS_CA_CERT_FILE=/run/secrets/corporate-ca.pem \
  -v /path/to/headers.json:/run/secrets/abs-headers.json:ro \
  -v /path/to/corporate-ca.pem:/run/secrets/corporate-ca.pem:ro \
  ghcr.io/jeeftor/abs-mcp:latest
```

## Audiobookshelf Client

Implement a small typed ABS REST client under `internal/abs`.

Required client behavior:

- Validate that `ABS_BASE_URL` includes a scheme and host.
- Send `Authorization: Bearer <token>` on every request.
- Allow sanitized extra headers, excluding `Authorization`.
- Redact the configured token from HTTP error messages.
- Support JSON GET, POST, and DELETE-style operations required by the MCP tools.
- Preserve raw JSON for source-backed endpoints where stable typed models are
  not yet worth maintaining.

Required client methods:

- `GetCurrentUser`
- `GetLibraries`
- `GetLibrary`
- `GetLibraryItems`
- `GetLibraryItemsWithOptions`
- `GetAllLibraryItems`
- `GetLibraryItem`
- `SearchLibrary`
- `GetLibraryStats`
- `GetLibraryFilterData`
- `GetItemMetadataObject`
- `ScanLibrary`
- `ScanItem`
- `RemoveLibraryItemsWithIssues`

## MCP Tools

Tool names must be stable and namespaced with `abs_`. Use typed inputs and
outputs instead of generic maps at MCP boundaries unless the ABS response is
intentionally raw JSON.

Read-only tools:

- `abs_health_check`: validate auth, return base URL, read-only status,
  authenticated username/user type, and visible library count.
- `abs_list_libraries`: return visible library summaries.
- `abs_get_library`: return one library by exact ID.
- `abs_list_library_items`: return a bounded page of items. Support limit,
  offset, sort, desc, filter, include, minified, and collapseSeries. Default to
  a small page size and cap page size.
- `abs_get_library_item`: return one item by exact ID.
- `abs_search_library`: search one library with a bounded result limit.
- `abs_get_library_stats`: return raw library stats.
- `abs_get_filter_data`: return raw filter data for one library.
- `abs_get_item_metadata_object`: return the raw metadata object for one item.

Mutating tools:

- `abs_scan_library`: trigger a library scan and require an explicit `force`
  boolean.
- `abs_scan_library_and_wait`: trigger a scan, then poll item totals until the
  expected count is seen or a timeout expires.
- `abs_scan_item`: rescan one directory-backed item by exact item ID.
- `abs_remove_library_items_with_issues`: remove missing or invalid rows only
  after exact confirmation; optionally check an expected issue count.

All mutating tools must return an error when `ABS_READ_ONLY=true`.

## MCP Resources

Expose JSON resources for state snapshots that are safe to read repeatedly:

- `abs://server/info`
- `abs://libraries`
- `abs://libraries/{library_id}`
- `abs://libraries/{library_id}/items{?limit,offset,sort,desc,filter,include,minified,collapseSeries}`
- `abs://libraries/{library_id}/stats`
- `abs://libraries/{library_id}/filterdata`
- `abs://items/{item_id}`
- `abs://items/{item_id}/metadata-object`
- `abs://api-inventory/current`
- `abs://fixture/status`

`abs://fixture/status` may report whether a fixture token is present and its
length, but it must never expose the token value.

## MCP Prompts

Provide repeatable operator workflows as MCP prompts:

- `abs_library_audit`: read-only audit of library health.
- `abs_scan_troubleshooting`: safe scan diagnosis workflow that respects
  read-only mode.
- `abs_api_update_review`: workflow for reviewing upstream Audiobookshelf API
  changes and updating tools/tests.

Prompts must direct agents to use read-only tools first and avoid mutating tools
unless the user explicitly requests mutation and read-only mode is disabled.

## API Inventory Workflow

Maintain a generated route inventory under `docs/api-inventory`.

Required scripts:

- `scripts/generate_abs_api_inventory.py`: parse Audiobookshelf source,
  especially `server/routers/ApiRouter.js` and controller methods.
- `scripts/diff_abs_api_inventory.py`: compare generated inventory against the
  baseline.
- `scripts/test_api_inventory.py`: unit tests for inventory parsing/diff logic.

Required make targets:

- `abs-api-inventory`
- `abs-api-inventory-from-router`
- `abs-api-inventory-diff`
- `abs-api-inventory-check`

When upstream ABS routes change, update the client, MCP schemas, docs, and
fixture tests together.

## Docker Fixture

Provide a resettable local fixture under `test/abs`.

Fixture requirements:

- Compose project name: `abs-mcp`.
- Plain ABS service on `http://localhost:13388`.
- Metadata-enabled ABS service on `http://localhost:13389`.
- Separate runtime/state/config paths so it does not collide with other local
  Audiobookshelf projects.
- Committed sanitized baseline SQLite databases.
- Disposable fixture credentials and token in `.env.testing`.
- Public-domain staged media only.
- Scripts to seed media, reset runtime state, restore baseline config, wait for
  ABS, scan libraries, configure a fresh ABS instance, and capture a new
  baseline intentionally.

Required make targets:

- `abs-dev-seed`
- `abs-dev-init`
- `abs-dev-configure`
- `abs-dev-up`
- `abs-dev-wait`
- `abs-dev-down`
- `abs-dev-reset`
- `abs-dev-reset-all`
- `abs-dev-scan`
- `abs-dev-reset-scan`
- `abs-ci-smoke`
- `abs-test-integration`

## Local MCP Client Config Helpers

Provide `scripts/write_mcp_dev_config.py` to generate local ignored MCP client
config files from `test/abs/.env.testing`.

Requirements:

- Default output `.mcp.dev.json` is read-only.
- Optional read-write output for mutation testing.
- Generated files must be owner-only because they contain a fixture token.
- CLI output must redact the token.
- Tests must verify config shape and redaction.

## CI and Distribution

Add GitHub Actions workflows for validation and release.

CI workflow requirements:

- Trigger on pushes and pull requests targeting `master`.
- Run `make test-unit`.
- Run `go test ./...`.
- Build the native binary with `make build`.
- Build the Docker image without pushing it.

Release workflow requirements:

- Trigger on version tags matching `v*`.
- Build native archives for:
  - `linux/amd64`
  - `linux/arm64`
  - `darwin/amd64`
  - `darwin/arm64`
  - `windows/amd64`
  - `windows/arm64`
- Attach release archives to the GitHub release.
- Build and push a multi-arch Docker image to GitHub Container Registry.
- Tag the image with the pushed version tag and semver-derived aliases.
- Use the repository `GITHUB_TOKEN`; do not require personal tokens.

## Safety Rules

- Never log or return bearer tokens, API keys, cookies, or raw Authorization
  headers.
- Prefer exact IDs over fuzzy names for mutating tools.
- Keep `ABS_READ_ONLY=true` as the default.
- Require exact confirmation for destructive cleanup tools.
- Cap list/search limits to keep MCP responses bounded.
- Treat the public Audiobookshelf API docs as historical context; verify against
  source and fixture behavior.

## Test Plan

Baseline verification:

```bash
make test-unit
go test ./...
make build
```

Docker packaging verification:

```bash
make docker-build
```

Docker-backed verification when fixture behavior matters:

```bash
make abs-test-integration
```

Expected coverage:

- Config loading from env.
- Config loading from `--env-file`.
- Config loading from Cobra/Viper flags.
- Flag-over-env-over-env-file precedence.
- Extra header validation and Authorization rejection.
- Header file and repeated `--header` merge behavior.
- Corporate/private CA bundle trust.
- Explicit insecure TLS skip-verify fallback.
- ABS client URL validation, auth header behavior, error redaction, and endpoint
  request shape.
- MCP server tool registration and JSON-RPC/protocol behavior.
- Resource URI parsing and JSON output.
- Prompt registration and content.
- Read-only blocking for mutating tools.
- Confirmation and issue-count safeguards for cleanup.
- Fixture status redaction.
- Python helper scripts and API inventory utilities.
- Dockerfile builds a runnable image with the generated API inventory included.
- GitHub Actions workflows cover CI and tag-based release distribution.

## Rebuild Order

1. Create the Go module and directory layout.
2. Implement config loading and tests.
3. Implement the ABS client and client tests.
4. Implement MCP tools, resources, and prompts with fake-client unit tests.
5. Add stdio command startup and command transport tests.
6. Add API inventory scripts, generated baseline, and script tests.
7. Add Docker fixture scripts, compose file, baseline config, and integration
   tests.
8. Add Dockerfile and GitHub Actions workflows for CI and release distribution.
9. Add README and operator docs.
10. Run `make test-unit`, `go test ./...`, `make docker-build`, and Docker
    integration tests when the fixture is available.
