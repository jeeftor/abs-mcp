# ABS API Source Sync Agent Brief

## Purpose

Keep the MCP server aligned with Audiobookshelf API changes by reading Audiobookshelf source, generating an endpoint inventory, and updating typed client methods, MCP schemas, docs, and tests together.

## Inputs

- Audiobookshelf source repository or checkout.
- Current MCP repo checkout.
- Current API inventory baseline, when present.
- Optional Docker ABS fixture for behavior verification.

## Source Priority

1. Audiobookshelf source code.
2. Behavior observed against a running ABS fixture.
3. Official Audiobookshelf user docs for current auth and setup concepts.
4. Old API docs only as historical hints.

## Workflow

1. Identify the Audiobookshelf source revision under review.
2. Inspect `server/routers/ApiRouter.js` for route declarations.
3. Inspect handler methods in `server/controllers/`.
4. Generate or update `docs/api-inventory/generated/abs-api-inventory.json` with `make abs-api-inventory` or `make abs-api-inventory-from-router`.
5. Diff generated inventory against the committed baseline with `make abs-api-inventory-diff`.
6. Classify endpoint changes:
   - new endpoint
   - removed endpoint
   - path or method changed
   - auth or permission changed
   - request schema changed
   - response schema changed
7. For endpoints mapped to MCP tools/resources, update:
   - ABS client method
   - request/response models
   - MCP input schema
   - MCP output shape
   - tests
   - docs
8. Run unit tests and relevant fixture round trips.
9. Summarize API changes and verification results.

## Required Output

- Source revision reviewed.
- Inventory diff summary.
- Files changed.
- Tests run.
- Any endpoint whose behavior remains uncertain.

## Stop Conditions

Stop and ask before proceeding when:

- An endpoint appears destructive and there is no fixture-safe test path.
- Auth semantics changed in a way that could leak or over-broaden access.
- A source change conflicts with observed fixture behavior.
- A generated inventory change would require broad MCP tool redesign.

## Quality Bar

- No MCP schema change without a test.
- No mutating endpoint exposure without read-only mode coverage.
- No reliance on the old API docs as the only evidence.
- No committed secrets, tokens, cookies, or local absolute credentials.
