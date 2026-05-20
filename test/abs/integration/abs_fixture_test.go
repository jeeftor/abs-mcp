//go:build abs_integration

package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jeeftor/abs-mcp/internal/abs"
	"github.com/jeeftor/abs-mcp/internal/mcpserver"
)

type fixtureEnv struct {
	PlainURL          string
	MetadataURL       string
	Token             string
	ExpectedAudiobook int
	ExpectedEbooks    int
}

func TestABSFixtureLibrariesAndItems(t *testing.T) {
	ctx := context.Background()
	repoRoot := findRepoRoot(t)
	env := loadFixtureEnv(t, filepath.Join(repoRoot, "test", "abs", ".env.testing"))

	if os.Getenv("ABS_INTEGRATION_SETUP") == "1" {
		runFixtureSetup(t, repoRoot)
	}

	servers := map[string]string{
		"plain":    env.PlainURL,
		"metadata": env.MetadataURL,
	}

	for name, baseURL := range servers {
		t.Run(name, func(t *testing.T) {
			client, err := abs.NewClient(strings.TrimRight(baseURL, "/"), env.Token)
			if err != nil {
				t.Fatalf("create ABS client: %v", err)
			}

			user, err := client.GetCurrentUser(ctx)
			if err != nil {
				t.Fatalf("get current user from %s fixture: %v", name, err)
			}
			if user.Username == "" {
				t.Fatalf("expected current user from %s fixture to include username", name)
			}

			libraries, err := client.GetLibraries(ctx)
			if err != nil {
				t.Fatalf("get libraries from %s fixture: %v", name, err)
			}
			byName := indexLibrariesByName(libraries)
			expectedCounts := map[string]int{
				"Audiobooks": env.ExpectedAudiobook,
				"Ebooks":     env.ExpectedEbooks,
			}

			for libraryName, expectedCount := range expectedCounts {
				lib, ok := byName[libraryName]
				if !ok {
					t.Fatalf("expected library %q in %s fixture, got %#v", libraryName, name, libraries)
				}

				roundTripLibrary, err := client.GetLibrary(ctx, lib.ID)
				if err != nil {
					t.Fatalf("get %s library %q by ID: %v", name, libraryName, err)
				}
				if roundTripLibrary.Name != libraryName {
					t.Fatalf("expected library ID %q to resolve to %q, got %q", lib.ID, libraryName, roundTripLibrary.Name)
				}

				items, err := client.GetLibraryItems(ctx, lib.ID, 100, 0)
				if err != nil {
					t.Fatalf("get %s library %q items: %v", name, libraryName, err)
				}
				if items.Total != expectedCount {
					t.Fatalf("expected %s library %q to have total %d, got %d", name, libraryName, expectedCount, items.Total)
				}
				if len(items.Results) != expectedCount {
					t.Fatalf("expected %s library %q to return %d results, got %d", name, libraryName, expectedCount, len(items.Results))
				}
				if len(items.Results) == 0 {
					t.Fatalf("expected %s library %q to contain at least one item", name, libraryName)
				}

				allItems, err := client.GetAllLibraryItems(ctx, lib.ID)
				if err != nil {
					t.Fatalf("get all %s library %q items: %v", name, libraryName, err)
				}
				if len(allItems) != expectedCount {
					t.Fatalf("expected all %s library %q items count %d, got %d", name, libraryName, expectedCount, len(allItems))
				}

				item, err := client.GetLibraryItem(ctx, items.Results[0].ID)
				if err != nil {
					t.Fatalf("get %s item %q: %v", name, items.Results[0].ID, err)
				}
				if item.ID != items.Results[0].ID {
					t.Fatalf("expected /api/items/%s to return same item ID, got %q", items.Results[0].ID, item.ID)
				}
				if item.LibraryID != "" && item.LibraryID != lib.ID {
					t.Fatalf("expected item %q library ID %q, got %q", item.ID, lib.ID, item.LibraryID)
				}
			}
		})
	}
}

func TestMCPServerAgainstABSFixture(t *testing.T) {
	ctx := context.Background()
	repoRoot := findRepoRoot(t)
	env := loadFixtureEnv(t, filepath.Join(repoRoot, "test", "abs", ".env.testing"))

	command := exec.CommandContext(ctx, "go", "run", "./cmd/abs-mcp")
	command.Dir = repoRoot
	command.Env = append(os.Environ(),
		"ABS_BASE_URL="+env.PlainURL,
		"ABS_API_KEY="+env.Token,
		"ABS_READ_ONLY=true",
	)

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "abs-mcp-fixture-test-client",
		Version: "0.1.0",
	}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: command}, nil)
	if err != nil {
		t.Fatalf("connect MCP server against ABS fixture: %v", err)
	}
	defer session.Close()

	healthResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "abs_health_check",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("call abs_health_check: %v", err)
	}
	if healthResult.IsError {
		t.Fatalf("abs_health_check returned tool error: %#v", healthResult.Content)
	}
	var health mcpserver.HealthOutput
	unmarshalStructuredOutput(t, healthResult.StructuredContent, &health)
	if !health.OK || !health.ReadOnly || health.LibraryCount != 2 {
		t.Fatalf("unexpected health output: %#v", health)
	}

	librariesResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "abs_list_libraries",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("call abs_list_libraries: %v", err)
	}
	if librariesResult.IsError {
		t.Fatalf("abs_list_libraries returned tool error: %#v", librariesResult.Content)
	}

	var libraries mcpserver.LibrariesOutput
	unmarshalStructuredOutput(t, librariesResult.StructuredContent, &libraries)
	byName := make(map[string]mcpserver.LibrarySummary, len(libraries.Libraries))
	for _, library := range libraries.Libraries {
		byName[library.Name] = library
	}

	audioLibrary, ok := byName["Audiobooks"]
	if !ok {
		t.Fatalf("Audiobooks library not found in MCP output: %#v", libraries.Libraries)
	}
	ebookLibrary, ok := byName["Ebooks"]
	if !ok {
		t.Fatalf("Ebooks library not found in MCP output: %#v", libraries.Libraries)
	}

	itemsResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "abs_list_library_items",
		Arguments: map[string]any{
			"libraryId": audioLibrary.ID,
			"limit":     env.ExpectedAudiobook,
		},
	})
	if err != nil {
		t.Fatalf("call abs_list_library_items: %v", err)
	}
	if itemsResult.IsError {
		t.Fatalf("abs_list_library_items returned tool error: %#v", itemsResult.Content)
	}

	var items mcpserver.LibraryItemsOutput
	unmarshalStructuredOutput(t, itemsResult.StructuredContent, &items)
	if items.Total != env.ExpectedAudiobook {
		t.Fatalf("Audiobooks total = %d, want %d", items.Total, env.ExpectedAudiobook)
	}
	if len(items.Items) == 0 {
		t.Fatal("expected at least one audiobook item")
	}

	filteredItemsResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "abs_list_library_items",
		Arguments: map[string]any{
			"libraryId": audioLibrary.ID,
			"limit":     1,
			"offset":    1,
			"sort":      "media.metadata.title",
			"desc":      false,
			"include":   []any{"progress"},
		},
	})
	if err != nil {
		t.Fatalf("call filtered abs_list_library_items: %v", err)
	}
	if filteredItemsResult.IsError {
		t.Fatalf("filtered abs_list_library_items returned tool error: %#v", filteredItemsResult.Content)
	}
	var filteredItems mcpserver.LibraryItemsOutput
	unmarshalStructuredOutput(t, filteredItemsResult.StructuredContent, &filteredItems)
	if filteredItems.Total != env.ExpectedAudiobook || filteredItems.Count != 1 || filteredItems.Page != 1 {
		t.Fatalf("unexpected filtered items output: %#v", filteredItems)
	}
	if filteredItems.Sort != "media.metadata.title" {
		t.Fatalf("filtered sort = %q, want media.metadata.title", filteredItems.Sort)
	}

	itemResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "abs_get_library_item",
		Arguments: map[string]any{
			"itemId": items.Items[0].ID,
		},
	})
	if err != nil {
		t.Fatalf("call abs_get_library_item: %v", err)
	}
	if itemResult.IsError {
		t.Fatalf("abs_get_library_item returned tool error: %#v", itemResult.Content)
	}
	var audioItem mcpserver.LibraryItemOutput
	unmarshalStructuredOutput(t, itemResult.StructuredContent, &audioItem)
	if audioItem.Item.ID != items.Items[0].ID {
		t.Fatalf("audio item ID = %q, want %q", audioItem.Item.ID, items.Items[0].ID)
	}
	if len(audioItem.Item.Files) == 0 {
		t.Fatalf("expected audiobook item to include file summaries: %#v", audioItem.Item)
	}

	ebookItemsResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "abs_list_library_items",
		Arguments: map[string]any{
			"libraryId": ebookLibrary.ID,
			"limit":     env.ExpectedEbooks,
		},
	})
	if err != nil {
		t.Fatalf("call ebook abs_list_library_items: %v", err)
	}
	if ebookItemsResult.IsError {
		t.Fatalf("ebook abs_list_library_items returned tool error: %#v", ebookItemsResult.Content)
	}
	var ebookItems mcpserver.LibraryItemsOutput
	unmarshalStructuredOutput(t, ebookItemsResult.StructuredContent, &ebookItems)
	if ebookItems.Total != env.ExpectedEbooks || len(ebookItems.Items) == 0 {
		t.Fatalf("unexpected ebook items output: %#v", ebookItems)
	}
	ebookItemResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "abs_get_library_item",
		Arguments: map[string]any{
			"itemId": ebookItems.Items[0].ID,
		},
	})
	if err != nil {
		t.Fatalf("call ebook abs_get_library_item: %v", err)
	}
	if ebookItemResult.IsError {
		t.Fatalf("ebook abs_get_library_item returned tool error: %#v", ebookItemResult.Content)
	}
	var ebookItem mcpserver.LibraryItemOutput
	unmarshalStructuredOutput(t, ebookItemResult.StructuredContent, &ebookItem)
	if ebookItem.Item.ID != ebookItems.Items[0].ID {
		t.Fatalf("ebook item ID = %q, want %q", ebookItem.Item.ID, ebookItems.Items[0].ID)
	}
	if len(ebookItem.Item.Files) == 0 {
		t.Fatalf("expected ebook item to include file summaries: %#v", ebookItem.Item)
	}

	searchResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "abs_search_library",
		Arguments: map[string]any{
			"libraryId": audioLibrary.ID,
			"query":     "Alice",
			"limit":     5,
		},
	})
	if err != nil {
		t.Fatalf("call abs_search_library: %v", err)
	}
	if searchResult.IsError {
		t.Fatalf("abs_search_library returned tool error: %#v", searchResult.Content)
	}

	statsResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "abs_get_library_stats",
		Arguments: map[string]any{"libraryId": audioLibrary.ID},
	})
	if err != nil {
		t.Fatalf("call abs_get_library_stats: %v", err)
	}
	if statsResult.IsError {
		t.Fatalf("abs_get_library_stats returned tool error: %#v", statsResult.Content)
	}

	filterDataResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "abs_get_filter_data",
		Arguments: map[string]any{"libraryId": audioLibrary.ID},
	})
	if err != nil {
		t.Fatalf("call abs_get_filter_data: %v", err)
	}
	if filterDataResult.IsError {
		t.Fatalf("abs_get_filter_data returned tool error: %#v", filterDataResult.Content)
	}

	metadataObjectResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "abs_get_item_metadata_object",
		Arguments: map[string]any{"itemId": items.Items[0].ID},
	})
	if err != nil {
		t.Fatalf("call abs_get_item_metadata_object: %v", err)
	}
	if metadataObjectResult.IsError {
		t.Fatalf("abs_get_item_metadata_object returned tool error: %#v", metadataObjectResult.Content)
	}

	scanResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "abs_scan_library",
		Arguments: map[string]any{
			"libraryId": audioLibrary.ID,
			"force":     true,
		},
	})
	if err != nil {
		t.Fatalf("call abs_scan_library: %v", err)
	}
	if !scanResult.IsError {
		t.Fatal("expected scan tool to be blocked by read-only mode")
	}

	scanWaitReadOnlyResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "abs_scan_library_and_wait",
		Arguments: map[string]any{
			"libraryId":     audioLibrary.ID,
			"expectedTotal": env.ExpectedAudiobook,
		},
	})
	if err != nil {
		t.Fatalf("call read-only abs_scan_library_and_wait: %v", err)
	}
	if !scanWaitReadOnlyResult.IsError {
		t.Fatal("expected scan-and-wait tool to be blocked by read-only mode")
	}

	scanItemReadOnlyResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "abs_scan_item",
		Arguments: map[string]any{
			"itemId": items.Items[0].ID,
		},
	})
	if err != nil {
		t.Fatalf("call read-only abs_scan_item: %v", err)
	}
	if !scanItemReadOnlyResult.IsError {
		t.Fatal("expected item scan tool to be blocked by read-only mode")
	}

	removeIssuesReadOnlyResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "abs_remove_library_items_with_issues",
		Arguments: map[string]any{
			"libraryId":    audioLibrary.ID,
			"confirmation": "remove issues from " + audioLibrary.ID,
		},
	})
	if err != nil {
		t.Fatalf("call read-only abs_remove_library_items_with_issues: %v", err)
	}
	if !removeIssuesReadOnlyResult.IsError {
		t.Fatal("expected remove-issues tool to be blocked by read-only mode")
	}

	resourceResult, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: "abs://libraries"})
	if err != nil {
		t.Fatalf("read abs://libraries: %v", err)
	}
	var resourceLibraries mcpserver.LibrariesOutput
	unmarshalResourceContent(t, resourceResult, &resourceLibraries)
	if resourceLibraries.Count != libraries.Count {
		t.Fatalf("resource library count = %d, tool count = %d", resourceLibraries.Count, libraries.Count)
	}

	fixtureStatusResult, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: "abs://fixture/status"})
	if err != nil {
		t.Fatalf("read abs://fixture/status: %v", err)
	}
	var fixtureStatus mcpserver.FixtureStatusOutput
	unmarshalResourceContent(t, fixtureStatusResult, &fixtureStatus)
	if !fixtureStatus.Exists || !fixtureStatus.ComposeFilePresent || !fixtureStatus.EnvFilePresent {
		t.Fatalf("unexpected fixture status: %#v", fixtureStatus)
	}
	if !fixtureStatus.TokenConfigured || fixtureStatus.TokenLength != len(env.Token) {
		t.Fatalf("unexpected fixture token status: %#v", fixtureStatus)
	}
	if strings.Contains(fixtureStatusResult.Contents[0].Text, env.Token) {
		t.Fatal("fixture status leaked ABS token")
	}

	badToken := env.Token + "-bad-token-secret"
	badTokenCommand := exec.CommandContext(ctx, "go", "run", "./cmd/abs-mcp")
	badTokenCommand.Dir = repoRoot
	badTokenCommand.Env = append(os.Environ(),
		"ABS_BASE_URL="+env.PlainURL,
		"ABS_API_KEY="+badToken,
		"ABS_READ_ONLY=true",
	)
	badTokenClient := mcp.NewClient(&mcp.Implementation{
		Name:    "abs-mcp-fixture-bad-token-test-client",
		Version: "0.1.0",
	}, nil)
	badTokenSession, err := badTokenClient.Connect(ctx, &mcp.CommandTransport{Command: badTokenCommand}, nil)
	if err != nil {
		t.Fatalf("connect bad-token MCP server against ABS fixture: %v", err)
	}
	defer badTokenSession.Close()
	badTokenResult, err := badTokenSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "abs_health_check",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("call bad-token abs_health_check: %v", err)
	}
	if !badTokenResult.IsError {
		t.Fatal("expected bad-token health check to return a tool error")
	}
	badTokenResultJSON, err := json.Marshal(badTokenResult)
	if err != nil {
		t.Fatalf("marshal bad-token tool error: %v", err)
	}
	if strings.Contains(string(badTokenResultJSON), badToken) {
		t.Fatal("bad-token health check leaked ABS token")
	}

	metadataCommand := exec.CommandContext(ctx, "go", "run", "./cmd/abs-mcp")
	metadataCommand.Dir = repoRoot
	metadataCommand.Env = append(os.Environ(),
		"ABS_BASE_URL="+env.MetadataURL,
		"ABS_API_KEY="+env.Token,
		"ABS_READ_ONLY=true",
	)
	metadataClient := mcp.NewClient(&mcp.Implementation{
		Name:    "abs-mcp-fixture-metadata-test-client",
		Version: "0.1.0",
	}, nil)
	metadataSession, err := metadataClient.Connect(ctx, &mcp.CommandTransport{Command: metadataCommand}, nil)
	if err != nil {
		t.Fatalf("connect metadata MCP server against ABS fixture: %v", err)
	}
	defer metadataSession.Close()
	metadataHealthResult, err := metadataSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "abs_health_check",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("call metadata abs_health_check: %v", err)
	}
	if metadataHealthResult.IsError {
		t.Fatalf("metadata abs_health_check returned tool error: %#v", metadataHealthResult.Content)
	}
	var metadataHealth mcpserver.HealthOutput
	unmarshalStructuredOutput(t, metadataHealthResult.StructuredContent, &metadataHealth)
	if !metadataHealth.OK || !metadataHealth.ReadOnly || metadataHealth.LibraryCount != 2 {
		t.Fatalf("unexpected metadata health output: %#v", metadataHealth)
	}

	prompts, err := session.ListPrompts(ctx, &mcp.ListPromptsParams{})
	if err != nil {
		t.Fatalf("list prompts: %v", err)
	}
	if !containsPrompt(prompts, "abs_library_audit") {
		t.Fatalf("abs_library_audit prompt not listed: %#v", prompts.Prompts)
	}

	prompt, err := session.GetPrompt(ctx, &mcp.GetPromptParams{
		Name:      "abs_library_audit",
		Arguments: map[string]string{"libraryId": audioLibrary.ID},
	})
	if err != nil {
		t.Fatalf("get abs_library_audit prompt: %v", err)
	}
	if !strings.Contains(promptMessageText(t, prompt), audioLibrary.ID) {
		t.Fatalf("prompt did not include library ID %q: %#v", audioLibrary.ID, prompt.Messages)
	}

	mutatingCommand := exec.CommandContext(ctx, "go", "run", "./cmd/abs-mcp")
	mutatingCommand.Dir = repoRoot
	mutatingCommand.Env = append(os.Environ(),
		"ABS_BASE_URL="+env.PlainURL,
		"ABS_API_KEY="+env.Token,
		"ABS_READ_ONLY=false",
	)

	mutatingClient := mcp.NewClient(&mcp.Implementation{
		Name:    "abs-mcp-fixture-mutating-test-client",
		Version: "0.1.0",
	}, nil)
	mutatingSession, err := mutatingClient.Connect(ctx, &mcp.CommandTransport{Command: mutatingCommand}, nil)
	if err != nil {
		t.Fatalf("connect mutating MCP server against ABS fixture: %v", err)
	}
	defer mutatingSession.Close()

	scanWaitResult, err := mutatingSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "abs_scan_library_and_wait",
		Arguments: map[string]any{
			"libraryId":                audioLibrary.ID,
			"force":                    false,
			"expectedTotal":            env.ExpectedAudiobook,
			"timeoutSeconds":           60,
			"pollIntervalMilliseconds": 1000,
		},
	})
	if err != nil {
		t.Fatalf("call mutating abs_scan_library_and_wait: %v", err)
	}
	if scanWaitResult.IsError {
		t.Fatalf("abs_scan_library_and_wait returned tool error: %#v", scanWaitResult.Content)
	}
	var scanWait mcpserver.ScanLibraryAndWaitOutput
	unmarshalStructuredOutput(t, scanWaitResult.StructuredContent, &scanWait)
	if !scanWait.Triggered || !scanWait.Completed || scanWait.TimedOut {
		t.Fatalf("unexpected scan-and-wait output: %#v", scanWait)
	}
	if scanWait.ObservedTotal < env.ExpectedAudiobook {
		t.Fatalf("scan observed total = %d, want at least %d", scanWait.ObservedTotal, env.ExpectedAudiobook)
	}

	scannableItemID := findScannableItemID(t, ctx, env.PlainURL, env.Token, audioLibrary.ID)
	scanItemResult, err := mutatingSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "abs_scan_item",
		Arguments: map[string]any{
			"itemId": scannableItemID,
		},
	})
	if err != nil {
		t.Fatalf("call mutating abs_scan_item: %v", err)
	}
	if scanItemResult.IsError {
		t.Fatalf("abs_scan_item returned tool error: %#v", scanItemResult.Content)
	}
	var scanItem mcpserver.ScanItemOutput
	unmarshalStructuredOutput(t, scanItemResult.StructuredContent, &scanItem)
	if !scanItem.Triggered || scanItem.ItemID != scannableItemID {
		t.Fatalf("unexpected item scan output: %#v", scanItem)
	}
	if scanItem.Result == "" {
		t.Fatalf("expected ABS item scan result, got %#v", scanItem)
	}

	removedFixtureFile := removeFirstLibraryMediaFile(t, ctx, repoRoot, env.PlainURL, env.Token, audioLibrary.ID)
	t.Logf("removed fixture file to create ABS missing-item issue: %s", removedFixtureFile)
	rescanAfterRemoveResult, err := mutatingSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "abs_scan_library_and_wait",
		Arguments: map[string]any{
			"libraryId":                audioLibrary.ID,
			"force":                    true,
			"expectedTotal":            env.ExpectedAudiobook,
			"timeoutSeconds":           60,
			"pollIntervalMilliseconds": 1000,
		},
	})
	if err != nil {
		t.Fatalf("call abs_scan_library_and_wait after fixture file removal: %v", err)
	}
	if rescanAfterRemoveResult.IsError {
		t.Fatalf("abs_scan_library_and_wait after fixture file removal returned tool error: %#v", rescanAfterRemoveResult.Content)
	}
	waitForIssueCount(t, ctx, env.PlainURL, env.Token, audioLibrary.ID, 1)

	removeIssuesResult, err := mutatingSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "abs_remove_library_items_with_issues",
		Arguments: map[string]any{
			"libraryId":          audioLibrary.ID,
			"confirmation":       "remove issues from " + audioLibrary.ID,
			"expectedIssueCount": 1,
		},
	})
	if err != nil {
		t.Fatalf("call mutating abs_remove_library_items_with_issues: %v", err)
	}
	if removeIssuesResult.IsError {
		t.Fatalf("abs_remove_library_items_with_issues returned tool error: %#v", removeIssuesResult.Content)
	}
	var removeIssues mcpserver.RemoveLibraryItemsWithIssuesOutput
	unmarshalStructuredOutput(t, removeIssuesResult.StructuredContent, &removeIssues)
	if !removeIssues.Triggered || removeIssues.IssueCountBefore != 1 || removeIssues.RemovedCount != 1 || removeIssues.RemainingIssueCount != 0 {
		t.Fatalf("unexpected remove-issues output: %#v", removeIssues)
	}
}

func indexLibrariesByName(libraries []abs.Library) map[string]abs.Library {
	byName := make(map[string]abs.Library, len(libraries))
	for _, lib := range libraries {
		byName[lib.Name] = lib
	}
	return byName
}

func findScannableItemID(t *testing.T, ctx context.Context, baseURL string, token string, libraryID string) string {
	t.Helper()

	client, err := abs.NewClient(strings.TrimRight(baseURL, "/"), token)
	if err != nil {
		t.Fatalf("create ABS client for item scan discovery: %v", err)
	}
	items, err := client.GetAllLibraryItems(ctx, libraryID)
	if err != nil {
		t.Fatalf("get all library items for item scan discovery: %v", err)
	}
	for _, item := range items {
		if !item.IsFile {
			return item.ID
		}
	}
	t.Skipf("ABS fixture has no directory-backed items in library %s; /api/items/:id/scan rejects file-style items", libraryID)
	return ""
}

func removeFirstLibraryMediaFile(
	t *testing.T,
	ctx context.Context,
	repoRoot string,
	baseURL string,
	token string,
	libraryID string,
) string {
	t.Helper()

	client, err := abs.NewClient(strings.TrimRight(baseURL, "/"), token)
	if err != nil {
		t.Fatalf("create ABS client for fixture file removal: %v", err)
	}
	items, err := client.GetAllLibraryItems(ctx, libraryID)
	if err != nil {
		t.Fatalf("get all library items for fixture file removal: %v", err)
	}
	for _, item := range items {
		detail, err := client.GetLibraryItem(ctx, item.ID)
		if err != nil {
			t.Fatalf("get library item %s for fixture file removal: %v", item.ID, err)
		}
		for _, libraryFile := range detail.LibraryFiles {
			if libraryFile.Metadata.Path == "" {
				continue
			}
			hostPath := filepath.Join(repoRoot, "test", "abs", "runtime", "plain", strings.TrimPrefix(libraryFile.Metadata.Path, "/"))
			if err := os.Remove(hostPath); err != nil {
				t.Fatalf("remove fixture media file %s: %v", hostPath, err)
			}
			return hostPath
		}
	}
	t.Fatal("could not find a library file path to remove")
	return ""
}

func waitForIssueCount(t *testing.T, ctx context.Context, baseURL string, token string, libraryID string, expected int) {
	t.Helper()

	client, err := abs.NewClient(strings.TrimRight(baseURL, "/"), token)
	if err != nil {
		t.Fatalf("create ABS client for issue polling: %v", err)
	}
	deadline := time.Now().Add(60 * time.Second)
	for {
		items, err := client.GetAllLibraryItems(ctx, libraryID)
		if err != nil {
			t.Fatalf("get all library items for issue polling: %v", err)
		}
		count := 0
		for _, item := range items {
			if item.IsMissing || item.IsInvalid {
				count++
			}
		}
		if count == expected {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for issue count %d, last count %d", expected, count)
		}
		select {
		case <-ctx.Done():
			t.Fatalf("context canceled while waiting for issue count: %v", ctx.Err())
		case <-time.After(time.Second):
		}
	}
}

func loadFixtureEnv(t *testing.T, path string) fixtureEnv {
	t.Helper()

	values := readDotEnv(t, path)
	return fixtureEnv{
		PlainURL:          requiredEnv(t, values, "ABS_PLAIN_URL"),
		MetadataURL:       requiredEnv(t, values, "ABS_METADATA_URL"),
		Token:             requiredEnv(t, values, "ABS_TOKEN"),
		ExpectedAudiobook: requiredIntEnv(t, values, "ABS_EXPECT_AUDIOBOOKS"),
		ExpectedEbooks:    requiredIntEnv(t, values, "ABS_EXPECT_BOOKS"),
	}
}

func readDotEnv(t *testing.T, path string) map[string]string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	values := make(map[string]string)
	for lineNumber, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			t.Fatalf("parse %s line %d: expected KEY=VALUE", path, lineNumber+1)
		}
		values[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"'`)
	}
	return values
}

func requiredEnv(t *testing.T, values map[string]string, key string) string {
	t.Helper()

	value := values[key]
	if value == "" {
		t.Fatalf("missing %s in test/abs/.env.testing", key)
	}
	return value
}

func requiredIntEnv(t *testing.T, values map[string]string, key string) int {
	t.Helper()

	value := requiredEnv(t, values, key)
	parsed, err := strconv.Atoi(value)
	if err != nil {
		t.Fatalf("parse %s=%q as integer: %v", key, value, err)
	}
	return parsed
}

func findRepoRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "test", "abs", ".env.testing")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repository root containing test/abs/.env.testing")
		}
		dir = parent
	}
}

func runFixtureSetup(t *testing.T, repoRoot string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "make", "abs-dev-reset-scan")
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run make abs-dev-reset-scan: %v\n%s", err, output)
	}
	fmt.Fprintln(os.Stderr, string(output))
}

func unmarshalStructuredOutput(t *testing.T, value any, target any) {
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
		t.Fatalf("resource content length = %d, want 1", len(result.Contents))
	}
	if err := json.Unmarshal([]byte(result.Contents[0].Text), target); err != nil {
		t.Fatalf("unmarshal resource content: %v\n%s", err, result.Contents[0].Text)
	}
}

func containsPrompt(result *mcp.ListPromptsResult, name string) bool {
	for _, prompt := range result.Prompts {
		if prompt.Name == name {
			return true
		}
	}
	return false
}

func promptMessageText(t *testing.T, result *mcp.GetPromptResult) string {
	t.Helper()
	if len(result.Messages) != 1 {
		t.Fatalf("prompt message length = %d, want 1", len(result.Messages))
	}
	content, ok := result.Messages[0].Content.(*mcp.TextContent)
	if !ok {
		t.Fatalf("prompt content type = %T, want *mcp.TextContent", result.Messages[0].Content)
	}
	return content.Text
}
