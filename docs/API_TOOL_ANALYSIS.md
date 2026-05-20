# Audiobookshelf API Tool Analysis

## MCP Mapping Approach

Do not map every Audiobookshelf endpoint directly to an MCP tool. The better flow is:

1. Inventory current ABS API routes from source.
2. Classify endpoints by user intent, risk, and response size.
3. Promote only useful, agent-safe workflows into MCP tools.
4. Expose stable read-only state as MCP resources.
5. Add prompts for multi-step operational workflows.
6. Back every accepted tool with unit tests and Docker fixture round trips.

## Source Inputs

Primary source:

- Audiobookshelf router source: `server/routers/ApiRouter.js`
- Audiobookshelf controllers under `server/controllers/`

Current generated baseline:

- Source ref: `2d0a5462d2234a8c1f853c9c23b790dc8e690fb5`
- Source commit date: `2026-05-17T19:31:45Z`
- Inventory: `docs/api-inventory/generated/abs-api-inventory.json`
- Route count: 198 total, 83 read-only by HTTP method, 115 mutating by HTTP method

Useful current route groups observed in `ApiRouter.js`:

- Libraries: `/libraries`, `/libraries/:id`, `/libraries/:id/items`, `/libraries/:id/search`, `/libraries/:id/stats`, `/libraries/:id/scan`, `/libraries/:id/issues`
- Items: `/items/:id`, `/items/:id/media`, `/items/:id/scan`, `/items/:id/metadata-object`, `/items/:id/file/:fileid`, `/items/:id/ebook/:fileid?`
- Current user: `/me`, `/me/progress/:libraryItemId/:episodeId?`, `/me/items-in-progress`
- Search: `/search/books`, `/search/authors`, `/search/covers`, `/search/chapters`, `/search/providers`
- Collections and playlists: `/collections`, `/playlists`
- Authors and series: `/authors/:id`, `/series/:id`
- Admin/server: `/settings`, `/tasks`, `/stats/server`, `/api-keys`, `/backups`, `/notifications`, `/tools/*`

## Candidate MCP Tools

### Tier 1: Build First

These are read-heavy, broadly useful, and fixture-testable.

- `abs_health_check`
  - API basis: `/api/me` plus a lightweight library call.
  - Purpose: validate URL/auth and return sanitized server/user capability state.
  - Mutates: no.

- `abs_list_libraries`
  - API basis: `GET /api/libraries`.
  - Purpose: discover library IDs, names, media types, and folders.
  - Mutates: no.

- `abs_get_library`
  - API basis: `GET /api/libraries/:id`.
  - Purpose: inspect one library before using its ID in later tools.
  - Mutates: no.

- `abs_list_library_items`
  - API basis: `GET /api/libraries/:id/items`.
  - Purpose: paginated item discovery with bounded output.
  - Mutates: no.

- `abs_get_library_item`
  - API basis: `GET /api/items/:id`.
  - Purpose: inspect item metadata, paths, media files, missing/invalid state.
  - Mutates: no.

- `abs_scan_library`
  - API basis: `POST /api/libraries/:id/scan`.
  - Purpose: trigger a library scan for round-trip tests and operations.
  - Mutates: yes. Must be blocked when `ABS_READ_ONLY=true`.

### Tier 2: Add After Source/Fixture Verification

- `abs_search_library`
  - API basis: `GET /api/libraries/:id/search`.
  - Purpose: scoped library search.
  - Mutates: no.

- `abs_get_library_stats`
  - API basis: `GET /api/libraries/:id/stats`.
  - Purpose: summarize library health and size.
  - Mutates: no.

- `abs_get_filter_data`
  - API basis: `GET /api/libraries/:id/filterdata`.
  - Purpose: discover genres, tags, authors, narrators, and series filters.
  - Mutates: no.

- `abs_get_item_metadata_object`
  - API basis: `GET /api/items/:id/metadata-object`.
  - Purpose: inspect ABS metadata sidecar payloads.
  - Mutates: no.

- `abs_scan_item`
  - API basis: `POST /api/items/:id/scan`.
  - Purpose: rescan a single item after targeted changes.
  - Mutates: yes. Must be blocked in read-only mode.

- `abs_remove_library_items_with_issues`
  - API basis: `DELETE /api/libraries/:id/issues`.
  - Purpose: cleanup missing/invalid rows after a verified workflow.
  - Mutates: yes. Require explicit confirmation input and fixture coverage.

### Tier 3: Defer

These are useful but riskier, larger, or less central to MCP-first workflows.

- Metadata mutation: `PATCH /api/items/:id/media`
- Cover upload/update/delete
- File download and ebook file retrieval
- Collections and playlists mutation
- Playback sessions and progress mutation
- Podcast download and matching operations
- Server settings, backups, notifications, API keys, cache, and tools endpoints

## Candidate MCP Resources

- `abs://server/me`
- `abs://libraries`
- `abs://libraries/{library_id}`
- `abs://libraries/{library_id}/items?limit=...`
- `abs://items/{item_id}`
- `abs://fixture/status`
- `abs://api-inventory/current`

Resources should stay read-only and bounded. Large item lists should require pagination.

## Candidate MCP Prompts

- `abs_library_audit`
  - Use libraries, stats/filter data, item lists, and missing-state checks to summarize library health.

- `abs_scan_troubleshooting`
  - Check fixture/server health, trigger scan when allowed, inspect tasks/items, and report likely scan issues.

- `abs_api_update_review`
  - Compare generated ABS route inventory to the committed baseline and identify MCP schema/test changes.

## Acceptance Rules

Before adding any tool:

- Identify the ABS route and controller source.
- Mark whether the operation mutates server state, filesystem state, metadata, or user data.
- Define bounded input and output schemas.
- Add read-only mode behavior if mutating.
- Add unit tests with a fake or `httptest` ABS client.
- Add a Docker fixture round trip when the behavior depends on real ABS.
