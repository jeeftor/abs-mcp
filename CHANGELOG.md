# Changelog

## 0.4.1 - 2026-06-22

### Fixed
- Accept the current Audiobookshelf backup-list response envelope while keeping compatibility with older raw-array responses.

## 0.4.0 - 2026-06-22

### Added
- Added ebook delivery tools for sending Audiobookshelf ebooks to saved ereader devices.
- Added ebook-only library search and sanitized ereader device listing helpers.
- Added a guarded query-based ebook send workflow that requires exactly one match and an exact confirmation string before delivery.

### Improved
- Updated MCP tool documentation and protocol coverage for the ebook delivery workflow.

## 0.3.0 - 2026-06-16

### Added
- Added optional Streamable HTTP transport with configurable bind address, endpoint path, and bearer-token protection while keeping stdio as the default transport.
- Added source-backed author, series, collection, current-user progress, and bookmark MCP tools.
- Added guarded metadata, progress, bookmark, collection, and playlist mutation tools behind the existing `ABS_READ_ONLY=false` gate.
- Added fixture and protocol coverage for the expanded MCP surface.

### Improved
- Updated public tool documentation, comparison notes, and MCP Registry metadata for the expanded runtime configuration and tool surface.
- Refreshed the Audiobookshelf API inventory source reference.

## 0.2.1 - 2026-05-20

### Added
- Added `abs_find_misorganized_items`, a read-only MCP tool that audits Audiobookshelf item folders against `author/title` and `author/series/title` layout conventions.
- Added official MCP Registry metadata and tag-based publishing support for GHCR Docker images, release archives, and registry publication.
- Added a project logo and expanded user-facing tool documentation.
- Added pre-commit checks for formatting, Go doc comments, `go vet`, tests, and script tests.

### Improved
- The MCP server now exposes its version through `internal/version`, with release builds overriding the value from the pushed tag.
- Release archives now include this changelog alongside the README, license, tool docs, and `server.json`.

### Fixed
- Removed the OCI package-level `version` field from MCP Registry metadata so registry publication accepts the GHCR image reference as the package version source.
