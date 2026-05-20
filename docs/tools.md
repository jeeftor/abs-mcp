# Audiobookshelf MCP Reference

This is the implemented MCP surface for `abs-mcp`. Outputs are JSON structured
content unless noted. Mutating tools are blocked when `ABS_READ_ONLY=true`, which
is the default.

Common ABS client failures include authentication or permission errors, missing
libraries or items, non-2xx Audiobookshelf HTTP responses, network failures, and
response decode errors.

## Tools

| Tool | Purpose | Key inputs | Output summary | Behavior | Common errors |
| --- | --- | --- | --- | --- | --- |
| `abs_health_check` | Validate authenticated ABS access. | None. | `ok`, configured `baseUrl`, `readOnly`, authenticated `username`/`userType`, visible `libraryCount`. | Read-only. | Current user or library list request fails. |
| `abs_list_libraries` | List libraries visible to the token. | None. | `libraries[]` with `id`, `name`, `mediaType`, `folders[]`; `count`. | Read-only. | Library list request fails. |
| `abs_get_library` | Get one library summary. | `libraryId` required. | `library` with `id`, `name`, `mediaType`, `folders[]`. | Read-only. | Missing `libraryId`; ABS cannot find or return the library. |
| `abs_list_library_items` | List one bounded page of library items. | `libraryId` required; optional `limit` default `25`, cap `100`; `offset` default `0`; `sort`; `desc`; `filter`; `include[]`; `minified`; `collapseSeries`. | `items[]` with item IDs, paths, missing/invalid flags, title/author/series, files, duration, size; `total`, `limit`, `offset`, `page`, `count`, requested sort/filter fields. | Read-only. | Missing `libraryId`; negative `limit` or `offset`; `offset` not a multiple of `limit`; ABS list request fails. |
| `abs_get_library_item` | Get one item summary. | `itemId` required. | `item` with ID, library ID, media type, path, missing/invalid flags, title/author/series, files, duration, size. | Read-only. | Missing `itemId`; ABS cannot find or return the item. |
| `abs_search_library` | Search one library. | `libraryId` and `query` required; optional `limit` default `12`, cap `50`. | `libraryId`, `query`, normalized `limit`, and raw ABS search `data`. | Read-only. | Missing `libraryId` or `query`; negative `limit`; ABS search request fails. |
| `abs_get_library_stats` | Read ABS stats for one library. | `libraryId` required. | `libraryId` and raw ABS stats `data`. | Read-only. | Missing `libraryId`; ABS stats request fails. |
| `abs_get_filter_data` | Read ABS filter data for one library. | `libraryId` required. | `libraryId` and raw ABS filter `data`. | Read-only. | Missing `libraryId`; ABS filterdata request fails. |
| `abs_get_item_metadata_object` | Read the raw metadata object for one item. | `itemId` required. | `itemId` and raw ABS metadata-object `data`. | Read-only. | Missing `itemId`; ABS metadata-object request fails, including permission failures. |
| `abs_find_misorganized_items` | Audit item folders against author/title or author/series/title layout conventions. | `libraryId` required; optional `convention` of `auto`, `author-title`, or `author-series-title`; optional `limit` default `50`, cap `200`; optional `includeOrganized`. | Counts for checked, organized, misorganized, and unclassifiable items; returned item findings with current/expected relative paths, reasons, confidence, and missing/invalid flags. | Read-only. | Missing `libraryId`; unsupported convention; negative `limit`; ABS library or item-list request fails. |
| `abs_scan_library` | Trigger a library scan. | `libraryId` required; `force` boolean. | `triggered`, `libraryId`, `force`. | Mutates ABS scan state; blocked by read-only mode. | `ABS_READ_ONLY=true`; missing `libraryId`; ABS scan request fails. |
| `abs_scan_library_and_wait` | Trigger a library scan, then poll item count. | `libraryId` required; `force`; optional `expectedTotal`; `timeoutSeconds` default `30`, cap `300`; `pollIntervalMilliseconds` default `1000`, cap `60000`. | `triggered`, `completed`, `timedOut`, `observedTotal`, `attempts`, timeout/poll settings, elapsed milliseconds. Timeout is returned as status, not an error. | Mutates ABS scan state; blocked by read-only mode. | `ABS_READ_ONLY=true`; missing `libraryId`; negative `expectedTotal`, timeout, or poll interval; scan or observation request fails. |
| `abs_scan_item` | Rescan one directory-backed item. | `itemId` required. | `triggered`, `itemId`, optional ABS `result`. | Mutates ABS scan state; blocked by read-only mode. | `ABS_READ_ONLY=true`; missing `itemId`; ABS item scan request fails. |
| `abs_update_item_metadata` | Planned item metadata update. | `itemId` required; `payload` reserved until source verification. | Not implemented yet. | Planned mutation; advertised for discovery; blocked by read-only mode. | `ABS_READ_ONLY=true`; missing `itemId`; not implemented until ABS source and fixture behavior are verified. |
| `abs_update_item_cover` | Planned item cover update. | `itemId` required; `payload` reserved until source verification. | Not implemented yet. | Planned mutation; advertised for discovery; blocked by read-only mode. | `ABS_READ_ONLY=true`; missing `itemId`; not implemented until ABS source and fixture behavior are verified. |
| `abs_remove_item_cover` | Planned item cover removal. | `itemId` required; `confirmation` must exactly equal `remove cover from <itemId>`. | Not implemented yet. | Planned destructive mutation; advertised for discovery; blocked by read-only mode and requires confirmation. | `ABS_READ_ONLY=true`; missing `itemId`; wrong confirmation; not implemented until ABS source and fixture behavior are verified. |
| `abs_match_item` | Planned item metadata match. | `itemId` required; optional `provider`; `payload` reserved until source verification. | Not implemented yet. | Planned mutation; advertised for discovery; blocked by read-only mode. | `ABS_READ_ONLY=true`; missing `itemId`; not implemented until ABS source and fixture behavior are verified. |
| `abs_update_item_chapters` | Planned item chapter update. | `itemId` required; `payload` reserved until source verification. | Not implemented yet. | Planned mutation; advertised for discovery; blocked by read-only mode. | `ABS_READ_ONLY=true`; missing `itemId`; not implemented until ABS source and fixture behavior are verified. |
| `abs_update_item_tracks` | Planned item track update. | `itemId` required; `payload` reserved until source verification. | Not implemented yet. | Planned mutation; advertised for discovery; blocked by read-only mode. | `ABS_READ_ONLY=true`; missing `itemId`; not implemented until ABS source and fixture behavior are verified. |
| `abs_create_collection` | Planned collection creation. | `name` required; `payload` reserved until source verification. | Not implemented yet. | Planned mutation; advertised for discovery; blocked by read-only mode. | `ABS_READ_ONLY=true`; missing `name`; not implemented until ABS source and fixture behavior are verified. |
| `abs_update_collection` | Planned collection update. | `collectionId` required; optional `name`; `payload` reserved until source verification. | Not implemented yet. | Planned mutation; advertised for discovery; blocked by read-only mode. | `ABS_READ_ONLY=true`; missing `collectionId`; not implemented until ABS source and fixture behavior are verified. |
| `abs_delete_collection` | Planned collection deletion. | `collectionId` required; `confirmation` must exactly equal `delete collection <collectionId>`. | Not implemented yet. | Planned destructive mutation; advertised for discovery; blocked by read-only mode and requires confirmation. | `ABS_READ_ONLY=true`; missing `collectionId`; wrong confirmation; not implemented until ABS source and fixture behavior are verified. |
| `abs_add_collection_item` | Planned collection membership add. | `collectionId` and `itemId` required. | Not implemented yet. | Planned mutation; advertised for discovery; blocked by read-only mode. | `ABS_READ_ONLY=true`; missing `collectionId` or `itemId`; not implemented until ABS source and fixture behavior are verified. |
| `abs_remove_collection_item` | Planned collection membership removal. | `collectionId` and `itemId` required; `confirmation` must exactly equal `remove item <itemId> from collection <collectionId>`. | Not implemented yet. | Planned destructive mutation; advertised for discovery; blocked by read-only mode and requires confirmation. | `ABS_READ_ONLY=true`; missing `collectionId` or `itemId`; wrong confirmation; not implemented until ABS source and fixture behavior are verified. |
| `abs_create_playlist` | Planned playlist creation. | `name` required; `payload` reserved until source verification. | Not implemented yet. | Planned mutation; advertised for discovery; blocked by read-only mode. | `ABS_READ_ONLY=true`; missing `name`; not implemented until ABS source and fixture behavior are verified. |
| `abs_update_playlist` | Planned playlist update. | `playlistId` required; optional `name`; `payload` reserved until source verification. | Not implemented yet. | Planned mutation; advertised for discovery; blocked by read-only mode. | `ABS_READ_ONLY=true`; missing `playlistId`; not implemented until ABS source and fixture behavior are verified. |
| `abs_delete_playlist` | Planned playlist deletion. | `playlistId` required; `confirmation` must exactly equal `delete playlist <playlistId>`. | Not implemented yet. | Planned destructive mutation; advertised for discovery; blocked by read-only mode and requires confirmation. | `ABS_READ_ONLY=true`; missing `playlistId`; wrong confirmation; not implemented until ABS source and fixture behavior are verified. |
| `abs_add_playlist_item` | Planned playlist membership add. | `playlistId` and `itemId` required; optional `episodeId`. | Not implemented yet. | Planned mutation; advertised for discovery; blocked by read-only mode. | `ABS_READ_ONLY=true`; missing `playlistId` or `itemId`; not implemented until ABS source and fixture behavior are verified. |
| `abs_remove_playlist_item` | Planned playlist membership removal. | `playlistId` and `itemId` required; optional `episodeId`; `confirmation` must exactly equal `remove item <itemId> from playlist <playlistId>`. | Not implemented yet. | Planned destructive mutation; advertised for discovery; blocked by read-only mode and requires confirmation. | `ABS_READ_ONLY=true`; missing `playlistId` or `itemId`; wrong confirmation; not implemented until ABS source and fixture behavior are verified. |
| `abs_remove_library_items_with_issues` | Remove missing or invalid items from one library after confirmation. | `libraryId` required; `confirmation` must exactly equal `remove issues from <libraryId>`; optional `expectedIssueCount`. | `triggered`, `libraryId`, issue count before cleanup, removed count, remaining issue count, up to 100 issue item IDs observed before cleanup. If no issues exist, `triggered=false`. | Mutates ABS library rows; blocked by read-only mode. | `ABS_READ_ONLY=true`; missing `libraryId`; wrong confirmation; negative or mismatched `expectedIssueCount`; list/delete/re-list request fails. |

## Resources

All resources return `application/json`.

| Resource | Purpose | Output summary | Errors |
| --- | --- | --- | --- |
| `abs://server/info` | Health summary. | Same shape as `abs_health_check`. | Wrong URI; health requests fail. |
| `abs://libraries` | Visible libraries. | Same shape as `abs_list_libraries`. | Wrong URI; library list request fails. |
| `abs://api-inventory/current` | Generated source-backed ABS API route inventory. | Inventory JSON, or `{ "available": false, "error": ... }` if loading failed. | Wrong URI; JSON marshaling failure. |
| `abs://fixture/status` | Local Docker fixture status without secrets. | Fixture directory/config presence, fixture URLs and SQLite paths from `.env.testing`, token configured flag and token length, expected item counts. | Wrong URI. |
| `abs://libraries/{library_id}` | One library by ID. | Same shape as `abs_get_library`. | Invalid URI; missing/not found library; ABS request fails. |
| `abs://libraries/{library_id}/items{?limit,offset}` | One bounded item page. | Same shape as `abs_list_library_items` for `libraryId`, `limit`, and `offset`. | Invalid URI; non-integer `limit`/`offset`; item-list validation or ABS request fails. |
| `abs://libraries/{library_id}/stats` | Raw library stats. | Same shape as `abs_get_library_stats`. | Invalid URI; ABS stats request fails. |
| `abs://libraries/{library_id}/filterdata` | Raw library filter data. | Same shape as `abs_get_filter_data`. | Invalid URI; ABS filterdata request fails. |
| `abs://items/{item_id}` | One item by ID. | Same shape as `abs_get_library_item`. | Invalid URI; missing/not found item; ABS request fails. |
| `abs://items/{item_id}/metadata-object` | Raw item metadata object. | Same shape as `abs_get_item_metadata_object`. | Invalid URI; ABS metadata-object request fails. |

## Prompts

| Prompt | Arguments | Purpose |
| --- | --- | --- |
| `abs_library_audit` | Optional `libraryId`. | Guides a read-only audit using health, library, item-list, and item-read calls; warns not to mutate unless explicitly requested and read-only mode is disabled. |
| `abs_scan_troubleshooting` | Optional `libraryId`. | Guides scan diagnosis, including before/after item reads, safe scan use, and confirmed cleanup only when requested. |
| `abs_api_update_review` | None. | Guides source inventory diff review, MCP schema/client/test updates, and verification against source and fixture behavior. |
