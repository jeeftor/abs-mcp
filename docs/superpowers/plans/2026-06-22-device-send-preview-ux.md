# Device Send Preview UX Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a read-only preview tool that makes ebook-to-device sending interactive and guarded before any email mutation is triggered.

**Architecture:** Reuse the existing ebook search and sanitized ereader device listing helpers. The preview tool returns compact candidate/device data and an exact confirmation string only when one ebook and one device are resolved. Existing send tools remain the mutation path.

**Tech Stack:** Go MCP server, existing Audiobookshelf client, Go tests, repo ABS Docker fixture matrix.

---

### Task 1: Core Preview Tool

**Files:**
- Modify: `internal/mcpserver/tools.go`
- Test: `internal/mcpserver/tools_test.go`

- [ ] Add `abs_preview_ebook_device_send` to the tool registry near the existing ebook/device tools.
- [ ] Add `PreviewEbookDeviceSendInput` with `libraryId`, `query`, optional `deviceName`, and optional `maxCandidates`.
- [ ] Add `PreviewEbookDeviceSendOutput` with `libraryId`, `query`, `deviceName`, `ready`, `confirmation`, `candidateCount`, `candidates`, `deviceCount`, `devices`, and `nextTool`.
- [ ] Write failing unit tests for exact match, ambiguous ebook matches, missing device, and read-only safety.
- [ ] Implement `PreviewEbookDeviceSend` using `searchEbooks` and `GetEmailSettings`; do not call `SendEbookToDevice`.
- [ ] Run `rtk go test ./internal/mcpserver -run 'TestServer.*Ebook.*Device|TestServerTools' -count=1`.

### Task 2: Protocol And Matrix Coverage

**Files:**
- Modify: `internal/mcpserver/protocol_test.go`
- Modify: `test/abs/integration/abs_fixture_test.go`

- [ ] Add protocol coverage that lists and calls `abs_preview_ebook_device_send`.
- [ ] Verify protocol output includes compact candidate data and does not mutate.
- [ ] Add ABS fixture coverage for `abs_search_ebooks` and preview with the existing ebook library data.
- [ ] Keep Docker fixture coverage read-only unless saved ereader device setup is already available from baseline config.
- [ ] Run `rtk go test ./internal/mcpserver -run TestProtocol -count=1`.
- [ ] Run `rtk go test -tags=abs_integration ./test/abs/integration -run '^$'`.

### Task 3: Docs And Verification

**Files:**
- Modify: `README.md`
- Modify: `docs/tools.md`
- Modify: `docs/API_TOOL_ANALYSIS.md`

- [ ] Document `abs_preview_ebook_device_send` as read-only and token-efficient.
- [ ] Explain the two-step flow: preview, then exact confirmed send.
- [ ] Run `rtk make test`.
- [ ] Run `rtk make build`.
- [ ] Run `rtk git diff --check`.
- [ ] Commit the finished feature.
