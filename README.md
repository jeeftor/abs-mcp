<p align="center">
  <img src="docs/assets/abs-mcp-logo.svg" width="112" alt="Audiobookshelf MCP logo">
</p>

<h1 align="center">Audiobookshelf MCP Server</h1>

<p align="center">
  A Go MCP server for inspecting and safely operating Audiobookshelf libraries.
</p>

`abs-mcp` exposes safe, typed MCP tools and resources for agents that need to
inspect Audiobookshelf libraries, diagnose scans, and optionally trigger bounded
maintenance workflows.

## Table of Contents

- [Highlights](#highlights)
- [Audiobook Organizer Compatibility](#audiobook-organizer-compatibility)
- [AI Generated Comparison - Last updated 2026-05-20](#ai-generated-comparison---last-updated-2026-05-20)
- [Quick Start](#quick-start)
- [MCP Surface](#mcp-surface)
- [Configuration](#configuration)
- [Safety](#safety)
- [Installation](#installation)
  - [Client Configs](#client-configs)
- [Local Development](#local-development)
- [Tests](#tests)
- [CI and Releases](#ci-and-releases)
- [MCP Registry](#mcp-registry)

## Highlights

- Read-only by default; scan and cleanup tools require `ABS_READ_ONLY=false`.
- Ships as a local stdio MCP server from a single Go binary or Docker image;
  Streamable HTTP is a good future fit for hosted or multi-client deployments.
- Supports env vars, Docker-style env files, extra headers, and custom TLS CA
  bundles.
- Includes source-backed Audiobookshelf API inventory resources and repeatable
  fixture tests.
- Publishes release binaries and a GHCR image suitable for official MCP
  Registry metadata.

## Audiobook Organizer Compatibility

I built this MCP server to work especially well alongside
[jeeftor/audiobook-organizer](https://github.com/jeeftor/audiobook-organizer).
While it exposes the normal Audiobookshelf MCP tools for inspecting libraries,
items, metadata, and server state, one of its most useful workflows is auditing
whether books are actually organized the way you expect on disk.

The `abs_find_misorganized_items` tool checks Audiobookshelf item paths against
supported folder layout conventions, including author/title and
author/series/title structures. It is audit-only, so it reports likely
misconfigured or misplaced books without moving or deleting files. That makes it
a good companion to Audiobook Organizer: use this MCP server to identify layout
problems from Audiobookshelf's perspective, then use Audiobook Organizer to
clean up or standardize the underlying files.

<!-- AI-GENERATED-COMPARISON:START -->
## AI Generated Comparison - Last updated 2026-05-20

This comparison is generated from public project READMEs, registry pages, and
this repository's current docs. It is descriptive rather than a recommendation.

| Server | Shape | Confirmed strengths | Safety posture | Difference from `jeeftor/abs-mcp` |
| --- | --- | --- | --- | --- |
| [`michaeldvinci/audiobookshelf-mcp`](https://github.com/michaeldvinci/audiobookshelf-mcp) | Go stdio server with release binaries. | Broad general Audiobookshelf management, including libraries, items, authors, collections, playlists, user info, sessions, podcasts, progress updates, and backups. | Exposes mutating tools; no global default read-only gate was found in the public README during this comparison pass. | Broader generic management surface, but less conservative. No public evidence was found for a misorganized-file audit, source-backed API inventory, MCP resources/prompts, cover removal/update, or chapter update tooling. |
| [`sandymac/audiobookshelf-mcp`](https://github.com/sandymac/audiobookshelf-mcp) | Rust server with stdio plus HTTP/SSE support. | Read/query surface for libraries, search, progress, stats, recent sessions, and optional progress/bookmark mutations. | Mutating tools are disabled by default and must be explicitly enabled. HTTP mode recommends bearer auth and TLS proxying. | Similar safety model, but narrower feature scope. No public evidence was found for metadata, cover, chapter repair tools, organizer-oriented audits, or fixture/API-inventory workflows. |
| [`sierikov/audiobookshelf-mcp`](https://github.com/sierikov/audiobookshelf-mcp) | Go server with read-oriented tooling and release binaries. | Read-only browsing and search across libraries, items, progress, stats, sessions, series, authors, and collections. | Public README presents it as read-only. | Useful inspection surface, but not a controlled repair workflow. No public evidence was found for mutating metadata, cover, chapter, cleanup, or organizer-focused audit tooling. |
| [`ForceConstant/audiobookshelf_mcp`](https://github.com/ForceConstant/audiobookshelf_mcp) | Generated OpenAPI MCP bridge with streamable HTTP and Docker-oriented files. | Broad generated API exposure from Audiobookshelf OpenAPI material. | Not determinable from the public README; generated API exposure may include mutating endpoints, but the README does not enumerate safety controls. | Less curated and less operator-specific. This project intentionally exposes bounded, typed tools with read-only gating and fixture-backed behavior checks instead of exposing every route directly. |

Weak or placeholder hits, such as
[`schmidt-software/mcp-audiobookshelf`](https://github.com/schmidt-software/mcp-audiobookshelf),
were excluded when no implementation or feature evidence was available.
<!-- AI-GENERATED-COMPARISON:END -->

## Quick Start

Run a downloaded release binary over MCP stdio:

```bash
ABS_BASE_URL=http://localhost:13378 \
ABS_API_KEY=... \
ABS_READ_ONLY=true \
/path/to/abs-mcp
```

Or run the container image:

```bash
docker run --rm -i \
  -e ABS_BASE_URL=http://host.docker.internal:13378 \
  -e ABS_API_KEY=... \
  -e ABS_READ_ONLY=true \
  ghcr.io/jeeftor/abs-mcp:0.1.1
```

For client-specific snippets, see [Client Configs](#client-configs).

## MCP Surface

### Tools

- `abs_health_check`
- `abs_list_libraries`
- `abs_get_library`
- `abs_list_library_items`
- `abs_get_library_item`
- `abs_search_library`
- `abs_get_library_stats`
- `abs_get_filter_data`
- `abs_get_item_metadata_object`
- `abs_find_misorganized_items`
- `abs_scan_library`
- `abs_scan_library_and_wait`
- `abs_scan_item`
- `abs_update_item_metadata`
- `abs_update_item_cover`
- `abs_remove_item_cover`
- `abs_match_item`
- `abs_update_item_chapters`
- `abs_update_item_tracks`
- `abs_create_collection`
- `abs_update_collection`
- `abs_delete_collection`
- `abs_add_collection_item`
- `abs_remove_collection_item`
- `abs_create_playlist`
- `abs_update_playlist`
- `abs_delete_playlist`
- `abs_add_playlist_item`
- `abs_remove_playlist_item`
- `abs_remove_library_items_with_issues`

Mutating tools are blocked by default because `ABS_READ_ONLY` defaults to
`true`. Scan tools, issue cleanup, `abs_update_item_cover`,
`abs_remove_item_cover`, and `abs_update_item_chapters` are implemented.
Remaining planned mutating tools, including broad metadata updates and item
matching, are advertised for discovery but return a not-implemented error after
read-only and confirmation checks until their ABS source and fixture behavior
are verified.

### Resources

- `abs://server/info`
- `abs://libraries`
- `abs://libraries/{library_id}`
- `abs://libraries/{library_id}/items?limit=...&offset=...`
- `abs://libraries/{library_id}/stats`
- `abs://libraries/{library_id}/filterdata`
- `abs://items/{item_id}`
- `abs://items/{item_id}/metadata-object`
- `abs://api-inventory/current`
- `abs://fixture/status`

### Prompts

- `abs_library_audit`
- `abs_scan_troubleshooting`
- `abs_api_update_review`

See [docs/tools.md](docs/tools.md) for tool inputs, output shapes, mutation
behavior, and common errors.

## Configuration

Environment variables and env files are the preferred configuration paths for
MCP clients, containers, and other launchers that inject secrets:

```bash
export ABS_BASE_URL=http://localhost:13388
export ABS_API_KEY=...
export ABS_READ_ONLY=true
export ABS_TIMEOUT=30s
export ABS_FIXTURE_DIR=test/abs
export ABS_EXTRA_HEADERS_FILE=/path/to/headers.json
export ABS_TLS_CA_CERT_FILE=/path/to/corporate-ca.pem
export ABS_TLS_INSECURE_SKIP_VERIFY=false
```

The server can load those same values from a Docker-style env file:

```bash
go run ./cmd/abs-mcp --env-file /path/to/.env
```

The server also accepts matching Cobra/Viper CLI flags. Precedence is explicit
CLI flags, then process environment variables, then `--env-file`, then built-in
defaults:

```bash
go run ./cmd/abs-mcp \
  --env-file /path/to/.env \
  --base-url http://localhost:13388 \
  --api-key ... \
  --read-only=true \
  --timeout 30s \
  --fixture-dir test/abs \
  --extra-headers-file /path/to/headers.json \
  --header 'CF-Access-Client-Id=...' \
  --header 'CF-Access-Client-Secret=...' \
  --tls-ca-cert-file /path/to/corporate-ca.pem
```

Prefer `ABS_API_KEY` over `--api-key` outside short local debugging sessions so
tokens do not land in shell history or process listings.

| Environment variable | CLI flag | Default |
| --- | --- | --- |
| n/a | `--env-file` | unset |
| `ABS_BASE_URL` | `--base-url` | required |
| `ABS_API_KEY` | `--api-key` | required |
| `ABS_READ_ONLY` | `--read-only` | `true` |
| `ABS_TIMEOUT` | `--timeout` | `30s` |
| `ABS_FIXTURE_DIR` | `--fixture-dir` | `test/abs` |
| `ABS_EXTRA_HEADERS_FILE` | `--extra-headers-file` | unset |
| n/a | `--header NAME=VALUE` | unset |
| `ABS_TLS_CA_CERT_FILE` | `--tls-ca-cert-file` | unset |
| `ABS_TLS_INSECURE_SKIP_VERIFY` | `--tls-insecure-skip-verify` | `false` |

`ABS_EXTRA_HEADERS_FILE` is optional. When set, it must point to a JSON object
of string header names to string values, for example
`{"X-Corp-Trace":"trace-1"}`. `Authorization` is rejected there; use
`ABS_API_KEY` for Audiobookshelf authentication.

`--env-file` supports simple Docker-style dotenv lines such as `KEY=value`,
`KEY="value"`, `KEY='value'`, blank lines, comments, and optional `export`
prefixes. Unknown keys are ignored by the MCP server.

Use `--header NAME=VALUE` for quick local header injection. It is repeatable,
and duplicate names override values from `ABS_EXTRA_HEADERS_FILE`. Prefer the
file for secrets such as Cloudflare Access credentials because CLI flags can
show up in shell history and process listings.

For private or corporate TLS certificates, prefer `ABS_TLS_CA_CERT_FILE` or
`--tls-ca-cert-file` with a PEM CA bundle. Use
`ABS_TLS_INSECURE_SKIP_VERIFY=true` or `--tls-insecure-skip-verify` only as a
temporary fallback while fixing local trust.

## Safety

`abs-mcp` defaults to read-only mode. With `ABS_READ_ONLY=true`, all mutating
tools are blocked before making Audiobookshelf API calls.

These tools can mutate Audiobookshelf state and require `ABS_READ_ONLY=false`:

- `abs_scan_library`
- `abs_scan_library_and_wait`
- `abs_scan_item`
- `abs_update_item_cover`
- `abs_remove_item_cover`
- `abs_update_item_chapters`
- `abs_update_item_metadata` (planned; not implemented)
- `abs_match_item` (planned; not implemented)
- `abs_update_item_tracks`
- `abs_create_collection`
- `abs_update_collection`
- `abs_delete_collection`
- `abs_add_collection_item`
- `abs_remove_collection_item`
- `abs_create_playlist`
- `abs_update_playlist`
- `abs_delete_playlist`
- `abs_add_playlist_item`
- `abs_remove_playlist_item`
- `abs_remove_library_items_with_issues`

The newly advertised item, collection, and playlist mutation tools are stubs:
they validate read-only mode and destructive confirmations, then return a clear
not-implemented error until their Audiobookshelf source and Docker fixture
behavior are verified.

`abs_remove_library_items_with_issues` also requires the exact confirmation
string `remove issues from <libraryId>` and can check an expected issue count
before it asks Audiobookshelf to remove missing or invalid items.

Future destructive tools should follow the same pattern: `ABS_READ_ONLY=false`
must be necessary but not sufficient, and the tool should require an explicit
confirmation input before deleting, removing, purging, replacing, overwriting,
or broadly batch-mutating Audiobookshelf data.

The server requires an Audiobookshelf base URL and API key or bearer token.
Prefer an Audiobookshelf API key with the least permissions needed for the
tools you plan to expose. Bearer tokens, API keys, raw `Authorization` headers,
and extra header values are not logged by this server. `Authorization` is
rejected in `ABS_EXTRA_HEADERS_FILE`; use `ABS_API_KEY` for Audiobookshelf
authentication.

## Installation

Download a release archive from the
[GitHub releases page](https://github.com/jeeftor/abs-mcp/releases), unpack it,
and point your MCP client at the `abs-mcp` binary. The server speaks MCP over
stdio.

For local development, you can also run the server from source:

```bash
go run ./cmd/abs-mcp
```

### Client Configs

Claude Desktop, Cursor, VS Code, and Windsurf all support stdio MCP servers
with a command, arguments, and environment values. Use an absolute binary path
and prefer `env` or `--env-file` for secrets.

Claude Desktop `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "Audiobookshelf": {
      "command": "/path/to/abs-mcp",
      "args": [
        "--env-file",
        "/path/to/abs-mcp.env",
        "--extra-headers-file",
        "/path/to/cf-headers.json"
      ]
    }
  }
}
```

Claude Code:

```bash
claude mcp add Audiobookshelf /path/to/abs-mcp \
  -e ABS_BASE_URL=http://localhost:13378 \
  -e ABS_API_KEY=... \
  -e ABS_READ_ONLY=true
```

Cursor `mcp.json`:

```json
{
  "mcpServers": {
    "Audiobookshelf": {
      "command": "/path/to/abs-mcp",
      "env": {
        "ABS_BASE_URL": "http://localhost:13378",
        "ABS_API_KEY": "...",
        "ABS_READ_ONLY": "true"
      }
    }
  }
}
```

VS Code MCP config:

```json
{
  "servers": {
    "Audiobookshelf": {
      "type": "stdio",
      "command": "/path/to/abs-mcp",
      "env": {
        "ABS_BASE_URL": "http://localhost:13378",
        "ABS_API_KEY": "...",
        "ABS_READ_ONLY": "true"
      }
    }
  }
}
```

Windsurf MCP config using an env file:

```json
{
  "mcpServers": {
    "Audiobookshelf": {
      "command": "/path/to/abs-mcp",
      "args": [
        "--env-file",
        "/path/to/abs-mcp.env"
      ]
    }
  }
}
```

Docker-based stdio config:

```json
{
  "mcpServers": {
    "Audiobookshelf": {
      "command": "docker",
      "args": [
        "run",
        "--rm",
        "-i",
        "-e",
        "ABS_BASE_URL=http://host.docker.internal:13378",
        "-e",
        "ABS_API_KEY",
        "-e",
        "ABS_READ_ONLY=true",
        "ghcr.io/jeeftor/abs-mcp:0.1.1"
      ],
      "env": {
        "ABS_API_KEY": "..."
      }
    }
  }
}
```

Run the container image directly:

```bash
docker run --rm -i \
  -e ABS_BASE_URL=http://host.docker.internal:13388 \
  -e ABS_API_KEY=... \
  -e ABS_READ_ONLY=true \
  ghcr.io/jeeftor/abs-mcp:0.1.1
```

With Cloudflare Access headers and a corporate/private CA bundle:

```bash
docker run --rm -i \
  -e ABS_BASE_URL=https://abs.example.com \
  -e ABS_API_KEY=... \
  -e ABS_EXTRA_HEADERS_FILE=/run/secrets/abs-headers.json \
  -e ABS_TLS_CA_CERT_FILE=/run/secrets/corporate-ca.pem \
  -v /path/to/headers.json:/run/secrets/abs-headers.json:ro \
  -v /path/to/corporate-ca.pem:/run/secrets/corporate-ca.pem:ro \
  ghcr.io/jeeftor/abs-mcp:0.1.1
```

Build a local image:

```bash
make docker-build
```

## Local Development

Bring up the repo-local Audiobookshelf fixture, scan the staged test media,
build the MCP server, and write a local MCP client config:

```bash
make dev
```

That target leaves Audiobookshelf running on `http://localhost:13388` and writes
`.mcp.dev.json`. The generated config points at `bin/abs-mcp`, includes the
fixture token from `test/abs/.env.testing`, sets `ABS_READ_ONLY=true`, and is
gitignored.

Use this command when you need mutating scan tools enabled in a local client:

```bash
make mcp-dev-config-read-write
```

Stop the fixture when done:

```bash
make abs-dev-down
```

Install the local pre-commit hooks:

```bash
prek install
prek install --hook-type commit-msg
```

Run all hooks manually:

```bash
prek run --all-files
```

## Tests

Run unit and protocol tests:

```bash
make test-unit
```

Run Docker-backed Audiobookshelf integration tests:

```bash
make abs-test-integration
```

The integration target resets and scans the repo-local ABS fixture before
running tests.

Stop fixture containers when done:

```bash
make abs-dev-down
```

## CI and Releases

GitHub Actions runs unit tests, Go package tests, a binary build, and a Docker
image build on pushes and pull requests to `master`.

Tags matching `v*` publish release archives for Linux, macOS, and Windows on
amd64 and arm64. The release workflow also publishes a multi-arch Docker image
to `ghcr.io/jeeftor/abs-mcp`.

## MCP Registry

This repository is prepared for the official MCP Registry using the OCI package
path:

- Registry name: `io.github.jeeftor/abs-mcp`
- Package: `ghcr.io/jeeftor/abs-mcp:<version>`
- Transport: `stdio`
- Metadata file: `server.json`

The Docker image includes the required MCP ownership label
`io.modelcontextprotocol.server.name=io.github.jeeftor/abs-mcp`.

On `v*` tags, the release workflow builds the immutable GHCR image tag, rewrites
`server.json` to the tag version, authenticates to the MCP Registry with GitHub
OIDC, and publishes with `mcp-publisher`. GitHub OIDC does not require a
dedicated registry secret.

After the official registry entry is published, downstream aggregators can pick
it up from the registry API. Glama is the next practical listing target.
Smithery should wait until this project either ships an MCPB bundle for stdio
distribution or adds a public Streamable HTTP transport with appropriate auth.
