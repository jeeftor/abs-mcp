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
	ListLibraryAuthors(context.Context, string, abs.CatalogListOptions) (abs.JSONValue, error)
	GetAuthor(context.Context, string, []string) (abs.JSONValue, error)
	ListLibrarySeries(context.Context, string, abs.CatalogListOptions) (abs.JSONValue, error)
	GetSeries(context.Context, string, []string) (abs.JSONValue, error)
	ListCollections(context.Context) (abs.JSONValue, error)
	GetCollection(context.Context, string, []string) (abs.JSONValue, error)
	GetItemsInProgress(context.Context, int) (abs.JSONValue, error)
	GetItemProgress(context.Context, string, string) (*abs.MediaProgress, error)
	ListBookmarks(context.Context) ([]abs.Bookmark, error)
	UpdateItemProgress(context.Context, string, string, abs.ProgressUpdatePayload) error
	CreateBookmark(context.Context, string, abs.BookmarkPayload) (*abs.Bookmark, error)
	UpdateBookmark(context.Context, string, abs.BookmarkPayload) (*abs.Bookmark, error)
	ListBackups(context.Context) ([]abs.Backup, error)
	CreateBackup(context.Context) (*abs.Backup, error)
	SendEbookToDevice(context.Context, abs.EbookDevicePayload) (abs.JSONValue, error)
	GetEmailSettings(context.Context) (*abs.EmailSettings, error)
	GetItemMetadataObject(context.Context, string) (abs.JSONValue, error)
	UpdateItemMetadata(context.Context, string, abs.ItemMetadataPayload) (abs.JSONValue, error)
	ScanLibrary(context.Context, string, bool) error
	RemoveLibraryItemsWithIssues(context.Context, string) error
	ScanItem(context.Context, string) (*abs.ScanItemResponse, error)
	UpdateItemCover(context.Context, string, string) (abs.JSONValue, error)
	RemoveItemCover(context.Context, string) error
	UpdateItemChapters(context.Context, string, []abs.Chapter) (abs.JSONValue, error)
	CreateCollection(context.Context, abs.CollectionPayload) (abs.JSONValue, error)
	UpdateCollection(context.Context, string, abs.CollectionPayload) (abs.JSONValue, error)
	AddCollectionItem(context.Context, string, string) (abs.JSONValue, error)
	CreatePlaylist(context.Context, abs.PlaylistPayload) (abs.JSONValue, error)
	UpdatePlaylist(context.Context, string, abs.PlaylistPayload) (abs.JSONValue, error)
	AddPlaylistItem(context.Context, string, abs.PlaylistItemPayload) (abs.JSONValue, error)
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
		Name:        "abs_search_ebooks",
		Title:       "Search Audiobookshelf ebooks",
		Description: "Search one Audiobookshelf library for items that have ebook files.",
	}, s.SearchEbooks)
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
		Name:        "abs_list_library_authors",
		Title:       "List Audiobookshelf library authors",
		Description: "List a bounded source-backed page of authors from one Audiobookshelf library.",
	}, s.ListLibraryAuthors)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_get_author",
		Title:       "Get Audiobookshelf author",
		Description: "Get one source-backed Audiobookshelf author object.",
	}, s.GetAuthor)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_list_library_series",
		Title:       "List Audiobookshelf library series",
		Description: "List a bounded source-backed page of series from one Audiobookshelf library.",
	}, s.ListLibrarySeries)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_get_series",
		Title:       "Get Audiobookshelf series",
		Description: "Get one source-backed Audiobookshelf series object.",
	}, s.GetSeries)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_list_collections",
		Title:       "List Audiobookshelf collections",
		Description: "List collections visible to the authenticated Audiobookshelf user.",
	}, s.ListCollections)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_get_collection",
		Title:       "Get Audiobookshelf collection",
		Description: "Get one source-backed Audiobookshelf collection object.",
	}, s.GetCollection)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_get_items_in_progress",
		Title:       "Get Audiobookshelf current-user items in progress",
		Description: "Get source-backed items in progress for the configured Audiobookshelf user.",
	}, s.GetItemsInProgress)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_get_item_progress",
		Title:       "Get Audiobookshelf current-user item progress",
		Description: "Get current-user progress for one Audiobookshelf library item or podcast episode.",
	}, s.GetItemProgress)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_list_bookmarks",
		Title:       "List Audiobookshelf current-user bookmarks",
		Description: "List bookmarks for the configured Audiobookshelf user, optionally filtered by library item ID.",
	}, s.ListBookmarks)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_update_item_progress",
		Title:       "Update Audiobookshelf current-user item progress",
		Description: "Create or update current-user progress for one Audiobookshelf library item or podcast episode. Blocked when ABS_READ_ONLY is true.",
	}, s.UpdateItemProgress)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_create_bookmark",
		Title:       "Create Audiobookshelf current-user bookmark",
		Description: "Create one bookmark for the configured Audiobookshelf user. Blocked when ABS_READ_ONLY is true.",
	}, s.CreateBookmark)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_update_bookmark",
		Title:       "Update Audiobookshelf current-user bookmark",
		Description: "Update one bookmark for the configured Audiobookshelf user. Blocked when ABS_READ_ONLY is true.",
	}, s.UpdateBookmark)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_list_backups",
		Title:       "List Audiobookshelf backups",
		Description: "List source-backed Audiobookshelf server backup records visible to the authenticated user.",
	}, s.ListBackups)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_create_backup",
		Title:       "Create Audiobookshelf backup",
		Description: "Create one Audiobookshelf server backup. Blocked when ABS_READ_ONLY is true.",
	}, s.CreateBackup)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_list_ereader_devices",
		Title:       "List Audiobookshelf ereader devices",
		Description: "List saved Audiobookshelf ereader device names from email settings without returning SMTP settings. Requires sufficient ABS permissions.",
	}, s.ListEreaderDevices)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_send_ebook_to_device",
		Title:       "Send Audiobookshelf ebook to device",
		Description: "Send one Audiobookshelf ebook item to a saved ereader device by device name. Blocked when ABS_READ_ONLY is true.",
	}, s.SendEbookToDevice)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_send_ebook_by_query",
		Title:       "Send Audiobookshelf ebook by query",
		Description: "Resolve exactly one ebook search match, require an exact confirmation string, then send it to a saved ereader device. Blocked when ABS_READ_ONLY is true.",
	}, s.SendEbookByQuery)
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
		Description: "Update selected source-verified metadata fields on one item. Blocked when ABS_READ_ONLY is true.",
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
		Description: "Create a collection with an initial source-verified item list. Blocked when ABS_READ_ONLY is true.",
	}, s.CreateCollection)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_update_collection",
		Title:       "Update Audiobookshelf collection",
		Description: "Update one collection name or description. Blocked when ABS_READ_ONLY is true.",
	}, s.UpdateCollection)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_delete_collection",
		Title:       "Delete Audiobookshelf collection",
		Description: "Planned destructive tool for deleting a collection. Requires exact confirmation, is blocked when ABS_READ_ONLY is true, and is not implemented until source and fixture behavior are verified.",
	}, s.DeleteCollection)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_add_collection_item",
		Title:       "Add Audiobookshelf collection item",
		Description: "Add one library item to a collection. Blocked when ABS_READ_ONLY is true.",
	}, s.AddCollectionItem)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_remove_collection_item",
		Title:       "Remove Audiobookshelf collection item",
		Description: "Planned destructive tool for removing an item from a collection. Requires exact confirmation, is blocked when ABS_READ_ONLY is true, and is not implemented until source and fixture behavior are verified.",
	}, s.RemoveCollectionItem)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_create_playlist",
		Title:       "Create Audiobookshelf playlist",
		Description: "Create a playlist with optional source-verified items. Blocked when ABS_READ_ONLY is true.",
	}, s.CreatePlaylist)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_update_playlist",
		Title:       "Update Audiobookshelf playlist",
		Description: "Update one playlist name or description. Blocked when ABS_READ_ONLY is true.",
	}, s.UpdatePlaylist)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_delete_playlist",
		Title:       "Delete Audiobookshelf playlist",
		Description: "Planned destructive tool for deleting a playlist. Requires exact confirmation, is blocked when ABS_READ_ONLY is true, and is not implemented until source and fixture behavior are verified.",
	}, s.DeletePlaylist)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "abs_add_playlist_item",
		Title:       "Add Audiobookshelf playlist item",
		Description: "Add one library item or podcast episode to a playlist. Blocked when ABS_READ_ONLY is true.",
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

// SearchEbooksInput selects one bounded ebook search.
type SearchEbooksInput struct {
	LibraryID string `json:"libraryId" jsonschema:"Audiobookshelf library ID."`
	Query     string `json:"query,omitempty" jsonschema:"Optional query matched against title, author, series, and ebook filename. Blank returns ebook items up to the limit."`
	Limit     int    `json:"limit,omitempty" jsonschema:"Maximum number of ebook results. Defaults to 25 and is capped at 100."`
}

// SearchEbooksOutput is returned by abs_search_ebooks.
type SearchEbooksOutput struct {
	LibraryID     string               `json:"libraryId" jsonschema:"Audiobookshelf library ID searched."`
	Query         string               `json:"query,omitempty" jsonschema:"Normalized query text used for matching."`
	Limit         int                  `json:"limit" jsonschema:"Maximum returned ebook results after normalization."`
	Items         []LibraryItemSummary `json:"items" jsonschema:"Matching ebook items."`
	CheckedCount  int                  `json:"checkedCount" jsonschema:"Number of library items checked."`
	EbookCount    int                  `json:"ebookCount" jsonschema:"Number of ebook items found before query filtering."`
	MatchedCount  int                  `json:"matchedCount" jsonschema:"Number of ebook items that matched the query."`
	ReturnedCount int                  `json:"returnedCount" jsonschema:"Number of ebook items returned."`
	Truncated     bool                 `json:"truncated" jsonschema:"Whether additional matches were omitted by the limit."`
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

// CatalogListInput selects a bounded page from source-backed catalog endpoints.
type CatalogListInput struct {
	LibraryID string   `json:"libraryId" jsonschema:"Audiobookshelf library ID."`
	Limit     int      `json:"limit,omitempty" jsonschema:"Maximum number of catalog records to return. Defaults to 25 and is capped at 100."`
	Offset    int      `json:"offset,omitempty" jsonschema:"Zero-based catalog offset. Must be a multiple of limit because ABS uses page-based pagination."`
	Sort      string   `json:"sort,omitempty" jsonschema:"ABS catalog sort key."`
	Desc      bool     `json:"desc,omitempty" jsonschema:"Whether to sort descending."`
	Filter    string   `json:"filter,omitempty" jsonschema:"ABS catalog filter expression."`
	Include   []string `json:"include,omitempty" jsonschema:"Optional ABS include values to request."`
	Minified  bool     `json:"minified,omitempty" jsonschema:"Whether to request minified ABS catalog records."`
}

// CatalogListOutput is returned by source-backed catalog list tools.
type CatalogListOutput struct {
	LibraryID string        `json:"libraryId"`
	Limit     int           `json:"limit"`
	Offset    int           `json:"offset"`
	Page      int           `json:"page"`
	Sort      string        `json:"sort,omitempty"`
	Desc      bool          `json:"desc,omitempty"`
	Filter    string        `json:"filter,omitempty"`
	Include   []string      `json:"include,omitempty"`
	Minified  bool          `json:"minified,omitempty"`
	Data      abs.JSONValue `json:"data" jsonschema:"Raw Audiobookshelf catalog response."`
}

// EntityInput identifies one source-backed entity with optional include values.
type EntityInput struct {
	ID      string   `json:"id" jsonschema:"Audiobookshelf entity ID."`
	Include []string `json:"include,omitempty" jsonschema:"Optional ABS include values to request."`
}

// EntityOutput is returned by source-backed entity read tools.
type EntityOutput struct {
	ID      string        `json:"id"`
	Include []string      `json:"include,omitempty"`
	Data    abs.JSONValue `json:"data" jsonschema:"Raw Audiobookshelf entity response."`
}

// RawOutput is returned by read-only tools that do not require input.
type RawOutput struct {
	Data abs.JSONValue `json:"data" jsonschema:"Raw Audiobookshelf response."`
}

// ItemsInProgressInput selects current-user items in progress.
type ItemsInProgressInput struct {
	Limit int `json:"limit,omitempty" jsonschema:"Maximum number of items in progress to return. Defaults to 25 and is capped at 100."`
}

// UserProgressScope describes which Audiobookshelf user's progress is represented.
type UserProgressScope struct {
	Scope  string `json:"scope" jsonschema:"Progress scope. Current-user tools always return currentUser."`
	UserID string `json:"userId,omitempty" jsonschema:"Audiobookshelf user ID returned by ABS when present."`
}

// ItemsInProgressOutput is returned by abs_get_items_in_progress.
type ItemsInProgressOutput struct {
	Limit int               `json:"limit" jsonschema:"Normalized maximum number of current-user items requested."`
	User  UserProgressScope `json:"user" jsonschema:"Audiobookshelf user scope for the returned progress data."`
	Data  abs.JSONValue     `json:"data" jsonschema:"Raw Audiobookshelf current-user items-in-progress response."`
}

// ItemProgressInput identifies current-user progress for one item or episode.
type ItemProgressInput struct {
	ItemID    string `json:"itemId" jsonschema:"Audiobookshelf library item ID."`
	EpisodeID string `json:"episodeId,omitempty" jsonschema:"Optional podcast episode ID."`
}

// ItemProgressOutput is returned by abs_get_item_progress.
type ItemProgressOutput struct {
	ItemID    string            `json:"itemId" jsonschema:"Audiobookshelf library item ID requested."`
	EpisodeID string            `json:"episodeId,omitempty" jsonschema:"Podcast episode ID requested, when present."`
	User      UserProgressScope `json:"user" jsonschema:"Audiobookshelf user scope for this progress object."`
	Progress  abs.MediaProgress `json:"progress" jsonschema:"Current-user Audiobookshelf progress object."`
}

// BookmarksInput filters current-user bookmarks.
type BookmarksInput struct {
	ItemID string `json:"itemId,omitempty" jsonschema:"Optional Audiobookshelf library item ID to filter bookmarks."`
}

// BookmarksOutput is returned by abs_list_bookmarks.
type BookmarksOutput struct {
	ItemID    string         `json:"itemId,omitempty" jsonschema:"Library item ID used to filter bookmarks, when present."`
	Bookmarks []abs.Bookmark `json:"bookmarks" jsonschema:"Current-user Audiobookshelf bookmarks."`
	Count     int            `json:"count" jsonschema:"Number of bookmarks returned."`
}

// UpdateItemProgressInput identifies a current-user progress update.
type UpdateItemProgressInput struct {
	ItemID                    string   `json:"itemId" jsonschema:"Audiobookshelf library item ID."`
	EpisodeID                 string   `json:"episodeId,omitempty" jsonschema:"Optional podcast episode ID."`
	Duration                  *float64 `json:"duration,omitempty" jsonschema:"Media duration in seconds."`
	Progress                  *float64 `json:"progress,omitempty" jsonschema:"Progress ratio from 0 to 1."`
	CurrentTime               *float64 `json:"currentTime,omitempty" jsonschema:"Current playback time in seconds."`
	IsFinished                *bool    `json:"isFinished,omitempty" jsonschema:"Whether the item is finished."`
	HideFromContinueListening *bool    `json:"hideFromContinueListening,omitempty" jsonschema:"Whether to hide this progress from continue listening."`
	EbookLocation             *string  `json:"ebookLocation,omitempty" jsonschema:"Ebook location marker."`
	EbookProgress             *float64 `json:"ebookProgress,omitempty" jsonschema:"Ebook progress ratio from 0 to 1."`
}

// ProgressMutationOutput is returned by abs_update_item_progress.
type ProgressMutationOutput struct {
	Triggered bool   `json:"triggered" jsonschema:"Whether an Audiobookshelf progress request was sent."`
	ItemID    string `json:"itemId" jsonschema:"Audiobookshelf library item ID requested."`
	EpisodeID string `json:"episodeId,omitempty" jsonschema:"Podcast episode ID requested, when present."`
}

// BookmarkMutationInput identifies one current-user bookmark create/update.
type BookmarkMutationInput struct {
	ItemID string  `json:"itemId" jsonschema:"Audiobookshelf library item ID."`
	Time   float64 `json:"time" jsonschema:"Bookmark time in seconds."`
	Title  string  `json:"title" jsonschema:"Bookmark title."`
}

// BookmarkMutationOutput is returned by bookmark mutation tools.
type BookmarkMutationOutput struct {
	Triggered bool         `json:"triggered" jsonschema:"Whether an Audiobookshelf bookmark request was sent."`
	ItemID    string       `json:"itemId" jsonschema:"Audiobookshelf library item ID requested."`
	Bookmark  abs.Bookmark `json:"bookmark" jsonschema:"Audiobookshelf bookmark returned by ABS."`
}

// BackupsOutput is returned by abs_list_backups.
type BackupsOutput struct {
	Backups []abs.Backup `json:"backups" jsonschema:"Audiobookshelf backup records."`
	Count   int          `json:"count" jsonschema:"Number of backups returned."`
}

// BackupMutationOutput is returned by backup mutation tools.
type BackupMutationOutput struct {
	Triggered bool       `json:"triggered" jsonschema:"Whether an Audiobookshelf backup request was sent."`
	Backup    abs.Backup `json:"backup" jsonschema:"Audiobookshelf backup returned by ABS."`
}

// EReaderDeviceSummary is a sanitized saved ereader device.
type EReaderDeviceSummary struct {
	Name               string   `json:"name" jsonschema:"Saved Audiobookshelf ereader device name."`
	Email              string   `json:"email,omitempty" jsonschema:"Redacted; this server does not return saved device email addresses."`
	AvailabilityOption string   `json:"availabilityOption,omitempty" jsonschema:"Audiobookshelf device availability option."`
	Users              []string `json:"users,omitempty" jsonschema:"Audiobookshelf user IDs allowed for this device when ABS exposes them."`
}

// EReaderDevicesOutput is returned by abs_list_ereader_devices.
type EReaderDevicesOutput struct {
	Devices []EReaderDeviceSummary `json:"devices" jsonschema:"Sanitized saved ereader devices."`
	Count   int                    `json:"count" jsonschema:"Number of devices returned."`
}

// SendEbookToDeviceInput identifies one ebook send request.
type SendEbookToDeviceInput struct {
	ItemID     string `json:"itemId" jsonschema:"Audiobookshelf library item ID containing an ebook file."`
	DeviceName string `json:"deviceName" jsonschema:"Exact saved Audiobookshelf ereader device name."`
}

// SendEbookToDeviceOutput is returned by abs_send_ebook_to_device.
type SendEbookToDeviceOutput struct {
	Triggered  bool          `json:"triggered" jsonschema:"Whether an Audiobookshelf email request was sent."`
	ItemID     string        `json:"itemId" jsonschema:"Audiobookshelf library item ID requested."`
	DeviceName string        `json:"deviceName" jsonschema:"Audiobookshelf ereader device name requested."`
	Data       abs.JSONValue `json:"data,omitempty" jsonschema:"Raw Audiobookshelf response, when returned by ABS."`
}

// SendEbookByQueryInput identifies one guarded query-based ebook send request.
type SendEbookByQueryInput struct {
	LibraryID     string `json:"libraryId" jsonschema:"Audiobookshelf library ID to search."`
	Query         string `json:"query" jsonschema:"Query matched against ebook title, author, series, and filename. Must resolve exactly one ebook."`
	DeviceName    string `json:"deviceName" jsonschema:"Exact saved Audiobookshelf ereader device name."`
	Confirmation  string `json:"confirmation" jsonschema:"Exact confirmation text. Must be: send ebook <resolvedItemId> to <deviceName>."`
	MaxCandidates int    `json:"maxCandidates,omitempty" jsonschema:"Maximum matching ebook candidates to inspect before treating the request as ambiguous. Defaults to 10 and is capped at 50."`
}

// SendEbookByQueryOutput is returned by abs_send_ebook_by_query.
type SendEbookByQueryOutput struct {
	Triggered  bool               `json:"triggered" jsonschema:"Whether an Audiobookshelf email request was sent."`
	LibraryID  string             `json:"libraryId" jsonschema:"Audiobookshelf library ID searched."`
	Query      string             `json:"query" jsonschema:"Query used to resolve the ebook item."`
	DeviceName string             `json:"deviceName" jsonschema:"Audiobookshelf ereader device name requested."`
	Item       LibraryItemSummary `json:"item" jsonschema:"Resolved ebook item sent to the device."`
	Data       abs.JSONValue      `json:"data,omitempty" jsonschema:"Raw Audiobookshelf response, when returned by ABS."`
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

// UpdateItemMetadataInput identifies one typed item metadata update request.
type UpdateItemMetadataInput struct {
	ItemID        string                `json:"itemId" jsonschema:"Audiobookshelf library item ID to mutate."`
	Title         *string               `json:"title,omitempty" jsonschema:"Book or podcast title."`
	Subtitle      *string               `json:"subtitle,omitempty" jsonschema:"Book subtitle."`
	Author        *string               `json:"author,omitempty" jsonschema:"Podcast author field."`
	Description   *string               `json:"description,omitempty" jsonschema:"Book or podcast description. Empty string clears the field."`
	PublishedYear *string               `json:"publishedYear,omitempty" jsonschema:"Book published year."`
	PublishedDate *string               `json:"publishedDate,omitempty" jsonschema:"Book published date."`
	Publisher     *string               `json:"publisher,omitempty" jsonschema:"Book publisher."`
	ISBN          *string               `json:"isbn,omitempty" jsonschema:"Book ISBN."`
	ASIN          *string               `json:"asin,omitempty" jsonschema:"Book ASIN."`
	Language      *string               `json:"language,omitempty" jsonschema:"Book or podcast language."`
	ReleaseDate   *string               `json:"releaseDate,omitempty" jsonschema:"Podcast release date."`
	Explicit      *bool                 `json:"explicit,omitempty" jsonschema:"Explicit content flag."`
	Abridged      *bool                 `json:"abridged,omitempty" jsonschema:"Book abridged flag."`
	Narrators     []string              `json:"narrators,omitempty" jsonschema:"Book narrator names. Empty array clears narrators."`
	Genres        []string              `json:"genres,omitempty" jsonschema:"Genre names. Empty array clears genres."`
	Tags          []string              `json:"tags,omitempty" jsonschema:"Item tags. Empty array clears tags."`
	Authors       []string              `json:"authors,omitempty" jsonschema:"Book author names. Empty array clears author links."`
	Series        []SeriesMetadataInput `json:"series,omitempty" jsonschema:"Book series entries. Empty array clears series links."`
}

// SeriesMetadataInput identifies one book series metadata update entry.
type SeriesMetadataInput struct {
	Name     string `json:"name" jsonschema:"Series name."`
	Sequence string `json:"sequence,omitempty" jsonschema:"Optional book sequence in the series."`
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

// CollectionInput identifies one collection create/update request.
type CollectionInput struct {
	CollectionID string   `json:"collectionId,omitempty" jsonschema:"Audiobookshelf collection ID for updates."`
	LibraryID    string   `json:"libraryId,omitempty" jsonschema:"Audiobookshelf library ID for collection creation."`
	Name         string   `json:"name,omitempty" jsonschema:"Collection name."`
	Description  string   `json:"description,omitempty" jsonschema:"Collection description."`
	ItemIDs      []string `json:"itemIds,omitempty" jsonschema:"Audiobookshelf library item IDs for collection creation."`
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

// PlaylistInput identifies one playlist create/update request.
type PlaylistInput struct {
	PlaylistID  string                    `json:"playlistId,omitempty" jsonschema:"Audiobookshelf playlist ID for updates."`
	LibraryID   string                    `json:"libraryId,omitempty" jsonschema:"Audiobookshelf library ID for playlist creation."`
	Name        string                    `json:"name,omitempty" jsonschema:"Playlist name."`
	Description string                    `json:"description,omitempty" jsonschema:"Playlist description."`
	Items       []PlaylistCreateItemInput `json:"items,omitempty" jsonschema:"Optional initial playlist items."`
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

// PlaylistCreateItemInput identifies one optional initial playlist item.
type PlaylistCreateItemInput struct {
	ItemID    string `json:"itemId" jsonschema:"Audiobookshelf library item ID to add."`
	EpisodeID string `json:"episodeId,omitempty" jsonschema:"Optional podcast episode ID when adding a podcast playlist item."`
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

// CatalogMutationOutput is returned by collection and playlist mutation tools.
type CatalogMutationOutput struct {
	Triggered bool          `json:"triggered" jsonschema:"Whether an Audiobookshelf mutation request was sent."`
	ID        string        `json:"id,omitempty" jsonschema:"Audiobookshelf collection or playlist ID returned by ABS, when available."`
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

// SearchEbooks searches one library for items with ebook files.
func (s *Server) SearchEbooks(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input SearchEbooksInput,
) (*mcp.CallToolResult, SearchEbooksOutput, error) {
	output, _, err := s.searchEbooks(ctx, input.LibraryID, input.Query, input.Limit)
	if err != nil {
		return nil, SearchEbooksOutput{}, err
	}
	return nil, output, nil
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

// ListLibraryAuthors returns a bounded page of source-backed ABS authors.
func (s *Server) ListLibraryAuthors(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input CatalogListInput,
) (*mcp.CallToolResult, CatalogListOutput, error) {
	options, output, err := catalogListRequest(input)
	if err != nil {
		return nil, CatalogListOutput{}, err
	}
	data, err := s.client.ListLibraryAuthors(ctx, input.LibraryID, options)
	if err != nil {
		return nil, CatalogListOutput{}, fmt.Errorf("list ABS library %q authors: %w", input.LibraryID, err)
	}
	output.Data = data
	return nil, output, nil
}

// GetAuthor returns one source-backed ABS author.
func (s *Server) GetAuthor(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input EntityInput,
) (*mcp.CallToolResult, EntityOutput, error) {
	include, output, err := entityRequest(input)
	if err != nil {
		return nil, EntityOutput{}, err
	}
	data, err := s.client.GetAuthor(ctx, input.ID, include)
	if err != nil {
		return nil, EntityOutput{}, fmt.Errorf("get ABS author %q: %w", input.ID, err)
	}
	output.Data = data
	return nil, output, nil
}

// ListLibrarySeries returns a bounded page of source-backed ABS series.
func (s *Server) ListLibrarySeries(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input CatalogListInput,
) (*mcp.CallToolResult, CatalogListOutput, error) {
	options, output, err := catalogListRequest(input)
	if err != nil {
		return nil, CatalogListOutput{}, err
	}
	data, err := s.client.ListLibrarySeries(ctx, input.LibraryID, options)
	if err != nil {
		return nil, CatalogListOutput{}, fmt.Errorf("list ABS library %q series: %w", input.LibraryID, err)
	}
	output.Data = data
	return nil, output, nil
}

// GetSeries returns one source-backed ABS series.
func (s *Server) GetSeries(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input EntityInput,
) (*mcp.CallToolResult, EntityOutput, error) {
	include, output, err := entityRequest(input)
	if err != nil {
		return nil, EntityOutput{}, err
	}
	data, err := s.client.GetSeries(ctx, input.ID, include)
	if err != nil {
		return nil, EntityOutput{}, fmt.Errorf("get ABS series %q: %w", input.ID, err)
	}
	output.Data = data
	return nil, output, nil
}

// ListCollections returns collections visible to the authenticated ABS user.
func (s *Server) ListCollections(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	_ EmptyInput,
) (*mcp.CallToolResult, RawOutput, error) {
	data, err := s.client.ListCollections(ctx)
	if err != nil {
		return nil, RawOutput{}, fmt.Errorf("list ABS collections: %w", err)
	}
	return nil, RawOutput{Data: data}, nil
}

// GetCollection returns one source-backed ABS collection.
func (s *Server) GetCollection(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input EntityInput,
) (*mcp.CallToolResult, EntityOutput, error) {
	include, output, err := entityRequest(input)
	if err != nil {
		return nil, EntityOutput{}, err
	}
	data, err := s.client.GetCollection(ctx, input.ID, include)
	if err != nil {
		return nil, EntityOutput{}, fmt.Errorf("get ABS collection %q: %w", input.ID, err)
	}
	output.Data = data
	return nil, output, nil
}

// GetItemsInProgress returns current-user items in progress.
func (s *Server) GetItemsInProgress(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input ItemsInProgressInput,
) (*mcp.CallToolResult, ItemsInProgressOutput, error) {
	limit, err := normalizeLimit(input.Limit)
	if err != nil {
		return nil, ItemsInProgressOutput{}, err
	}
	data, err := s.client.GetItemsInProgress(ctx, limit)
	if err != nil {
		return nil, ItemsInProgressOutput{}, fmt.Errorf("get ABS current-user items in progress: %w", err)
	}
	return nil, ItemsInProgressOutput{Limit: limit, User: currentUserProgressScope(""), Data: data}, nil
}

// GetItemProgress returns current-user progress for one item or episode.
func (s *Server) GetItemProgress(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input ItemProgressInput,
) (*mcp.CallToolResult, ItemProgressOutput, error) {
	if input.ItemID == "" {
		return nil, ItemProgressOutput{}, fmt.Errorf("itemId is required")
	}
	progress, err := s.client.GetItemProgress(ctx, input.ItemID, input.EpisodeID)
	if err != nil {
		return nil, ItemProgressOutput{}, fmt.Errorf("get ABS current-user progress for item %q: %w", input.ItemID, err)
	}
	return nil, ItemProgressOutput{
		ItemID:    input.ItemID,
		EpisodeID: input.EpisodeID,
		User:      currentUserProgressScope(progress.UserID),
		Progress:  *progress,
	}, nil
}

// ListBookmarks returns current-user bookmarks, optionally filtered by item ID.
func (s *Server) ListBookmarks(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input BookmarksInput,
) (*mcp.CallToolResult, BookmarksOutput, error) {
	bookmarks, err := s.client.ListBookmarks(ctx)
	if err != nil {
		return nil, BookmarksOutput{}, fmt.Errorf("list ABS current-user bookmarks: %w", err)
	}
	if input.ItemID != "" {
		filtered := make([]abs.Bookmark, 0, len(bookmarks))
		for _, bookmark := range bookmarks {
			if bookmark.LibraryItemID == input.ItemID {
				filtered = append(filtered, bookmark)
			}
		}
		bookmarks = filtered
	}
	return nil, BookmarksOutput{ItemID: input.ItemID, Bookmarks: bookmarks, Count: len(bookmarks)}, nil
}

// UpdateItemProgress creates or updates current-user progress for one item or episode.
func (s *Server) UpdateItemProgress(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input UpdateItemProgressInput,
) (*mcp.CallToolResult, ProgressMutationOutput, error) {
	if err := s.requireMutatingTool("abs_update_item_progress"); err != nil {
		return nil, ProgressMutationOutput{}, err
	}
	itemID := strings.TrimSpace(input.ItemID)
	episodeID := strings.TrimSpace(input.EpisodeID)
	if itemID == "" {
		return nil, ProgressMutationOutput{}, fmt.Errorf("itemId is required")
	}
	payload, err := buildProgressUpdatePayload(input)
	if err != nil {
		return nil, ProgressMutationOutput{}, err
	}
	if err := s.client.UpdateItemProgress(ctx, itemID, episodeID, payload); err != nil {
		return nil, ProgressMutationOutput{}, fmt.Errorf("update ABS current-user progress for item %q: %w", itemID, err)
	}
	return nil, ProgressMutationOutput{Triggered: true, ItemID: itemID, EpisodeID: episodeID}, nil
}

// CreateBookmark creates one current-user bookmark.
func (s *Server) CreateBookmark(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input BookmarkMutationInput,
) (*mcp.CallToolResult, BookmarkMutationOutput, error) {
	if err := s.requireMutatingTool("abs_create_bookmark"); err != nil {
		return nil, BookmarkMutationOutput{}, err
	}
	itemID, payload, err := buildBookmarkPayload(input)
	if err != nil {
		return nil, BookmarkMutationOutput{}, err
	}
	bookmark, err := s.client.CreateBookmark(ctx, itemID, payload)
	if err != nil {
		return nil, BookmarkMutationOutput{}, fmt.Errorf("create ABS current-user bookmark for item %q: %w", itemID, err)
	}
	return nil, BookmarkMutationOutput{Triggered: true, ItemID: itemID, Bookmark: *bookmark}, nil
}

// UpdateBookmark updates one current-user bookmark.
func (s *Server) UpdateBookmark(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input BookmarkMutationInput,
) (*mcp.CallToolResult, BookmarkMutationOutput, error) {
	if err := s.requireMutatingTool("abs_update_bookmark"); err != nil {
		return nil, BookmarkMutationOutput{}, err
	}
	itemID, payload, err := buildBookmarkPayload(input)
	if err != nil {
		return nil, BookmarkMutationOutput{}, err
	}
	bookmark, err := s.client.UpdateBookmark(ctx, itemID, payload)
	if err != nil {
		return nil, BookmarkMutationOutput{}, fmt.Errorf("update ABS current-user bookmark for item %q: %w", itemID, err)
	}
	return nil, BookmarkMutationOutput{Triggered: true, ItemID: itemID, Bookmark: *bookmark}, nil
}

// ListBackups returns Audiobookshelf server backups.
func (s *Server) ListBackups(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	_ EmptyInput,
) (*mcp.CallToolResult, BackupsOutput, error) {
	backups, err := s.client.ListBackups(ctx)
	if err != nil {
		return nil, BackupsOutput{}, fmt.Errorf("list ABS backups: %w", err)
	}
	return nil, BackupsOutput{Backups: backups, Count: len(backups)}, nil
}

// CreateBackup creates one Audiobookshelf server backup.
func (s *Server) CreateBackup(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	_ EmptyInput,
) (*mcp.CallToolResult, BackupMutationOutput, error) {
	if err := s.requireMutatingTool("abs_create_backup"); err != nil {
		return nil, BackupMutationOutput{}, err
	}
	backup, err := s.client.CreateBackup(ctx)
	if err != nil {
		return nil, BackupMutationOutput{}, fmt.Errorf("create ABS backup: %w", err)
	}
	return nil, BackupMutationOutput{Triggered: true, Backup: *backup}, nil
}

// ListEreaderDevices returns saved ereader device names without SMTP settings.
func (s *Server) ListEreaderDevices(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	_ EmptyInput,
) (*mcp.CallToolResult, EReaderDevicesOutput, error) {
	settings, err := s.client.GetEmailSettings(ctx)
	if err != nil {
		return nil, EReaderDevicesOutput{}, fmt.Errorf("get ABS email settings for ereader devices: %w", err)
	}
	devices := make([]EReaderDeviceSummary, 0, len(settings.EReaderDevices))
	for _, device := range settings.EReaderDevices {
		name := strings.TrimSpace(device.Name)
		if name == "" {
			continue
		}
		devices = append(devices, EReaderDeviceSummary{
			Name:               name,
			AvailabilityOption: strings.TrimSpace(device.AvailabilityOption),
			Users:              append([]string(nil), device.Users...),
		})
	}
	return nil, EReaderDevicesOutput{Devices: devices, Count: len(devices)}, nil
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

// SendEbookToDevice sends one ebook item to a saved ereader device.
func (s *Server) SendEbookToDevice(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input SendEbookToDeviceInput,
) (*mcp.CallToolResult, SendEbookToDeviceOutput, error) {
	if err := s.requireMutatingTool("abs_send_ebook_to_device"); err != nil {
		return nil, SendEbookToDeviceOutput{}, err
	}
	itemID := strings.TrimSpace(input.ItemID)
	deviceName := strings.TrimSpace(input.DeviceName)
	if itemID == "" {
		return nil, SendEbookToDeviceOutput{}, fmt.Errorf("itemId is required")
	}
	if deviceName == "" {
		return nil, SendEbookToDeviceOutput{}, fmt.Errorf("deviceName is required")
	}
	data, err := s.client.SendEbookToDevice(ctx, abs.EbookDevicePayload{
		LibraryItemID: itemID,
		DeviceName:    deviceName,
	})
	if err != nil {
		return nil, SendEbookToDeviceOutput{}, fmt.Errorf("send ABS ebook item %q to device %q: %w", itemID, deviceName, err)
	}
	return nil, SendEbookToDeviceOutput{
		Triggered:  true,
		ItemID:     itemID,
		DeviceName: deviceName,
		Data:       data,
	}, nil
}

// SendEbookByQuery resolves one ebook by query and sends it after exact confirmation.
func (s *Server) SendEbookByQuery(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input SendEbookByQueryInput,
) (*mcp.CallToolResult, SendEbookByQueryOutput, error) {
	if err := s.requireMutatingTool("abs_send_ebook_by_query"); err != nil {
		return nil, SendEbookByQueryOutput{}, err
	}
	libraryID := strings.TrimSpace(input.LibraryID)
	query := strings.TrimSpace(input.Query)
	deviceName := strings.TrimSpace(input.DeviceName)
	if libraryID == "" {
		return nil, SendEbookByQueryOutput{}, fmt.Errorf("libraryId is required")
	}
	if query == "" {
		return nil, SendEbookByQueryOutput{}, fmt.Errorf("query is required")
	}
	if deviceName == "" {
		return nil, SendEbookByQueryOutput{}, fmt.Errorf("deviceName is required")
	}
	limit, err := normalizeQuerySendCandidateLimit(input.MaxCandidates)
	if err != nil {
		return nil, SendEbookByQueryOutput{}, err
	}

	searchOutput, matches, err := s.searchEbooks(ctx, libraryID, query, limit)
	if err != nil {
		return nil, SendEbookByQueryOutput{}, err
	}
	if searchOutput.MatchedCount == 0 {
		return nil, SendEbookByQueryOutput{}, fmt.Errorf("query %q matched no ebooks in library %q", query, libraryID)
	}
	if searchOutput.MatchedCount != 1 {
		return nil, SendEbookByQueryOutput{}, fmt.Errorf("query %q matched %d ebooks in library %q; call abs_search_ebooks and send by exact itemId instead", query, searchOutput.MatchedCount, libraryID)
	}
	item := matches[0]
	expectedConfirmation := fmt.Sprintf("send ebook %s to %s", item.ID, deviceName)
	if input.Confirmation != expectedConfirmation {
		return nil, SendEbookByQueryOutput{}, fmt.Errorf("confirmation must exactly equal %q", expectedConfirmation)
	}

	data, err := s.client.SendEbookToDevice(ctx, abs.EbookDevicePayload{
		LibraryItemID: item.ID,
		DeviceName:    deviceName,
	})
	if err != nil {
		return nil, SendEbookByQueryOutput{}, fmt.Errorf("send ABS ebook item %q to device %q: %w", item.ID, deviceName, err)
	}
	return nil, SendEbookByQueryOutput{
		Triggered:  true,
		LibraryID:  libraryID,
		Query:      query,
		DeviceName: deviceName,
		Item:       summarizeItem(item),
		Data:       data,
	}, nil
}

// UpdateItemMetadata updates selected source-verified metadata fields.
func (s *Server) UpdateItemMetadata(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input UpdateItemMetadataInput,
) (*mcp.CallToolResult, ItemMutationOutput, error) {
	if err := s.requireMutatingTool("abs_update_item_metadata"); err != nil {
		return nil, ItemMutationOutput{}, err
	}
	itemID := strings.TrimSpace(input.ItemID)
	if itemID == "" {
		return nil, ItemMutationOutput{}, fmt.Errorf("itemId is required")
	}
	payload, err := buildItemMetadataPayload(input)
	if err != nil {
		return nil, ItemMutationOutput{}, err
	}
	data, err := s.client.UpdateItemMetadata(ctx, itemID, payload)
	if err != nil {
		return nil, ItemMutationOutput{}, fmt.Errorf("update ABS item %q metadata: %w", itemID, err)
	}
	return nil, ItemMutationOutput{Triggered: true, ItemID: itemID, Data: data}, nil
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

// CreateCollection creates one collection with an initial item list.
func (s *Server) CreateCollection(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input CollectionInput,
) (*mcp.CallToolResult, CatalogMutationOutput, error) {
	if err := s.requireMutatingTool("abs_create_collection"); err != nil {
		return nil, CatalogMutationOutput{}, err
	}
	libraryID := strings.TrimSpace(input.LibraryID)
	name := strings.TrimSpace(input.Name)
	description := strings.TrimSpace(input.Description)
	if libraryID == "" {
		return nil, CatalogMutationOutput{}, fmt.Errorf("libraryId is required")
	}
	if name == "" {
		return nil, CatalogMutationOutput{}, fmt.Errorf("name is required")
	}
	itemIDs, err := normalizeRequiredIDs(input.ItemIDs, "itemIds")
	if err != nil {
		return nil, CatalogMutationOutput{}, err
	}
	payload := abs.CollectionPayload{
		LibraryID:   libraryID,
		Name:        name,
		Description: description,
		Books:       itemIDs,
	}
	data, err := s.client.CreateCollection(ctx, payload)
	if err != nil {
		return nil, CatalogMutationOutput{}, fmt.Errorf("create ABS collection: %w", err)
	}
	return nil, CatalogMutationOutput{Triggered: true, ID: jsonValueID(data), Data: data}, nil
}

// UpdateCollection updates one collection's non-membership fields.
func (s *Server) UpdateCollection(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input CollectionInput,
) (*mcp.CallToolResult, CatalogMutationOutput, error) {
	if err := s.requireMutatingTool("abs_update_collection"); err != nil {
		return nil, CatalogMutationOutput{}, err
	}
	collectionID := strings.TrimSpace(input.CollectionID)
	name := strings.TrimSpace(input.Name)
	description := strings.TrimSpace(input.Description)
	if collectionID == "" {
		return nil, CatalogMutationOutput{}, fmt.Errorf("collectionId is required")
	}
	if name == "" && description == "" {
		return nil, CatalogMutationOutput{}, fmt.Errorf("name or description is required")
	}
	payload := abs.CollectionPayload{Name: name, Description: description}
	data, err := s.client.UpdateCollection(ctx, collectionID, payload)
	if err != nil {
		return nil, CatalogMutationOutput{}, fmt.Errorf("update ABS collection %q: %w", collectionID, err)
	}
	id := jsonValueID(data)
	if id == "" {
		id = collectionID
	}
	return nil, CatalogMutationOutput{Triggered: true, ID: id, Data: data}, nil
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

// AddCollectionItem adds one item to a collection.
func (s *Server) AddCollectionItem(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input CollectionItemInput,
) (*mcp.CallToolResult, CatalogMutationOutput, error) {
	if err := s.requireMutatingTool("abs_add_collection_item"); err != nil {
		return nil, CatalogMutationOutput{}, err
	}
	collectionID := strings.TrimSpace(input.CollectionID)
	itemID := strings.TrimSpace(input.ItemID)
	if collectionID == "" {
		return nil, CatalogMutationOutput{}, fmt.Errorf("collectionId is required")
	}
	if itemID == "" {
		return nil, CatalogMutationOutput{}, fmt.Errorf("itemId is required")
	}
	data, err := s.client.AddCollectionItem(ctx, collectionID, itemID)
	if err != nil {
		return nil, CatalogMutationOutput{}, fmt.Errorf("add ABS collection %q item %q: %w", collectionID, itemID, err)
	}
	id := jsonValueID(data)
	if id == "" {
		id = collectionID
	}
	return nil, CatalogMutationOutput{Triggered: true, ID: id, Data: data}, nil
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

// CreatePlaylist creates one playlist.
func (s *Server) CreatePlaylist(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input PlaylistInput,
) (*mcp.CallToolResult, CatalogMutationOutput, error) {
	if err := s.requireMutatingTool("abs_create_playlist"); err != nil {
		return nil, CatalogMutationOutput{}, err
	}
	libraryID := strings.TrimSpace(input.LibraryID)
	name := strings.TrimSpace(input.Name)
	description := strings.TrimSpace(input.Description)
	if libraryID == "" {
		return nil, CatalogMutationOutput{}, fmt.Errorf("libraryId is required")
	}
	if name == "" {
		return nil, CatalogMutationOutput{}, fmt.Errorf("name is required")
	}
	items, err := normalizePlaylistItems(input.Items)
	if err != nil {
		return nil, CatalogMutationOutput{}, err
	}
	payload := abs.PlaylistPayload{
		LibraryID:   libraryID,
		Name:        name,
		Description: description,
		Items:       items,
	}
	data, err := s.client.CreatePlaylist(ctx, payload)
	if err != nil {
		return nil, CatalogMutationOutput{}, fmt.Errorf("create ABS playlist: %w", err)
	}
	return nil, CatalogMutationOutput{Triggered: true, ID: jsonValueID(data), Data: data}, nil
}

// UpdatePlaylist updates one playlist's non-membership fields.
func (s *Server) UpdatePlaylist(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input PlaylistInput,
) (*mcp.CallToolResult, CatalogMutationOutput, error) {
	if err := s.requireMutatingTool("abs_update_playlist"); err != nil {
		return nil, CatalogMutationOutput{}, err
	}
	playlistID := strings.TrimSpace(input.PlaylistID)
	name := strings.TrimSpace(input.Name)
	description := strings.TrimSpace(input.Description)
	if playlistID == "" {
		return nil, CatalogMutationOutput{}, fmt.Errorf("playlistId is required")
	}
	if name == "" && description == "" {
		return nil, CatalogMutationOutput{}, fmt.Errorf("name or description is required")
	}
	payload := abs.PlaylistPayload{Name: name, Description: description}
	data, err := s.client.UpdatePlaylist(ctx, playlistID, payload)
	if err != nil {
		return nil, CatalogMutationOutput{}, fmt.Errorf("update ABS playlist %q: %w", playlistID, err)
	}
	id := jsonValueID(data)
	if id == "" {
		id = playlistID
	}
	return nil, CatalogMutationOutput{Triggered: true, ID: id, Data: data}, nil
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

// AddPlaylistItem adds one item or episode to a playlist.
func (s *Server) AddPlaylistItem(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input PlaylistItemInput,
) (*mcp.CallToolResult, CatalogMutationOutput, error) {
	if err := s.requireMutatingTool("abs_add_playlist_item"); err != nil {
		return nil, CatalogMutationOutput{}, err
	}
	playlistID := strings.TrimSpace(input.PlaylistID)
	itemID := strings.TrimSpace(input.ItemID)
	if playlistID == "" {
		return nil, CatalogMutationOutput{}, fmt.Errorf("playlistId is required")
	}
	if itemID == "" {
		return nil, CatalogMutationOutput{}, fmt.Errorf("itemId is required")
	}
	payload := abs.PlaylistItemPayload{
		LibraryItemID: itemID,
		EpisodeID:     strings.TrimSpace(input.EpisodeID),
	}
	data, err := s.client.AddPlaylistItem(ctx, playlistID, payload)
	if err != nil {
		return nil, CatalogMutationOutput{}, fmt.Errorf("add ABS playlist %q item %q: %w", playlistID, itemID, err)
	}
	id := jsonValueID(data)
	if id == "" {
		id = playlistID
	}
	return nil, CatalogMutationOutput{Triggered: true, ID: id, Data: data}, nil
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

func buildProgressUpdatePayload(input UpdateItemProgressInput) (abs.ProgressUpdatePayload, error) {
	payload := abs.ProgressUpdatePayload{
		Duration:                  input.Duration,
		Progress:                  input.Progress,
		CurrentTime:               input.CurrentTime,
		IsFinished:                input.IsFinished,
		HideFromContinueListening: input.HideFromContinueListening,
		EbookProgress:             input.EbookProgress,
	}
	if input.EbookLocation != nil {
		location := strings.TrimSpace(*input.EbookLocation)
		payload.EbookLocation = &location
	}
	if payload.Duration == nil &&
		payload.Progress == nil &&
		payload.CurrentTime == nil &&
		payload.IsFinished == nil &&
		payload.HideFromContinueListening == nil &&
		payload.EbookLocation == nil &&
		payload.EbookProgress == nil {
		return abs.ProgressUpdatePayload{}, fmt.Errorf("at least one progress field is required")
	}
	if err := validateNonNegativeFloat(payload.Duration, "duration"); err != nil {
		return abs.ProgressUpdatePayload{}, err
	}
	if err := validateNonNegativeFloat(payload.CurrentTime, "currentTime"); err != nil {
		return abs.ProgressUpdatePayload{}, err
	}
	if err := validateRatio(payload.Progress, "progress"); err != nil {
		return abs.ProgressUpdatePayload{}, err
	}
	if err := validateRatio(payload.EbookProgress, "ebookProgress"); err != nil {
		return abs.ProgressUpdatePayload{}, err
	}
	return payload, nil
}

func buildBookmarkPayload(input BookmarkMutationInput) (string, abs.BookmarkPayload, error) {
	itemID := strings.TrimSpace(input.ItemID)
	title := strings.TrimSpace(input.Title)
	if itemID == "" {
		return "", abs.BookmarkPayload{}, fmt.Errorf("itemId is required")
	}
	if input.Time < 0 {
		return "", abs.BookmarkPayload{}, fmt.Errorf("time must be greater than or equal to 0")
	}
	if title == "" {
		return "", abs.BookmarkPayload{}, fmt.Errorf("title is required")
	}
	return itemID, abs.BookmarkPayload{Time: input.Time, Title: title}, nil
}

func validateNonNegativeFloat(value *float64, fieldName string) error {
	if value != nil && *value < 0 {
		return fmt.Errorf("%s must be greater than or equal to 0", fieldName)
	}
	return nil
}

func validateRatio(value *float64, fieldName string) error {
	if value != nil && (*value < 0 || *value > 1) {
		return fmt.Errorf("%s must be between 0 and 1", fieldName)
	}
	return nil
}

func buildItemMetadataPayload(input UpdateItemMetadataInput) (abs.ItemMetadataPayload, error) {
	var payload abs.ItemMetadataPayload
	metadata := &abs.ItemMetadataFields{}
	hasMetadata := false

	setString := func(target **string, value *string, fieldName string, requireNonBlank bool) error {
		if value == nil {
			return nil
		}
		normalized := strings.TrimSpace(*value)
		if requireNonBlank && normalized == "" {
			return fmt.Errorf("%s must not be blank", fieldName)
		}
		*target = &normalized
		hasMetadata = true
		return nil
	}

	if err := setString(&metadata.Title, input.Title, "title", true); err != nil {
		return abs.ItemMetadataPayload{}, err
	}
	if err := setString(&metadata.Subtitle, input.Subtitle, "subtitle", false); err != nil {
		return abs.ItemMetadataPayload{}, err
	}
	if err := setString(&metadata.Author, input.Author, "author", true); err != nil {
		return abs.ItemMetadataPayload{}, err
	}
	if err := setString(&metadata.Description, input.Description, "description", false); err != nil {
		return abs.ItemMetadataPayload{}, err
	}
	if err := setString(&metadata.PublishedYear, input.PublishedYear, "publishedYear", false); err != nil {
		return abs.ItemMetadataPayload{}, err
	}
	if err := setString(&metadata.PublishedDate, input.PublishedDate, "publishedDate", false); err != nil {
		return abs.ItemMetadataPayload{}, err
	}
	if err := setString(&metadata.Publisher, input.Publisher, "publisher", false); err != nil {
		return abs.ItemMetadataPayload{}, err
	}
	if err := setString(&metadata.ISBN, input.ISBN, "isbn", false); err != nil {
		return abs.ItemMetadataPayload{}, err
	}
	if err := setString(&metadata.ASIN, input.ASIN, "asin", false); err != nil {
		return abs.ItemMetadataPayload{}, err
	}
	if err := setString(&metadata.Language, input.Language, "language", false); err != nil {
		return abs.ItemMetadataPayload{}, err
	}
	if err := setString(&metadata.ReleaseDate, input.ReleaseDate, "releaseDate", false); err != nil {
		return abs.ItemMetadataPayload{}, err
	}
	if input.Explicit != nil {
		metadata.Explicit = input.Explicit
		hasMetadata = true
	}
	if input.Abridged != nil {
		metadata.Abridged = input.Abridged
		hasMetadata = true
	}
	if input.Narrators != nil {
		narrators, err := normalizeMetadataStringSlice(input.Narrators, "narrators")
		if err != nil {
			return abs.ItemMetadataPayload{}, err
		}
		metadata.Narrators = &narrators
		hasMetadata = true
	}
	if input.Genres != nil {
		genres, err := normalizeMetadataStringSlice(input.Genres, "genres")
		if err != nil {
			return abs.ItemMetadataPayload{}, err
		}
		metadata.Genres = &genres
		hasMetadata = true
	}
	if input.Authors != nil {
		authors, err := normalizeMetadataAuthors(input.Authors)
		if err != nil {
			return abs.ItemMetadataPayload{}, err
		}
		metadata.Authors = &authors
		hasMetadata = true
	}
	if input.Series != nil {
		series, err := normalizeMetadataSeries(input.Series)
		if err != nil {
			return abs.ItemMetadataPayload{}, err
		}
		metadata.Series = &series
		hasMetadata = true
	}
	if input.Tags != nil {
		tags, err := normalizeMetadataStringSlice(input.Tags, "tags")
		if err != nil {
			return abs.ItemMetadataPayload{}, err
		}
		payload.Tags = &tags
	}
	if hasMetadata {
		payload.Metadata = metadata
	}
	if !hasMetadata && payload.Tags == nil {
		return abs.ItemMetadataPayload{}, fmt.Errorf("at least one metadata field is required")
	}
	return payload, nil
}

func normalizeMetadataStringSlice(values []string, fieldName string) ([]string, error) {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, fmt.Errorf("%s must not contain blank values", fieldName)
		}
		normalized = append(normalized, value)
	}
	return normalized, nil
}

func normalizeMetadataAuthors(values []string) ([]abs.ItemMetadataAuthorPayload, error) {
	normalized := make([]abs.ItemMetadataAuthorPayload, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, fmt.Errorf("authors must not contain blank names")
		}
		normalized = append(normalized, abs.ItemMetadataAuthorPayload{Name: value})
	}
	return normalized, nil
}

func normalizeMetadataSeries(values []SeriesMetadataInput) ([]abs.ItemMetadataSeriesPayload, error) {
	normalized := make([]abs.ItemMetadataSeriesPayload, 0, len(values))
	for _, value := range values {
		name := strings.TrimSpace(value.Name)
		if name == "" {
			return nil, fmt.Errorf("series.name is required")
		}
		normalized = append(normalized, abs.ItemMetadataSeriesPayload{
			Name:     name,
			Sequence: strings.TrimSpace(value.Sequence),
		})
	}
	return normalized, nil
}

func normalizeRequiredIDs(values []string, fieldName string) ([]string, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("%s is required", fieldName)
	}
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, fmt.Errorf("%s must not contain blank IDs", fieldName)
		}
		normalized = append(normalized, value)
	}
	return normalized, nil
}

func normalizePlaylistItems(inputs []PlaylistCreateItemInput) ([]abs.PlaylistItemPayload, error) {
	items := make([]abs.PlaylistItemPayload, 0, len(inputs))
	for _, input := range inputs {
		itemID := strings.TrimSpace(input.ItemID)
		if itemID == "" {
			return nil, fmt.Errorf("items.itemId is required")
		}
		items = append(items, abs.PlaylistItemPayload{
			LibraryItemID: itemID,
			EpisodeID:     strings.TrimSpace(input.EpisodeID),
		})
	}
	return items, nil
}

func jsonValueID(value abs.JSONValue) string {
	switch typed := value.(type) {
	case map[string]any:
		return stringValue(typed["id"])
	case map[string]string:
		return typed["id"]
	}

	body, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	var response struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return ""
	}
	return response.ID
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return ""
	}
}

func currentUserProgressScope(userID string) UserProgressScope {
	return UserProgressScope{Scope: "currentUser", UserID: userID}
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

func normalizeQuerySendCandidateLimit(limit int) (int, error) {
	if limit == 0 {
		return 10, nil
	}
	if limit < 0 {
		return 0, fmt.Errorf("maxCandidates must be greater than 0")
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

func (s *Server) searchEbooks(ctx context.Context, libraryID string, query string, limitInput int) (SearchEbooksOutput, []abs.LibraryItem, error) {
	libraryID = strings.TrimSpace(libraryID)
	query = strings.TrimSpace(query)
	if libraryID == "" {
		return SearchEbooksOutput{}, nil, fmt.Errorf("libraryId is required")
	}
	limit, err := normalizeLimit(limitInput)
	if err != nil {
		return SearchEbooksOutput{}, nil, err
	}
	items, err := s.client.GetAllLibraryItems(ctx, libraryID)
	if err != nil {
		return SearchEbooksOutput{}, nil, fmt.Errorf("list ABS library %q items for ebook search: %w", libraryID, err)
	}

	output := SearchEbooksOutput{
		LibraryID:    libraryID,
		Query:        query,
		Limit:        limit,
		CheckedCount: len(items),
	}
	matches := make([]abs.LibraryItem, 0)
	for _, item := range items {
		if !itemHasEbook(item) {
			continue
		}
		output.EbookCount++
		if !ebookMatchesQuery(item, query) {
			continue
		}
		output.MatchedCount++
		matches = append(matches, item)
		if len(output.Items) < limit {
			output.Items = append(output.Items, summarizeItem(item))
		} else {
			output.Truncated = true
		}
	}
	output.ReturnedCount = len(output.Items)
	return output, matches, nil
}

func itemHasEbook(item abs.LibraryItem) bool {
	if item.Media.EbookFile != nil {
		return true
	}
	for _, file := range item.LibraryFiles {
		if strings.EqualFold(file.FileType, "ebook") {
			return true
		}
	}
	return false
}

func ebookMatchesQuery(item abs.LibraryItem, query string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return true
	}
	haystack := strings.ToLower(strings.Join(ebookSearchFields(item), " "))
	for _, term := range strings.Fields(query) {
		if !strings.Contains(haystack, term) {
			return false
		}
	}
	return true
}

func ebookSearchFields(item abs.LibraryItem) []string {
	fields := []string{
		item.ID,
		item.Media.Metadata.Title,
		item.Media.Metadata.AuthorName,
		item.AuthorNamesFirstLast,
		item.Media.Metadata.SeriesName,
		item.Media.Metadata.ISBN,
		item.Media.Metadata.ASIN,
	}
	if item.Media.EbookFile != nil {
		fields = append(fields,
			item.Media.EbookFile.Metadata.Filename,
			item.Media.EbookFile.Metadata.RelPath,
			item.Media.EbookFile.Metadata.Path,
		)
	}
	for _, file := range item.LibraryFiles {
		if strings.EqualFold(file.FileType, "ebook") {
			fields = append(fields, file.Metadata.Filename, file.Metadata.RelPath, file.Metadata.Path)
		}
	}
	for _, author := range item.Media.Metadata.Authors {
		fields = append(fields, author.Name)
	}
	for _, series := range item.Media.Metadata.Series {
		fields = append(fields, series.Name)
	}
	return fields
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

func catalogListRequest(input CatalogListInput) (abs.CatalogListOptions, CatalogListOutput, error) {
	if input.LibraryID == "" {
		return abs.CatalogListOptions{}, CatalogListOutput{}, fmt.Errorf("libraryId is required")
	}
	limit, err := normalizeLimit(input.Limit)
	if err != nil {
		return abs.CatalogListOptions{}, CatalogListOutput{}, err
	}
	if input.Offset < 0 {
		return abs.CatalogListOptions{}, CatalogListOutput{}, fmt.Errorf("offset must be greater than or equal to 0")
	}
	page, err := pageFromOffset(input.Offset, limit)
	if err != nil {
		return abs.CatalogListOptions{}, CatalogListOutput{}, err
	}
	include := sanitizeInclude(input.Include)
	options := abs.CatalogListOptions{
		Limit:    limit,
		Page:     page,
		Sort:     input.Sort,
		Desc:     input.Desc,
		Filter:   input.Filter,
		Include:  include,
		Minified: input.Minified,
	}
	output := CatalogListOutput{
		LibraryID: input.LibraryID,
		Limit:     limit,
		Offset:    input.Offset,
		Page:      page,
		Sort:      input.Sort,
		Desc:      input.Desc,
		Filter:    input.Filter,
		Include:   include,
		Minified:  input.Minified,
	}
	return options, output, nil
}

func entityRequest(input EntityInput) ([]string, EntityOutput, error) {
	if input.ID == "" {
		return nil, EntityOutput{}, fmt.Errorf("id is required")
	}
	include := sanitizeInclude(input.Include)
	return include, EntityOutput{ID: input.ID, Include: include}, nil
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
