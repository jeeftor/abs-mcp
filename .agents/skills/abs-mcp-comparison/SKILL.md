---
name: abs-mcp-comparison
description: Use when maintaining this Audiobookshelf MCP server's README comparison against other Audiobookshelf MCP servers, refreshing the AI Generated Comparison timestamp, researching public Audiobookshelf MCP alternatives, or suggesting feature gaps to consider from competitor evidence.
---

# ABS MCP Comparison

## Purpose

Keep the README comparison against other Audiobookshelf MCP servers current,
evidence-backed, and clearly marked as AI-generated.

## Workflow

1. Search for public Audiobookshelf MCP servers. Use subagents when the user
   asks for parallel research or when there are multiple independent targets.
2. Prefer direct evidence from project READMEs, releases, registry entries, and
   package metadata. Treat aggregator pages as secondary evidence.
3. Compare each server against this repository's current README, `docs/tools.md`,
   `docs/API_TOOL_ANALYSIS.md`, and `.agents/skills/`.
4. Update the README section between:
   - `<!-- AI-GENERATED-COMPARISON:START -->`
   - `<!-- AI-GENERATED-COMPARISON:END -->`
5. Set the heading to `## AI Generated Comparison - Last updated YYYY-MM-DD`
   using the current local date.
6. Keep the README comparison and candidate gaps descriptive. Do not rank
   servers or make a recommendation in the README.
7. After updating the README, report separate feature gaps or candidate work to
   the Codex user. Suggestions should be evidence-based and should not imply
   they are already planned.
8. Include a brief operator-focused overview of mutating versus non-mutating
   coverage. Distinguish read-only inspection/search resources from tools that
   can change Audiobookshelf state, and call out any safety gates such as
   read-only mode, explicit enable flags, or destructive confirmations.
9. Add a brief `Candidate gaps from this comparison` subsection inside the
   README generated block. Include only evidence-backed gaps, mark them as
   candidates, and link existing GitHub issues when they exist.
10. Convert the feature gaps into concise candidate GitHub issue titles with
   one-sentence scopes. Ask the user whether they want issues created for all
   candidates, a selected subset, or none. Do not create issues unless the user
   explicitly confirms.

## Comparison Criteria

For each credible public server, capture:

- Repository or registry URL.
- Runtime and transport.
- Read-only and mutating tool coverage.
- Safety controls, especially read-only mode and destructive confirmations.
- Distribution path, such as release binaries, Docker, source build, package
  registry, or MCP registry entry.
- Maintenance signals, such as last visible activity and releases.
- Notable features this server lacks or implements differently.

## Search Seeds

Start with current known public targets, then search for new ones:

- `michaeldvinci/audiobookshelf-mcp`
- `sandymac/audiobookshelf-mcp`
- `sierikov/audiobookshelf-mcp`
- `ForceConstant/audiobookshelf_mcp`
- `schmidt-software/mcp-audiobookshelf`
- Search queries such as `Audiobookshelf MCP server`, `site:github.com
  audiobookshelf mcp`, `site:npmjs.com audiobookshelf mcp`, and `site:pypi.org
  audiobookshelf mcp`.

## This Repository's Distinguishing Evidence

Check the current files instead of relying on memory:

- `README.md`: public positioning, MCP surface, install paths, and comparison.
- `docs/tools.md`: tool inputs, output shapes, read-only behavior, and stubs.
- `docs/API_TOOL_ANALYSIS.md`: source-backed API inventory and candidate
  endpoint rationale.
- `.agents/skills/abs-api-source-sync/SKILL.md`: API sync workflow.

Common differentiators to verify before stating:

- `ABS_READ_ONLY=true` by default.
- Destructive tools require exact confirmation strings.
- `abs_find_misorganized_items` audits file layout without moving files.
- Source-backed API inventory and repo-local Docker fixture workflow exist.
- MCP resources and prompts are documented.

## Output Requirements

When using this skill, report:

- The comparison timestamp written to README.
- Sources used.
- Credible servers found and weak or placeholder hits excluded.
- Files changed.
- Tests or checks run.
- Brief mutating versus non-mutating coverage overview for this server and
  notable peers.
- Feature gaps or candidate additions documented in the README.
- Candidate issue titles and scopes for the feature gaps.
- A direct offer such as: `I found N candidate gaps. I can create GitHub issues
  for all of them, only the ones you choose, or none.`

## Stop Conditions

Stop and ask before proceeding when:

- A comparison claim depends on private credentials or a private repository.
- A source contradicts this repository's current implementation.
- A README update would require claiming an unimplemented feature.
- The next action would create, edit, label, close, or otherwise mutate GitHub
  issues and the user has not explicitly confirmed that action.
