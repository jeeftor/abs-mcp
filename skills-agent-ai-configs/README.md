# Skills and Agent Configs

This directory stores historical repo-local briefs for future agents that
maintain the Audiobookshelf MCP server. They are intentionally plain Markdown so
they can be adapted into Codex skills, MCP prompts, or other agent
configuration formats later.

Repo-scoped Codex skills live under `.agents/skills/`. Put new repo-local
skills there so Codex can discover them with the repository context. Install or
copy a skill into `${CODEX_HOME:-$HOME/.codex}/skills/` only when it should be
available globally outside this repository.

Current skills:

- `.agents/skills/abs-api-source-sync/`: Detect Audiobookshelf API changes from
  source and update this MCP server safely.
- `.agents/skills/abs-mcp-comparison/`: Refresh the README comparison against
  other Audiobookshelf MCP servers and suggest feature gaps.

Historical briefs:

- `abs-api-source-sync.md`: How an agent should detect Audiobookshelf API changes from source and update this MCP server safely.
- `abs-fixture-roundtrip.md`: How an agent should use the Docker fixture for MCP and ABS behavior tests.

Rules for future configs:

- Keep each brief focused on one repeatable workflow.
- Include inputs, outputs, required checks, and stop conditions.
- Prefer source-backed and fixture-backed evidence over stale public docs.
- Rerun `.agents/skills/abs-mcp-comparison/` periodically, and after material
  MCP surface changes, so the README comparison does not drift.
- Never include real tokens or local secrets.
