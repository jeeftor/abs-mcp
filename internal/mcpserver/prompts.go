package mcpserver

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RegisterPrompts adds the server's prompt surface.
func (s *Server) RegisterPrompts(server *mcp.Server) {
	server.AddPrompt(&mcp.Prompt{
		Name:        "abs_library_audit",
		Title:       "Audiobookshelf library audit",
		Description: "Guide an agent through auditing Audiobookshelf library health using safe read-only tools.",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "libraryId",
				Description: "Optional Audiobookshelf library ID to focus the audit.",
			},
		},
	}, s.LibraryAuditPrompt)
	server.AddPrompt(&mcp.Prompt{
		Name:        "abs_scan_troubleshooting",
		Title:       "Audiobookshelf scan troubleshooting",
		Description: "Guide an agent through diagnosing scan issues and using scan tools safely.",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "libraryId",
				Description: "Optional Audiobookshelf library ID with scan issues.",
			},
		},
	}, s.ScanTroubleshootingPrompt)
	server.AddPrompt(&mcp.Prompt{
		Name:        "abs_api_update_review",
		Title:       "Audiobookshelf API update review",
		Description: "Guide an agent through reviewing Audiobookshelf source API changes against MCP tools and tests.",
	}, s.APIUpdateReviewPrompt)
}

// LibraryAuditPrompt returns the abs_library_audit prompt.
func (s *Server) LibraryAuditPrompt(_ context.Context, request *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	libraryID := request.Params.Arguments["libraryId"]
	scope := "all visible libraries"
	if libraryID != "" {
		scope = fmt.Sprintf("library `%s`", libraryID)
	}

	return promptResult(
		"Audit Audiobookshelf library health.",
		fmt.Sprintf(`Audit %s in Audiobookshelf.

Use these MCP calls in order:
1. Call abs_health_check and confirm authentication, read-only mode, and library count.
2. Call abs_list_libraries to identify target libraries and mounted folders.
3. For each target library, call abs_get_library and abs_list_library_items with a bounded limit.
4. Inspect item summaries for isMissing, isInvalid, unexpected paths, empty titles, and missing authors.
5. Read specific suspicious items with abs_get_library_item.
6. Summarize findings with library IDs, item IDs, counts, and recommended next checks.

Do not call mutating tools unless the user explicitly asks and ABS_READ_ONLY is false.`, scope),
	), nil
}

// ScanTroubleshootingPrompt returns the abs_scan_troubleshooting prompt.
func (s *Server) ScanTroubleshootingPrompt(_ context.Context, request *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	libraryID := request.Params.Arguments["libraryId"]
	target := "the affected library"
	if libraryID != "" {
		target = fmt.Sprintf("library `%s`", libraryID)
	}

	return promptResult(
		"Troubleshoot Audiobookshelf scan behavior.",
		fmt.Sprintf(`Troubleshoot scans for %s.

Use these MCP calls in order:
1. Call abs_health_check and verify whether read-only mode blocks scan actions.
2. Call abs_get_library for the target library when a libraryId is known; otherwise call abs_list_libraries first.
3. Call abs_list_library_items before scanning and note total, missing, and invalid state.
4. If mutation is allowed and the user requested it, prefer abs_scan_library_and_wait with force, expectedTotal, and timeoutSeconds set explicitly.
5. Use abs_scan_item only when the issue is isolated to one directory-backed item ID.
6. Re-read abs_list_library_items after the scan observation and compare totals and item state.
7. Use abs_scan_library only when the caller wants a fire-and-forget scan request.
8. Use abs_remove_library_items_with_issues only after the user explicitly asks to remove missing or invalid rows; set expectedIssueCount when known and use the exact confirmation text.
9. Report whether the issue appears to be auth, library discovery, missing files, invalid metadata, or scan timing.

Never bypass read-only mode. Do not remove missing items as part of this workflow.`, target),
	), nil
}

// APIUpdateReviewPrompt returns the abs_api_update_review prompt.
func (s *Server) APIUpdateReviewPrompt(context.Context, *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return promptResult(
		"Review Audiobookshelf API changes for MCP impact.",
		`Review Audiobookshelf API changes before changing MCP tools.

Use this workflow:
1. Identify the Audiobookshelf source revision under review.
2. Generate the API inventory with make abs-api-inventory or make abs-api-inventory-from-router.
3. Run make abs-api-inventory-diff and classify added, removed, and changed routes.
4. For changed routes mapped to MCP tools or resources, inspect controller behavior and update typed client models.
5. Update MCP tool/resource schemas and tests together.
6. Run go test ./... and the relevant Docker-backed integration test.
7. Summarize source revision, route changes, MCP surface changes, and verification results.

Use source and fixture behavior as authoritative. Treat the old public API docs as historical context only.`,
	), nil
}

func promptResult(description string, text string) *mcp.GetPromptResult {
	return &mcp.GetPromptResult{
		Description: description,
		Messages: []*mcp.PromptMessage{
			{
				Role:    "user",
				Content: &mcp.TextContent{Text: text},
			},
		},
	}
}
