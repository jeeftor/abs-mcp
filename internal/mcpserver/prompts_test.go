package mcpserver

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestLibraryAuditPrompt(t *testing.T) {
	t.Parallel()

	result, err := newTestServer().LibraryAuditPrompt(context.Background(), getPromptRequest("abs_library_audit", map[string]string{
		"libraryId": "lib-audio",
	}))
	if err != nil {
		t.Fatalf("LibraryAuditPrompt failed: %v", err)
	}
	text := promptText(t, result)
	if !strings.Contains(text, "library `lib-audio`") {
		t.Fatalf("prompt does not include library ID: %s", text)
	}
	if !strings.Contains(text, "abs_health_check") || !strings.Contains(text, "abs_list_library_items") {
		t.Fatalf("prompt does not include expected tools: %s", text)
	}
}

func TestScanTroubleshootingPrompt(t *testing.T) {
	t.Parallel()

	result, err := newTestServer().ScanTroubleshootingPrompt(context.Background(), getPromptRequest("abs_scan_troubleshooting", nil))
	if err != nil {
		t.Fatalf("ScanTroubleshootingPrompt failed: %v", err)
	}
	text := promptText(t, result)
	if !strings.Contains(text, "Never bypass read-only mode") {
		t.Fatalf("prompt missing read-only guidance: %s", text)
	}
	if !strings.Contains(text, "abs_scan_library_and_wait") ||
		!strings.Contains(text, "abs_scan_library") ||
		!strings.Contains(text, "abs_scan_item") ||
		!strings.Contains(text, "abs_remove_library_items_with_issues") {
		t.Fatalf("prompt missing scan tools: %s", text)
	}
}

func TestAPIUpdateReviewPrompt(t *testing.T) {
	t.Parallel()

	result, err := newTestServer().APIUpdateReviewPrompt(context.Background(), getPromptRequest("abs_api_update_review", nil))
	if err != nil {
		t.Fatalf("APIUpdateReviewPrompt failed: %v", err)
	}
	text := promptText(t, result)
	if !strings.Contains(text, "make abs-api-inventory-diff") {
		t.Fatalf("prompt missing inventory diff command: %s", text)
	}
	if !strings.Contains(text, "old public API docs as historical context only") {
		t.Fatalf("prompt missing source priority guidance: %s", text)
	}
}

func getPromptRequest(name string, args map[string]string) *mcp.GetPromptRequest {
	if args == nil {
		args = map[string]string{}
	}
	return &mcp.GetPromptRequest{Params: &mcp.GetPromptParams{Name: name, Arguments: args}}
}

func promptText(t *testing.T, result *mcp.GetPromptResult) string {
	t.Helper()
	if len(result.Messages) != 1 {
		t.Fatalf("message count = %d, want 1", len(result.Messages))
	}
	content, ok := result.Messages[0].Content.(*mcp.TextContent)
	if !ok {
		t.Fatalf("content type = %T, want *mcp.TextContent", result.Messages[0].Content)
	}
	return content.Text
}
