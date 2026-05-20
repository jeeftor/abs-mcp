package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jeeftor/abs-mcp/internal/abs"
	"github.com/jeeftor/abs-mcp/internal/config"
	"github.com/jeeftor/abs-mcp/internal/version"
)

// ABSClient is the subset of the Audiobookshelf client used by MCP tools.
type ABSClient interface {
	GetCurrentUser(context.Context) (*abs.User, error)
	GetLibraries(context.Context) ([]abs.Library, error)
	GetLibrary(context.Context, string) (*abs.Library, error)
	GetLibraryItems(context.Context, string, int, int) (*abs.LibraryItemsResponse, error)
	GetLibraryItemsWithOptions(context.Context, string, abs.LibraryItemsOptions) (*abs.LibraryItemsResponse, error)
	GetAllLibraryItems(context.Context, string) ([]abs.LibraryItem, error)
	GetLibraryItem(context.Context, string) (*abs.LibraryItem, error)
	SearchLibrary(context.Context, string, string, int) (abs.JSONValue, error)
	GetLibraryStats(context.Context, string) (abs.JSONValue, error)
	GetLibraryFilterData(context.Context, string) (abs.JSONValue, error)
	GetItemMetadataObject(context.Context, string) (abs.JSONValue, error)
	ScanLibrary(context.Context, string, bool) error
	RemoveLibraryItemsWithIssues(context.Context, string) error
	ScanItem(context.Context, string) (*abs.ScanItemResponse, error)
	UpdateItemCover(context.Context, string, string) (abs.JSONValue, error)
	RemoveItemCover(context.Context, string) error
	UpdateItemChapters(context.Context, string, []abs.Chapter) (abs.JSONValue, error)
}

// Server owns MCP tool handlers and their dependencies.
type Server struct {
	cfg          config.Config
	client       ABSClient
	apiInventory any
}

// New creates a testable MCP server wrapper.
func New(cfg config.Config, client ABSClient) *Server {
	server := &Server{
		cfg:    cfg,
		client: client,
	}
	server.apiInventory = loadAPIInventory()
	return server
}

// MCPServer returns an SDK MCP server with all implemented tools registered.
func (s *Server) MCPServer() *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "abs-mcp",
		Title:   "Audiobookshelf MCP",
		Version: version.Version,
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_health_check",
		Title:       "Check Audiobookshelf health",
		Description: "Validate Audiobookshelf authentication and return a sanitized server summary.",
	}, s.HealthCheck)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_list_libraries",
		Title:       "List Audiobookshelf libraries",
		Description: "List Audiobookshelf library IDs, names, media types, and folders.",
	}, s.ListLibraries)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_get_library",
		Title:       "Get Audiobookshelf library",
		Description: "Get one Audiobookshelf library by ID.",
	}, s.GetLibrary)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_list_library_items",
		Title:       "List Audiobookshelf library items",
		Description: "List a bounded page of items from one Audiobookshelf library.",
	}, s.ListLibraryItems)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_get_library_item",
		Title:       "Get Audiobookshelf library item",
		Description: "Get one Audiobookshelf library item by ID.",
	}, s.GetLibraryItem)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_search_library",
		Title:       "Search Audiobookshelf library",
		Description: "Search one Audiobookshelf library with a bounded result limit.",
	}, s.SearchLibrary)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_get_library_stats",
		Title:       "Get Audiobookshelf library stats",
		Description: "Get source-backed Audiobookshelf stats for one library.",
	}, s.GetLibraryStats)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_get_filter_data",
		Title:       "Get Audiobookshelf filter data",
		Description: "Get filter data for one Audiobookshelf library.",
	}, s.GetLibraryFilterData)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_get_item_metadata_object",
		Title:       "Get Audiobookshelf item metadata object",
		Description: "Get the raw ABS metadata object for one audiobook item. Requires sufficient ABS permissions.",
	}, s.GetItemMetadataObject)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_find_misorganized_items",
		Title:       "Find misorganized Audiobookshelf items",
		Description: "Audit item paths against author/title or author/series/title layout conventions without moving files.",
	}, s.FindMisorganizedItems)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_scan_library",
		Title:       "Scan Audiobookshelf library",
		Description: "Trigger an Audiobookshelf library scan. Blocked when ABS_READ_ONLY is true.",
	}, s.ScanLibrary)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_scan_library_and_wait",
		Title:       "Scan Audiobookshelf library and wait",
		Description: "Trigger a library scan, then poll the library item count until the expected total is observed or a timeout is reached. Blocked when ABS_READ_ONLY is true.",
	}, s.ScanLibraryAndWait)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_scan_item",
		Title:       "Scan Audiobookshelf item",
		Description: "Rescan one directory-backed Audiobookshelf library item by ID. Blocked when ABS_READ_ONLY is true.",
	}, s.ScanItem)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_update_item_metadata",
		Title:       "Update Audiobookshelf item metadata",
		Description: "Planned tool for updating selected metadata on one item. Registered for discovery, blocked when ABS_READ_ONLY is true, and not implemented until safe typed fields are source-verified.",
	}, s.UpdateItemMetadata)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_update_item_cover",
		Title:       "Update Audiobookshelf item cover",
		Description: "Update one Audiobookshelf item cover from an ABS-visible path using PATCH. Blocked when ABS_READ_ONLY is true.",
	}, s.UpdateItemCover)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_remove_item_cover",
		Title:       "Remove Audiobookshelf item cover",
		Description: "Remove one Audiobookshelf item cover. Requires exact confirmation and is blocked when ABS_READ_ONLY is true.",
	}, s.RemoveItemCover)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_match_item",
		Title:       "Match Audiobookshelf item metadata",
		Description: "Planned tool for running Audiobookshelf item matching. Registered for discovery, blocked when ABS_READ_ONLY is true, and not implemented until a restricted source-verified input is available.",
	}, s.MatchItem)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_update_item_chapters",
		Title:       "Update Audiobookshelf item chapters",
		Description: "Replace one Audiobookshelf item chapter list with typed chapters after an expected-count guard. Blocked when ABS_READ_ONLY is true.",
	}, s.UpdateItemChapters)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_update_item_tracks",
		Title:       "Update Audiobookshelf item tracks",
		Description: "Planned tool for replacing item tracks. Registered for discovery, blocked when ABS_READ_ONLY is true, and not implemented until source and fixture behavior are verified.",
	}, s.UpdateItemTracks)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_create_collection",
		Title:       "Create Audiobookshelf collection",
		Description: "Planned tool for creating a collection. Registered for discovery, blocked when ABS_READ_ONLY is true, and not implemented until source and fixture behavior are verified.",
	}, s.CreateCollection)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_update_collection",
		Title:       "Update Audiobookshelf collection",
		Description: "Planned tool for updating a collection. Registered for discovery, blocked when ABS_READ_ONLY is true, and not implemented until source and fixture behavior are verified.",
	}, s.UpdateCollection)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_delete_collection",
		Title:       "Delete Audiobookshelf collection",
		Description: "Planned destructive tool for deleting a collection. Requires exact confirmation, is blocked when ABS_READ_ONLY is true, and is not implemented until source and fixture behavior are verified.",
	}, s.DeleteCollection)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_add_collection_item",
		Title:       "Add Audiobookshelf collection item",
		Description: "Planned tool for adding an item to a collection. Registered for discovery, blocked when ABS_READ_ONLY is true, and not implemented until source and fixture behavior are verified.",
	}, s.AddCollectionItem)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_remove_collection_item",
		Title:       "Remove Audiobookshelf collection item",
		Description: "Planned destructive tool for removing an item from a collection. Requires exact confirmation, is blocked when ABS_READ_ONLY is true, and is not implemented until source and fixture behavior are verified.",
	}, s.RemoveCollectionItem)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_create_playlist",
		Title:       "Create Audiobookshelf playlist",
		Description: "Planned tool for creating a playlist. Registered for discovery, blocked when ABS_READ_ONLY is true, and not implemented until source and fixture behavior are verified.",
	}, s.CreatePlaylist)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_update_playlist",
		Title:       "Update Audiobookshelf playlist",
		Description: "Planned tool for updating a playlist. Registered for discovery, blocked when ABS_READ_ONLY is true, and not implemented until source and fixture behavior are verified.",
	}, s.UpdatePlaylist)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_delete_playlist",
		Title:       "Delete Audiobookshelf playlist",
		Description: "Planned destructive tool for deleting a playlist. Requires exact confirmation, is blocked when ABS_READ_ONLY is true, and is not implemented until source and fixture behavior are verified.",
	}, s.DeletePlaylist)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_add_playlist_item",
		Title:       "Add Audiobookshelf playlist item",
		Description: "Planned tool for adding an item to a playlist. Registered for discovery, blocked when ABS_READ_ONLY is true, and not implemented until source and fixture behavior are verified.",
	}, s.AddPlaylistItem)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_remove_playlist_item",
		Title:       "Remove Audiobookshelf playlist item",
		Description: "Planned destructive tool for removing an item from a playlist. Requires exact confirmation, is blocked when ABS_READ_ONLY is true, and is not implemented until source and fixture behavior are verified.",
	}, s.RemovePlaylistItem)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_remove_library_items_with_issues",
		Title:       "Remove Audiobookshelf library items with issues",
		Description: "Remove missing or invalid items from one Audiobookshelf library. Requires exact confirmation and is blocked when ABS_READ_ONLY is true.",
	}, s.RemoveLibraryItemsWithIssues)

	s.RegisterResources(server)
	s.RegisterPrompts(server)

	return server
}

func loadAPIInventory() any {
	data, err := os.ReadFile("docs/api-inventory/generated/abs-api-inventory.json")
	if err != nil {
		return map[string]any{
			"available": false,
			"error":     err.Error(),
		}
	}
	var inventory any
	if err := json.Unmarshal(data, &inventory); err != nil {
		return map[string]any{
			"available": false,
			"error":     err.Error(),
		}
	}
	return inventory
}

// EmptyInput is used for tools without input arguments.
type EmptyInput struct{}

// HealthOutput is returned by abs_health_check.
type HealthOutput struct {
	OK           bool   `json:"ok" jsonschema:"Whether Audiobookshelf responded to authenticated requests."`
	BaseURL      string `json:"baseUrl" jsonschema:"Configured Audiobookshelf base URL."`
	ReadOnly     bool   `json:"readOnly" jsonschema:"Whether mutating tools are blocked by configuration."`
	Username     string `json:"username" jsonschema:"Authenticated Audiobookshelf username."`
	UserType     string `json:"userType" jsonschema:"Authenticated Audiobookshelf user type."`
	LibraryCount int    `json:"libraryCount" jsonschema:"Number of visible Audiobookshelf libraries."`
}

// LibrariesOutput is returned by abs_list_libraries.
type LibrariesOutput struct {
	Libraries []LibrarySummary `json:"libraries" jsonschema:"Audiobookshelf libraries visible to the token."`
	Count     int              `json:"count" jsonschema:"Number of libraries returned."`
}

// LibraryInput identifies one ABS library.
type LibraryInput struct {
	LibraryID string `json:"libraryId" jsonschema:"Audiobookshelf library ID."`
}

// LibraryOutput is returned by abs_get_library.
type LibraryOutput struct {
	Library LibrarySummary `json:"library" jsonschema:"Audiobookshelf library summary."`
}

// LibraryItemsInput selects a bounded page of library items.
type LibraryItemsInput struct {
	LibraryID      string   `json:"libraryId" jsonschema:"Audiobookshelf library ID."`
	Limit          int      `json:"limit,omitempty" jsonschema:"Maximum number of items to return. Defaults to 25 and is capped at 100."`
	Offset         int      `json:"offset,omitempty" jsonschema:"Zero-based item offset. Must be a multiple of limit because ABS uses page-based pagination."`
	Sort           string   `json:"sort,omitempty" jsonschema:"ABS sort key, such as media.metadata.title."`
	Desc           bool     `json:"desc,omitempty" jsonschema:"Whether to sort descending."`
	Filter         string   `json:"filter,omitempty" jsonschema:"ABS filter expression from filterdata, such as issues.true."`
	Include        []string `json:"include,omitempty" jsonschema:"Optional ABS include values to request."`
	Minified       bool     `json:"minified,omitempty" jsonschema:"Whether to request minified ABS items."`
	CollapseSeries bool     `json:"collapseSeries,omitempty" jsonschema:"Whether ABS should collapse series when supported by the filter."`
}

// LibraryItemsOutput is returned by abs_list_library_items.
type LibraryItemsOutput struct {
	Items  []LibraryItemSummary `json:"items" jsonschema:"Library items in the requested page."`
	Total  int                  `json:"total" jsonschema:"Total item count reported by Audiobookshelf."`
	Limit  int                  `json:"limit" jsonschema:"Page size used for the request."`
	Offset int                  `json:"offset" jsonschema:"Offset used for the request."`
	Page   int                  `json:"page" jsonschema:"ABS page used for the request."`
	Count  int                  `json:"count" jsonschema:"Number of items returned in this response."`
	Sort   string               `json:"sort,omitempty" jsonschema:"ABS sort key used for this request."`
	Desc   bool                 `json:"desc,omitempty" jsonschema:"Whether descending sort was requested."`
	Filter string               `json:"filter,omitempty" jsonschema:"ABS filter used for this request."`
}

// LibraryItemInput identifies one ABS item.
type LibraryItemInput struct {
	ItemID string `json:"itemId" jsonschema:"Audiobookshelf library item ID."`
}

// LibraryItemOutput is returned by abs_get_library_item.
type LibraryItemOutput struct {
	Item LibraryItemSummary `json:"item" jsonschema:"Audiobookshelf library item summary."`
}

// SearchLibraryInput selects a bounded library search.
type SearchLibraryInput struct {
	LibraryID string `json:"libraryId" jsonschema:"Audiobookshelf library ID."`
	Query     string `json:"query" jsonschema:"Search query text."`
	Limit     int    `json:"limit,omitempty" jsonschema:"Maximum number of search results. Defaults to 12 and is capped at 50."`
}

// SearchLibraryOutput is returned by abs_search_library.
type SearchLibraryOutput struct {
	LibraryID string        `json:"libraryId"`
	Query     string        `json:"query"`
	Limit     int           `json:"limit"`
	Data      abs.JSONValue `json:"data" jsonschema:"Raw Audiobookshelf search response."`
}

// LibraryRawInput identifies one library for raw read-only endpoints.
type LibraryRawInput struct {
	LibraryID string `json:"libraryId" jsonschema:"Audiobookshelf library ID."`
}

// LibraryRawOutput is returned by raw library inspection tools.
type LibraryRawOutput struct {
	LibraryID string        `json:"libraryId"`
	Data      abs.JSONValue `json:"data" jsonschema:"Raw Audiobookshelf response."`
}

// MetadataObjectOutput is returned by abs_get_item_metadata_object.
type MetadataObjectOutput struct {
	ItemID string        `json:"itemId"`
	Data   abs.JSONValue `json:"data" jsonschema:"Raw Audiobookshelf metadata object response."`
}

// FindMisorganizedItemsInput selects one read-only library layout audit.
type FindMisorganizedItemsInput struct {
	LibraryID        string `json:"libraryId" jsonschema:"Audiobookshelf library ID to audit."`
	Convention       string `json:"convention,omitempty" jsonschema:"Layout convention: auto, author-title, or author-series-title. Defaults to auto."`
	Limit            int    `json:"limit,omitempty" jsonschema:"Maximum findings to return. Defaults to 50 and is capped at 200."`
	IncludeOrganized bool   `json:"includeOrganized,omitempty" jsonschema:"Whether to include organized items in the returned item list."`
}

// FindMisorganizedItemsOutput is returned by abs_find_misorganized_items.
type FindMisorganizedItemsOutput struct {
	LibraryID            string            `json:"libraryId" jsonschema:"Audiobookshelf library ID audited."`
	Convention           string            `json:"convention" jsonschema:"Layout convention used for the audit."`
	CheckedCount         int               `json:"checkedCount" jsonschema:"Number of items checked."`
	OrganizedCount       int               `json:"organizedCount" jsonschema:"Number of items that matched the expected layout."`
	MisorganizedCount    int               `json:"misorganizedCount" jsonschema:"Number of items that did not match the expected layout."`
	UnclassifiableCount  int               `json:"unclassifiableCount" jsonschema:"Number of items that could not be classified due to missing metadata or path data."`
	ReturnedCount        int               `json:"returnedCount" jsonschema:"Number of item findings returned."`
	Limit                int               `json:"limit" jsonschema:"Maximum findings requested after normalization."`
	Items                []LayoutAuditItem `json:"items" jsonschema:"Layout audit findings."`
	Truncated            bool              `json:"truncated" jsonschema:"Whether additional findings were omitted by the limit."`
	SummaryByReason      map[string]int    `json:"summaryByReason" jsonschema:"Finding counts grouped by reason."`
	SupportedConventions []string          `json:"supportedConventions" jsonschema:"Layout conventions supported by this tool."`
	LibraryFolders       []FolderSummary   `json:"libraryFolders,omitempty" jsonschema:"Library root folders used to derive relative paths."`
}

// LayoutAuditItem describes one item path classification.
type LayoutAuditItem struct {
	ItemID          string   `json:"itemId" jsonschema:"Audiobookshelf library item ID."`
	Title           string   `json:"title,omitempty" jsonschema:"Metadata title used for expected path calculation."`
	Author          string   `json:"author,omitempty" jsonschema:"Metadata author used for expected path calculation."`
	Series          string   `json:"series,omitempty" jsonschema:"Metadata series used for expected path calculation."`
	CurrentRelPath  string   `json:"currentRelPath,omitempty" jsonschema:"Current item path relative to the library folder when known."`
	ExpectedRelPath string   `json:"expectedRelPath,omitempty" jsonschema:"Expected item directory for the selected layout convention."`
	Convention      string   `json:"convention" jsonschema:"Layout convention used for this item."`
	Organized       bool     `json:"organized" jsonschema:"Whether the current path matches the expected path."`
	Classifiable    bool     `json:"classifiable" jsonschema:"Whether enough metadata and path data was available to classify the item."`
	Confidence      string   `json:"confidence" jsonschema:"Confidence level for the classification: high, medium, or low."`
	Reasons         []string `json:"reasons,omitempty" jsonschema:"Machine-readable reasons for misorganization or uncertainty."`
	IsMissing       bool     `json:"isMissing" jsonschema:"Whether ABS marks the item as missing."`
	IsInvalid       bool     `json:"isInvalid" jsonschema:"Whether ABS marks the item as invalid."`
}

// ScanLibraryInput identifies one ABS library scan request.
type ScanLibraryInput struct {
	LibraryID string `json:"libraryId" jsonschema:"Audiobookshelf library ID to scan."`
	Force     bool   `json:"force" jsonschema:"Whether to request a forced scan."`
}

// ScanLibraryOutput is returned by abs_scan_library.
type ScanLibraryOutput struct {
	Triggered bool   `json:"triggered" jsonschema:"Whether the scan request was sent."`
	LibraryID string `json:"libraryId" jsonschema:"Audiobookshelf library ID requested for scanning."`
	Force     bool   `json:"force" jsonschema:"Whether the scan was requested with force=true."`
}

// RemoveLibraryItemsWithIssuesInput identifies one confirmed cleanup request.
type RemoveLibraryItemsWithIssuesInput struct {
	LibraryID          string `json:"libraryId" jsonschema:"Audiobookshelf library ID to clean."`
	Confirmation       string `json:"confirmation" jsonschema:"Exact confirmation text. Must be: remove issues from <libraryId>."`
	ExpectedIssueCount int    `json:"expectedIssueCount,omitempty" jsonschema:"Optional expected number of missing or invalid items. When non-zero, cleanup is blocked unless it matches the observed issue count."`
}

// RemoveLibraryItemsWithIssuesOutput is returned by abs_remove_library_items_with_issues.
type RemoveLibraryItemsWithIssuesOutput struct {
	Triggered           bool     `json:"triggered" jsonschema:"Whether the ABS delete request was sent."`
	LibraryID           string   `json:"libraryId" jsonschema:"Audiobookshelf library ID requested for cleanup."`
	IssueCountBefore    int      `json:"issueCountBefore" jsonschema:"Missing or invalid item count observed before cleanup."`
	RemovedCount        int      `json:"removedCount" jsonschema:"Number of issue items expected to have been removed."`
	RemainingIssueCount int      `json:"remainingIssueCount" jsonschema:"Missing or invalid item count observed after cleanup."`
	IssueItemIDs        []string `json:"issueItemIds,omitempty" jsonschema:"IDs of missing or invalid items observed before cleanup, capped at 100."`
}

// ScanLibraryAndWaitInput identifies a scan request and bounded polling window.
type ScanLibraryAndWaitInput struct {
	LibraryID                string `json:"libraryId" jsonschema:"Audiobookshelf library ID to scan."`
	Force                    bool   `json:"force" jsonschema:"Whether to request a forced scan."`
	ExpectedTotal            int    `json:"expectedTotal,omitempty" jsonschema:"Optional minimum item total to wait for. If omitted, the tool observes item count once after triggering the scan."`
	TimeoutSeconds           int    `json:"timeoutSeconds,omitempty" jsonschema:"Maximum seconds to wait. Defaults to 30 and is capped at 300."`
	PollIntervalMilliseconds int    `json:"pollIntervalMilliseconds,omitempty" jsonschema:"Milliseconds between item-count polls. Defaults to 1000 and is capped at 60000."`
}

// ScanLibraryAndWaitOutput is returned by abs_scan_library_and_wait.
type ScanLibraryAndWaitOutput struct {
	Triggered           bool   `json:"triggered" jsonschema:"Whether the scan request was sent."`
	Completed           bool   `json:"completed" jsonschema:"Whether the expected total was observed before timeout."`
	TimedOut            bool   `json:"timedOut" jsonschema:"Whether polling reached the timeout before completion."`
	LibraryID           string `json:"libraryId" jsonschema:"Audiobookshelf library ID requested for scanning."`
	Force               bool   `json:"force" jsonschema:"Whether the scan was requested with force=true."`
	ExpectedTotal       int    `json:"expectedTotal,omitempty" jsonschema:"Minimum item total requested by the caller."`
	ObservedTotal       int    `json:"observedTotal" jsonschema:"Last total item count observed from Audiobookshelf."`
	Attempts            int    `json:"attempts" jsonschema:"Number of item-count polls performed."`
	TimeoutSeconds      int    `json:"timeoutSeconds" jsonschema:"Timeout seconds used for this request."`
	PollIntervalMillis  int    `json:"pollIntervalMilliseconds" jsonschema:"Poll interval milliseconds used for this request."`
	ElapsedMilliseconds int64  `json:"elapsedMilliseconds" jsonschema:"Elapsed polling time in milliseconds."`
}

// ScanItemInput identifies one ABS item scan request.
type ScanItemInput struct {
	ItemID string `json:"itemId" jsonschema:"Audiobookshelf library item ID to scan."`
}

// ScanItemOutput is returned by abs_scan_item.
type ScanItemOutput struct {
	Triggered bool   `json:"triggered" jsonschema:"Whether the scan request was sent."`
	ItemID    string `json:"itemId" jsonschema:"Audiobookshelf library item ID requested for scanning."`
	Result    string `json:"result,omitempty" jsonschema:"Audiobookshelf scan result string, when returned by ABS."`
}

// ItemPayloadInput identifies one item mutation with a caller-provided payload.
type ItemPayloadInput struct {
	ItemID  string        `json:"itemId" jsonschema:"Audiobookshelf library item ID to mutate."`
	Payload abs.JSONValue `json:"payload,omitempty" jsonschema:"Planned mutation payload. Exact shape is not committed until source and fixture behavior are verified."`
}

// ConfirmedItemInput identifies one destructive item mutation.
type ConfirmedItemInput struct {
	ItemID       string `json:"itemId" jsonschema:"Audiobookshelf library item ID to mutate."`
	Confirmation string `json:"confirmation" jsonschema:"Exact confirmation text required by the tool."`
}

// UpdateItemCoverInput identifies one item cover update request.
type UpdateItemCoverInput struct {
	ItemID string `json:"itemId" jsonschema:"Audiobookshelf library item ID to mutate."`
	Cover  string `json:"cover" jsonschema:"ABS-visible cover path to set on the item."`
}

// ChapterInput is one typed item chapter.
type ChapterInput struct {
	Title string  `json:"title" jsonschema:"Chapter title."`
	Start float64 `json:"start" jsonschema:"Chapter start time in seconds."`
	End   float64 `json:"end" jsonschema:"Chapter end time in seconds."`
}

// UpdateItemChaptersInput identifies one guarded chapter replacement request.
type UpdateItemChaptersInput struct {
	ItemID               string         `json:"itemId" jsonschema:"Audiobookshelf library item ID to mutate."`
	Chapters             []ChapterInput `json:"chapters" jsonschema:"Complete replacement chapter list."`
	ExpectedChapterCount int            `json:"expectedChapterCount" jsonschema:"Expected chapter count. Must exactly equal len(chapters)."`
}

// MatchItemInput identifies one item match request.
type MatchItemInput struct {
	ItemID       string        `json:"itemId" jsonschema:"Audiobookshelf library item ID to match."`
	Provider     string        `json:"provider,omitempty" jsonschema:"Optional metadata provider to use when matching."`
	Payload      abs.JSONValue `json:"payload,omitempty" jsonschema:"Planned match payload. Exact shape is not committed until source and fixture behavior are verified."`
	Confirmation string        `json:"confirmation,omitempty" jsonschema:"Reserved for source-verified overwrite confirmation if matching is destructive."`
}

// CollectionInput identifies one planned collection create/update request.
type CollectionInput struct {
	CollectionID string        `json:"collectionId,omitempty" jsonschema:"Audiobookshelf collection ID for updates."`
	Name         string        `json:"name,omitempty" jsonschema:"Collection name."`
	Payload      abs.JSONValue `json:"payload,omitempty" jsonschema:"Planned collection payload. Exact shape is not committed until source and fixture behavior are verified."`
}

// ConfirmedCollectionInput identifies one planned destructive collection request.
type ConfirmedCollectionInput struct {
	CollectionID string `json:"collectionId" jsonschema:"Audiobookshelf collection ID to mutate."`
	Confirmation string `json:"confirmation" jsonschema:"Exact confirmation text required by the tool."`
}

// CollectionItemInput identifies one planned collection item membership request.
type CollectionItemInput struct {
	CollectionID string `json:"collectionId" jsonschema:"Audiobookshelf collection ID to mutate."`
	ItemID       string `json:"itemId" jsonschema:"Audiobookshelf library item ID to add or remove."`
	Confirmation string `json:"confirmation,omitempty" jsonschema:"Exact confirmation text required for removal."`
}

// PlaylistInput identifies one planned playlist create/update request.
type PlaylistInput struct {
	PlaylistID string        `json:"playlistId,omitempty" jsonschema:"Audiobookshelf playlist ID for updates."`
	Name       string        `json:"name,omitempty" jsonschema:"Playlist name."`
	Payload    abs.JSONValue `json:"payload,omitempty" jsonschema:"Planned playlist payload. Exact shape is not committed until source and fixture behavior are verified."`
}

// ConfirmedPlaylistInput identifies one planned destructive playlist request.
type ConfirmedPlaylistInput struct {
	PlaylistID   string `json:"playlistId" jsonschema:"Audiobookshelf playlist ID to mutate."`
	Confirmation string `json:"confirmation" jsonschema:"Exact confirmation text required by the tool."`
}

// PlaylistItemInput identifies one planned playlist item membership request.
type PlaylistItemInput struct {
	PlaylistID   string `json:"playlistId" jsonschema:"Audiobookshelf playlist ID to mutate."`
	ItemID       string `json:"itemId" jsonschema:"Audiobookshelf library item ID to add or remove."`
	EpisodeID    string `json:"episodeId,omitempty" jsonschema:"Optional podcast episode ID when mutating a podcast playlist item."`
	Confirmation string `json:"confirmation,omitempty" jsonschema:"Exact confirmation text required for removal."`
}

// PlannedMutationOutput is reserved for future implemented mutating tools.
type PlannedMutationOutput struct {
	Triggered   bool   `json:"triggered" jsonschema:"Whether an Audiobookshelf mutation request was sent."`
	Tool        string `json:"tool" jsonschema:"MCP tool name."`
	Route       string `json:"route" jsonschema:"Audiobookshelf API route planned for this tool."`
	Implemented bool   `json:"implemented" jsonschema:"Whether this planned mutation is implemented."`
}

// ItemMutationOutput is returned by typed item mutation tools.
type ItemMutationOutput struct {
	Triggered bool          `json:"triggered" jsonschema:"Whether an Audiobookshelf mutation request was sent."`
	ItemID    string        `json:"itemId" jsonschema:"Audiobookshelf library item ID requested for mutation."`
	Data      abs.JSONValue `json:"data,omitempty" jsonschema:"Raw Audiobookshelf response, when returned by ABS."`
}

// RemoveItemCoverOutput is returned by abs_remove_item_cover.
type RemoveItemCoverOutput struct {
	Triggered bool   `json:"triggered" jsonschema:"Whether the cover removal request was sent."`
	ItemID    string `json:"itemId" jsonschema:"Audiobookshelf library item ID requested for cover removal."`
}

// LibrarySummary is a compact library shape suitable for MCP output.
type LibrarySummary struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	MediaType string          `json:"mediaType"`
	Folders   []FolderSummary `json:"folders"`
}

// FolderSummary is a compact library folder shape.
type FolderSummary struct {
	ID       string `json:"id,omitempty"`
	Path     string `json:"path,omitempty"`
	FullPath string `json:"fullPath,omitempty"`
}

// LibraryItemSummary is a compact item shape suitable for bounded MCP output.
type LibraryItemSummary struct {
	ID        string             `json:"id"`
	LibraryID string             `json:"libraryId"`
	MediaType string             `json:"mediaType"`
	Path      string             `json:"path"`
	RelPath   string             `json:"relPath,omitempty"`
	IsMissing bool               `json:"isMissing"`
	IsInvalid bool               `json:"isInvalid"`
	Title     string             `json:"title,omitempty"`
	Author    string             `json:"author,omitempty"`
	Series    string             `json:"series,omitempty"`
	Files     []MediaFileSummary `json:"files,omitempty"`
	Duration  float64            `json:"duration,omitempty"`
	Size      int64              `json:"size,omitempty"`
}

// MediaFileSummary is a compact file shape for one ABS library item.
type MediaFileSummary struct {
	FileType string `json:"fileType,omitempty"`
	Filename string `json:"filename,omitempty"`
	Path     string `json:"path,omitempty"`
	RelPath  string `json:"relPath,omitempty"`
	Size     int64  `json:"size,omitempty"`
}

// HealthCheck validates authentication and basic library access.
func (s *Server) HealthCheck(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	_ EmptyInput,
) (*mcp.CallToolResult, HealthOutput, error) {
	user, err := s.client.GetCurrentUser(ctx)
	if err != nil {
		return nil, HealthOutput{}, fmt.Errorf("get current ABS user: %w", err)
	}
	libraries, err := s.client.GetLibraries(ctx)
	if err != nil {
		return nil, HealthOutput{}, fmt.Errorf("list ABS libraries: %w", err)
	}

	return nil, HealthOutput{
		OK:           true,
		BaseURL:      s.cfg.ABSBaseURL,
		ReadOnly:     s.cfg.ReadOnly,
		Username:     user.Username,
		UserType:     user.Type,
		LibraryCount: len(libraries),
	}, nil
}

// ListLibraries returns visible ABS libraries.
func (s *Server) ListLibraries(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	_ EmptyInput,
) (*mcp.CallToolResult, LibrariesOutput, error) {
	libraries, err := s.client.GetLibraries(ctx)
	if err != nil {
		return nil, LibrariesOutput{}, fmt.Errorf("list ABS libraries: %w", err)
	}
	summaries := summarizeLibraries(libraries)
	return nil, LibrariesOutput{Libraries: summaries, Count: len(summaries)}, nil
}

// GetLibrary returns one ABS library.
func (s *Server) GetLibrary(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input LibraryInput,
) (*mcp.CallToolResult, LibraryOutput, error) {
	if input.LibraryID == "" {
		return nil, LibraryOutput{}, fmt.Errorf("libraryId is required")
	}
	library, err := s.client.GetLibrary(ctx, input.LibraryID)
	if err != nil {
		return nil, LibraryOutput{}, fmt.Errorf("get ABS library %q: %w", input.LibraryID, err)
	}
	return nil, LibraryOutput{Library: summarizeLibrary(*library)}, nil
}

// ListLibraryItems returns a bounded page of ABS library items.
func (s *Server) ListLibraryItems(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input LibraryItemsInput,
) (*mcp.CallToolResult, LibraryItemsOutput, error) {
	if input.LibraryID == "" {
		return nil, LibraryItemsOutput{}, fmt.Errorf("libraryId is required")
	}
	limit, err := normalizeLimit(input.Limit)
	if err != nil {
		return nil, LibraryItemsOutput{}, err
	}
	if input.Offset < 0 {
		return nil, LibraryItemsOutput{}, fmt.Errorf("offset must be greater than or equal to 0")
	}
	page, err := pageFromOffset(input.Offset, limit)
	if err != nil {
		return nil, LibraryItemsOutput{}, err
	}

	response, err := s.client.GetLibraryItemsWithOptions(ctx, input.LibraryID, abs.LibraryItemsOptions{
		Limit:          limit,
		Page:           page,
		Sort:           input.Sort,
		Desc:           input.Desc,
		Filter:         input.Filter,
		Include:        sanitizeInclude(input.Include),
		Minified:       input.Minified,
		CollapseSeries: input.CollapseSeries,
	})
	if err != nil {
		return nil, LibraryItemsOutput{}, fmt.Errorf("list ABS library %q items: %w", input.LibraryID, err)
	}
	items := summarizeItems(response.Results)
	offset := response.Offset
	if offset == 0 && page > 0 {
		offset = page * limit
	}
	return nil, LibraryItemsOutput{
		Items:  items,
		Total:  response.Total,
		Limit:  response.Limit,
		Offset: offset,
		Page:   page,
		Count:  len(items),
		Sort:   input.Sort,
		Desc:   input.Desc,
		Filter: input.Filter,
	}, nil
}

// GetLibraryItem returns one ABS library item.
func (s *Server) GetLibraryItem(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input LibraryItemInput,
) (*mcp.CallToolResult, LibraryItemOutput, error) {
	if input.ItemID == "" {
		return nil, LibraryItemOutput{}, fmt.Errorf("itemId is required")
	}
	item, err := s.client.GetLibraryItem(ctx, input.ItemID)
	if err != nil {
		return nil, LibraryItemOutput{}, fmt.Errorf("get ABS item %q: %w", input.ItemID, err)
	}
	return nil, LibraryItemOutput{Item: summarizeItem(*item)}, nil
}

// SearchLibrary searches one ABS library.
func (s *Server) SearchLibrary(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input SearchLibraryInput,
) (*mcp.CallToolResult, SearchLibraryOutput, error) {
	if input.LibraryID == "" {
		return nil, SearchLibraryOutput{}, fmt.Errorf("libraryId is required")
	}
	if input.Query == "" {
		return nil, SearchLibraryOutput{}, fmt.Errorf("query is required")
	}
	limit, err := normalizeSearchLimit(input.Limit)
	if err != nil {
		return nil, SearchLibraryOutput{}, err
	}
	data, err := s.client.SearchLibrary(ctx, input.LibraryID, input.Query, limit)
	if err != nil {
		return nil, SearchLibraryOutput{}, fmt.Errorf("search ABS library %q: %w", input.LibraryID, err)
	}
	return nil, SearchLibraryOutput{
		LibraryID: input.LibraryID,
		Query:     input.Query,
		Limit:     limit,
		Data:      data,
	}, nil
}

// GetLibraryStats returns raw ABS stats for one library.
func (s *Server) GetLibraryStats(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input LibraryRawInput,
) (*mcp.CallToolResult, LibraryRawOutput, error) {
	if input.LibraryID == "" {
		return nil, LibraryRawOutput{}, fmt.Errorf("libraryId is required")
	}
	data, err := s.client.GetLibraryStats(ctx, input.LibraryID)
	if err != nil {
		return nil, LibraryRawOutput{}, fmt.Errorf("get ABS library %q stats: %w", input.LibraryID, err)
	}
	return nil, LibraryRawOutput{LibraryID: input.LibraryID, Data: data}, nil
}

// GetLibraryFilterData returns raw ABS filter data for one library.
func (s *Server) GetLibraryFilterData(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input LibraryRawInput,
) (*mcp.CallToolResult, LibraryRawOutput, error) {
	if input.LibraryID == "" {
		return nil, LibraryRawOutput{}, fmt.Errorf("libraryId is required")
	}
	data, err := s.client.GetLibraryFilterData(ctx, input.LibraryID)
	if err != nil {
		return nil, LibraryRawOutput{}, fmt.Errorf("get ABS library %q filter data: %w", input.LibraryID, err)
	}
	return nil, LibraryRawOutput{LibraryID: input.LibraryID, Data: data}, nil
}

// GetItemMetadataObject returns the raw ABS metadata object for one item.
func (s *Server) GetItemMetadataObject(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input LibraryItemInput,
) (*mcp.CallToolResult, MetadataObjectOutput, error) {
	if input.ItemID == "" {
		return nil, MetadataObjectOutput{}, fmt.Errorf("itemId is required")
	}
	data, err := s.client.GetItemMetadataObject(ctx, input.ItemID)
	if err != nil {
		return nil, MetadataObjectOutput{}, fmt.Errorf("get ABS item %q metadata object: %w", input.ItemID, err)
	}
	return nil, MetadataObjectOutput{ItemID: input.ItemID, Data: data}, nil
}

// FindMisorganizedItems audits item paths against expected library layout conventions.
func (s *Server) FindMisorganizedItems(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input FindMisorganizedItemsInput,
) (*mcp.CallToolResult, FindMisorganizedItemsOutput, error) {
	if input.LibraryID == "" {
		return nil, FindMisorganizedItemsOutput{}, fmt.Errorf("libraryId is required")
	}
	convention, err := normalizeLayoutConvention(input.Convention)
	if err != nil {
		return nil, FindMisorganizedItemsOutput{}, err
	}
	limit, err := normalizeLayoutLimit(input.Limit)
	if err != nil {
		return nil, FindMisorganizedItemsOutput{}, err
	}

	library, err := s.client.GetLibrary(ctx, input.LibraryID)
	if err != nil {
		return nil, FindMisorganizedItemsOutput{}, fmt.Errorf("get ABS library %q: %w", input.LibraryID, err)
	}
	items, err := s.client.GetAllLibraryItems(ctx, input.LibraryID)
	if err != nil {
		return nil, FindMisorganizedItemsOutput{}, fmt.Errorf("list ABS library %q items for layout audit: %w", input.LibraryID, err)
	}

	output := FindMisorganizedItemsOutput{
		LibraryID:            input.LibraryID,
		Convention:           convention,
		CheckedCount:         len(items),
		Limit:                limit,
		SummaryByReason:      map[string]int{},
		SupportedConventions: supportedLayoutConventions(),
		LibraryFolders:       summarizeLibrary(*library).Folders,
	}
	for _, item := range items {
		auditItem := auditItemLayout(item, *library, convention)
		if auditItem.Organized {
			output.OrganizedCount++
		} else if auditItem.Classifiable {
			output.MisorganizedCount++
		} else {
			output.UnclassifiableCount++
		}
		for _, reason := range auditItem.Reasons {
			output.SummaryByReason[reason]++
		}
		if input.IncludeOrganized || !auditItem.Organized {
			if len(output.Items) < limit {
				output.Items = append(output.Items, auditItem)
			} else {
				output.Truncated = true
			}
		}
	}
	output.ReturnedCount = len(output.Items)
	return nil, output, nil
}

// ScanLibrary triggers one ABS library scan when mutating tools are enabled.
func (s *Server) ScanLibrary(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input ScanLibraryInput,
) (*mcp.CallToolResult, ScanLibraryOutput, error) {
	if s.cfg.ReadOnly {
		return nil, ScanLibraryOutput{}, readOnlyToolError("abs_scan_library")
	}
	if input.LibraryID == "" {
		return nil, ScanLibraryOutput{}, fmt.Errorf("libraryId is required")
	}
	if err := s.client.ScanLibrary(ctx, input.LibraryID, input.Force); err != nil {
		return nil, ScanLibraryOutput{}, fmt.Errorf("scan ABS library %q: %w", input.LibraryID, err)
	}
	return nil, ScanLibraryOutput{
		Triggered: true,
		LibraryID: input.LibraryID,
		Force:     input.Force,
	}, nil
}

// RemoveLibraryItemsWithIssues removes missing or invalid library items after explicit confirmation.
func (s *Server) RemoveLibraryItemsWithIssues(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input RemoveLibraryItemsWithIssuesInput,
) (*mcp.CallToolResult, RemoveLibraryItemsWithIssuesOutput, error) {
	if s.cfg.ReadOnly {
		return nil, RemoveLibraryItemsWithIssuesOutput{}, readOnlyToolError("abs_remove_library_items_with_issues")
	}
	if input.LibraryID == "" {
		return nil, RemoveLibraryItemsWithIssuesOutput{}, fmt.Errorf("libraryId is required")
	}
	expectedConfirmation := fmt.Sprintf("remove issues from %s", input.LibraryID)
	if input.Confirmation != expectedConfirmation {
		return nil, RemoveLibraryItemsWithIssuesOutput{}, fmt.Errorf("confirmation must exactly equal %q", expectedConfirmation)
	}
	if input.ExpectedIssueCount < 0 {
		return nil, RemoveLibraryItemsWithIssuesOutput{}, fmt.Errorf("expectedIssueCount must be greater than or equal to 0")
	}

	itemsBefore, err := s.client.GetAllLibraryItems(ctx, input.LibraryID)
	if err != nil {
		return nil, RemoveLibraryItemsWithIssuesOutput{}, fmt.Errorf("list ABS library %q items before issue cleanup: %w", input.LibraryID, err)
	}
	issueIDs := issueItemIDs(itemsBefore)
	issueCount := len(issueIDs)
	if input.ExpectedIssueCount != 0 && input.ExpectedIssueCount != issueCount {
		return nil, RemoveLibraryItemsWithIssuesOutput{}, fmt.Errorf("expectedIssueCount %d does not match observed issue count %d", input.ExpectedIssueCount, issueCount)
	}

	output := RemoveLibraryItemsWithIssuesOutput{
		LibraryID:        input.LibraryID,
		IssueCountBefore: issueCount,
		IssueItemIDs:     capStrings(issueIDs, 100),
	}
	if issueCount == 0 {
		return nil, output, nil
	}

	if err := s.client.RemoveLibraryItemsWithIssues(ctx, input.LibraryID); err != nil {
		return nil, RemoveLibraryItemsWithIssuesOutput{}, fmt.Errorf("remove ABS library %q items with issues: %w", input.LibraryID, err)
	}
	itemsAfter, err := s.client.GetAllLibraryItems(ctx, input.LibraryID)
	if err != nil {
		return nil, RemoveLibraryItemsWithIssuesOutput{}, fmt.Errorf("list ABS library %q items after issue cleanup: %w", input.LibraryID, err)
	}
	output.Triggered = true
	output.RemovedCount = issueCount
	output.RemainingIssueCount = len(issueItemIDs(itemsAfter))
	return nil, output, nil
}

// ScanLibraryAndWait triggers one ABS library scan and observes item count afterward.
func (s *Server) ScanLibraryAndWait(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input ScanLibraryAndWaitInput,
) (*mcp.CallToolResult, ScanLibraryAndWaitOutput, error) {
	if s.cfg.ReadOnly {
		return nil, ScanLibraryAndWaitOutput{}, readOnlyToolError("abs_scan_library_and_wait")
	}
	if input.LibraryID == "" {
		return nil, ScanLibraryAndWaitOutput{}, fmt.Errorf("libraryId is required")
	}
	if input.ExpectedTotal < 0 {
		return nil, ScanLibraryAndWaitOutput{}, fmt.Errorf("expectedTotal must be greater than or equal to 0")
	}
	timeoutSeconds, pollIntervalMillis, err := normalizeScanWait(input.TimeoutSeconds, input.PollIntervalMilliseconds)
	if err != nil {
		return nil, ScanLibraryAndWaitOutput{}, err
	}

	if err := s.client.ScanLibrary(ctx, input.LibraryID, input.Force); err != nil {
		return nil, ScanLibraryAndWaitOutput{}, fmt.Errorf("scan ABS library %q: %w", input.LibraryID, err)
	}

	start := time.Now()
	timeout := time.Duration(timeoutSeconds) * time.Second
	pollInterval := time.Duration(pollIntervalMillis) * time.Millisecond
	deadline := start.Add(timeout)
	output := ScanLibraryAndWaitOutput{
		Triggered:          true,
		LibraryID:          input.LibraryID,
		Force:              input.Force,
		ExpectedTotal:      input.ExpectedTotal,
		TimeoutSeconds:     timeoutSeconds,
		PollIntervalMillis: pollIntervalMillis,
	}

	for {
		response, err := s.client.GetLibraryItems(ctx, input.LibraryID, 1, 0)
		if err != nil {
			return nil, ScanLibraryAndWaitOutput{}, fmt.Errorf("observe ABS library %q item count: %w", input.LibraryID, err)
		}
		output.Attempts++
		output.ObservedTotal = response.Total
		output.ElapsedMilliseconds = time.Since(start).Milliseconds()
		if input.ExpectedTotal == 0 || response.Total >= input.ExpectedTotal {
			output.Completed = true
			return nil, output, nil
		}
		if !time.Now().Before(deadline) {
			output.TimedOut = true
			return nil, output, nil
		}

		wait := pollInterval
		if remaining := time.Until(deadline); remaining < wait {
			wait = remaining
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return nil, ScanLibraryAndWaitOutput{}, ctx.Err()
		case <-timer.C:
		}
	}
}

// ScanItem rescans one directory-backed ABS library item when mutating tools are enabled.
func (s *Server) ScanItem(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input ScanItemInput,
) (*mcp.CallToolResult, ScanItemOutput, error) {
	if s.cfg.ReadOnly {
		return nil, ScanItemOutput{}, readOnlyToolError("abs_scan_item")
	}
	if input.ItemID == "" {
		return nil, ScanItemOutput{}, fmt.Errorf("itemId is required")
	}
	response, err := s.client.ScanItem(ctx, input.ItemID)
	if err != nil {
		return nil, ScanItemOutput{}, fmt.Errorf("scan ABS item %q: %w", input.ItemID, err)
	}
	return nil, ScanItemOutput{
		Triggered: true,
		ItemID:    input.ItemID,
		Result:    response.Result,
	}, nil
}

// UpdateItemMetadata is a planned metadata mutation tool gated by read-only mode.
func (s *Server) UpdateItemMetadata(
	_ context.Context,
	_ *mcp.CallToolRequest,
	input ItemPayloadInput,
) (*mcp.CallToolResult, PlannedMutationOutput, error) {
	if err := s.requireMutatingTool("abs_update_item_metadata"); err != nil {
		return nil, PlannedMutationOutput{}, err
	}
	if input.ItemID == "" {
		return nil, PlannedMutationOutput{}, fmt.Errorf("itemId is required")
	}
	return nil, PlannedMutationOutput{}, plannedToolError("abs_update_item_metadata", "PATCH /api/items/:id/media")
}

// UpdateItemCover updates an item cover from an ABS-visible path.
func (s *Server) UpdateItemCover(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input UpdateItemCoverInput,
) (*mcp.CallToolResult, ItemMutationOutput, error) {
	if err := s.requireMutatingTool("abs_update_item_cover"); err != nil {
		return nil, ItemMutationOutput{}, err
	}
	if input.ItemID == "" {
		return nil, ItemMutationOutput{}, fmt.Errorf("itemId is required")
	}
	if strings.TrimSpace(input.Cover) == "" {
		return nil, ItemMutationOutput{}, fmt.Errorf("cover is required")
	}
	data, err := s.client.UpdateItemCover(ctx, input.ItemID, input.Cover)
	if err != nil {
		return nil, ItemMutationOutput{}, fmt.Errorf("update ABS item %q cover: %w", input.ItemID, err)
	}
	return nil, ItemMutationOutput{Triggered: true, ItemID: input.ItemID, Data: data}, nil
}

// RemoveItemCover removes an item cover after exact confirmation.
func (s *Server) RemoveItemCover(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input ConfirmedItemInput,
) (*mcp.CallToolResult, RemoveItemCoverOutput, error) {
	if err := s.requireMutatingTool("abs_remove_item_cover"); err != nil {
		return nil, RemoveItemCoverOutput{}, err
	}
	if input.ItemID == "" {
		return nil, RemoveItemCoverOutput{}, fmt.Errorf("itemId is required")
	}
	expectedConfirmation := fmt.Sprintf("remove cover from %s", input.ItemID)
	if input.Confirmation != expectedConfirmation {
		return nil, RemoveItemCoverOutput{}, fmt.Errorf("confirmation must exactly equal %q", expectedConfirmation)
	}
	if err := s.client.RemoveItemCover(ctx, input.ItemID); err != nil {
		return nil, RemoveItemCoverOutput{}, fmt.Errorf("remove ABS item %q cover: %w", input.ItemID, err)
	}
	return nil, RemoveItemCoverOutput{Triggered: true, ItemID: input.ItemID}, nil
}

// MatchItem is a planned item matching tool gated by read-only mode.
func (s *Server) MatchItem(
	_ context.Context,
	_ *mcp.CallToolRequest,
	input MatchItemInput,
) (*mcp.CallToolResult, PlannedMutationOutput, error) {
	if err := s.requireMutatingTool("abs_match_item"); err != nil {
		return nil, PlannedMutationOutput{}, err
	}
	if input.ItemID == "" {
		return nil, PlannedMutationOutput{}, fmt.Errorf("itemId is required")
	}
	return nil, PlannedMutationOutput{}, plannedToolError("abs_match_item", "POST /api/items/:id/match")
}

// UpdateItemChapters replaces an item chapter list after an expected-count guard.
func (s *Server) UpdateItemChapters(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input UpdateItemChaptersInput,
) (*mcp.CallToolResult, ItemMutationOutput, error) {
	if err := s.requireMutatingTool("abs_update_item_chapters"); err != nil {
		return nil, ItemMutationOutput{}, err
	}
	if input.ItemID == "" {
		return nil, ItemMutationOutput{}, fmt.Errorf("itemId is required")
	}
	if len(input.Chapters) == 0 {
		return nil, ItemMutationOutput{}, fmt.Errorf("chapters must contain at least one chapter")
	}
	if input.ExpectedChapterCount != len(input.Chapters) {
		return nil, ItemMutationOutput{}, fmt.Errorf("expectedChapterCount %d does not match chapter count %d", input.ExpectedChapterCount, len(input.Chapters))
	}
	chapters := make([]abs.Chapter, 0, len(input.Chapters))
	for index, chapter := range input.Chapters {
		if strings.TrimSpace(chapter.Title) == "" {
			return nil, ItemMutationOutput{}, fmt.Errorf("chapters[%d].title is required", index)
		}
		if chapter.Start < 0 {
			return nil, ItemMutationOutput{}, fmt.Errorf("chapters[%d].start must be greater than or equal to 0", index)
		}
		if chapter.End < chapter.Start {
			return nil, ItemMutationOutput{}, fmt.Errorf("chapters[%d].end must be greater than or equal to start", index)
		}
		chapters = append(chapters, abs.Chapter{
			Title: strings.TrimSpace(chapter.Title),
			Start: chapter.Start,
			End:   chapter.End,
		})
	}
	data, err := s.client.UpdateItemChapters(ctx, input.ItemID, chapters)
	if err != nil {
		return nil, ItemMutationOutput{}, fmt.Errorf("update ABS item %q chapters: %w", input.ItemID, err)
	}
	return nil, ItemMutationOutput{Triggered: true, ItemID: input.ItemID, Data: data}, nil
}

// UpdateItemTracks is a planned track mutation tool gated by read-only mode.
func (s *Server) UpdateItemTracks(
	_ context.Context,
	_ *mcp.CallToolRequest,
	input ItemPayloadInput,
) (*mcp.CallToolResult, PlannedMutationOutput, error) {
	if err := s.requireMutatingTool("abs_update_item_tracks"); err != nil {
		return nil, PlannedMutationOutput{}, err
	}
	if input.ItemID == "" {
		return nil, PlannedMutationOutput{}, fmt.Errorf("itemId is required")
	}
	return nil, PlannedMutationOutput{}, plannedToolError("abs_update_item_tracks", "PATCH /api/items/:id/tracks")
}

// CreateCollection is a planned collection mutation tool gated by read-only mode.
func (s *Server) CreateCollection(
	_ context.Context,
	_ *mcp.CallToolRequest,
	input CollectionInput,
) (*mcp.CallToolResult, PlannedMutationOutput, error) {
	if err := s.requireMutatingTool("abs_create_collection"); err != nil {
		return nil, PlannedMutationOutput{}, err
	}
	if input.Name == "" {
		return nil, PlannedMutationOutput{}, fmt.Errorf("name is required")
	}
	return nil, PlannedMutationOutput{}, plannedToolError("abs_create_collection", "POST /api/collections")
}

// UpdateCollection is a planned collection mutation tool gated by read-only mode.
func (s *Server) UpdateCollection(
	_ context.Context,
	_ *mcp.CallToolRequest,
	input CollectionInput,
) (*mcp.CallToolResult, PlannedMutationOutput, error) {
	if err := s.requireMutatingTool("abs_update_collection"); err != nil {
		return nil, PlannedMutationOutput{}, err
	}
	if input.CollectionID == "" {
		return nil, PlannedMutationOutput{}, fmt.Errorf("collectionId is required")
	}
	return nil, PlannedMutationOutput{}, plannedToolError("abs_update_collection", "PATCH /api/collections/:id")
}

// DeleteCollection is a planned destructive collection mutation tool gated by read-only mode.
func (s *Server) DeleteCollection(
	_ context.Context,
	_ *mcp.CallToolRequest,
	input ConfirmedCollectionInput,
) (*mcp.CallToolResult, PlannedMutationOutput, error) {
	if err := s.requireMutatingTool("abs_delete_collection"); err != nil {
		return nil, PlannedMutationOutput{}, err
	}
	if input.CollectionID == "" {
		return nil, PlannedMutationOutput{}, fmt.Errorf("collectionId is required")
	}
	expectedConfirmation := fmt.Sprintf("delete collection %s", input.CollectionID)
	if input.Confirmation != expectedConfirmation {
		return nil, PlannedMutationOutput{}, fmt.Errorf("confirmation must exactly equal %q", expectedConfirmation)
	}
	return nil, PlannedMutationOutput{}, plannedToolError("abs_delete_collection", "DELETE /api/collections/:id")
}

// AddCollectionItem is a planned collection membership tool gated by read-only mode.
func (s *Server) AddCollectionItem(
	_ context.Context,
	_ *mcp.CallToolRequest,
	input CollectionItemInput,
) (*mcp.CallToolResult, PlannedMutationOutput, error) {
	if err := s.requireMutatingTool("abs_add_collection_item"); err != nil {
		return nil, PlannedMutationOutput{}, err
	}
	if input.CollectionID == "" {
		return nil, PlannedMutationOutput{}, fmt.Errorf("collectionId is required")
	}
	if input.ItemID == "" {
		return nil, PlannedMutationOutput{}, fmt.Errorf("itemId is required")
	}
	return nil, PlannedMutationOutput{}, plannedToolError("abs_add_collection_item", "POST /api/collections/:id/book")
}

// RemoveCollectionItem is a planned destructive collection membership tool gated by read-only mode.
func (s *Server) RemoveCollectionItem(
	_ context.Context,
	_ *mcp.CallToolRequest,
	input CollectionItemInput,
) (*mcp.CallToolResult, PlannedMutationOutput, error) {
	if err := s.requireMutatingTool("abs_remove_collection_item"); err != nil {
		return nil, PlannedMutationOutput{}, err
	}
	if input.CollectionID == "" {
		return nil, PlannedMutationOutput{}, fmt.Errorf("collectionId is required")
	}
	if input.ItemID == "" {
		return nil, PlannedMutationOutput{}, fmt.Errorf("itemId is required")
	}
	expectedConfirmation := fmt.Sprintf("remove item %s from collection %s", input.ItemID, input.CollectionID)
	if input.Confirmation != expectedConfirmation {
		return nil, PlannedMutationOutput{}, fmt.Errorf("confirmation must exactly equal %q", expectedConfirmation)
	}
	return nil, PlannedMutationOutput{}, plannedToolError("abs_remove_collection_item", "DELETE /api/collections/:id/book/:bookId")
}

// CreatePlaylist is a planned playlist mutation tool gated by read-only mode.
func (s *Server) CreatePlaylist(
	_ context.Context,
	_ *mcp.CallToolRequest,
	input PlaylistInput,
) (*mcp.CallToolResult, PlannedMutationOutput, error) {
	if err := s.requireMutatingTool("abs_create_playlist"); err != nil {
		return nil, PlannedMutationOutput{}, err
	}
	if input.Name == "" {
		return nil, PlannedMutationOutput{}, fmt.Errorf("name is required")
	}
	return nil, PlannedMutationOutput{}, plannedToolError("abs_create_playlist", "POST /api/playlists")
}

// UpdatePlaylist is a planned playlist mutation tool gated by read-only mode.
func (s *Server) UpdatePlaylist(
	_ context.Context,
	_ *mcp.CallToolRequest,
	input PlaylistInput,
) (*mcp.CallToolResult, PlannedMutationOutput, error) {
	if err := s.requireMutatingTool("abs_update_playlist"); err != nil {
		return nil, PlannedMutationOutput{}, err
	}
	if input.PlaylistID == "" {
		return nil, PlannedMutationOutput{}, fmt.Errorf("playlistId is required")
	}
	return nil, PlannedMutationOutput{}, plannedToolError("abs_update_playlist", "PATCH /api/playlists/:id")
}

// DeletePlaylist is a planned destructive playlist mutation tool gated by read-only mode.
func (s *Server) DeletePlaylist(
	_ context.Context,
	_ *mcp.CallToolRequest,
	input ConfirmedPlaylistInput,
) (*mcp.CallToolResult, PlannedMutationOutput, error) {
	if err := s.requireMutatingTool("abs_delete_playlist"); err != nil {
		return nil, PlannedMutationOutput{}, err
	}
	if input.PlaylistID == "" {
		return nil, PlannedMutationOutput{}, fmt.Errorf("playlistId is required")
	}
	expectedConfirmation := fmt.Sprintf("delete playlist %s", input.PlaylistID)
	if input.Confirmation != expectedConfirmation {
		return nil, PlannedMutationOutput{}, fmt.Errorf("confirmation must exactly equal %q", expectedConfirmation)
	}
	return nil, PlannedMutationOutput{}, plannedToolError("abs_delete_playlist", "DELETE /api/playlists/:id")
}

// AddPlaylistItem is a planned playlist membership tool gated by read-only mode.
func (s *Server) AddPlaylistItem(
	_ context.Context,
	_ *mcp.CallToolRequest,
	input PlaylistItemInput,
) (*mcp.CallToolResult, PlannedMutationOutput, error) {
	if err := s.requireMutatingTool("abs_add_playlist_item"); err != nil {
		return nil, PlannedMutationOutput{}, err
	}
	if input.PlaylistID == "" {
		return nil, PlannedMutationOutput{}, fmt.Errorf("playlistId is required")
	}
	if input.ItemID == "" {
		return nil, PlannedMutationOutput{}, fmt.Errorf("itemId is required")
	}
	return nil, PlannedMutationOutput{}, plannedToolError("abs_add_playlist_item", "POST /api/playlists/:id/item")
}

// RemovePlaylistItem is a planned destructive playlist membership tool gated by read-only mode.
func (s *Server) RemovePlaylistItem(
	_ context.Context,
	_ *mcp.CallToolRequest,
	input PlaylistItemInput,
) (*mcp.CallToolResult, PlannedMutationOutput, error) {
	if err := s.requireMutatingTool("abs_remove_playlist_item"); err != nil {
		return nil, PlannedMutationOutput{}, err
	}
	if input.PlaylistID == "" {
		return nil, PlannedMutationOutput{}, fmt.Errorf("playlistId is required")
	}
	if input.ItemID == "" {
		return nil, PlannedMutationOutput{}, fmt.Errorf("itemId is required")
	}
	expectedConfirmation := fmt.Sprintf("remove item %s from playlist %s", input.ItemID, input.PlaylistID)
	if input.Confirmation != expectedConfirmation {
		return nil, PlannedMutationOutput{}, fmt.Errorf("confirmation must exactly equal %q", expectedConfirmation)
	}
	return nil, PlannedMutationOutput{}, plannedToolError("abs_remove_playlist_item", "DELETE /api/playlists/:id/item/:libraryItemId/:episodeId?")
}

func (s *Server) requireMutatingTool(toolName string) error {
	if s.cfg.ReadOnly {
		return readOnlyToolError(toolName)
	}
	return nil
}

func readOnlyToolError(toolName string) error {
	return fmt.Errorf("%s is not usable while ABS_READ_ONLY=true. To use this mutating tool, restart abs-mcp with ABS_READ_ONLY=false or --read-only=false after confirming the Audiobookshelf operation is safe", toolName)
}

func plannedToolError(toolName string, route string) error {
	return fmt.Errorf("%s is registered but not implemented yet; planned ABS route %s requires source and fixture verification before this MCP server will mutate Audiobookshelf", toolName, route)
}

func normalizeLimit(limit int) (int, error) {
	if limit == 0 {
		return 25, nil
	}
	if limit < 0 {
		return 0, fmt.Errorf("limit must be greater than 0")
	}
	if limit > 100 {
		return 100, nil
	}
	return limit, nil
}

func normalizeSearchLimit(limit int) (int, error) {
	if limit == 0 {
		return 12, nil
	}
	if limit < 0 {
		return 0, fmt.Errorf("limit must be greater than 0")
	}
	if limit > 50 {
		return 50, nil
	}
	return limit, nil
}

func normalizeLayoutLimit(limit int) (int, error) {
	if limit == 0 {
		return 50, nil
	}
	if limit < 0 {
		return 0, fmt.Errorf("limit must be greater than 0")
	}
	if limit > 200 {
		return 200, nil
	}
	return limit, nil
}

func supportedLayoutConventions() []string {
	return []string{"auto", "author-title", "author-series-title"}
}

func normalizeLayoutConvention(convention string) (string, error) {
	convention = strings.TrimSpace(strings.ToLower(convention))
	if convention == "" {
		return "auto", nil
	}
	for _, supported := range supportedLayoutConventions() {
		if convention == supported {
			return convention, nil
		}
	}
	return "", fmt.Errorf("convention must be one of: %s", strings.Join(supportedLayoutConventions(), ", "))
}

func pageFromOffset(offset int, limit int) (int, error) {
	if offset == 0 {
		return 0, nil
	}
	if limit <= 0 {
		return 0, fmt.Errorf("limit must be greater than 0 when offset is set")
	}
	if offset%limit != 0 {
		return 0, fmt.Errorf("offset must be a multiple of limit because ABS uses page-based pagination")
	}
	return offset / limit, nil
}

func auditItemLayout(item abs.LibraryItem, library abs.Library, convention string) LayoutAuditItem {
	author := itemAuthor(item)
	title := item.Media.Metadata.Title
	series := itemSeries(item)
	currentRelPath := itemRelPath(item, library)
	itemConvention := convention
	if itemConvention == "auto" {
		itemConvention = "author-title"
		if series != "" {
			itemConvention = "author-series-title"
		}
	}

	result := LayoutAuditItem{
		ItemID:         item.ID,
		Title:          title,
		Author:         author,
		Series:         series,
		CurrentRelPath: currentRelPath,
		Convention:     itemConvention,
		Confidence:     "high",
		IsMissing:      item.IsMissing,
		IsInvalid:      item.IsInvalid,
	}

	var expectedParts []string
	if author == "" {
		result.Reasons = append(result.Reasons, "metadata_missing_author")
	}
	if title == "" {
		result.Reasons = append(result.Reasons, "metadata_missing_title")
	}
	if currentRelPath == "" {
		result.Reasons = append(result.Reasons, "path_missing")
	}
	if itemConvention == "author-series-title" && series == "" {
		result.Reasons = append(result.Reasons, "metadata_missing_series")
	}

	if author != "" {
		expectedParts = append(expectedParts, cleanLayoutSegment(author))
	}
	if itemConvention == "author-series-title" && series != "" {
		expectedParts = append(expectedParts, cleanLayoutSegment(series))
	}
	if title != "" {
		expectedParts = append(expectedParts, cleanLayoutSegment(title))
	}
	result.ExpectedRelPath = strings.Join(expectedParts, "/")

	if len(result.Reasons) > 0 {
		result.Classifiable = false
		result.Confidence = "low"
		return result
	}

	currentParts := splitLayoutPath(currentRelPath)
	expectedParts = splitLayoutPath(result.ExpectedRelPath)
	result.Classifiable = true
	result.Reasons = layoutMismatchReasons(currentParts, expectedParts, itemConvention)
	result.Organized = len(result.Reasons) == 0
	if !result.Organized && len(currentParts) >= len(expectedParts) {
		result.Confidence = "medium"
	}
	return result
}

func itemAuthor(item abs.LibraryItem) string {
	return firstNonEmpty(item.Media.Metadata.AuthorName, item.AuthorNamesFirstLast)
}

func itemSeries(item abs.LibraryItem) string {
	if item.Media.Metadata.SeriesName != "" {
		return item.Media.Metadata.SeriesName
	}
	if len(item.Media.Metadata.Series) > 0 {
		return item.Media.Metadata.Series[0].Name
	}
	return ""
}

func itemRelPath(item abs.LibraryItem, library abs.Library) string {
	if item.RelPath != "" {
		return trimLayoutPath(item.RelPath)
	}
	itemPath := trimLayoutPath(item.Path)
	for _, folder := range library.Folders {
		for _, root := range []string{folder.FullPath, folder.Path} {
			root = trimLayoutPath(root)
			if root == "" {
				continue
			}
			if itemPath == root {
				return ""
			}
			if strings.HasPrefix(itemPath, root+"/") {
				return strings.TrimPrefix(itemPath, root+"/")
			}
		}
	}
	return itemPath
}

func layoutMismatchReasons(currentParts []string, expectedParts []string, convention string) []string {
	reasons := make([]string, 0)
	if len(currentParts) < len(expectedParts) {
		reasons = append(reasons, "path_too_shallow")
	}
	if len(currentParts) > len(expectedParts) {
		reasons = append(reasons, "path_has_extra_directories")
	}
	if !sameLayoutPart(currentParts, expectedParts, 0) {
		reasons = append(reasons, "author_directory_mismatch")
	}
	titleIndex := len(expectedParts) - 1
	if !sameLayoutPart(currentParts, expectedParts, titleIndex) {
		reasons = append(reasons, "title_directory_mismatch")
	}
	if convention == "author-series-title" && !sameLayoutPart(currentParts, expectedParts, 1) {
		reasons = append(reasons, "series_directory_mismatch")
	}
	return reasons
}

func sameLayoutPart(currentParts []string, expectedParts []string, index int) bool {
	if index < 0 || index >= len(currentParts) || index >= len(expectedParts) {
		return false
	}
	return normalizeLayoutPart(currentParts[index]) == normalizeLayoutPart(expectedParts[index])
}

func splitLayoutPath(value string) []string {
	value = trimLayoutPath(value)
	if value == "" {
		return nil
	}
	return strings.Split(value, "/")
}

func trimLayoutPath(value string) string {
	value = strings.ReplaceAll(value, "\\", "/")
	value = path.Clean("/" + value)
	return strings.Trim(value, "/")
}

func cleanLayoutSegment(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "/", "-"))
	value = strings.ReplaceAll(value, "\\", "-")
	value = strings.Join(strings.Fields(value), " ")
	if value == "" {
		return "Unknown"
	}
	return value
}

func normalizeLayoutPart(value string) string {
	value = strings.ToLower(cleanLayoutSegment(value))
	replacer := strings.NewReplacer(":", "", ";", "", ",", "", ".", "", "'", "", "\"", "", "!", "", "?", "", "&", "and")
	value = replacer.Replace(value)
	return strings.Join(strings.Fields(value), " ")
}

func sanitizeInclude(values []string) []string {
	includes := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(strings.ToLower(value))
		if value != "" {
			includes = append(includes, value)
		}
	}
	return includes
}

func normalizeScanWait(timeoutSeconds int, pollIntervalMillis int) (int, int, error) {
	if timeoutSeconds < 0 {
		return 0, 0, fmt.Errorf("timeoutSeconds must be greater than or equal to 0")
	}
	if pollIntervalMillis < 0 {
		return 0, 0, fmt.Errorf("pollIntervalMilliseconds must be greater than or equal to 0")
	}
	if timeoutSeconds == 0 {
		timeoutSeconds = 30
	}
	if timeoutSeconds > 300 {
		timeoutSeconds = 300
	}
	if pollIntervalMillis == 0 {
		pollIntervalMillis = 1000
	}
	if pollIntervalMillis > 60000 {
		pollIntervalMillis = 60000
	}
	return timeoutSeconds, pollIntervalMillis, nil
}

func issueItemIDs(items []abs.LibraryItem) []string {
	ids := make([]string, 0)
	for _, item := range items {
		if item.IsMissing || item.IsInvalid {
			ids = append(ids, item.ID)
		}
	}
	return ids
}

func capStrings(values []string, limit int) []string {
	if len(values) <= limit {
		return values
	}
	return values[:limit]
}

func summarizeLibraries(libraries []abs.Library) []LibrarySummary {
	summaries := make([]LibrarySummary, 0, len(libraries))
	for _, library := range libraries {
		summaries = append(summaries, summarizeLibrary(library))
	}
	return summaries
}

func summarizeLibrary(library abs.Library) LibrarySummary {
	folders := make([]FolderSummary, 0, len(library.Folders))
	for _, folder := range library.Folders {
		folders = append(folders, FolderSummary{
			ID:       folder.ID,
			Path:     folder.Path,
			FullPath: folder.FullPath,
		})
	}
	return LibrarySummary{
		ID:        library.ID,
		Name:      library.Name,
		MediaType: library.MediaType,
		Folders:   folders,
	}
}

func summarizeItems(items []abs.LibraryItem) []LibraryItemSummary {
	summaries := make([]LibraryItemSummary, 0, len(items))
	for _, item := range items {
		summaries = append(summaries, summarizeItem(item))
	}
	return summaries
}

func summarizeItem(item abs.LibraryItem) LibraryItemSummary {
	return LibraryItemSummary{
		ID:        item.ID,
		LibraryID: item.LibraryID,
		MediaType: item.MediaType,
		Path:      item.Path,
		RelPath:   item.RelPath,
		IsMissing: item.IsMissing,
		IsInvalid: item.IsInvalid,
		Title:     item.Media.Metadata.Title,
		Author:    firstNonEmpty(item.Media.Metadata.AuthorName, item.AuthorNamesFirstLast),
		Series:    item.Media.Metadata.SeriesName,
		Files:     summarizeMediaFiles(item),
		Duration:  item.Media.Duration,
		Size:      item.Media.Size,
	}
}

func summarizeMediaFiles(item abs.LibraryItem) []MediaFileSummary {
	files := make([]MediaFileSummary, 0, len(item.LibraryFiles)+len(item.Media.AudioFiles)+1)
	for _, file := range item.LibraryFiles {
		files = append(files, summarizeLibraryFile(file))
	}
	if len(files) > 0 {
		return files
	}
	for _, file := range item.Media.AudioFiles {
		files = append(files, summarizeLibraryFile(file.LibraryFile))
	}
	if item.Media.EbookFile != nil {
		files = append(files, summarizeLibraryFile(item.Media.EbookFile.LibraryFile))
	}
	return files
}

func summarizeLibraryFile(file abs.LibraryFile) MediaFileSummary {
	return MediaFileSummary{
		FileType: file.FileType,
		Filename: file.Metadata.Filename,
		Path:     file.Metadata.Path,
		RelPath:  file.Metadata.RelPath,
		Size:     file.Metadata.Size,
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
