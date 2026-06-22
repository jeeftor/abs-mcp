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
	if !toolNames(tools)["abs_search_ebooks"] {
		t.Fatalf("expected abs_search_ebooks in tools: %#v", tools.Tools)
	}
	if !toolNames(tools)["abs_find_misorganized_items"] {
		t.Fatalf("expected abs_find_misorganized_items in tools: %#v", tools.Tools)
	}
	for _, toolName := range []string{
		"abs_list_library_authors",
		"abs_get_author",
		"abs_list_library_series",
		"abs_get_series",
		"abs_list_collections",
		"abs_get_collection",
		"abs_get_items_in_progress",
		"abs_get_listening_stats",
		"abs_list_listening_sessions",
		"abs_get_item_progress",
		"abs_list_bookmarks",
		"abs_list_backups",
		"abs_list_ereader_devices",
	} {
		if !toolNames(tools)[toolName] {
			t.Fatalf("expected %s in tools: %#v", toolName, tools.Tools)
		}
	}
	for _, toolName := range []string{
		"abs_update_item_metadata",
		"abs_update_item_progress",
		"abs_create_bookmark",
		"abs_update_bookmark",
		"abs_create_backup",
		"abs_send_ebook_to_device",
		"abs_send_ebook_by_query",
		"abs_update_item_cover",
		"abs_remove_item_cover",
		"abs_match_item",
		"abs_update_item_chapters",
		"abs_update_item_tracks",
		"abs_create_collection",
		"abs_update_collection",
		"abs_delete_collection",
		"abs_add_collection_item",
		"abs_remove_collection_item",
		"abs_create_playlist",
		"abs_update_playlist",
		"abs_delete_playlist",
		"abs_add_playlist_item",
		"abs_remove_playlist_item",
	} {
		if !toolNames(tools)[toolName] {
			t.Fatalf("expected %s in tools: %#v", toolName, tools.Tools)
		}
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

	ebookSearchResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "abs_search_ebooks",
		Arguments: map[string]any{
			"libraryId": "lib-books",
			"query":     "alice.epub",
			"limit":     10,
		},
	})
	if err != nil {
		t.Fatalf("call abs_search_ebooks: %v", err)
	}
	if ebookSearchResult.IsError {
		t.Fatalf("abs_search_ebooks returned tool error: %#v", ebookSearchResult.Content)
	}

	deviceResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "abs_list_ereader_devices",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("call abs_list_ereader_devices: %v", err)
	}
	if deviceResult.IsError {
		t.Fatalf("abs_list_ereader_devices returned tool error: %#v", deviceResult.Content)
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

	authorsResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "abs_list_library_authors",
		Arguments: map[string]any{
			"libraryId": "lib-audio",
			"limit":     100,
			"offset":    100,
			"include":   []any{"items", "series"},
		},
	})
	if err != nil {
		t.Fatalf("call abs_list_library_authors: %v", err)
	}
	if authorsResult.IsError {
		t.Fatalf("abs_list_library_authors returned tool error: %#v", authorsResult.Content)
	}
	var authorsOutput CatalogListOutput
	marshalStructuredOutput(t, authorsResult.StructuredContent, &authorsOutput)
	if authorsOutput.Page != 1 || authorsOutput.Limit != 100 || authorsOutput.Data == nil {
		t.Fatalf("unexpected authors output: %#v", authorsOutput)
	}

	for _, call := range []struct {
		name string
		args map[string]any
	}{
		{name: "abs_get_author", args: map[string]any{"id": "author-1", "include": []any{"items", "series"}}},
		{name: "abs_list_library_series", args: map[string]any{"libraryId": "lib-audio", "limit": 25, "offset": 50}},
		{name: "abs_get_series", args: map[string]any{"id": "series-1", "include": []any{"progress"}}},
		{name: "abs_list_collections", args: map[string]any{}},
		{name: "abs_get_collection", args: map[string]any{"id": "col-1", "include": []any{"items"}}},
		{name: "abs_get_items_in_progress", args: map[string]any{"limit": 3}},
		{name: "abs_get_listening_stats", args: map[string]any{}},
		{name: "abs_list_listening_sessions", args: map[string]any{"limit": 1000, "page": 2}},
		{name: "abs_get_item_progress", args: map[string]any{"itemId": "item-1"}},
		{name: "abs_list_bookmarks", args: map[string]any{"itemId": "item-1"}},
	} {
		catalogResult, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name:      call.name,
			Arguments: call.args,
		})
		if err != nil {
			t.Fatalf("call %s: %v", call.name, err)
		}
		if catalogResult.IsError {
			t.Fatalf("%s returned tool error: %#v", call.name, catalogResult.Content)
		}
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

	metadataUpdateReadOnlyResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "abs_update_item_metadata",
		Arguments: map[string]any{
			"itemId": "item-1",
		},
	})
	if err != nil {
		t.Fatalf("call abs_update_item_metadata: %v", err)
	}
	if !metadataUpdateReadOnlyResult.IsError {
		t.Fatal("expected abs_update_item_metadata to be a tool error in read-only mode")
	}

	sendEbookReadOnlyResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "abs_send_ebook_to_device",
		Arguments: map[string]any{
			"itemId":     "book-1",
			"deviceName": "Kindle",
		},
	})
	if err != nil {
		t.Fatalf("call abs_send_ebook_to_device: %v", err)
	}
	if !sendEbookReadOnlyResult.IsError {
		t.Fatal("expected abs_send_ebook_to_device to be a tool error in read-only mode")
	}

	sendEbookByQueryReadOnlyResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "abs_send_ebook_by_query",
		Arguments: map[string]any{
			"libraryId":    "lib-books",
			"query":        "alice.epub",
			"deviceName":   "Kindle",
			"confirmation": "send ebook book-1 to Kindle",
		},
	})
	if err != nil {
		t.Fatalf("call abs_send_ebook_by_query: %v", err)
	}
	if !sendEbookByQueryReadOnlyResult.IsError {
		t.Fatal("expected abs_send_ebook_by_query to be a tool error in read-only mode")
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

func TestMCPProtocolItemMutatingTools(t *testing.T) {
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
		Name:    "abs-mcp-item-mutation-test-client",
		Version: "0.1.0",
	}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("connect MCP client: %v", err)
	}
	defer session.Close()

	coverResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "abs_update_item_cover",
		Arguments: map[string]any{
			"itemId": "item-1",
			"cover":  "/covers/alice.jpg",
		},
	})
	if err != nil {
		t.Fatalf("call abs_update_item_cover: %v", err)
	}
	if coverResult.IsError {
		t.Fatalf("abs_update_item_cover returned tool error: %#v", coverResult.Content)
	}
	var coverOutput ItemMutationOutput
	marshalStructuredOutput(t, coverResult.StructuredContent, &coverOutput)
	if !coverOutput.Triggered || coverOutput.ItemID != "item-1" {
		t.Fatalf("unexpected cover update output: %#v", coverOutput)
	}

	chaptersResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "abs_update_item_chapters",
		Arguments: map[string]any{
			"itemId": "item-1",
			"chapters": []any{
				map[string]any{"title": "Intro", "start": 0, "end": 12.5},
			},
			"expectedChapterCount": 1,
		},
	})
	if err != nil {
		t.Fatalf("call abs_update_item_chapters: %v", err)
	}
	if chaptersResult.IsError {
		t.Fatalf("abs_update_item_chapters returned tool error: %#v", chaptersResult.Content)
	}
	var chaptersOutput ItemMutationOutput
	marshalStructuredOutput(t, chaptersResult.StructuredContent, &chaptersOutput)
	if !chaptersOutput.Triggered || chaptersOutput.ItemID != "item-1" {
		t.Fatalf("unexpected chapter update output: %#v", chaptersOutput)
	}

	metadataResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "abs_update_item_metadata",
		Arguments: map[string]any{
			"itemId":      "item-1",
			"title":       "Alice Updated",
			"description": "Clean description",
			"explicit":    true,
			"authors":     []any{"Lewis Carroll"},
			"series": []any{
				map[string]any{"name": "Alice Books", "sequence": "1"},
			},
			"genres": []any{"fiction"},
			"tags":   []any{"favorite"},
		},
	})
	if err != nil {
		t.Fatalf("call abs_update_item_metadata: %v", err)
	}
	if metadataResult.IsError {
		t.Fatalf("abs_update_item_metadata returned tool error: %s", contentText(metadataResult.Content))
	}
	var metadataOutput ItemMutationOutput
	marshalStructuredOutput(t, metadataResult.StructuredContent, &metadataOutput)
	if !metadataOutput.Triggered || metadataOutput.ItemID != "item-1" {
		t.Fatalf("unexpected metadata update output: %#v", metadataOutput)
	}

	progressResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "abs_update_item_progress",
		Arguments: map[string]any{
			"itemId":      "item-1",
			"episodeId":   "episode-1",
			"currentTime": 42.5,
			"progress":    0.5,
		},
	})
	if err != nil {
		t.Fatalf("call abs_update_item_progress: %v", err)
	}
	if progressResult.IsError {
		t.Fatalf("abs_update_item_progress returned tool error: %s", contentText(progressResult.Content))
	}
	var progressOutput ProgressMutationOutput
	marshalStructuredOutput(t, progressResult.StructuredContent, &progressOutput)
	if !progressOutput.Triggered || progressOutput.ItemID != "item-1" || progressOutput.EpisodeID != "episode-1" {
		t.Fatalf("unexpected progress update output: %#v", progressOutput)
	}

	bookmarkResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "abs_create_bookmark",
		Arguments: map[string]any{
			"itemId": "item-1",
			"time":   12.5,
			"title":  "Start",
		},
	})
	if err != nil {
		t.Fatalf("call abs_create_bookmark: %v", err)
	}
	if bookmarkResult.IsError {
		t.Fatalf("abs_create_bookmark returned tool error: %s", contentText(bookmarkResult.Content))
	}
	var bookmarkOutput BookmarkMutationOutput
	marshalStructuredOutput(t, bookmarkResult.StructuredContent, &bookmarkOutput)
	if !bookmarkOutput.Triggered || bookmarkOutput.Bookmark.Title != "Start" {
		t.Fatalf("unexpected bookmark output: %#v", bookmarkOutput)
	}

	updateBookmarkResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "abs_update_bookmark",
		Arguments: map[string]any{
			"itemId": "item-1",
			"time":   12.5,
			"title":  "Updated",
		},
	})
	if err != nil {
		t.Fatalf("call abs_update_bookmark: %v", err)
	}
	if updateBookmarkResult.IsError {
		t.Fatalf("abs_update_bookmark returned tool error: %s", contentText(updateBookmarkResult.Content))
	}
	var updateBookmarkOutput BookmarkMutationOutput
	marshalStructuredOutput(t, updateBookmarkResult.StructuredContent, &updateBookmarkOutput)
	if !updateBookmarkOutput.Triggered || updateBookmarkOutput.Bookmark.Title != "Updated" {
		t.Fatalf("unexpected update bookmark output: %#v", updateBookmarkOutput)
	}

	createBackupResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "abs_create_backup",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("call abs_create_backup: %v", err)
	}
	if createBackupResult.IsError {
		t.Fatalf("abs_create_backup returned tool error: %s", contentText(createBackupResult.Content))
	}
	var createBackupOutput BackupMutationOutput
	marshalStructuredOutput(t, createBackupResult.StructuredContent, &createBackupOutput)
	if !createBackupOutput.Triggered || createBackupOutput.Backup.ID != "backup-created" {
		t.Fatalf("unexpected create backup output: %#v", createBackupOutput)
	}

	createCollectionResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "abs_create_collection",
		Arguments: map[string]any{
			"libraryId":   "lib-audio",
			"name":        "Favorites",
			"description": "Good books",
			"itemIds":     []any{"item-1"},
		},
	})
	if err != nil {
		t.Fatalf("call abs_create_collection: %v", err)
	}
	if createCollectionResult.IsError {
		t.Fatalf("abs_create_collection returned tool error: %#v", createCollectionResult.Content)
	}
	var createCollectionOutput CatalogMutationOutput
	marshalStructuredOutput(t, createCollectionResult.StructuredContent, &createCollectionOutput)
	if !createCollectionOutput.Triggered || createCollectionOutput.ID != "col-1" {
		t.Fatalf("unexpected create collection output: %#v", createCollectionOutput)
	}

	updateCollectionResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "abs_update_collection",
		Arguments: map[string]any{
			"collectionId": "col-1",
			"name":         "Updated favorites",
		},
	})
	if err != nil {
		t.Fatalf("call abs_update_collection: %v", err)
	}
	if updateCollectionResult.IsError {
		t.Fatalf("abs_update_collection returned tool error: %#v", updateCollectionResult.Content)
	}
	var updateCollectionOutput CatalogMutationOutput
	marshalStructuredOutput(t, updateCollectionResult.StructuredContent, &updateCollectionOutput)
	if !updateCollectionOutput.Triggered || updateCollectionOutput.ID != "col-1" {
		t.Fatalf("unexpected update collection output: %#v", updateCollectionOutput)
	}

	addCollectionResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "abs_add_collection_item",
		Arguments: map[string]any{
			"collectionId": "col-1",
			"itemId":       "item-1",
		},
	})
	if err != nil {
		t.Fatalf("call abs_add_collection_item: %v", err)
	}
	if addCollectionResult.IsError {
		t.Fatalf("abs_add_collection_item returned tool error: %#v", addCollectionResult.Content)
	}
	var addCollectionOutput CatalogMutationOutput
	marshalStructuredOutput(t, addCollectionResult.StructuredContent, &addCollectionOutput)
	if !addCollectionOutput.Triggered || addCollectionOutput.ID != "col-1" {
		t.Fatalf("unexpected add collection output: %#v", addCollectionOutput)
	}

	deleteCollectionResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "abs_delete_collection",
		Arguments: map[string]any{
			"collectionId": "col-1",
			"confirmation": "delete collection col-1",
		},
	})
	if err != nil {
		t.Fatalf("call abs_delete_collection: %v", err)
	}
	if deleteCollectionResult.IsError {
		t.Fatalf("abs_delete_collection returned tool error: %#v", deleteCollectionResult.Content)
	}
	var deleteCollectionOutput CatalogMutationOutput
	marshalStructuredOutput(t, deleteCollectionResult.StructuredContent, &deleteCollectionOutput)
	if !deleteCollectionOutput.Triggered || deleteCollectionOutput.ID != "col-1" {
		t.Fatalf("unexpected delete collection output: %#v", deleteCollectionOutput)
	}

	removeCollectionResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "abs_remove_collection_item",
		Arguments: map[string]any{
			"collectionId": "col-1",
			"itemId":       "item-1",
			"confirmation": "remove item item-1 from collection col-1",
		},
	})
	if err != nil {
		t.Fatalf("call abs_remove_collection_item: %v", err)
	}
	if removeCollectionResult.IsError {
		t.Fatalf("abs_remove_collection_item returned tool error: %#v", removeCollectionResult.Content)
	}
	var removeCollectionOutput CatalogMutationOutput
	marshalStructuredOutput(t, removeCollectionResult.StructuredContent, &removeCollectionOutput)
	if !removeCollectionOutput.Triggered || removeCollectionOutput.ID != "col-1" {
		t.Fatalf("unexpected remove collection output: %#v", removeCollectionOutput)
	}

	createPlaylistResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "abs_create_playlist",
		Arguments: map[string]any{
			"libraryId":   "lib-audio",
			"name":        "Queue",
			"description": "Listen next",
			"items": []any{
				map[string]any{"itemId": "item-1", "episodeId": "episode-1"},
			},
		},
	})
	if err != nil {
		t.Fatalf("call abs_create_playlist: %v", err)
	}
	if createPlaylistResult.IsError {
		t.Fatalf("abs_create_playlist returned tool error: %s", contentText(createPlaylistResult.Content))
	}
	var createPlaylistOutput CatalogMutationOutput
	marshalStructuredOutput(t, createPlaylistResult.StructuredContent, &createPlaylistOutput)
	if !createPlaylistOutput.Triggered || createPlaylistOutput.ID != "pl-1" {
		t.Fatalf("unexpected create playlist output: %#v", createPlaylistOutput)
	}

	updatePlaylistResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "abs_update_playlist",
		Arguments: map[string]any{
			"playlistId": "pl-1",
			"name":       "Updated queue",
		},
	})
	if err != nil {
		t.Fatalf("call abs_update_playlist: %v", err)
	}
	if updatePlaylistResult.IsError {
		t.Fatalf("abs_update_playlist returned tool error: %#v", updatePlaylistResult.Content)
	}
	var updatePlaylistOutput CatalogMutationOutput
	marshalStructuredOutput(t, updatePlaylistResult.StructuredContent, &updatePlaylistOutput)
	if !updatePlaylistOutput.Triggered || updatePlaylistOutput.ID != "pl-1" {
		t.Fatalf("unexpected update playlist output: %#v", updatePlaylistOutput)
	}

	addPlaylistResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "abs_add_playlist_item",
		Arguments: map[string]any{
			"playlistId": "pl-1",
			"itemId":     "item-1",
			"episodeId":  "episode-1",
		},
	})
	if err != nil {
		t.Fatalf("call abs_add_playlist_item: %v", err)
	}
	if addPlaylistResult.IsError {
		t.Fatalf("abs_add_playlist_item returned tool error: %#v", addPlaylistResult.Content)
	}
	var addPlaylistOutput CatalogMutationOutput
	marshalStructuredOutput(t, addPlaylistResult.StructuredContent, &addPlaylistOutput)
	if !addPlaylistOutput.Triggered || addPlaylistOutput.ID != "pl-1" {
		t.Fatalf("unexpected add playlist output: %#v", addPlaylistOutput)
	}

	deletePlaylistResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "abs_delete_playlist",
		Arguments: map[string]any{
			"playlistId":   "pl-1",
			"confirmation": "delete playlist pl-1",
		},
	})
	if err != nil {
		t.Fatalf("call abs_delete_playlist: %v", err)
	}
	if deletePlaylistResult.IsError {
		t.Fatalf("abs_delete_playlist returned tool error: %#v", deletePlaylistResult.Content)
	}
	var deletePlaylistOutput CatalogMutationOutput
	marshalStructuredOutput(t, deletePlaylistResult.StructuredContent, &deletePlaylistOutput)
	if !deletePlaylistOutput.Triggered || deletePlaylistOutput.ID != "pl-1" {
		t.Fatalf("unexpected delete playlist output: %#v", deletePlaylistOutput)
	}

	removePlaylistResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "abs_remove_playlist_item",
		Arguments: map[string]any{
			"playlistId":   "pl-1",
			"itemId":       "item-1",
			"episodeId":    "episode-1",
			"confirmation": "remove item item-1 from playlist pl-1",
		},
	})
	if err != nil {
		t.Fatalf("call abs_remove_playlist_item: %v", err)
	}
	if removePlaylistResult.IsError {
		t.Fatalf("abs_remove_playlist_item returned tool error: %#v", removePlaylistResult.Content)
	}
	var removePlaylistOutput CatalogMutationOutput
	marshalStructuredOutput(t, removePlaylistResult.StructuredContent, &removePlaylistOutput)
	if !removePlaylistOutput.Triggered || removePlaylistOutput.ID != "pl-1" {
		t.Fatalf("unexpected remove playlist output: %#v", removePlaylistOutput)
	}

	removeResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "abs_remove_item_cover",
		Arguments: map[string]any{
			"itemId":       "item-1",
			"confirmation": "remove cover from item-1",
		},
	})
	if err != nil {
		t.Fatalf("call abs_remove_item_cover: %v", err)
	}
	if removeResult.IsError {
		t.Fatalf("abs_remove_item_cover returned tool error: %#v", removeResult.Content)
	}
	var removeOutput RemoveItemCoverOutput
	marshalStructuredOutput(t, removeResult.StructuredContent, &removeOutput)
	if !removeOutput.Triggered || removeOutput.ItemID != "item-1" {
		t.Fatalf("unexpected remove cover output: %#v", removeOutput)
	}

	if fakeClient.updateItemCoverID != "item-1" || fakeClient.updateItemChaptersID != "item-1" || fakeClient.removeItemCoverID != "item-1" {
		t.Fatalf("unexpected fake client calls: cover=%q chapters=%q remove=%q", fakeClient.updateItemCoverID, fakeClient.updateItemChaptersID, fakeClient.removeItemCoverID)
	}
	if fakeClient.addCollectionID != "col-1" || fakeClient.deleteCollectionID != "col-1" || fakeClient.removeCollectionID != "col-1" {
		t.Fatalf("unexpected collection mutation calls: add=%q delete=%q remove=%q", fakeClient.addCollectionID, fakeClient.deleteCollectionID, fakeClient.removeCollectionID)
	}
	if fakeClient.addPlaylistID != "pl-1" || fakeClient.deletePlaylistID != "pl-1" || fakeClient.removePlaylistID != "pl-1" {
		t.Fatalf("unexpected playlist mutation calls: add=%q delete=%q remove=%q", fakeClient.addPlaylistID, fakeClient.deletePlaylistID, fakeClient.removePlaylistID)
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

func contentText(contents []mcp.Content) string {
	parts := make([]string, 0, len(contents))
	for _, content := range contents {
		if text, ok := content.(*mcp.TextContent); ok {
			parts = append(parts, text.Text)
		}
	}
	return strings.Join(parts, "\n")
}
