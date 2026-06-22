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

- Source ref: `e70e4b9d40a6251897e114c4154add8c05ad0944`
- Source commit date: `2026-05-30T20:43:50Z`
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

- `abs_search_ebooks`
  - API basis: `GET /api/libraries/:id/items`, filtered locally to items with
    ebook files.
  - Purpose: find ebook item IDs before send-to-device workflows.
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

- `abs_get_items_in_progress`
  - API basis: `GET /api/me/items-in-progress`.
  - Purpose: inspect in-progress items for the configured ABS user.
  - Mutates: no.

- `abs_get_item_progress`
  - API basis: `GET /api/me/progress/:id/:episodeId?`.
  - Purpose: inspect progress for one item or podcast episode for the configured ABS user.
  - Mutates: no.

- `abs_list_bookmarks`
  - API basis: `GET /api/me`, extracting only `bookmarks`.
  - Purpose: inspect bookmarks for the configured ABS user without exposing the full user payload.
  - Mutates: no.

- `abs_list_backups`
  - API basis: `GET /api/backups`.
  - Purpose: inspect server backup records visible to the configured ABS token.
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

- Broad or raw metadata mutation beyond the typed allowlist
- Cover upload
- File download and ebook file retrieval
- Destructive collection and playlist removal/deletion
- Playback sessions and progress mutation
- Podcast download and matching operations
- Server settings, backups, notifications, API keys, cache, and tools endpoints

## Mutating Endpoint Backlog

Current inventory has 115 mutating routes. All future tools mapped to these
routes must require `ABS_READ_ONLY=false`. Destructive routes also require an
explicit confirmation input, preferably an exact phrase containing the relevant
ABS ID and an expected affected-record count when the server can preview it.

Already exposed:

- `abs_scan_library`, `abs_scan_library_and_wait`: `POST /api/libraries/:id/scan`
- `abs_scan_item`: `POST /api/items/:id/scan`
- `abs_remove_library_items_with_issues`: `DELETE /api/libraries/:id/issues`,
  with exact confirmation and optional expected issue count.
- `abs_update_item_metadata`: `PATCH /api/items/:id/media`, restricted to a
  typed allowlist of source-verified catalog fields.
- `abs_update_item_cover`: `PATCH /api/items/:id/cover`
- `abs_remove_item_cover`: `DELETE /api/items/:id/cover`, with exact
  confirmation.
- `abs_update_item_chapters`: `POST /api/items/:id/chapters`, with an expected
  chapter-count guard.
- `abs_create_collection`, `abs_update_collection`, `abs_add_collection_item`:
  `POST /api/collections`, `PATCH /api/collections/:id`, and
  `POST /api/collections/:id/book`.
- `abs_create_playlist`, `abs_update_playlist`, `abs_add_playlist_item`:
  `POST /api/playlists`, `PATCH /api/playlists/:id`, and
  `POST /api/playlists/:id/item`.
- `abs_update_item_progress`: `PATCH /api/me/progress/:libraryItemId/:episodeId?`,
  scoped to the configured ABS user.
- `abs_create_bookmark`, `abs_update_bookmark`: `POST|PATCH
  /api/me/item/:id/bookmark`, scoped to the configured ABS user.
- `abs_create_backup`: `POST /api/backups`, blocked by read-only mode and
  limited to backup creation only.
- `abs_send_ebook_to_device`: `POST /api/emails/send-ebook-to-device`, blocked
  by read-only mode and limited to sending an existing ebook item to a saved
  device name that Audiobookshelf verifies the configured user can access.
- `abs_send_ebook_by_query`: local ebook search plus
  `POST /api/emails/send-ebook-to-device`, blocked by read-only mode, limited to
  exactly one query match, and guarded by an exact confirmation string containing
  the resolved item ID and device name.
- `abs_list_ereader_devices`: `GET /api/emails/settings`, read-only but
  admin-scoped in Audiobookshelf; returns only sanitized ereader device metadata
  and intentionally omits SMTP settings and saved device email addresses.

High-fit future candidates:

- Additional raw metadata payload support for `PATCH /api/items/:id/media`
  should remain deferred unless a source-verified workflow requires fields that
  are not covered by the typed `abs_update_item_metadata` allowlist.
- Cover upload: `POST /api/items/:id/cover`. Upload behavior needs source and
  fixture proof before exposure.
- `abs_match_item`: `POST /api/items/:id/match`. Potentially useful after
  misorganization or metadata audits; needs source review of overwrite behavior
  before exposure.
- `abs_update_item_tracks`: `PATCH /api/items/:id/tracks`. Useful for repair
  workflows, but schema and media-type behavior need source and fixture proof.
- Destructive collection and playlist management: `DELETE /api/collections/:id`,
  `DELETE /api/collections/:id/book/:bookId`, `DELETE /api/playlists/:id`, and
  `DELETE /api/playlists/:id/item/:libraryItemId/:episodeId?`. Delete/remove
  workflows require confirmation and fixture proof.
- Destructive current-user progress/bookmark cleanup:
  `DELETE /me/progress/:id` and `DELETE /me/item/:id/bookmark/:time`. These
  should require confirmation because they remove user state.

Lower-fit or admin-heavy candidates:

- Library create/update/delete/order and `POST /api/libraries/:id/remove-metadata`.
  These are admin-level operations; delete and metadata removal require strong
  confirmation and fixture-safe tests.
- Author, series, genre, tag, narrator, and sorting-prefix mutations. Useful for
  catalog cleanup, but many are global or broad; rename/delete operations need
  preview and confirmation.
- Podcast download, match, episode update, OPML, RSS feed, and share mutations.
  These are workflow-specific and should wait for a concrete user workflow.
- Backup restore/apply, delete, upload, download, and backup-path changes.
  These are destructive, file-transfer oriented, or broad server configuration
  workflows and should stay deferred until a concrete operator workflow is
  requested and fixture-tested.
- `POST /api/tools/*`, cache purge, settings, auth settings, users, API keys,
  notifications, email settings other than sanitized ereader device listing and
  narrow ebook send routes, sessions, upload, watcher, and server admin
  endpoints. These should stay deferred unless an explicit admin workflow is
  requested and tested in the fixture.

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
