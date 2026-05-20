# Skills and Agent Configs

This directory stores repo-local briefs for future agents that maintain the Audiobookshelf MCP server. They are intentionally plain Markdown so they can be adapted into Codex skills, MCP prompts, or other agent configuration formats later.

Current briefs:

- `abs-api-source-sync.md`: How an agent should detect Audiobookshelf API changes from source and update this MCP server safely.
- `abs-fixture-roundtrip.md`: How an agent should use the Docker fixture for MCP and ABS behavior tests.

Rules for future configs:

- Keep each brief focused on one repeatable workflow.
- Include inputs, outputs, required checks, and stop conditions.
- Prefer source-backed and fixture-backed evidence over stale public docs.
- Never include real tokens or local secrets.
