# Audiobookshelf MCP Server

`abs-mcp` is a Go MCP server for inspecting and safely operating an Audiobookshelf instance.

The current implementation exposes the first planned tool slice:

- `abs_health_check`
- `abs_list_libraries`
- `abs_get_library`
- `abs_list_library_items`
- `abs_get_library_item`
- `abs_search_library`
- `abs_get_library_stats`
- `abs_get_filter_data`
- `abs_get_item_metadata_object`
- `abs_scan_library`
- `abs_scan_library_and_wait`
- `abs_scan_item`
- `abs_remove_library_items_with_issues`

Scan tools are blocked by default because `ABS_READ_ONLY` defaults to `true`.

It also exposes these MCP resources:

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

Prompts:

- `abs_library_audit`
- `abs_scan_troubleshooting`
- `abs_api_update_review`

## Configuration

Environment variables are the preferred configuration path for MCP clients,
containers, and other launchers that inject secrets:

```bash
export ABS_BASE_URL=http://localhost:13388
export ABS_API_KEY=...
export ABS_READ_ONLY=true
export ABS_TIMEOUT=30s
export ABS_FIXTURE_DIR=test/abs
export ABS_EXTRA_HEADERS_FILE=/path/to/headers.json
```

The server also accepts matching Cobra/Viper CLI flags. Explicit flags override
environment variables:

```bash
go run ./cmd/abs-mcp \
  --base-url http://localhost:13388 \
  --api-key ... \
  --read-only=true \
  --timeout 30s \
  --fixture-dir test/abs \
  --extra-headers-file /path/to/headers.json
```

Prefer `ABS_API_KEY` over `--api-key` outside short local debugging sessions so
tokens do not land in shell history or process listings.

| Environment variable | CLI flag | Default |
| --- | --- | --- |
| `ABS_BASE_URL` | `--base-url` | required |
| `ABS_API_KEY` | `--api-key` | required |
| `ABS_READ_ONLY` | `--read-only` | `true` |
| `ABS_TIMEOUT` | `--timeout` | `30s` |
| `ABS_FIXTURE_DIR` | `--fixture-dir` | `test/abs` |
| `ABS_EXTRA_HEADERS_FILE` | `--extra-headers-file` | unset |

`ABS_EXTRA_HEADERS_FILE` is optional. When set, it must point to a JSON object of string header names to string values, for example `{"X-Corp-Trace":"trace-1"}`. `Authorization` is rejected there; use `ABS_API_KEY` for Audiobookshelf authentication.

Run over MCP stdio:

```bash
go run ./cmd/abs-mcp
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

## Tests

Run unit and protocol tests:

```bash
make test-unit
```

Run Docker-backed Audiobookshelf integration tests:

```bash
make abs-test-integration
```

The integration target resets and scans the repo-local ABS fixture before running tests.

Stop fixture containers when done:

```bash
make abs-dev-down
```
