package mcpserver

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jeeftor/abs-mcp/internal/config"
	"github.com/jeeftor/abs-mcp/internal/version"
)

func TestMCPProtocolListsAndCallsTools(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- newTestServer().MCPServer().Run(ctx, serverTransport)
	}()

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "abs-mcp-test-client",
		Version: "0.1.0",
	}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("connect MCP client: %v", err)
	}
	defer session.Close()
	if got := session.InitializeResult().ServerInfo.Version; got != version.Version {
		t.Fatalf("server version = %q, want %q", got, version.Version)
	}

	tools, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	if !toolNames(tools)["abs_health_check"] {
		t.Fatalf("expected abs_health_check in tools: %#v", tools.Tools)
	}
	if !toolNames(tools)["abs_scan_library"] {
		t.Fatalf("expected abs_scan_library in tools: %#v", tools.Tools)
	}
	if !toolNames(tools)["abs_scan_library_and_wait"] {
		t.Fatalf("expected abs_scan_library_and_wait in tools: %#v", tools.Tools)
	}
	if !toolNames(tools)["abs_scan_item"] {
		t.Fatalf("expected abs_scan_item in tools: %#v", tools.Tools)
	}
	if !toolNames(tools)["abs_remove_library_items_with_issues"] {
		t.Fatalf("expected abs_remove_library_items_with_issues in tools: %#v", tools.Tools)
	}
	if !toolNames(tools)["abs_search_library"] {
		t.Fatalf("expected abs_search_library in tools: %#v", tools.Tools)
	}
	if !toolNames(tools)["abs_find_misorganized_items"] {
		t.Fatalf("expected abs_find_misorganized_items in tools: %#v", tools.Tools)
	}

	resources, err := session.ListResources(ctx, &mcp.ListResourcesParams{})
	if err != nil {
		t.Fatalf("list resources: %v", err)
	}
	if !resourceURIs(resources)["abs://libraries"] {
		t.Fatalf("expected abs://libraries resource: %#v", resources.Resources)
	}
	if !resourceURIs(resources)["abs://fixture/status"] {
		t.Fatalf("expected abs://fixture/status resource: %#v", resources.Resources)
	}

	templates, err := session.ListResourceTemplates(ctx, &mcp.ListResourceTemplatesParams{})
	if err != nil {
		t.Fatalf("list resource templates: %v", err)
	}
	if !resourceTemplateURIs(templates)["abs://items/{item_id}"] {
		t.Fatalf("expected abs://items/{item_id} template: %#v", templates.ResourceTemplates)
	}

	readResult, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: "abs://libraries"})
	if err != nil {
		t.Fatalf("read abs://libraries: %v", err)
	}
	var resourceOutput LibrariesOutput
	unmarshalResourceContent(t, readResult, &resourceOutput)
	if resourceOutput.Count != 2 {
		t.Fatalf("resource library count = %d, want 2", resourceOutput.Count)
	}

	fixtureResult, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: "abs://fixture/status"})
	if err != nil {
		t.Fatalf("read abs://fixture/status: %v", err)
	}
	var fixtureOutput FixtureStatusOutput
	unmarshalResourceContent(t, fixtureResult, &fixtureOutput)
	if fixtureOutput.Configured {
		t.Fatalf("test server should not configure fixture dir by default: %#v", fixtureOutput)
	}

	prompts, err := session.ListPrompts(ctx, &mcp.ListPromptsParams{})
	if err != nil {
		t.Fatalf("list prompts: %v", err)
	}
	if !promptNames(prompts)["abs_library_audit"] {
		t.Fatalf("expected abs_library_audit prompt: %#v", prompts.Prompts)
	}

	prompt, err := session.GetPrompt(ctx, &mcp.GetPromptParams{
		Name:      "abs_library_audit",
		Arguments: map[string]string{"libraryId": "lib-audio"},
	})
	if err != nil {
		t.Fatalf("get abs_library_audit prompt: %v", err)
	}
	if !strings.Contains(protocolPromptText(t, prompt), "library `lib-audio`") {
		t.Fatalf("prompt did not include requested library ID: %#v", prompt.Messages)
	}

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "abs_list_libraries",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("call abs_list_libraries: %v", err)
	}
	if result.IsError {
		t.Fatalf("abs_list_libraries returned tool error: %#v", result.Content)
	}

	var output LibrariesOutput
	marshalStructuredOutput(t, result.StructuredContent, &output)
	if output.Count != 2 {
		t.Fatalf("library count = %d, want 2", output.Count)
	}

	itemsResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "abs_list_library_items",
		Arguments: map[string]any{
			"libraryId":      "lib-audio",
			"limit":          2,
			"offset":         2,
			"sort":           "media.metadata.title",
			"desc":           true,
			"filter":         "issues.true",
			"include":        []any{"rssfeed", "progress"},
			"minified":       true,
			"collapseSeries": true,
		},
	})
	if err != nil {
		t.Fatalf("call abs_list_library_items: %v", err)
	}
	if itemsResult.IsError {
		t.Fatalf("abs_list_library_items returned tool error: %#v", itemsResult.Content)
	}
	var itemOutput LibraryItemsOutput
	marshalStructuredOutput(t, itemsResult.StructuredContent, &itemOutput)
	if itemOutput.Page != 1 || itemOutput.Sort != "media.metadata.title" || !itemOutput.Desc || itemOutput.Filter != "issues.true" {
		t.Fatalf("unexpected item list output: %#v", itemOutput)
	}

	searchResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "abs_search_library",
		Arguments: map[string]any{
			"libraryId": "lib-audio",
			"query":     "alice",
			"limit":     3,
		},
	})
	if err != nil {
		t.Fatalf("call abs_search_library: %v", err)
	}
	if searchResult.IsError {
		t.Fatalf("abs_search_library returned tool error: %#v", searchResult.Content)
	}

	layoutResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "abs_find_misorganized_items",
		Arguments: map[string]any{"libraryId": "lib-audio"},
	})
	if err != nil {
		t.Fatalf("call abs_find_misorganized_items: %v", err)
	}
	if layoutResult.IsError {
		t.Fatalf("abs_find_misorganized_items returned tool error: %#v", layoutResult.Content)
	}
	var layoutOutput FindMisorganizedItemsOutput
	marshalStructuredOutput(t, layoutResult.StructuredContent, &layoutOutput)
	if layoutOutput.CheckedCount != 3 || layoutOutput.MisorganizedCount != 3 {
		t.Fatalf("unexpected layout audit output: %#v", layoutOutput)
	}

	statsResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "abs_get_library_stats",
		Arguments: map[string]any{"libraryId": "lib-audio"},
	})
	if err != nil {
		t.Fatalf("call abs_get_library_stats: %v", err)
	}
	if statsResult.IsError {
		t.Fatalf("abs_get_library_stats returned tool error: %#v", statsResult.Content)
	}

	scanResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "abs_scan_library",
		Arguments: map[string]any{
			"libraryId": "lib-audio",
			"force":     true,
		},
	})
	if err != nil {
		t.Fatalf("call abs_scan_library: %v", err)
	}
	if !scanResult.IsError {
		t.Fatal("expected abs_scan_library to be a tool error in read-only mode")
	}

	scanWaitResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "abs_scan_library_and_wait",
		Arguments: map[string]any{
			"libraryId":     "lib-audio",
			"expectedTotal": 3,
		},
	})
	if err != nil {
		t.Fatalf("call abs_scan_library_and_wait: %v", err)
	}
	if !scanWaitResult.IsError {
		t.Fatal("expected abs_scan_library_and_wait to be a tool error in read-only mode")
	}

	scanItemReadOnlyResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "abs_scan_item",
		Arguments: map[string]any{
			"itemId": "item-1",
		},
	})
	if err != nil {
		t.Fatalf("call abs_scan_item: %v", err)
	}
	if !scanItemReadOnlyResult.IsError {
		t.Fatal("expected abs_scan_item to be a tool error in read-only mode")
	}

	removeIssuesReadOnlyResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "abs_remove_library_items_with_issues",
		Arguments: map[string]any{
			"libraryId":    "lib-audio",
			"confirmation": "remove issues from lib-audio",
		},
	})
	if err != nil {
		t.Fatalf("call abs_remove_library_items_with_issues: %v", err)
	}
	if !removeIssuesReadOnlyResult.IsError {
		t.Fatal("expected abs_remove_library_items_with_issues to be a tool error in read-only mode")
	}

	if err := session.Close(); err != nil {
		t.Fatalf("close MCP session: %v", err)
	}
	select {
	case err := <-serverErr:
		if err != nil {
			t.Fatalf("server Run returned error: %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("server did not stop after client close: %v", ctx.Err())
	}
}

func TestMCPProtocolRemoveLibraryItemsWithIssues(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fakeClient := newFakeABSClient()
	fakeClient.items["lib-audio"][0].IsMissing = true
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverErr := make(chan error, 1)
	go func() {
		server := New(config.Config{ABSBaseURL: "http://abs", ReadOnly: false}, fakeClient)
		serverErr <- server.MCPServer().Run(ctx, serverTransport)
	}()

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "abs-mcp-remove-issues-test-client",
		Version: "0.1.0",
	}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("connect MCP client: %v", err)
	}
	defer session.Close()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "abs_remove_library_items_with_issues",
		Arguments: map[string]any{
			"libraryId":          "lib-audio",
			"confirmation":       "remove issues from lib-audio",
			"expectedIssueCount": 1,
		},
	})
	if err != nil {
		t.Fatalf("call abs_remove_library_items_with_issues: %v", err)
	}
	if result.IsError {
		t.Fatalf("abs_remove_library_items_with_issues returned tool error: %#v", result.Content)
	}

	var output RemoveLibraryItemsWithIssuesOutput
	marshalStructuredOutput(t, result.StructuredContent, &output)
	if !output.Triggered || output.IssueCountBefore != 1 || output.RemovedCount != 1 || output.RemainingIssueCount != 0 {
		t.Fatalf("unexpected remove issues output: %#v", output)
	}

	if err := session.Close(); err != nil {
		t.Fatalf("close MCP session: %v", err)
	}
	select {
	case err := <-serverErr:
		if err != nil {
			t.Fatalf("server Run returned error: %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("server did not stop after client close: %v", ctx.Err())
	}
}

func TestMCPProtocolScanLibraryAndWait(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fakeClient := newFakeABSClient()
	fakeClient.libraryItemTotals = []int{1, 3}
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverErr := make(chan error, 1)
	go func() {
		server := New(config.Config{ABSBaseURL: "http://abs", ReadOnly: false}, fakeClient)
		serverErr <- server.MCPServer().Run(ctx, serverTransport)
	}()

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "abs-mcp-scan-wait-test-client",
		Version: "0.1.0",
	}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("connect MCP client: %v", err)
	}
	defer session.Close()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "abs_scan_library_and_wait",
		Arguments: map[string]any{
			"libraryId":                "lib-audio",
			"force":                    true,
			"expectedTotal":            3,
			"timeoutSeconds":           1,
			"pollIntervalMilliseconds": 1,
		},
	})
	if err != nil {
		t.Fatalf("call abs_scan_library_and_wait: %v", err)
	}
	if result.IsError {
		t.Fatalf("abs_scan_library_and_wait returned tool error: %#v", result.Content)
	}

	var output ScanLibraryAndWaitOutput
	marshalStructuredOutput(t, result.StructuredContent, &output)
	if !output.Triggered || !output.Completed || output.TimedOut {
		t.Fatalf("unexpected scan wait output: %#v", output)
	}
	if output.ObservedTotal != 3 || output.Attempts != 2 {
		t.Fatalf("observed total/attempts = %d/%d, want 3/2", output.ObservedTotal, output.Attempts)
	}

	if err := session.Close(); err != nil {
		t.Fatalf("close MCP session: %v", err)
	}
	select {
	case err := <-serverErr:
		if err != nil {
			t.Fatalf("server Run returned error: %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("server did not stop after client close: %v", ctx.Err())
	}
}

func TestMCPProtocolScanItem(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fakeClient := newFakeABSClient()
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverErr := make(chan error, 1)
	go func() {
		server := New(config.Config{ABSBaseURL: "http://abs", ReadOnly: false}, fakeClient)
		serverErr <- server.MCPServer().Run(ctx, serverTransport)
	}()

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "abs-mcp-scan-item-test-client",
		Version: "0.1.0",
	}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("connect MCP client: %v", err)
	}
	defer session.Close()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "abs_scan_item",
		Arguments: map[string]any{
			"itemId": "item-1",
		},
	})
	if err != nil {
		t.Fatalf("call abs_scan_item: %v", err)
	}
	if result.IsError {
		t.Fatalf("abs_scan_item returned tool error: %#v", result.Content)
	}

	var output ScanItemOutput
	marshalStructuredOutput(t, result.StructuredContent, &output)
	if !output.Triggered || output.ItemID != "item-1" || output.Result != "SUCCESS" {
		t.Fatalf("unexpected scan item output: %#v", output)
	}

	if err := session.Close(); err != nil {
		t.Fatalf("close MCP session: %v", err)
	}
	select {
	case err := <-serverErr:
		if err != nil {
			t.Fatalf("server Run returned error: %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("server did not stop after client close: %v", ctx.Err())
	}
}

func toolNames(result *mcp.ListToolsResult) map[string]bool {
	names := make(map[string]bool, len(result.Tools))
	for _, tool := range result.Tools {
		names[tool.Name] = true
	}
	return names
}

func resourceURIs(result *mcp.ListResourcesResult) map[string]bool {
	uris := make(map[string]bool, len(result.Resources))
	for _, resource := range result.Resources {
		uris[resource.URI] = true
	}
	return uris
}

func resourceTemplateURIs(result *mcp.ListResourceTemplatesResult) map[string]bool {
	uris := make(map[string]bool, len(result.ResourceTemplates))
	for _, resource := range result.ResourceTemplates {
		uris[resource.URITemplate] = true
	}
	return uris
}

func promptNames(result *mcp.ListPromptsResult) map[string]bool {
	names := make(map[string]bool, len(result.Prompts))
	for _, prompt := range result.Prompts {
		names[prompt.Name] = true
	}
	return names
}

func marshalStructuredOutput(t *testing.T, value any, target any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal structured output: %v", err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatalf("unmarshal structured output: %v", err)
	}
}

func unmarshalResourceContent(t *testing.T, result *mcp.ReadResourceResult, target any) {
	t.Helper()
	if len(result.Contents) != 1 {
		t.Fatalf("resource contents length = %d, want 1", len(result.Contents))
	}
	if err := json.Unmarshal([]byte(result.Contents[0].Text), target); err != nil {
		t.Fatalf("unmarshal resource content: %v\n%s", err, result.Contents[0].Text)
	}
}

func protocolPromptText(t *testing.T, result *mcp.GetPromptResult) string {
	t.Helper()
	if len(result.Messages) != 1 {
		t.Fatalf("prompt messages length = %d, want 1", len(result.Messages))
	}
	content, ok := result.Messages[0].Content.(*mcp.TextContent)
	if !ok {
		t.Fatalf("prompt content type = %T, want *mcp.TextContent", result.Messages[0].Content)
	}
	return content.Text
}
