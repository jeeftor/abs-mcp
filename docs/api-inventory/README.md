# Audiobookshelf API Inventory

This directory holds source-generated API route inventory files for Audiobookshelf.

Generate from a full Audiobookshelf checkout:

```bash
ABS_SOURCE_DIR=/path/to/audiobookshelf ABS_SOURCE_REF=<commit-or-tag> make abs-api-inventory
```

Generate from a single router file:

```bash
ABS_ROUTER_FILE=/path/to/ApiRouter.js ABS_SOURCE_REF=<commit-or-tag> make abs-api-inventory-from-router
```

Outputs:

- `generated/abs-api-inventory.json`
- `generated/abs-api-inventory.md`
- `baseline/abs-api-inventory.json`

Compare the current generated inventory to the baseline:

```bash
make abs-api-inventory-diff
```

Fail when the generated inventory differs from the baseline:

```bash
make abs-api-inventory-check
```

The generated inventory is an input to MCP tool design. Do not expose every route as a tool. Use `docs/API_TOOL_ANALYSIS.md` to classify routes into tools, resources, prompts, or deferred endpoints.
