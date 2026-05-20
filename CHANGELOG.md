# Changelog

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
