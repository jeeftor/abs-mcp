---
name: abs-api-source-sync
description: Use when maintaining this Audiobookshelf MCP server against Audiobookshelf API source, generating or diffing the API inventory, deciding which ABS endpoints should become MCP tools/resources/prompts, or auditing mutating tool candidates gated by ABS_READ_ONLY.
---

# ABS API Source Sync

## Purpose

Keep this MCP server aligned with Audiobookshelf API source while exposing only
bounded, typed, agent-safe workflows.

## Source Priority

1. Audiobookshelf source code, especially `server/routers/ApiRouter.js` and
   handlers under `server/controllers/`.
2. Behavior verified against the repo-local Docker ABS fixture.
3. Current Audiobookshelf user docs for setup and auth concepts.
4. Old API docs only as historical hints.

## Workflow

1. Identify the Audiobookshelf source revision under review.
2. Generate or update inventory with `make abs-api-inventory` or
   `make abs-api-inventory-from-router`.
3. Compare generated inventory to baseline with `make abs-api-inventory-diff`.
4. Classify changes as new, removed, path/method changed, auth changed, request
   schema changed, or response schema changed.
5. For endpoints mapped to MCP surface, update the typed ABS client, MCP schema,
   output shape, docs, and tests together.
6. If the MCP tool surface changes materially, rerun the
   `$abs-mcp-comparison` skill so the README comparison and feature-gap
   suggestions stay current.
7. Run unit tests and relevant fixture round trips.

## MCP Mapping Rules

- Do not expose every ABS route directly.
- Prefer resources for bounded read-only state snapshots.
- Prefer tools for actions, bounded queries, and repeatable operational
  workflows.
- Every tool schema must document required inputs, optional inputs, output
  shape, error cases, and whether it mutates ABS state.
- Every mutating tool must be blocked when `ABS_READ_ONLY=true`.
- Destructive tools, including delete, remove, purge, overwrite, replace, and
  broad batch operations, must require an explicit confirmation input. Prefer an
  exact phrase that includes the relevant ABS ID, and add expected-count checks
  when the tool can preview affected records.
- Long operations should return job/status information or use a separate status
  read instead of blocking indefinitely.

## Required Output

When using this skill, report:

- Audiobookshelf source revision reviewed.
- Inventory diff or current inventory basis.
- Candidate MCP tools/resources/prompts and why they were accepted or deferred.
- Files changed.
- Tests run.
- Whether `$abs-mcp-comparison` was rerun or intentionally deferred.
- Any endpoint whose behavior remains uncertain.

## Stop Conditions

Stop and ask before proceeding when:

- An endpoint appears destructive and no fixture-safe test path exists.
- Auth semantics changed in a way that could leak or over-broaden access.
- Source behavior conflicts with fixture behavior.
- A generated inventory change requires broad MCP tool redesign.
