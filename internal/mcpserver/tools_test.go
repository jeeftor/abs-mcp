package mcpserver

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jeeftor/abs-mcp/internal/abs"
	"github.com/jeeftor/abs-mcp/internal/config"
)

func TestHealthCheck(t *testing.T) {
	t.Parallel()

	server := newTestServer()
	_, output, err := server.HealthCheck(context.Background(), nil, EmptyInput{})
	if err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}
	if !output.OK {
		t.Fatal("OK = false, want true")
	}
	if output.Username != "root" {
		t.Fatalf("Username = %q, want root", output.Username)
	}
	if output.LibraryCount != 2 {
		t.Fatalf("LibraryCount = %d, want 2", output.LibraryCount)
	}
	if !output.ReadOnly {
		t.Fatal("ReadOnly = false, want true")
	}
}

func TestListLibraries(t *testing.T) {
	t.Parallel()

	server := newTestServer()
	_, output, err := server.ListLibraries(context.Background(), nil, EmptyInput{})
	if err != nil {
		t.Fatalf("ListLibraries failed: %v", err)
	}
	if output.Count != 2 {
		t.Fatalf("Count = %d, want 2", output.Count)
	}
	if output.Libraries[0].Name != "Audiobooks" {
		t.Fatalf("first library = %q, want Audiobooks", output.Libraries[0].Name)
	}
	if output.Libraries[0].Folders[0].FullPath != "/audiobooks" {
		t.Fatalf("folder full path = %q, want /audiobooks", output.Libraries[0].Folders[0].FullPath)
	}
}

func TestGetLibrary(t *testing.T) {
	t.Parallel()

	server := newTestServer()
	_, output, err := server.GetLibrary(context.Background(), nil, LibraryInput{LibraryID: "lib-books"})
	if err != nil {
		t.Fatalf("GetLibrary failed: %v", err)
	}
	if output.Library.Name != "Ebooks" {
		t.Fatalf("library name = %q, want Ebooks", output.Library.Name)
	}
}

func TestGetLibraryRequiresID(t *testing.T) {
	t.Parallel()

	server := newTestServer()
	if _, _, err := server.GetLibrary(context.Background(), nil, LibraryInput{}); err == nil {
		t.Fatal("expected missing libraryId error")
	}
}

func TestListLibraryItems(t *testing.T) {
	t.Parallel()

	server := newTestServer()
	_, output, err := server.ListLibraryItems(context.Background(), nil, LibraryItemsInput{
		LibraryID: "lib-audio",
		Limit:     2,
		Offset:    2,
	})
	if err != nil {
		t.Fatalf("ListLibraryItems failed: %v", err)
	}
	if output.Total != 3 || output.Count != 1 {
		t.Fatalf("Total/Count = %d/%d, want 3/1", output.Total, output.Count)
	}
	if output.Limit != 2 || output.Offset != 2 || output.Page != 1 {
		t.Fatalf("Limit/Offset/Page = %d/%d/%d, want 2/2/1", output.Limit, output.Offset, output.Page)
	}
	if output.Items[0].Title != "Carol" {
		t.Fatalf("first title = %q, want Carol", output.Items[0].Title)
	}
}

func TestListLibraryItemsWithFilters(t *testing.T) {
	t.Parallel()

	client := newFakeABSClient()
	server := New(config.Config{ABSBaseURL: "http://abs", ReadOnly: true}, client)
	_, output, err := server.ListLibraryItems(context.Background(), nil, LibraryItemsInput{
		LibraryID:      "lib-audio",
		Limit:          2,
		Offset:         2,
		Sort:           "media.metadata.title",
		Desc:           true,
		Filter:         "issues.true",
		Include:        []string{"rssfeed", " Progress "},
		Minified:       true,
		CollapseSeries: true,
	})
	if err != nil {
		t.Fatalf("ListLibraryItems failed: %v", err)
	}
	if output.Sort != "media.metadata.title" || !output.Desc || output.Filter != "issues.true" {
		t.Fatalf("unexpected output filters: %#v", output)
	}
	if client.lastLibraryItemsOptions.Page != 1 {
		t.Fatalf("Page = %d, want 1", client.lastLibraryItemsOptions.Page)
	}
	if client.lastLibraryItemsOptions.Sort != "media.metadata.title" || !client.lastLibraryItemsOptions.Desc {
		t.Fatalf("unexpected sort options: %#v", client.lastLibraryItemsOptions)
	}
	if client.lastLibraryItemsOptions.Filter != "issues.true" {
		t.Fatalf("Filter = %q, want issues.true", client.lastLibraryItemsOptions.Filter)
	}
	if strings.Join(client.lastLibraryItemsOptions.Include, ",") != "rssfeed,progress" {
		t.Fatalf("Include = %#v, want rssfeed,progress", client.lastLibraryItemsOptions.Include)
	}
	if !client.lastLibraryItemsOptions.Minified || !client.lastLibraryItemsOptions.CollapseSeries {
		t.Fatalf("unexpected boolean options: %#v", client.lastLibraryItemsOptions)
	}
}

func TestListLibraryItemsDefaultsAndCapsLimit(t *testing.T) {
	t.Parallel()

	server := newTestServer()
	_, output, err := server.ListLibraryItems(context.Background(), nil, LibraryItemsInput{
		LibraryID: "lib-audio",
		Limit:     1000,
	})
	if err != nil {
		t.Fatalf("ListLibraryItems failed: %v", err)
	}
	if output.Limit != 100 {
		t.Fatalf("Limit = %d, want capped 100", output.Limit)
	}

	_, output, err = server.ListLibraryItems(context.Background(), nil, LibraryItemsInput{LibraryID: "lib-audio"})
	if err != nil {
		t.Fatalf("ListLibraryItems default failed: %v", err)
	}
	if output.Limit != 25 {
		t.Fatalf("default Limit = %d, want 25", output.Limit)
	}
}

func TestListLibraryItemsRejectsBadInput(t *testing.T) {
	t.Parallel()

	server := newTestServer()
	if _, _, err := server.ListLibraryItems(context.Background(), nil, LibraryItemsInput{}); err == nil {
		t.Fatal("expected missing libraryId error")
	}
	if _, _, err := server.ListLibraryItems(context.Background(), nil, LibraryItemsInput{
		LibraryID: "lib-audio",
		Limit:     -1,
	}); err == nil {
		t.Fatal("expected negative limit error")
	}
	if _, _, err := server.ListLibraryItems(context.Background(), nil, LibraryItemsInput{
		LibraryID: "lib-audio",
		Offset:    -1,
	}); err == nil {
		t.Fatal("expected negative offset error")
	}
	if _, _, err := server.ListLibraryItems(context.Background(), nil, LibraryItemsInput{
		LibraryID: "lib-audio",
		Limit:     2,
		Offset:    1,
	}); err == nil {
		t.Fatal("expected unaligned offset error")
	}
}

func TestGetLibraryItem(t *testing.T) {
	t.Parallel()

	server := newTestServer()
	_, output, err := server.GetLibraryItem(context.Background(), nil, LibraryItemInput{ItemID: "item-1"})
	if err != nil {
		t.Fatalf("GetLibraryItem failed: %v", err)
	}
	if output.Item.Title != "Alice" {
		t.Fatalf("title = %q, want Alice", output.Item.Title)
	}
	if output.Item.Author != "Lewis Carroll" {
		t.Fatalf("author = %q, want Lewis Carroll", output.Item.Author)
	}
	if len(output.Item.Files) != 1 {
		t.Fatalf("file count = %d, want 1: %#v", len(output.Item.Files), output.Item.Files)
	}
	if output.Item.Files[0].Filename != "alice.m4b" {
		t.Fatalf("filename = %q, want alice.m4b", output.Item.Files[0].Filename)
	}
}

func TestGetLibraryItemRequiresID(t *testing.T) {
	t.Parallel()

	server := newTestServer()
	if _, _, err := server.GetLibraryItem(context.Background(), nil, LibraryItemInput{}); err == nil {
		t.Fatal("expected missing itemId error")
	}
}

func TestSearchLibrary(t *testing.T) {
	t.Parallel()

	server := newTestServer()
	_, output, err := server.SearchLibrary(context.Background(), nil, SearchLibraryInput{
		LibraryID: "lib-audio",
		Query:     "alice",
		Limit:     1000,
	})
	if err != nil {
		t.Fatalf("SearchLibrary failed: %v", err)
	}
	if output.Limit != 50 {
		t.Fatalf("Limit = %d, want capped 50", output.Limit)
	}
	if output.Data == nil {
		t.Fatal("expected search data")
	}
}

func TestSearchLibraryRejectsBadInput(t *testing.T) {
	t.Parallel()

	server := newTestServer()
	if _, _, err := server.SearchLibrary(context.Background(), nil, SearchLibraryInput{Query: "alice"}); err == nil {
		t.Fatal("expected missing libraryId error")
	}
	if _, _, err := server.SearchLibrary(context.Background(), nil, SearchLibraryInput{LibraryID: "lib-audio"}); err == nil {
		t.Fatal("expected missing query error")
	}
	if _, _, err := server.SearchLibrary(context.Background(), nil, SearchLibraryInput{
		LibraryID: "lib-audio",
		Query:     "alice",
		Limit:     -1,
	}); err == nil {
		t.Fatal("expected negative limit error")
	}
}

func TestGetLibraryStats(t *testing.T) {
	t.Parallel()

	server := newTestServer()
	_, output, err := server.GetLibraryStats(context.Background(), nil, LibraryRawInput{LibraryID: "lib-audio"})
	if err != nil {
		t.Fatalf("GetLibraryStats failed: %v", err)
	}
	if output.Data == nil {
		t.Fatal("expected stats data")
	}
}

func TestGetLibraryFilterData(t *testing.T) {
	t.Parallel()

	server := newTestServer()
	_, output, err := server.GetLibraryFilterData(context.Background(), nil, LibraryRawInput{LibraryID: "lib-audio"})
	if err != nil {
		t.Fatalf("GetLibraryFilterData failed: %v", err)
	}
	if output.Data == nil {
		t.Fatal("expected filter data")
	}
}

func TestGetItemMetadataObject(t *testing.T) {
	t.Parallel()

	server := newTestServer()
	_, output, err := server.GetItemMetadataObject(context.Background(), nil, LibraryItemInput{ItemID: "item-1"})
	if err != nil {
		t.Fatalf("GetItemMetadataObject failed: %v", err)
	}
	if output.Data == nil {
		t.Fatal("expected metadata object data")
	}
}

func TestFindMisorganizedItems(t *testing.T) {
	t.Parallel()

	client := newFakeABSClient()
	client.items["lib-audio"] = []abs.LibraryItem{
		{
			ID:        "organized",
			LibraryID: "lib-audio",
			Path:      "/audiobooks/Lewis Carroll/Alice",
			RelPath:   "Lewis Carroll/Alice",
			MediaType: "book",
			Media:     abs.Media{Metadata: abs.Metadata{Title: "Alice", AuthorName: "Lewis Carroll"}},
		},
		{
			ID:        "series-ok",
			LibraryID: "lib-audio",
			Path:      "/audiobooks/Lewis Carroll/Alice Books/Alice",
			RelPath:   "Lewis Carroll/Alice Books/Alice",
			MediaType: "book",
			Media: abs.Media{Metadata: abs.Metadata{
				Title:      "Alice",
				AuthorName: "Lewis Carroll",
				SeriesName: "Alice Books",
			}},
		},
		{
			ID:        "flat",
			LibraryID: "lib-audio",
			Path:      "/audiobooks/Alice",
			RelPath:   "Alice",
			MediaType: "book",
			Media:     abs.Media{Metadata: abs.Metadata{Title: "Alice", AuthorName: "Lewis Carroll"}},
		},
		{
			ID:        "wrong-series",
			LibraryID: "lib-audio",
			Path:      "/audiobooks/Lewis Carroll/Alice",
			RelPath:   "Lewis Carroll/Alice",
			MediaType: "book",
			Media: abs.Media{Metadata: abs.Metadata{
				Title:      "Alice",
				AuthorName: "Lewis Carroll",
				SeriesName: "Alice Books",
			}},
		},
		{
			ID:        "missing-author",
			LibraryID: "lib-audio",
			Path:      "/audiobooks/Unknown/Alice",
			RelPath:   "Unknown/Alice",
			MediaType: "book",
			Media:     abs.Media{Metadata: abs.Metadata{Title: "Alice"}},
		},
	}

	server := New(config.Config{ABSBaseURL: "http://abs", ReadOnly: true}, client)
	_, output, err := server.FindMisorganizedItems(context.Background(), nil, FindMisorganizedItemsInput{
		LibraryID: "lib-audio",
	})
	if err != nil {
		t.Fatalf("FindMisorganizedItems failed: %v", err)
	}
	if output.CheckedCount != 5 || output.OrganizedCount != 2 || output.MisorganizedCount != 2 || output.UnclassifiableCount != 1 {
		t.Fatalf("unexpected layout counts: %#v", output)
	}
	if output.ReturnedCount != 3 {
		t.Fatalf("returned count = %d, want 3", output.ReturnedCount)
	}
	findings := layoutFindingsByID(output.Items)
	if !contains(findings["flat"].Reasons, "path_too_shallow") || !contains(findings["flat"].Reasons, "author_directory_mismatch") {
		t.Fatalf("flat reasons = %#v", findings["flat"].Reasons)
	}
	if !contains(findings["wrong-series"].Reasons, "path_too_shallow") || !contains(findings["wrong-series"].Reasons, "series_directory_mismatch") {
		t.Fatalf("wrong-series reasons = %#v", findings["wrong-series"].Reasons)
	}
	if !contains(findings["missing-author"].Reasons, "metadata_missing_author") {
		t.Fatalf("missing-author reasons = %#v", findings["missing-author"].Reasons)
	}
}

func TestFindMisorganizedItemsIncludeOrganizedAndLimit(t *testing.T) {
	t.Parallel()

	server := newTestServer()
	_, output, err := server.FindMisorganizedItems(context.Background(), nil, FindMisorganizedItemsInput{
		LibraryID:        "lib-audio",
		Convention:       "author-title",
		Limit:            2,
		IncludeOrganized: true,
	})
	if err != nil {
		t.Fatalf("FindMisorganizedItems failed: %v", err)
	}
	if output.ReturnedCount != 2 || !output.Truncated {
		t.Fatalf("expected two truncated findings, got %#v", output)
	}
	if output.Convention != "author-title" {
		t.Fatalf("convention = %q, want author-title", output.Convention)
	}
}

func TestFindMisorganizedItemsRejectsBadInput(t *testing.T) {
	t.Parallel()

	server := newTestServer()
	tests := map[string]FindMisorganizedItemsInput{
		"missing library": {Convention: "auto"},
		"bad convention":  {LibraryID: "lib-audio", Convention: "flat"},
		"bad limit":       {LibraryID: "lib-audio", Limit: -1},
	}
	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, _, err := server.FindMisorganizedItems(context.Background(), nil, input); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestRawToolsRequireIDs(t *testing.T) {
	t.Parallel()

	server := newTestServer()
	if _, _, err := server.GetLibraryStats(context.Background(), nil, LibraryRawInput{}); err == nil {
		t.Fatal("expected stats missing libraryId error")
	}
	if _, _, err := server.GetLibraryFilterData(context.Background(), nil, LibraryRawInput{}); err == nil {
		t.Fatal("expected filterdata missing libraryId error")
	}
	if _, _, err := server.GetItemMetadataObject(context.Background(), nil, LibraryItemInput{}); err == nil {
		t.Fatal("expected metadata-object missing itemId error")
	}
}

func TestScanLibraryBlockedInReadOnlyMode(t *testing.T) {
	t.Parallel()

	server := newTestServer()
	if _, _, err := server.ScanLibrary(context.Background(), nil, ScanLibraryInput{
		LibraryID: "lib-audio",
		Force:     true,
	}); err == nil || !strings.Contains(err.Error(), "--read-only=false") {
		t.Fatalf("expected actionable read-only error, got %v", err)
	}
}

func TestScanLibrary(t *testing.T) {
	t.Parallel()

	client := newFakeABSClient()
	server := New(config.Config{ABSBaseURL: "http://abs", ReadOnly: false}, client)
	_, output, err := server.ScanLibrary(context.Background(), nil, ScanLibraryInput{
		LibraryID: "lib-audio",
		Force:     true,
	})
	if err != nil {
		t.Fatalf("ScanLibrary failed: %v", err)
	}
	if !output.Triggered {
		t.Fatal("Triggered = false, want true")
	}
	if client.scanLibraryID != "lib-audio" {
		t.Fatalf("scanLibraryID = %q, want lib-audio", client.scanLibraryID)
	}
	if !client.scanForce {
		t.Fatal("scanForce = false, want true")
	}
}

func TestScanLibraryRequiresID(t *testing.T) {
	t.Parallel()

	server := New(config.Config{ABSBaseURL: "http://abs", ReadOnly: false}, newFakeABSClient())
	if _, _, err := server.ScanLibrary(context.Background(), nil, ScanLibraryInput{}); err == nil {
		t.Fatal("expected missing libraryId error")
	}
}

func TestRemoveLibraryItemsWithIssuesBlockedInReadOnlyMode(t *testing.T) {
	t.Parallel()

	server := newTestServer()
	if _, _, err := server.RemoveLibraryItemsWithIssues(context.Background(), nil, RemoveLibraryItemsWithIssuesInput{
		LibraryID:    "lib-audio",
		Confirmation: "remove issues from lib-audio",
	}); err == nil || !strings.Contains(err.Error(), "--read-only=false") {
		t.Fatalf("expected actionable read-only error, got %v", err)
	}
}

func TestRemoveLibraryItemsWithIssuesRequiresConfirmation(t *testing.T) {
	t.Parallel()

	server := New(config.Config{ABSBaseURL: "http://abs", ReadOnly: false}, newFakeABSClient())
	for _, input := range []RemoveLibraryItemsWithIssuesInput{
		{},
		{LibraryID: "lib-audio"},
		{LibraryID: "lib-audio", Confirmation: "yes"},
		{LibraryID: "lib-audio", Confirmation: "remove issues from lib-audio", ExpectedIssueCount: -1},
	} {
		if _, _, err := server.RemoveLibraryItemsWithIssues(context.Background(), nil, input); err == nil {
			t.Fatalf("expected validation error for %#v", input)
		}
	}
}

func TestRemoveLibraryItemsWithIssues(t *testing.T) {
	t.Parallel()

	client := newFakeABSClient()
	client.items["lib-audio"][0].IsMissing = true
	client.items["lib-audio"][2].IsInvalid = true
	server := New(config.Config{ABSBaseURL: "http://abs", ReadOnly: false}, client)

	_, output, err := server.RemoveLibraryItemsWithIssues(context.Background(), nil, RemoveLibraryItemsWithIssuesInput{
		LibraryID:          "lib-audio",
		Confirmation:       "remove issues from lib-audio",
		ExpectedIssueCount: 2,
	})
	if err != nil {
		t.Fatalf("RemoveLibraryItemsWithIssues failed: %v", err)
	}
	if !output.Triggered {
		t.Fatal("Triggered = false, want true")
	}
	if output.IssueCountBefore != 2 || output.RemovedCount != 2 || output.RemainingIssueCount != 0 {
		t.Fatalf("unexpected cleanup counts: %#v", output)
	}
	if !client.removeIssuesCalled || client.removeIssuesLibraryID != "lib-audio" {
		t.Fatalf("remove issues call = %v/%q, want true/lib-audio", client.removeIssuesCalled, client.removeIssuesLibraryID)
	}
}

func TestRemoveLibraryItemsWithIssuesNoIssuesDoesNotCallABSDelete(t *testing.T) {
	t.Parallel()

	client := newFakeABSClient()
	server := New(config.Config{ABSBaseURL: "http://abs", ReadOnly: false}, client)

	_, output, err := server.RemoveLibraryItemsWithIssues(context.Background(), nil, RemoveLibraryItemsWithIssuesInput{
		LibraryID:    "lib-audio",
		Confirmation: "remove issues from lib-audio",
	})
	if err != nil {
		t.Fatalf("RemoveLibraryItemsWithIssues failed: %v", err)
	}
	if output.Triggered {
		t.Fatal("Triggered = true, want false when there are no issues")
	}
	if client.removeIssuesCalled {
		t.Fatal("RemoveLibraryItemsWithIssues called ABS delete despite no issues")
	}
}

func TestRemoveLibraryItemsWithIssuesExpectedCountMismatch(t *testing.T) {
	t.Parallel()

	client := newFakeABSClient()
	client.items["lib-audio"][0].IsMissing = true
	server := New(config.Config{ABSBaseURL: "http://abs", ReadOnly: false}, client)

	if _, _, err := server.RemoveLibraryItemsWithIssues(context.Background(), nil, RemoveLibraryItemsWithIssuesInput{
		LibraryID:          "lib-audio",
		Confirmation:       "remove issues from lib-audio",
		ExpectedIssueCount: 2,
	}); err == nil {
		t.Fatal("expected count mismatch error")
	}
	if client.removeIssuesCalled {
		t.Fatal("RemoveLibraryItemsWithIssues called ABS delete after count mismatch")
	}
}

func TestScanItemBlockedInReadOnlyMode(t *testing.T) {
	t.Parallel()

	server := newTestServer()
	if _, _, err := server.ScanItem(context.Background(), nil, ScanItemInput{ItemID: "item-1"}); err == nil || !strings.Contains(err.Error(), "--read-only=false") {
		t.Fatalf("expected actionable read-only error, got %v", err)
	}
}

func TestScanItem(t *testing.T) {
	t.Parallel()

	client := newFakeABSClient()
	server := New(config.Config{ABSBaseURL: "http://abs", ReadOnly: false}, client)
	_, output, err := server.ScanItem(context.Background(), nil, ScanItemInput{ItemID: "item-1"})
	if err != nil {
		t.Fatalf("ScanItem failed: %v", err)
	}
	if !output.Triggered {
		t.Fatal("Triggered = false, want true")
	}
	if output.ItemID != "item-1" {
		t.Fatalf("ItemID = %q, want item-1", output.ItemID)
	}
	if output.Result != "SUCCESS" {
		t.Fatalf("Result = %q, want SUCCESS", output.Result)
	}
	if client.scanItemID != "item-1" {
		t.Fatalf("scanItemID = %q, want item-1", client.scanItemID)
	}
}

func TestScanItemRequiresID(t *testing.T) {
	t.Parallel()

	server := New(config.Config{ABSBaseURL: "http://abs", ReadOnly: false}, newFakeABSClient())
	if _, _, err := server.ScanItem(context.Background(), nil, ScanItemInput{}); err == nil {
		t.Fatal("expected missing itemId error")
	}
}

func TestPlannedMutatingToolsBlockedInReadOnlyMode(t *testing.T) {
	t.Parallel()

	server := newTestServer()
	tests := map[string]func() error{
		"abs_update_item_metadata": func() error {
			_, _, err := server.UpdateItemMetadata(context.Background(), nil, ItemPayloadInput{ItemID: "item-1"})
			return err
		},
		"abs_update_item_cover": func() error {
			_, _, err := server.UpdateItemCover(context.Background(), nil, ItemPayloadInput{ItemID: "item-1"})
			return err
		},
		"abs_remove_item_cover": func() error {
			_, _, err := server.RemoveItemCover(context.Background(), nil, ConfirmedItemInput{ItemID: "item-1", Confirmation: "remove cover from item-1"})
			return err
		},
		"abs_match_item": func() error {
			_, _, err := server.MatchItem(context.Background(), nil, MatchItemInput{ItemID: "item-1"})
			return err
		},
		"abs_update_item_chapters": func() error {
			_, _, err := server.UpdateItemChapters(context.Background(), nil, ItemPayloadInput{ItemID: "item-1"})
			return err
		},
		"abs_update_item_tracks": func() error {
			_, _, err := server.UpdateItemTracks(context.Background(), nil, ItemPayloadInput{ItemID: "item-1"})
			return err
		},
		"abs_create_collection": func() error {
			_, _, err := server.CreateCollection(context.Background(), nil, CollectionInput{Name: "Favorites"})
			return err
		},
		"abs_update_collection": func() error {
			_, _, err := server.UpdateCollection(context.Background(), nil, CollectionInput{CollectionID: "col-1"})
			return err
		},
		"abs_delete_collection": func() error {
			_, _, err := server.DeleteCollection(context.Background(), nil, ConfirmedCollectionInput{CollectionID: "col-1", Confirmation: "delete collection col-1"})
			return err
		},
		"abs_add_collection_item": func() error {
			_, _, err := server.AddCollectionItem(context.Background(), nil, CollectionItemInput{CollectionID: "col-1", ItemID: "item-1"})
			return err
		},
		"abs_remove_collection_item": func() error {
			_, _, err := server.RemoveCollectionItem(context.Background(), nil, CollectionItemInput{CollectionID: "col-1", ItemID: "item-1", Confirmation: "remove item item-1 from collection col-1"})
			return err
		},
		"abs_create_playlist": func() error {
			_, _, err := server.CreatePlaylist(context.Background(), nil, PlaylistInput{Name: "Queue"})
			return err
		},
		"abs_update_playlist": func() error {
			_, _, err := server.UpdatePlaylist(context.Background(), nil, PlaylistInput{PlaylistID: "pl-1"})
			return err
		},
		"abs_delete_playlist": func() error {
			_, _, err := server.DeletePlaylist(context.Background(), nil, ConfirmedPlaylistInput{PlaylistID: "pl-1", Confirmation: "delete playlist pl-1"})
			return err
		},
		"abs_add_playlist_item": func() error {
			_, _, err := server.AddPlaylistItem(context.Background(), nil, PlaylistItemInput{PlaylistID: "pl-1", ItemID: "item-1"})
			return err
		},
		"abs_remove_playlist_item": func() error {
			_, _, err := server.RemovePlaylistItem(context.Background(), nil, PlaylistItemInput{PlaylistID: "pl-1", ItemID: "item-1", Confirmation: "remove item item-1 from playlist pl-1"})
			return err
		},
	}

	for toolName, call := range tests {
		err := call()
		if err == nil {
			t.Fatalf("%s: expected read-only error", toolName)
		}
		if !strings.Contains(err.Error(), "ABS_READ_ONLY=true") || !strings.Contains(err.Error(), "--read-only=false") {
			t.Fatalf("%s: expected actionable read-only error, got %v", toolName, err)
		}
	}
}

func TestPlannedMutatingToolsValidateInputBeforeImplementation(t *testing.T) {
	t.Parallel()

	server := New(config.Config{ABSBaseURL: "http://abs", ReadOnly: false}, newFakeABSClient())
	if _, _, err := server.UpdateItemMetadata(context.Background(), nil, ItemPayloadInput{}); err == nil || !strings.Contains(err.Error(), "itemId") {
		t.Fatalf("expected missing itemId error, got %v", err)
	}
	if _, _, err := server.RemoveItemCover(context.Background(), nil, ConfirmedItemInput{ItemID: "item-1", Confirmation: "yes"}); err == nil || !strings.Contains(err.Error(), "remove cover from item-1") {
		t.Fatalf("expected cover confirmation error, got %v", err)
	}
	if _, _, err := server.DeleteCollection(context.Background(), nil, ConfirmedCollectionInput{CollectionID: "col-1", Confirmation: "yes"}); err == nil || !strings.Contains(err.Error(), "delete collection col-1") {
		t.Fatalf("expected collection confirmation error, got %v", err)
	}
	if _, _, err := server.RemoveCollectionItem(context.Background(), nil, CollectionItemInput{CollectionID: "col-1", ItemID: "item-1", Confirmation: "yes"}); err == nil || !strings.Contains(err.Error(), "remove item item-1 from collection col-1") {
		t.Fatalf("expected collection item confirmation error, got %v", err)
	}
	if _, _, err := server.DeletePlaylist(context.Background(), nil, ConfirmedPlaylistInput{PlaylistID: "pl-1", Confirmation: "yes"}); err == nil || !strings.Contains(err.Error(), "delete playlist pl-1") {
		t.Fatalf("expected playlist confirmation error, got %v", err)
	}
	if _, _, err := server.RemovePlaylistItem(context.Background(), nil, PlaylistItemInput{PlaylistID: "pl-1", ItemID: "item-1", Confirmation: "yes"}); err == nil || !strings.Contains(err.Error(), "remove item item-1 from playlist pl-1") {
		t.Fatalf("expected playlist item confirmation error, got %v", err)
	}
}

func TestPlannedMutatingToolsAreNotImplementedWithReadOnlyDisabled(t *testing.T) {
	t.Parallel()

	server := New(config.Config{ABSBaseURL: "http://abs", ReadOnly: false}, newFakeABSClient())
	tests := map[string]func() error{
		"abs_update_item_metadata": func() error {
			_, _, err := server.UpdateItemMetadata(context.Background(), nil, ItemPayloadInput{ItemID: "item-1"})
			return err
		},
		"abs_remove_item_cover": func() error {
			_, _, err := server.RemoveItemCover(context.Background(), nil, ConfirmedItemInput{ItemID: "item-1", Confirmation: "remove cover from item-1"})
			return err
		},
		"abs_create_collection": func() error {
			_, _, err := server.CreateCollection(context.Background(), nil, CollectionInput{Name: "Favorites"})
			return err
		},
		"abs_delete_playlist": func() error {
			_, _, err := server.DeletePlaylist(context.Background(), nil, ConfirmedPlaylistInput{PlaylistID: "pl-1", Confirmation: "delete playlist pl-1"})
			return err
		},
	}

	for toolName, call := range tests {
		err := call()
		if err == nil || !strings.Contains(err.Error(), "not implemented yet") {
			t.Fatalf("%s: expected not implemented error, got %v", toolName, err)
		}
	}
}

func TestScanLibraryAndWaitBlockedInReadOnlyMode(t *testing.T) {
	t.Parallel()

	server := newTestServer()
	if _, _, err := server.ScanLibraryAndWait(context.Background(), nil, ScanLibraryAndWaitInput{
		LibraryID:     "lib-audio",
		ExpectedTotal: 3,
	}); err == nil || !strings.Contains(err.Error(), "--read-only=false") {
		t.Fatalf("expected actionable read-only error, got %v", err)
	}
}

func TestScanLibraryAndWaitCompletesAfterPolling(t *testing.T) {
	t.Parallel()

	client := newFakeABSClient()
	client.libraryItemTotals = []int{1, 2, 3}
	server := New(config.Config{ABSBaseURL: "http://abs", ReadOnly: false}, client)

	_, output, err := server.ScanLibraryAndWait(context.Background(), nil, ScanLibraryAndWaitInput{
		LibraryID:                "lib-audio",
		Force:                    true,
		ExpectedTotal:            3,
		TimeoutSeconds:           1,
		PollIntervalMilliseconds: 1,
	})
	if err != nil {
		t.Fatalf("ScanLibraryAndWait failed: %v", err)
	}
	if !output.Triggered || !output.Completed || output.TimedOut {
		t.Fatalf("unexpected scan status: %#v", output)
	}
	if output.ObservedTotal != 3 {
		t.Fatalf("ObservedTotal = %d, want 3", output.ObservedTotal)
	}
	if output.Attempts != 3 {
		t.Fatalf("Attempts = %d, want 3", output.Attempts)
	}
	if client.scanLibraryID != "lib-audio" || !client.scanForce {
		t.Fatalf("scan request = %q/%v, want lib-audio/true", client.scanLibraryID, client.scanForce)
	}
}

func TestScanLibraryAndWaitWithoutExpectedTotalObservesOnce(t *testing.T) {
	t.Parallel()

	client := newFakeABSClient()
	server := New(config.Config{ABSBaseURL: "http://abs", ReadOnly: false}, client)

	_, output, err := server.ScanLibraryAndWait(context.Background(), nil, ScanLibraryAndWaitInput{
		LibraryID: "lib-audio",
	})
	if err != nil {
		t.Fatalf("ScanLibraryAndWait failed: %v", err)
	}
	if !output.Completed || output.Attempts != 1 {
		t.Fatalf("unexpected scan status: %#v", output)
	}
	if output.ObservedTotal != 3 {
		t.Fatalf("ObservedTotal = %d, want 3", output.ObservedTotal)
	}
}

func TestScanLibraryAndWaitTimeoutReturnsStatus(t *testing.T) {
	t.Parallel()

	client := newFakeABSClient()
	client.libraryItemTotals = []int{1, 1, 1}
	server := New(config.Config{ABSBaseURL: "http://abs", ReadOnly: false}, client)

	_, output, err := server.ScanLibraryAndWait(context.Background(), nil, ScanLibraryAndWaitInput{
		LibraryID:                "lib-audio",
		ExpectedTotal:            3,
		TimeoutSeconds:           1,
		PollIntervalMilliseconds: 1,
	})
	if err != nil {
		t.Fatalf("ScanLibraryAndWait failed: %v", err)
	}
	if output.Completed || !output.TimedOut {
		t.Fatalf("unexpected scan timeout status: %#v", output)
	}
	if output.ObservedTotal != 1 {
		t.Fatalf("ObservedTotal = %d, want 1", output.ObservedTotal)
	}
}

func TestScanLibraryAndWaitRejectsBadInput(t *testing.T) {
	t.Parallel()

	server := New(config.Config{ABSBaseURL: "http://abs", ReadOnly: false}, newFakeABSClient())
	tests := map[string]ScanLibraryAndWaitInput{
		"missing library":         {},
		"negative expected total": {LibraryID: "lib-audio", ExpectedTotal: -1},
		"negative timeout":        {LibraryID: "lib-audio", TimeoutSeconds: -1},
		"negative poll interval":  {LibraryID: "lib-audio", PollIntervalMilliseconds: -1},
	}
	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, _, err := server.ScanLibraryAndWait(context.Background(), nil, input); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestToolHandlerPropagatesClientError(t *testing.T) {
	t.Parallel()

	server := New(config.Config{ABSBaseURL: "http://abs", ReadOnly: true}, &fakeABSClient{
		err: errors.New("boom"),
	})
	if _, _, err := server.ListLibraries(context.Background(), nil, EmptyInput{}); err == nil {
		t.Fatal("expected client error")
	}
}

func newTestServer() *Server {
	return New(config.Config{ABSBaseURL: "http://abs", ReadOnly: true}, newFakeABSClient())
}

func layoutFindingsByID(items []LayoutAuditItem) map[string]LayoutAuditItem {
	findings := make(map[string]LayoutAuditItem, len(items))
	for _, item := range items {
		findings[item.ItemID] = item
	}
	return findings
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

type fakeABSClient struct {
	user                    abs.User
	libraries               []abs.Library
	items                   map[string][]abs.LibraryItem
	libraryItemTotals       []int
	getLibraryItemsCalls    int
	lastLibraryItemsOptions abs.LibraryItemsOptions
	scanLibraryID           string
	scanForce               bool
	scanItemID              string
	removeIssuesCalled      bool
	removeIssuesLibraryID   string
	err                     error
}

func newFakeABSClient() *fakeABSClient {
	return &fakeABSClient{
		user: abs.User{ID: "user-1", Username: "root", Type: "root", IsActive: true},
		libraries: []abs.Library{
			{
				ID:        "lib-audio",
				Name:      "Audiobooks",
				MediaType: "book",
				Folders:   []abs.Folder{{ID: "folder-audio", FullPath: "/audiobooks"}},
			},
			{
				ID:        "lib-books",
				Name:      "Ebooks",
				MediaType: "book",
				Folders:   []abs.Folder{{ID: "folder-books", FullPath: "/books"}},
			},
		},
		items: map[string][]abs.LibraryItem{
			"lib-audio": {
				{
					ID:        "item-0",
					LibraryID: "lib-audio",
					Path:      "/audiobooks/zero",
					MediaType: "book",
					Media:     abs.Media{Metadata: abs.Metadata{Title: "Zero", AuthorName: "Author Zero"}},
				},
				{
					ID:        "item-1",
					LibraryID: "lib-audio",
					Path:      "/audiobooks/alice",
					MediaType: "book",
					Media:     abs.Media{Metadata: abs.Metadata{Title: "Alice", AuthorName: "Lewis Carroll"}},
					LibraryFiles: []abs.LibraryFile{
						{
							FileType: "audio",
							Metadata: abs.FileMetadata{
								Filename: "alice.m4b",
								Path:     "/audiobooks/alice/alice.m4b",
								RelPath:  "alice/alice.m4b",
								Size:     123,
							},
						},
					},
				},
				{
					ID:        "item-2",
					LibraryID: "lib-audio",
					Path:      "/audiobooks/carol",
					MediaType: "book",
					Media:     abs.Media{Metadata: abs.Metadata{Title: "Carol", AuthorName: "Charles Dickens"}},
				},
			},
			"lib-books": {
				{
					ID:        "book-1",
					LibraryID: "lib-books",
					Path:      "/books/alice",
					MediaType: "book",
					Media:     abs.Media{Metadata: abs.Metadata{Title: "Alice", AuthorName: "Lewis Carroll"}},
				},
			},
		},
	}
}

func (f *fakeABSClient) GetCurrentUser(context.Context) (*abs.User, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &f.user, nil
}

func (f *fakeABSClient) GetLibraries(context.Context) ([]abs.Library, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.libraries, nil
}

func (f *fakeABSClient) GetLibrary(_ context.Context, libraryID string) (*abs.Library, error) {
	if f.err != nil {
		return nil, f.err
	}
	for _, library := range f.libraries {
		if library.ID == libraryID {
			return &library, nil
		}
	}
	return nil, errors.New("library not found")
}

func (f *fakeABSClient) GetLibraryItems(
	_ context.Context,
	libraryID string,
	limit int,
	offset int,
) (*abs.LibraryItemsResponse, error) {
	page := 0
	if limit > 0 && offset > 0 {
		page = offset / limit
	}
	return f.GetLibraryItemsWithOptions(context.Background(), libraryID, abs.LibraryItemsOptions{Limit: limit, Page: page})
}

func (f *fakeABSClient) GetLibraryItemsWithOptions(
	_ context.Context,
	libraryID string,
	options abs.LibraryItemsOptions,
) (*abs.LibraryItemsResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.lastLibraryItemsOptions = options
	allItems := f.items[libraryID]
	total := len(allItems)
	if len(f.libraryItemTotals) > 0 {
		index := f.getLibraryItemsCalls
		if index >= len(f.libraryItemTotals) {
			index = len(f.libraryItemTotals) - 1
		}
		total = f.libraryItemTotals[index]
	}
	f.getLibraryItemsCalls++
	limit := options.Limit
	if limit == 0 {
		limit = len(allItems)
	}
	offset := options.Page * limit
	if offset > len(allItems) {
		offset = len(allItems)
	}
	end := offset + limit
	if end > len(allItems) {
		end = len(allItems)
	}
	return &abs.LibraryItemsResponse{
		Results: allItems[offset:end],
		Total:   total,
		Limit:   limit,
		Page:    options.Page,
		Offset:  offset,
	}, nil
}

func (f *fakeABSClient) GetAllLibraryItems(_ context.Context, libraryID string) ([]abs.LibraryItem, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.items[libraryID], nil
}

func (f *fakeABSClient) GetLibraryItem(_ context.Context, itemID string) (*abs.LibraryItem, error) {
	if f.err != nil {
		return nil, f.err
	}
	for _, items := range f.items {
		for _, item := range items {
			if item.ID == itemID {
				return &item, nil
			}
		}
	}
	return nil, errors.New("item not found")
}

func (f *fakeABSClient) SearchLibrary(_ context.Context, libraryID string, query string, limit int) (abs.JSONValue, error) {
	if f.err != nil {
		return nil, f.err
	}
	return map[string]any{
		"libraryId": libraryID,
		"query":     query,
		"limit":     limit,
		"results":   []any{map[string]any{"id": "item-1"}},
	}, nil
}

func (f *fakeABSClient) GetLibraryStats(_ context.Context, libraryID string) (abs.JSONValue, error) {
	if f.err != nil {
		return nil, f.err
	}
	return map[string]any{"libraryId": libraryID, "totalItems": len(f.items[libraryID])}, nil
}

func (f *fakeABSClient) GetLibraryFilterData(_ context.Context, libraryID string) (abs.JSONValue, error) {
	if f.err != nil {
		return nil, f.err
	}
	return map[string]any{"libraryId": libraryID, "genres": []any{"fiction"}}, nil
}

func (f *fakeABSClient) GetItemMetadataObject(_ context.Context, itemID string) (abs.JSONValue, error) {
	if f.err != nil {
		return nil, f.err
	}
	return map[string]any{"itemId": itemID, "title": "Alice"}, nil
}

func (f *fakeABSClient) ScanLibrary(_ context.Context, libraryID string, force bool) error {
	if f.err != nil {
		return f.err
	}
	f.scanLibraryID = libraryID
	f.scanForce = force
	return nil
}

func (f *fakeABSClient) ScanItem(_ context.Context, itemID string) (*abs.ScanItemResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.scanItemID = itemID
	return &abs.ScanItemResponse{Result: "SUCCESS"}, nil
}

func (f *fakeABSClient) RemoveLibraryItemsWithIssues(_ context.Context, libraryID string) error {
	if f.err != nil {
		return f.err
	}
	f.removeIssuesCalled = true
	f.removeIssuesLibraryID = libraryID
	items := f.items[libraryID]
	kept := items[:0]
	for _, item := range items {
		if !item.IsMissing && !item.IsInvalid {
			kept = append(kept, item)
		}
	}
	f.items[libraryID] = kept
	return nil
}
