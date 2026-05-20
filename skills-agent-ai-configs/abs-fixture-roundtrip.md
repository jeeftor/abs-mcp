# ABS Fixture Round-Trip Agent Brief

## Purpose

Use a resettable Audiobookshelf Docker fixture to prove MCP tools behave correctly against real ABS instances.

## Fixture

Fixture path:

```text
test/abs
```

Known services:

- Plain metadata mode: `http://localhost:13388`
- Sidecar metadata mode: `http://localhost:13389`

Known Docker identity:

- Compose project: `abs-mcp`
- Plain container: `abs-mcp-abs-plain`
- Metadata-enabled container: `abs-mcp-abs-metadata`

Known libraries:

- `Audiobooks` mounted at `/audiobooks`
- `Ebooks` mounted at `/books`

## Workflow

1. Confirm Docker is available.
2. Reset the fixture using its native script or make target.
3. Wait for both ABS instances.
4. Load fixture environment from `.env.testing` unless the user provided overrides.
5. Run MCP server protocol tests against the plain instance.
6. Run the same read-only tests against the metadata-enabled instance.
7. Run mutating tests only when `ABS_READ_ONLY=false` and the test resets state before and after.
8. Capture only sanitized outputs.

## Minimum Round Trips

- Health check succeeds.
- Libraries list includes `Audiobooks` and `Ebooks`.
- Library item list works after scan.
- A forced scan can be triggered when mutating tools are enabled.
- Read-only mode blocks scan and cleanup tools.
- Bad token returns a safe auth error.

## Stop Conditions

Stop and report when:

- The fixture path is missing.
- Docker cannot start the services.
- Ports `13388` or `13389` are already owned by unrelated processes.
- Fixture reset fails.
- A token appears in command output, logs, or snapshots.

## Notes

- Do not edit the sibling Audiobook Organizer fixture unless explicitly requested.
- Prefer MCP-specific additions in this repo over changing the original fixture.
