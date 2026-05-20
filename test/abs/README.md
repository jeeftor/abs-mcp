# Audiobookshelf MCP Test Harness

This directory provides resettable local Audiobookshelf instances for MCP integration work.

It is based on the reusable Docker fixture from `/Users/Shared/Docker/audiobook-organizer/test/abs`, but it uses its own Docker identity so it can run without touching the sibling project's containers:

- Compose project name: `abs-mcp`
- Plain container: `abs-mcp-abs-plain`
- Metadata-enabled container: `abs-mcp-abs-metadata`
- Plain server: <http://localhost:13388>
- Metadata-enabled server: <http://localhost:13389>

The default image pulls the upstream Audiobookshelf image through the corporate proxy:

```bash
make abs-dev-up
```

Override `ABS_IMAGE` when testing a local Audiobookshelf build or when running outside the corporate proxy:

```bash
ABS_IMAGE=ghcr.io/advplyr/audiobookshelf:latest make abs-dev-up
```

## Layout

```text
test/abs/
  docker-compose.yml
  baseline-config/  # committed sanitized ABS config databases
  staging-data/     # ignored durable public-domain fixture cache
  runtime/          # ignored throwaway library copies mounted into ABS
  state/            # ignored live ABS config and metadata cache
  scripts/
```

The fixture runs two separate ABS services:

- `abs-plain` uses `runtime/plain` and has `storeMetadataWithItem=false`.
- `abs-metadata` uses `runtime/metadata` and has `storeMetadataWithItem=true`.

Each service has two libraries:

- `Audiobooks`: `/audiobooks`
- `Ebooks`: `/books`

## Commands

Start Docker Desktop or another Docker daemon first.

Seed public-domain test media into `staging-data/` and refresh runtime copies:

```bash
make abs-dev-seed
```

Reset state from the committed baseline, start both ABS services, and wait:

```bash
make abs-dev-reset
```

Reset, start, and scan both libraries:

```bash
make abs-dev-reset-scan
```

Start already-prepared services:

```bash
make abs-dev-up
make abs-dev-wait
```

Stop the services:

```bash
make abs-dev-down
```

Run the CI-style fixture smoke path:

```bash
make abs-ci-smoke
```

This seeds public-domain media, resets runtime/state, restores the committed baseline config, starts both services, waits for them, and scans both libraries.

## Local Credentials

The baseline fixture uses disposable local credentials:

```text
username: root
password: password
```

The committed `.env.testing` contains a disposable API token for the committed baseline databases. Do not replace it with a personal token.

## Test Data

The seed script downloads public-domain fixtures:

- LibriVox M4B: Alice's Adventures in Wonderland (Abridged)
- LibriVox M4B: A Christmas Carol
- Project Gutenberg EPUB: Alice's Adventures in Wonderland
- Project Gutenberg EPUB: Frankenstein
- Project Gutenberg EPUB: Pride and Prejudice

Runtime paths are intentionally messy so MCP tests can inspect realistic ABS item paths:

```text
runtime/plain/audiobooks/unsorted-audio/drop-001/not-alice.m4b
runtime/plain/audiobooks/loose/holiday_story_final.m4b
runtime/plain/books/imported/ebook-001.epub
runtime/plain/books/random/shelley-book.epub
runtime/plain/books/to-sort/austen.epub
```

The metadata-enabled runtime mirrors those paths and adds `metadata.json` sidecars.

## Useful Paths

```text
Plain ABS URL:        http://localhost:13388
Metadata ABS URL:     http://localhost:13389
Plain ABS SQLite:     test/abs/state/plain/config/absdatabase.sqlite
Metadata ABS SQLite:  test/abs/state/metadata-enabled/config/absdatabase.sqlite
Plain audiobooks:     test/abs/runtime/plain/audiobooks
Plain books:          test/abs/runtime/plain/books
Metadata audiobooks:  test/abs/runtime/metadata/audiobooks
Metadata books:       test/abs/runtime/metadata/books
Container paths:
  /audiobooks
  /books
```

## Baseline Maintenance

Use the committed baseline for normal development. To rebuild it intentionally:

```bash
make abs-dev-init
make abs-dev-configure
make abs-dev-capture-baseline
```

Only commit sanitized local fixture state. Do not capture a real personal Audiobookshelf library.
