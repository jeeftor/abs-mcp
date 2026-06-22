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

func TestListBackups(t *testing.T) {
	t.Parallel()

	server := newTestServer()
	_, output, err := server.ListBackups(context.Background(), nil, EmptyInput{})
	if err != nil {
		t.Fatalf("ListBackups failed: %v", err)
	}
	if output.Count != 1 {
		t.Fatalf("Count = %d, want 1", output.Count)
	}
	if output.Backups[0].ID != "backup-1" {
		t.Fatalf("first backup ID = %q, want backup-1", output.Backups[0].ID)
	}
}

func TestListEreaderDevices(t *testing.T) {
	t.Parallel()

	server := newTestServer()
	_, output, err := server.ListEreaderDevices(context.Background(), nil, EmptyInput{})
	if err != nil {
		t.Fatalf("ListEreaderDevices failed: %v", err)
	}
	if output.Count != 2 {
		t.Fatalf("Count = %d, want 2", output.Count)
	}
	if output.Devices[0].Name != "Kindle" {
		t.Fatalf("first device name = %q, want Kindle", output.Devices[0].Name)
	}
	if output.Devices[0].Email != "" {
		t.Fatalf("device email = %q, want redacted empty email", output.Devices[0].Email)
	}
}

func TestCreateBackup(t *testing.T) {
	t.Parallel()

	client := newFakeABSClient()
	server := New(config.Config{ABSBaseURL: "http://abs", ReadOnly: false}, client)
	_, output, err := server.CreateBackup(context.Background(), nil, EmptyInput{})
	if err != nil {
		t.Fatalf("CreateBackup failed: %v", err)
	}
	if !output.Triggered || output.Backup.ID != "backup-created" {
		t.Fatalf("unexpected create backup output: %#v", output)
	}
	if !client.createBackupCalled {
		t.Fatal("expected CreateBackup to call ABS client")
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

func TestSearchEbooks(t *testing.T) {
	t.Parallel()

	server := newTestServer()
	_, output, err := server.SearchEbooks(context.Background(), nil, SearchEbooksInput{
		LibraryID: "lib-books",
		Query:     "alice.epub",
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("SearchEbooks failed: %v", err)
	}
	if output.CheckedCount != 3 {
		t.Fatalf("CheckedCount = %d, want 3", output.CheckedCount)
	}
	if output.MatchedCount != 1 || output.ReturnedCount != 1 {
		t.Fatalf("Matched/Returned = %d/%d, want 1/1", output.MatchedCount, output.ReturnedCount)
	}
	if output.Items[0].ID != "book-1" {
		t.Fatalf("first item ID = %q, want book-1", output.Items[0].ID)
	}
}

func TestSearchEbooksRecognizesEbookFileExtensions(t *testing.T) {
	t.Parallel()

	client := newFakeABSClient()
	client.items["lib-books"] = append(client.items["lib-books"], abs.LibraryItem{
		ID:        "book-extension",
		LibraryID: "lib-books",
		Path:      "/books/fixture-only",
		MediaType: "book",
		Media:     abs.Media{Metadata: abs.Metadata{Title: "Fixture Only", AuthorName: "Test Author"}},
		LibraryFiles: []abs.LibraryFile{
			{
				Metadata: abs.FileMetadata{
					Filename: "fixture-only.epub",
					Path:     "/books/fixture-only/fixture-only.epub",
					RelPath:  "fixture-only/fixture-only.epub",
				},
			},
		},
	})
	server := New(config.Config{ABSBaseURL: "http://abs", ReadOnly: true}, client)

	_, output, err := server.SearchEbooks(context.Background(), nil, SearchEbooksInput{
		LibraryID: "lib-books",
		Query:     "fixture-only.epub",
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("SearchEbooks failed: %v", err)
	}
	if output.MatchedCount != 1 || output.Items[0].ID != "book-extension" {
		t.Fatalf("unexpected extension-backed ebook search output: %#v", output)
	}
}

func TestSearchEbooksRecognizesEbookLibraryItemsWithoutFileDetails(t *testing.T) {
	t.Parallel()

	client := newFakeABSClient()
	client.items["lib-books"] = []abs.LibraryItem{
		{
			ID:        "book-fixture",
			LibraryID: "lib-books",
			Path:      "/books/to-sort/austen",
			RelPath:   "to-sort/austen",
			MediaType: "book",
			Media:     abs.Media{Metadata: abs.Metadata{Title: "Austen Fixture", AuthorName: "Jane Austen"}},
		},
	}
	server := New(config.Config{ABSBaseURL: "http://abs", ReadOnly: true}, client)

	_, output, err := server.SearchEbooks(context.Background(), nil, SearchEbooksInput{
		LibraryID: "lib-books",
		Query:     "austen",
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("SearchEbooks failed: %v", err)
	}
	if output.MatchedCount != 1 || output.Items[0].ID != "book-fixture" {
		t.Fatalf("unexpected ebook-library fallback search output: %#v", output)
	}
}

func TestSearchEbooksRejectsBadInput(t *testing.T) {
	t.Parallel()

	server := newTestServer()
	if _, _, err := server.SearchEbooks(context.Background(), nil, SearchEbooksInput{}); err == nil {
		t.Fatal("expected missing libraryId error")
	}
	if _, _, err := server.SearchEbooks(context.Background(), nil, SearchEbooksInput{
		LibraryID: "lib-books",
		Limit:     -1,
	}); err == nil {
		t.Fatal("expected negative limit error")
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

func TestListLibraryAuthors(t *testing.T) {
	t.Parallel()

	client := newFakeABSClient()
	server := New(config.Config{ABSBaseURL: "http://abs", ReadOnly: true}, client)
	_, output, err := server.ListLibraryAuthors(context.Background(), nil, CatalogListInput{
		LibraryID: "lib-audio",
		Limit:     1000,
		Offset:    100,
		Sort:      "name",
		Desc:      true,
		Filter:    "authors.name.Lewis",
		Include:   []string{"Items", " series "},
		Minified:  true,
	})
	if err != nil {
		t.Fatalf("ListLibraryAuthors failed: %v", err)
	}
	if output.LibraryID != "lib-audio" || output.Limit != 100 || output.Offset != 100 || output.Page != 1 {
		t.Fatalf("unexpected authors output: %#v", output)
	}
	if client.lastCatalogListOptions.Limit != 100 || client.lastCatalogListOptions.Page != 1 {
		t.Fatalf("unexpected catalog options: %#v", client.lastCatalogListOptions)
	}
	if strings.Join(client.lastCatalogListOptions.Include, ",") != "items,series" {
		t.Fatalf("Include = %#v, want items,series", client.lastCatalogListOptions.Include)
	}
	if output.Data == nil {
		t.Fatal("expected author list data")
	}
}

func TestGetAuthor(t *testing.T) {
	t.Parallel()

	client := newFakeABSClient()
	server := New(config.Config{ABSBaseURL: "http://abs", ReadOnly: true}, client)
	_, output, err := server.GetAuthor(context.Background(), nil, EntityInput{
		ID:      "author-1",
		Include: []string{"items", "series"},
	})
	if err != nil {
		t.Fatalf("GetAuthor failed: %v", err)
	}
	if output.ID != "author-1" || output.Data == nil {
		t.Fatalf("unexpected author output: %#v", output)
	}
	if client.lastInclude[0] != "items" || client.lastInclude[1] != "series" {
		t.Fatalf("lastInclude = %#v, want items,series", client.lastInclude)
	}
}

func TestListLibrarySeries(t *testing.T) {
	t.Parallel()

	client := newFakeABSClient()
	server := New(config.Config{ABSBaseURL: "http://abs", ReadOnly: true}, client)
	_, output, err := server.ListLibrarySeries(context.Background(), nil, CatalogListInput{
		LibraryID: "lib-audio",
		Limit:     25,
		Offset:    50,
		Sort:      "name",
		Desc:      true,
		Filter:    "series.name.Alice",
		Include:   []string{"progress", "rssfeed"},
		Minified:  true,
	})
	if err != nil {
		t.Fatalf("ListLibrarySeries failed: %v", err)
	}
	if output.LibraryID != "lib-audio" || output.Limit != 25 || output.Offset != 50 || output.Page != 2 {
		t.Fatalf("unexpected series output: %#v", output)
	}
	if client.lastCatalogListOptions.Page != 2 || client.lastCatalogListOptions.Filter != "series.name.Alice" {
		t.Fatalf("unexpected catalog options: %#v", client.lastCatalogListOptions)
	}
	if output.Data == nil {
		t.Fatal("expected series list data")
	}
}

func TestGetSeries(t *testing.T) {
	t.Parallel()

	client := newFakeABSClient()
	server := New(config.Config{ABSBaseURL: "http://abs", ReadOnly: true}, client)
	_, output, err := server.GetSeries(context.Background(), nil, EntityInput{
		ID:      "series-1",
		Include: []string{"progress", "rssfeed"},
	})
	if err != nil {
		t.Fatalf("GetSeries failed: %v", err)
	}
	if output.ID != "series-1" || output.Data == nil {
		t.Fatalf("unexpected series output: %#v", output)
	}
}

func TestListCollections(t *testing.T) {
	t.Parallel()

	server := newTestServer()
	_, output, err := server.ListCollections(context.Background(), nil, EmptyInput{})
	if err != nil {
		t.Fatalf("ListCollections failed: %v", err)
	}
	if output.Data == nil {
		t.Fatal("expected collections data")
	}
}

func TestGetCollection(t *testing.T) {
	t.Parallel()

	client := newFakeABSClient()
	server := New(config.Config{ABSBaseURL: "http://abs", ReadOnly: true}, client)
	_, output, err := server.GetCollection(context.Background(), nil, EntityInput{
		ID:      "col-1",
		Include: []string{"items"},
	})
	if err != nil {
		t.Fatalf("GetCollection failed: %v", err)
	}
	if output.ID != "col-1" || output.Data == nil {
		t.Fatalf("unexpected collection output: %#v", output)
	}
}

func TestGetItemsInProgress(t *testing.T) {
	t.Parallel()

	client := newFakeABSClient()
	server := New(config.Config{ABSBaseURL: "http://abs", ReadOnly: true}, client)
	_, output, err := server.GetItemsInProgress(context.Background(), nil, ItemsInProgressInput{Limit: 1000})
	if err != nil {
		t.Fatalf("GetItemsInProgress failed: %v", err)
	}
	if output.Limit != 100 {
		t.Fatalf("Limit = %d, want 100", output.Limit)
	}
	if output.User.Scope != "currentUser" {
		t.Fatalf("User.Scope = %q, want currentUser", output.User.Scope)
	}
	if client.itemsInProgressLimit != 100 {
		t.Fatalf("itemsInProgressLimit = %d, want 100", client.itemsInProgressLimit)
	}
	if output.Data == nil {
		t.Fatal("expected items-in-progress data")
	}
}

func TestGetListeningStats(t *testing.T) {
	t.Parallel()

	client := newFakeABSClient()
	server := New(config.Config{ABSBaseURL: "http://abs", ReadOnly: true}, client)
	_, output, err := server.GetListeningStats(context.Background(), nil, EmptyInput{})
	if err != nil {
		t.Fatalf("GetListeningStats failed: %v", err)
	}
	if output.User.Scope != "currentUser" {
		t.Fatalf("User.Scope = %q, want currentUser", output.User.Scope)
	}
	if !client.getListeningStatsCalled {
		t.Fatal("expected GetListeningStats client call")
	}
	if output.Data == nil {
		t.Fatal("expected listening stats data")
	}
}

func TestListListeningSessions(t *testing.T) {
	t.Parallel()

	client := newFakeABSClient()
	server := New(config.Config{ABSBaseURL: "http://abs", ReadOnly: true}, client)
	_, output, err := server.ListListeningSessions(context.Background(), nil, ListeningSessionsInput{
		Limit: 1000,
		Page:  2,
	})
	if err != nil {
		t.Fatalf("ListListeningSessions failed: %v", err)
	}
	if output.User.Scope != "currentUser" {
		t.Fatalf("User.Scope = %q, want currentUser", output.User.Scope)
	}
	if output.Limit != 100 || output.Page != 2 {
		t.Fatalf("unexpected pagination: %#v", output)
	}
	if client.listListeningSessionsLimit != 100 || client.listListeningSessionsPage != 2 {
		t.Fatalf("unexpected client pagination: %d/%d", client.listListeningSessionsLimit, client.listListeningSessionsPage)
	}
	if output.Data == nil {
		t.Fatal("expected listening sessions data")
	}
}

func TestListListeningSessionsDefaults(t *testing.T) {
	t.Parallel()

	client := newFakeABSClient()
	server := New(config.Config{ABSBaseURL: "http://abs", ReadOnly: true}, client)
	_, output, err := server.ListListeningSessions(context.Background(), nil, ListeningSessionsInput{})
	if err != nil {
		t.Fatalf("ListListeningSessions failed: %v", err)
	}
	if output.Limit != 25 || output.Page != 0 {
		t.Fatalf("Limit/Page = %d/%d, want 25/0", output.Limit, output.Page)
	}
	if client.listListeningSessionsLimit != 25 || client.listListeningSessionsPage != 0 {
		t.Fatalf("client Limit/Page = %d/%d, want 25/0", client.listListeningSessionsLimit, client.listListeningSessionsPage)
	}
}

func TestGetItemProgress(t *testing.T) {
	t.Parallel()

	server := newTestServer()
	_, output, err := server.GetItemProgress(context.Background(), nil, ItemProgressInput{
		ItemID:    "item-1",
		EpisodeID: "episode-1",
	})
	if err != nil {
		t.Fatalf("GetItemProgress failed: %v", err)
	}
	if output.ItemID != "item-1" || output.EpisodeID != "episode-1" {
		t.Fatalf("unexpected progress output IDs: %#v", output)
	}
	if output.User.Scope != "currentUser" || output.User.UserID != "user-1" {
		t.Fatalf("unexpected progress user scope: %#v", output.User)
	}
	if output.Progress.ID != "progress-1" || output.Progress.CurrentTime != 42 {
		t.Fatalf("unexpected progress: %#v", output.Progress)
	}
}

func TestListBookmarks(t *testing.T) {
	t.Parallel()

	server := newTestServer()
	_, output, err := server.ListBookmarks(context.Background(), nil, BookmarksInput{ItemID: "item-1"})
	if err != nil {
		t.Fatalf("ListBookmarks failed: %v", err)
	}
	if output.Count != 1 {
		t.Fatalf("Count = %d, want 1", output.Count)
	}
	if output.Bookmarks[0].LibraryItemID != "item-1" || output.Bookmarks[0].Title != "Start" {
		t.Fatalf("unexpected bookmark: %#v", output.Bookmarks[0])
	}
}

func TestProgressAndBookmarkMutations(t *testing.T) {
	t.Parallel()

	client := newFakeABSClient()
	server := New(config.Config{ABSBaseURL: "http://abs", ReadOnly: false}, client)
	currentTime := 42.5
	progress := 0.5

	_, progressOutput, err := server.UpdateItemProgress(context.Background(), nil, UpdateItemProgressInput{
		ItemID:      "item-1",
		EpisodeID:   "episode-1",
		CurrentTime: &currentTime,
		Progress:    &progress,
	})
	if err != nil {
		t.Fatalf("UpdateItemProgress failed: %v", err)
	}
	if !progressOutput.Triggered || progressOutput.ItemID != "item-1" || progressOutput.EpisodeID != "episode-1" {
		t.Fatalf("unexpected progress output: %#v", progressOutput)
	}
	if client.updateItemProgressID != "item-1" || client.updateItemProgressEpisodeID != "episode-1" {
		t.Fatalf("unexpected progress call: %q/%q", client.updateItemProgressID, client.updateItemProgressEpisodeID)
	}
	if client.updateItemProgressPayload.CurrentTime == nil || *client.updateItemProgressPayload.CurrentTime != currentTime {
		t.Fatalf("unexpected progress payload: %#v", client.updateItemProgressPayload)
	}

	_, created, err := server.CreateBookmark(context.Background(), nil, BookmarkMutationInput{
		ItemID: "item-1",
		Time:   12.5,
		Title:  "Start",
	})
	if err != nil {
		t.Fatalf("CreateBookmark failed: %v", err)
	}
	if !created.Triggered || created.Bookmark.Title != "Start" {
		t.Fatalf("unexpected create bookmark output: %#v", created)
	}

	_, updated, err := server.UpdateBookmark(context.Background(), nil, BookmarkMutationInput{
		ItemID: "item-1",
		Time:   12.5,
		Title:  "Updated",
	})
	if err != nil {
		t.Fatalf("UpdateBookmark failed: %v", err)
	}
	if !updated.Triggered || updated.Bookmark.Title != "Updated" {
		t.Fatalf("unexpected update bookmark output: %#v", updated)
	}
	if client.updateBookmarkPayload.Title != "Updated" {
		t.Fatalf("unexpected bookmark payload: %#v", client.updateBookmarkPayload)
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
	if _, _, err := server.ListLibraryAuthors(context.Background(), nil, CatalogListInput{}); err == nil {
		t.Fatal("expected authors missing libraryId error")
	}
	if _, _, err := server.ListLibrarySeries(context.Background(), nil, CatalogListInput{LibraryID: "lib-audio", Limit: 2, Offset: 1}); err == nil {
		t.Fatal("expected series unaligned offset error")
	}
	if _, _, err := server.GetAuthor(context.Background(), nil, EntityInput{}); err == nil {
		t.Fatal("expected missing author ID error")
	}
	if _, _, err := server.GetSeries(context.Background(), nil, EntityInput{}); err == nil {
		t.Fatal("expected missing series ID error")
	}
	if _, _, err := server.GetCollection(context.Background(), nil, EntityInput{}); err == nil {
		t.Fatal("expected missing collection ID error")
	}
	if _, _, err := server.GetItemsInProgress(context.Background(), nil, ItemsInProgressInput{Limit: -1}); err == nil {
		t.Fatal("expected negative progress limit error")
	}
	if _, _, err := server.GetItemProgress(context.Background(), nil, ItemProgressInput{}); err == nil {
		t.Fatal("expected missing item progress ID error")
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

func TestSendEbookToDeviceBlockedInReadOnlyMode(t *testing.T) {
	t.Parallel()

	server := newTestServer()
	if _, _, err := server.SendEbookToDevice(context.Background(), nil, SendEbookToDeviceInput{
		ItemID:     "book-1",
		DeviceName: "Kindle",
	}); err == nil || !strings.Contains(err.Error(), "--read-only=false") {
		t.Fatalf("expected actionable read-only error, got %v", err)
	}
}

func TestSendEbookToDevice(t *testing.T) {
	t.Parallel()

	client := newFakeABSClient()
	server := New(config.Config{ABSBaseURL: "http://abs", ReadOnly: false}, client)
	_, output, err := server.SendEbookToDevice(context.Background(), nil, SendEbookToDeviceInput{
		ItemID:     " book-1 ",
		DeviceName: " Kindle ",
	})
	if err != nil {
		t.Fatalf("SendEbookToDevice failed: %v", err)
	}
	if !output.Triggered {
		t.Fatal("Triggered = false, want true")
	}
	if output.ItemID != "book-1" {
		t.Fatalf("ItemID = %q, want book-1", output.ItemID)
	}
	if output.DeviceName != "Kindle" {
		t.Fatalf("DeviceName = %q, want Kindle", output.DeviceName)
	}
	if client.sendEbookPayload.LibraryItemID != "book-1" {
		t.Fatalf("LibraryItemID = %q, want book-1", client.sendEbookPayload.LibraryItemID)
	}
	if client.sendEbookPayload.DeviceName != "Kindle" {
		t.Fatalf("DeviceName payload = %q, want Kindle", client.sendEbookPayload.DeviceName)
	}
}

func TestSendEbookToDeviceRequiresInput(t *testing.T) {
	t.Parallel()

	server := New(config.Config{ABSBaseURL: "http://abs", ReadOnly: false}, newFakeABSClient())
	for _, input := range []SendEbookToDeviceInput{
		{},
		{ItemID: "book-1"},
		{DeviceName: "Kindle"},
	} {
		if _, _, err := server.SendEbookToDevice(context.Background(), nil, input); err == nil {
			t.Fatalf("expected validation error for %#v", input)
		}
	}
}

func TestPreviewEbookDeviceSendReady(t *testing.T) {
	t.Parallel()

	client := newFakeABSClient()
	server := New(config.Config{ABSBaseURL: "http://abs", ReadOnly: true}, client)
	_, output, err := server.PreviewEbookDeviceSend(context.Background(), nil, PreviewEbookDeviceSendInput{
		LibraryID:     "lib-books",
		Query:         "alice.epub",
		DeviceName:    " Kindle ",
		MaxCandidates: 10,
	})
	if err != nil {
		t.Fatalf("PreviewEbookDeviceSend failed: %v", err)
	}
	if !output.Ready {
		t.Fatalf("Ready = false, want true: %#v", output)
	}
	if output.Confirmation != "send ebook book-1 to Kindle" {
		t.Fatalf("Confirmation = %q, want exact send text", output.Confirmation)
	}
	if output.NextTool != "abs_send_ebook_by_query" {
		t.Fatalf("NextTool = %q, want abs_send_ebook_by_query", output.NextTool)
	}
	if output.CandidateCount != 1 || len(output.Candidates) != 1 || output.Candidates[0].ID != "book-1" {
		t.Fatalf("unexpected candidates: %#v", output.Candidates)
	}
	if output.DeviceCount != 2 || len(output.Devices) != 2 || output.Devices[0].Email != "" {
		t.Fatalf("unexpected sanitized devices: %#v", output.Devices)
	}
	if client.sendEbookPayload.LibraryItemID != "" {
		t.Fatalf("preview sent ebook unexpectedly: %#v", client.sendEbookPayload)
	}
}

func TestPreviewEbookDeviceSendRejectsAmbiguousEbooks(t *testing.T) {
	t.Parallel()

	server := newTestServer()
	_, output, err := server.PreviewEbookDeviceSend(context.Background(), nil, PreviewEbookDeviceSendInput{
		LibraryID:  "lib-books",
		Query:      "alice",
		DeviceName: "Kindle",
	})
	if err != nil {
		t.Fatalf("PreviewEbookDeviceSend failed: %v", err)
	}
	if output.Ready {
		t.Fatalf("Ready = true for ambiguous query: %#v", output)
	}
	if output.CandidateCount != 2 || output.NextTool != "abs_search_ebooks" || output.Confirmation != "" {
		t.Fatalf("unexpected ambiguous preview output: %#v", output)
	}
}

func TestPreviewEbookDeviceSendRejectsMissingDevice(t *testing.T) {
	t.Parallel()

	server := newTestServer()
	_, output, err := server.PreviewEbookDeviceSend(context.Background(), nil, PreviewEbookDeviceSendInput{
		LibraryID:  "lib-books",
		Query:      "alice.epub",
		DeviceName: "Nook",
	})
	if err != nil {
		t.Fatalf("PreviewEbookDeviceSend failed: %v", err)
	}
	if output.Ready {
		t.Fatalf("Ready = true for missing device: %#v", output)
	}
	if output.CandidateCount != 1 || output.NextTool != "abs_list_ereader_devices" || output.Confirmation != "" {
		t.Fatalf("unexpected missing-device preview output: %#v", output)
	}
}

func TestPreviewEbookDeviceSendWorksInReadOnlyMode(t *testing.T) {
	t.Parallel()

	server := newTestServer()
	_, output, err := server.PreviewEbookDeviceSend(context.Background(), nil, PreviewEbookDeviceSendInput{
		LibraryID:  "lib-books",
		Query:      "alice.epub",
		DeviceName: "Kindle",
	})
	if err != nil {
		t.Fatalf("PreviewEbookDeviceSend failed in read-only mode: %v", err)
	}
	if !output.Ready || output.Confirmation != "send ebook book-1 to Kindle" {
		t.Fatalf("unexpected read-only preview output: %#v", output)
	}
}

func TestSendEbookByQuery(t *testing.T) {
	t.Parallel()

	client := newFakeABSClient()
	server := New(config.Config{ABSBaseURL: "http://abs", ReadOnly: false}, client)
	_, output, err := server.SendEbookByQuery(context.Background(), nil, SendEbookByQueryInput{
		LibraryID:     "lib-books",
		Query:         "alice.epub",
		DeviceName:    " Kindle ",
		Confirmation:  "send ebook book-1 to Kindle",
		MaxCandidates: 10,
	})
	if err != nil {
		t.Fatalf("SendEbookByQuery failed: %v", err)
	}
	if !output.Triggered {
		t.Fatal("Triggered = false, want true")
	}
	if output.Item.ID != "book-1" {
		t.Fatalf("Item.ID = %q, want book-1", output.Item.ID)
	}
	if output.DeviceName != "Kindle" {
		t.Fatalf("DeviceName = %q, want Kindle", output.DeviceName)
	}
	if client.sendEbookPayload.LibraryItemID != "book-1" || client.sendEbookPayload.DeviceName != "Kindle" {
		t.Fatalf("send payload = %#v, want book-1/Kindle", client.sendEbookPayload)
	}
}

func TestSendEbookByQueryRequiresConfirmationForResolvedItem(t *testing.T) {
	t.Parallel()

	client := newFakeABSClient()
	server := New(config.Config{ABSBaseURL: "http://abs", ReadOnly: false}, client)
	_, _, err := server.SendEbookByQuery(context.Background(), nil, SendEbookByQueryInput{
		LibraryID:    "lib-books",
		Query:        "alice.epub",
		DeviceName:   "Kindle",
		Confirmation: "send it",
	})
	if err == nil || !strings.Contains(err.Error(), "send ebook book-1 to Kindle") {
		t.Fatalf("expected exact confirmation error, got %v", err)
	}
	if client.sendEbookPayload.LibraryItemID != "" {
		t.Fatalf("sent despite bad confirmation: %#v", client.sendEbookPayload)
	}
}

func TestSendEbookByQueryRejectsAmbiguousMatches(t *testing.T) {
	t.Parallel()

	client := newFakeABSClient()
	server := New(config.Config{ABSBaseURL: "http://abs", ReadOnly: false}, client)
	if _, _, err := server.SendEbookByQuery(context.Background(), nil, SendEbookByQueryInput{
		LibraryID:    "lib-books",
		Query:        "alice",
		DeviceName:   "Kindle",
		Confirmation: "send ebook book-1 to Kindle",
	}); err == nil || !strings.Contains(err.Error(), "matched 2 ebooks") {
		t.Fatalf("expected ambiguous match error, got %v", err)
	}
	if client.sendEbookPayload.LibraryItemID != "" {
		t.Fatalf("sent despite ambiguous match: %#v", client.sendEbookPayload)
	}
}

func TestSendEbookByQueryBlockedInReadOnlyMode(t *testing.T) {
	t.Parallel()

	server := newTestServer()
	if _, _, err := server.SendEbookByQuery(context.Background(), nil, SendEbookByQueryInput{
		LibraryID:    "lib-books",
		Query:        "alice",
		DeviceName:   "Kindle",
		Confirmation: "send ebook book-1 to Kindle",
	}); err == nil || !strings.Contains(err.Error(), "--read-only=false") {
		t.Fatalf("expected actionable read-only error, got %v", err)
	}
}

func TestUpdateItemCover(t *testing.T) {
	t.Parallel()

	client := newFakeABSClient()
	server := New(config.Config{ABSBaseURL: "http://abs", ReadOnly: false}, client)

	_, output, err := server.UpdateItemCover(context.Background(), nil, UpdateItemCoverInput{
		ItemID: "item-1",
		Cover:  "/covers/alice.jpg",
	})
	if err != nil {
		t.Fatalf("UpdateItemCover failed: %v", err)
	}
	if !output.Triggered || output.ItemID != "item-1" {
		t.Fatalf("unexpected cover update output: %#v", output)
	}
	if client.updateItemCoverID != "item-1" {
		t.Fatalf("updateItemCoverID = %q, want item-1", client.updateItemCoverID)
	}
	if client.updateItemCoverPath != "/covers/alice.jpg" {
		t.Fatalf("updateItemCoverPath = %q, want /covers/alice.jpg", client.updateItemCoverPath)
	}
}

func TestRemoveItemCover(t *testing.T) {
	t.Parallel()

	client := newFakeABSClient()
	server := New(config.Config{ABSBaseURL: "http://abs", ReadOnly: false}, client)

	_, output, err := server.RemoveItemCover(context.Background(), nil, ConfirmedItemInput{
		ItemID:       "item-1",
		Confirmation: "remove cover from item-1",
	})
	if err != nil {
		t.Fatalf("RemoveItemCover failed: %v", err)
	}
	if !output.Triggered || output.ItemID != "item-1" {
		t.Fatalf("unexpected remove cover output: %#v", output)
	}
	if client.removeItemCoverID != "item-1" {
		t.Fatalf("removeItemCoverID = %q, want item-1", client.removeItemCoverID)
	}
}

func TestUpdateItemChapters(t *testing.T) {
	t.Parallel()

	client := newFakeABSClient()
	server := New(config.Config{ABSBaseURL: "http://abs", ReadOnly: false}, client)

	_, output, err := server.UpdateItemChapters(context.Background(), nil, UpdateItemChaptersInput{
		ItemID: "item-1",
		Chapters: []ChapterInput{
			{Title: "Intro", Start: 0, End: 12.5},
			{Title: "Chapter 1", Start: 12.5, End: 60},
		},
		ExpectedChapterCount: 2,
	})
	if err != nil {
		t.Fatalf("UpdateItemChapters failed: %v", err)
	}
	if !output.Triggered || output.ItemID != "item-1" {
		t.Fatalf("unexpected chapter update output: %#v", output)
	}
	if client.updateItemChaptersID != "item-1" {
		t.Fatalf("updateItemChaptersID = %q, want item-1", client.updateItemChaptersID)
	}
	if len(client.updateItemChapters) != 2 {
		t.Fatalf("chapter count = %d, want 2", len(client.updateItemChapters))
	}
	if client.updateItemChapters[0].Title != "Intro" {
		t.Fatalf("first chapter title = %q, want Intro", client.updateItemChapters[0].Title)
	}
}

func TestUpdateItemMetadata(t *testing.T) {
	t.Parallel()

	client := newFakeABSClient()
	server := New(config.Config{ABSBaseURL: "http://abs", ReadOnly: false}, client)
	title := "Alice Updated"
	description := "Clean description"
	explicit := true

	_, output, err := server.UpdateItemMetadata(context.Background(), nil, UpdateItemMetadataInput{
		ItemID:      "item-1",
		Title:       &title,
		Description: &description,
		Explicit:    &explicit,
		Genres:      []string{"fiction", "classic"},
		Authors:     []string{"Lewis Carroll"},
		Series:      []SeriesMetadataInput{{Name: "Alice Books", Sequence: "1"}},
		Tags:        []string{"favorite"},
	})
	if err != nil {
		t.Fatalf("UpdateItemMetadata failed: %v", err)
	}
	if !output.Triggered || output.ItemID != "item-1" || output.Data == nil {
		t.Fatalf("unexpected metadata update output: %#v", output)
	}
	if client.updateItemMetadataID != "item-1" {
		t.Fatalf("updateItemMetadataID = %q, want item-1", client.updateItemMetadataID)
	}
	payload := client.updateItemMetadataPayload
	if payload.Metadata == nil || payload.Metadata.Title == nil || *payload.Metadata.Title != title {
		t.Fatalf("unexpected title payload: %#v", payload)
	}
	if payload.Metadata.Description == nil || *payload.Metadata.Description != description {
		t.Fatalf("unexpected description payload: %#v", payload)
	}
	if payload.Metadata.Explicit == nil || !*payload.Metadata.Explicit {
		t.Fatalf("unexpected explicit payload: %#v", payload)
	}
	if payload.Metadata.Authors == nil || len(*payload.Metadata.Authors) != 1 || (*payload.Metadata.Authors)[0].Name != "Lewis Carroll" {
		t.Fatalf("unexpected authors payload: %#v", payload)
	}
	if payload.Metadata.Series == nil || len(*payload.Metadata.Series) != 1 || (*payload.Metadata.Series)[0].Name != "Alice Books" {
		t.Fatalf("unexpected series payload: %#v", payload)
	}
	if payload.Tags == nil || len(*payload.Tags) != 1 || (*payload.Tags)[0] != "favorite" {
		t.Fatalf("unexpected tags payload: %#v", payload)
	}
}

func TestCollectionMutations(t *testing.T) {
	t.Parallel()

	client := newFakeABSClient()
	server := New(config.Config{ABSBaseURL: "http://abs", ReadOnly: false}, client)

	_, created, err := server.CreateCollection(context.Background(), nil, CollectionInput{
		LibraryID:   "lib-audio",
		Name:        "Favorites",
		Description: "Good books",
		ItemIDs:     []string{"item-1"},
	})
	if err != nil {
		t.Fatalf("CreateCollection failed: %v", err)
	}
	if !created.Triggered || created.ID != "col-1" || created.Data == nil {
		t.Fatalf("unexpected create collection output: %#v", created)
	}
	if client.createCollectionPayload.Name != "Favorites" || client.createCollectionPayload.LibraryID != "lib-audio" {
		t.Fatalf("unexpected create collection payload: %#v", client.createCollectionPayload)
	}

	_, updated, err := server.UpdateCollection(context.Background(), nil, CollectionInput{
		CollectionID: "col-1",
		Name:         "Updated",
		Description:  "New description",
	})
	if err != nil {
		t.Fatalf("UpdateCollection failed: %v", err)
	}
	if !updated.Triggered || updated.ID != "col-1" {
		t.Fatalf("unexpected update collection output: %#v", updated)
	}
	if client.updateCollectionID != "col-1" || client.updateCollectionPayload.Name != "Updated" {
		t.Fatalf("unexpected update collection call: %q %#v", client.updateCollectionID, client.updateCollectionPayload)
	}

	_, added, err := server.AddCollectionItem(context.Background(), nil, CollectionItemInput{
		CollectionID: "col-1",
		ItemID:       "item-1",
	})
	if err != nil {
		t.Fatalf("AddCollectionItem failed: %v", err)
	}
	if !added.Triggered || added.ID != "col-1" {
		t.Fatalf("unexpected add collection output: %#v", added)
	}
	if client.addCollectionID != "col-1" || client.addCollectionItemID != "item-1" {
		t.Fatalf("unexpected add collection call: %q/%q", client.addCollectionID, client.addCollectionItemID)
	}

	_, deleted, err := server.DeleteCollection(context.Background(), nil, ConfirmedCollectionInput{
		CollectionID: "col-1",
		Confirmation: "delete collection col-1",
	})
	if err != nil {
		t.Fatalf("DeleteCollection failed: %v", err)
	}
	if !deleted.Triggered || deleted.ID != "col-1" {
		t.Fatalf("unexpected delete collection output: %#v", deleted)
	}
	if client.deleteCollectionID != "col-1" {
		t.Fatalf("unexpected delete collection call: %q", client.deleteCollectionID)
	}

	_, removed, err := server.RemoveCollectionItem(context.Background(), nil, CollectionItemInput{
		CollectionID: "col-1",
		ItemID:       "item-1",
		Confirmation: "remove item item-1 from collection col-1",
	})
	if err != nil {
		t.Fatalf("RemoveCollectionItem failed: %v", err)
	}
	if !removed.Triggered || removed.ID != "col-1" {
		t.Fatalf("unexpected remove collection item output: %#v", removed)
	}
	if client.removeCollectionID != "col-1" || client.removeCollectionItemID != "item-1" {
		t.Fatalf("unexpected remove collection call: %q/%q", client.removeCollectionID, client.removeCollectionItemID)
	}
}

func TestPlaylistMutations(t *testing.T) {
	t.Parallel()

	client := newFakeABSClient()
	server := New(config.Config{ABSBaseURL: "http://abs", ReadOnly: false}, client)

	_, created, err := server.CreatePlaylist(context.Background(), nil, PlaylistInput{
		LibraryID:   "lib-audio",
		Name:        "Queue",
		Description: "Listen next",
		Items:       []PlaylistCreateItemInput{{ItemID: "item-1", EpisodeID: "episode-1"}},
	})
	if err != nil {
		t.Fatalf("CreatePlaylist failed: %v", err)
	}
	if !created.Triggered || created.ID != "pl-1" || created.Data == nil {
		t.Fatalf("unexpected create playlist output: %#v", created)
	}
	if client.createPlaylistPayload.Name != "Queue" || client.createPlaylistPayload.Items[0].LibraryItemID != "item-1" {
		t.Fatalf("unexpected create playlist payload: %#v", client.createPlaylistPayload)
	}

	_, updated, err := server.UpdatePlaylist(context.Background(), nil, PlaylistInput{
		PlaylistID:  "pl-1",
		Name:        "Updated",
		Description: "New description",
	})
	if err != nil {
		t.Fatalf("UpdatePlaylist failed: %v", err)
	}
	if !updated.Triggered || updated.ID != "pl-1" {
		t.Fatalf("unexpected update playlist output: %#v", updated)
	}
	if client.updatePlaylistID != "pl-1" || client.updatePlaylistPayload.Name != "Updated" {
		t.Fatalf("unexpected update playlist call: %q %#v", client.updatePlaylistID, client.updatePlaylistPayload)
	}

	_, added, err := server.AddPlaylistItem(context.Background(), nil, PlaylistItemInput{
		PlaylistID: "pl-1",
		ItemID:     "item-1",
		EpisodeID:  "episode-1",
	})
	if err != nil {
		t.Fatalf("AddPlaylistItem failed: %v", err)
	}
	if !added.Triggered || added.ID != "pl-1" {
		t.Fatalf("unexpected add playlist output: %#v", added)
	}
	if client.addPlaylistID != "pl-1" || client.addPlaylistItem.LibraryItemID != "item-1" || client.addPlaylistItem.EpisodeID != "episode-1" {
		t.Fatalf("unexpected add playlist call: %q %#v", client.addPlaylistID, client.addPlaylistItem)
	}

	_, deleted, err := server.DeletePlaylist(context.Background(), nil, ConfirmedPlaylistInput{
		PlaylistID:   "pl-1",
		Confirmation: "delete playlist pl-1",
	})
	if err != nil {
		t.Fatalf("DeletePlaylist failed: %v", err)
	}
	if !deleted.Triggered || deleted.ID != "pl-1" {
		t.Fatalf("unexpected delete playlist output: %#v", deleted)
	}
	if client.deletePlaylistID != "pl-1" {
		t.Fatalf("unexpected delete playlist call: %q", client.deletePlaylistID)
	}

	_, removed, err := server.RemovePlaylistItem(context.Background(), nil, PlaylistItemInput{
		PlaylistID:   "pl-1",
		ItemID:       "item-1",
		EpisodeID:    "episode-1",
		Confirmation: "remove item item-1 from playlist pl-1",
	})
	if err != nil {
		t.Fatalf("RemovePlaylistItem failed: %v", err)
	}
	if !removed.Triggered || removed.ID != "pl-1" {
		t.Fatalf("unexpected remove playlist item output: %#v", removed)
	}
	if client.removePlaylistID != "pl-1" || client.removePlaylistItem.LibraryItemID != "item-1" || client.removePlaylistItem.EpisodeID != "episode-1" {
		t.Fatalf("unexpected remove playlist call: %q %#v", client.removePlaylistID, client.removePlaylistItem)
	}
}

func TestPlannedMutatingToolsBlockedInReadOnlyMode(t *testing.T) {
	t.Parallel()

	server := newTestServer()
	metadataTitle := "Alice Updated"
	tests := map[string]func() error{
		"abs_update_item_metadata": func() error {
			_, _, err := server.UpdateItemMetadata(context.Background(), nil, UpdateItemMetadataInput{ItemID: "item-1", Title: &metadataTitle})
			return err
		},
		"abs_update_item_progress": func() error {
			currentTime := 42.5
			_, _, err := server.UpdateItemProgress(context.Background(), nil, UpdateItemProgressInput{ItemID: "item-1", CurrentTime: &currentTime})
			return err
		},
		"abs_create_bookmark": func() error {
			_, _, err := server.CreateBookmark(context.Background(), nil, BookmarkMutationInput{ItemID: "item-1", Time: 12.5, Title: "Start"})
			return err
		},
		"abs_update_bookmark": func() error {
			_, _, err := server.UpdateBookmark(context.Background(), nil, BookmarkMutationInput{ItemID: "item-1", Time: 12.5, Title: "Start"})
			return err
		},
		"abs_create_backup": func() error {
			_, _, err := server.CreateBackup(context.Background(), nil, EmptyInput{})
			return err
		},
		"abs_update_item_cover": func() error {
			_, _, err := server.UpdateItemCover(context.Background(), nil, UpdateItemCoverInput{ItemID: "item-1", Cover: "/covers/alice.jpg"})
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
			_, _, err := server.UpdateItemChapters(context.Background(), nil, UpdateItemChaptersInput{
				ItemID:               "item-1",
				Chapters:             []ChapterInput{{Title: "Intro", Start: 0, End: 1}},
				ExpectedChapterCount: 1,
			})
			return err
		},
		"abs_update_item_tracks": func() error {
			_, _, err := server.UpdateItemTracks(context.Background(), nil, ItemPayloadInput{ItemID: "item-1"})
			return err
		},
		"abs_create_collection": func() error {
			_, _, err := server.CreateCollection(context.Background(), nil, CollectionInput{LibraryID: "lib-audio", Name: "Favorites", ItemIDs: []string{"item-1"}})
			return err
		},
		"abs_update_collection": func() error {
			_, _, err := server.UpdateCollection(context.Background(), nil, CollectionInput{CollectionID: "col-1", Name: "Updated"})
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
			_, _, err := server.CreatePlaylist(context.Background(), nil, PlaylistInput{LibraryID: "lib-audio", Name: "Queue"})
			return err
		},
		"abs_update_playlist": func() error {
			_, _, err := server.UpdatePlaylist(context.Background(), nil, PlaylistInput{PlaylistID: "pl-1", Name: "Updated"})
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
	if _, _, err := server.UpdateItemMetadata(context.Background(), nil, UpdateItemMetadataInput{}); err == nil || !strings.Contains(err.Error(), "itemId") {
		t.Fatalf("expected missing itemId error, got %v", err)
	}
	if _, _, err := server.UpdateItemMetadata(context.Background(), nil, UpdateItemMetadataInput{ItemID: "item-1"}); err == nil || !strings.Contains(err.Error(), "metadata field") {
		t.Fatalf("expected missing metadata field error, got %v", err)
	}
	if _, _, err := server.UpdateItemProgress(context.Background(), nil, UpdateItemProgressInput{}); err == nil || !strings.Contains(err.Error(), "itemId") {
		t.Fatalf("expected missing progress itemId error, got %v", err)
	}
	if _, _, err := server.UpdateItemProgress(context.Background(), nil, UpdateItemProgressInput{ItemID: "item-1"}); err == nil || !strings.Contains(err.Error(), "progress field") {
		t.Fatalf("expected missing progress field error, got %v", err)
	}
	badProgress := 1.5
	if _, _, err := server.UpdateItemProgress(context.Background(), nil, UpdateItemProgressInput{ItemID: "item-1", Progress: &badProgress}); err == nil || !strings.Contains(err.Error(), "progress") {
		t.Fatalf("expected bad progress error, got %v", err)
	}
	if _, _, err := server.CreateBookmark(context.Background(), nil, BookmarkMutationInput{ItemID: "item-1", Time: -1, Title: "Start"}); err == nil || !strings.Contains(err.Error(), "time") {
		t.Fatalf("expected bookmark time error, got %v", err)
	}
	if _, _, err := server.UpdateBookmark(context.Background(), nil, BookmarkMutationInput{ItemID: "item-1", Time: 1, Title: " "}); err == nil || !strings.Contains(err.Error(), "title") {
		t.Fatalf("expected bookmark title error, got %v", err)
	}
	if _, _, err := server.UpdateItemCover(context.Background(), nil, UpdateItemCoverInput{ItemID: "item-1"}); err == nil || !strings.Contains(err.Error(), "cover") {
		t.Fatalf("expected missing cover error, got %v", err)
	}
	if _, _, err := server.UpdateItemChapters(context.Background(), nil, UpdateItemChaptersInput{
		ItemID:               "item-1",
		Chapters:             []ChapterInput{{Title: "Intro", Start: 0, End: 1}},
		ExpectedChapterCount: 2,
	}); err == nil || !strings.Contains(err.Error(), "expectedChapterCount") {
		t.Fatalf("expected chapter count guard error, got %v", err)
	}
	if _, _, err := server.MatchItem(context.Background(), nil, MatchItemInput{}); err == nil || !strings.Contains(err.Error(), "itemId") {
		t.Fatalf("expected missing itemId error, got %v", err)
	}
	if _, _, err := server.RemoveItemCover(context.Background(), nil, ConfirmedItemInput{ItemID: "item-1", Confirmation: "yes"}); err == nil || !strings.Contains(err.Error(), "remove cover from item-1") {
		t.Fatalf("expected cover confirmation error, got %v", err)
	}
	if _, _, err := server.CreateCollection(context.Background(), nil, CollectionInput{Name: "Favorites"}); err == nil || !strings.Contains(err.Error(), "libraryId") {
		t.Fatalf("expected collection libraryId error, got %v", err)
	}
	if _, _, err := server.CreateCollection(context.Background(), nil, CollectionInput{LibraryID: "lib-audio", Name: "Favorites"}); err == nil || !strings.Contains(err.Error(), "itemIds") {
		t.Fatalf("expected collection itemIds error, got %v", err)
	}
	if _, _, err := server.UpdateCollection(context.Background(), nil, CollectionInput{CollectionID: "col-1"}); err == nil || !strings.Contains(err.Error(), "name or description") {
		t.Fatalf("expected collection update field error, got %v", err)
	}
	if _, _, err := server.AddCollectionItem(context.Background(), nil, CollectionItemInput{CollectionID: "col-1"}); err == nil || !strings.Contains(err.Error(), "itemId") {
		t.Fatalf("expected collection item error, got %v", err)
	}
	if _, _, err := server.CreatePlaylist(context.Background(), nil, PlaylistInput{Name: "Queue"}); err == nil || !strings.Contains(err.Error(), "libraryId") {
		t.Fatalf("expected playlist libraryId error, got %v", err)
	}
	if _, _, err := server.UpdatePlaylist(context.Background(), nil, PlaylistInput{PlaylistID: "pl-1"}); err == nil || !strings.Contains(err.Error(), "name or description") {
		t.Fatalf("expected playlist update field error, got %v", err)
	}
	if _, _, err := server.AddPlaylistItem(context.Background(), nil, PlaylistItemInput{PlaylistID: "pl-1"}); err == nil || !strings.Contains(err.Error(), "itemId") {
		t.Fatalf("expected playlist item error, got %v", err)
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
		"abs_match_item": func() error {
			_, _, err := server.MatchItem(context.Background(), nil, MatchItemInput{ItemID: "item-1"})
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
	user                        abs.User
	libraries                   []abs.Library
	items                       map[string][]abs.LibraryItem
	libraryItemTotals           []int
	getLibraryItemsCalls        int
	lastLibraryItemsOptions     abs.LibraryItemsOptions
	lastCatalogListOptions      abs.CatalogListOptions
	lastInclude                 []string
	itemsInProgressLimit        int
	getListeningStatsCalled     bool
	listListeningSessionsLimit  int
	listListeningSessionsPage   int
	updateItemProgressID        string
	updateItemProgressEpisodeID string
	updateItemProgressPayload   abs.ProgressUpdatePayload
	createBookmarkID            string
	createBookmarkPayload       abs.BookmarkPayload
	updateBookmarkID            string
	updateBookmarkPayload       abs.BookmarkPayload
	createBackupCalled          bool
	sendEbookPayload            abs.EbookDevicePayload
	scanLibraryID               string
	scanForce                   bool
	scanItemID                  string
	updateItemMetadataID        string
	updateItemMetadataPayload   abs.ItemMetadataPayload
	updateItemCoverID           string
	updateItemCoverPath         string
	removeItemCoverID           string
	updateItemChaptersID        string
	updateItemChapters          []abs.Chapter
	createCollectionPayload     abs.CollectionPayload
	updateCollectionID          string
	updateCollectionPayload     abs.CollectionPayload
	addCollectionID             string
	addCollectionItemID         string
	deleteCollectionID          string
	removeCollectionID          string
	removeCollectionItemID      string
	createPlaylistPayload       abs.PlaylistPayload
	updatePlaylistID            string
	updatePlaylistPayload       abs.PlaylistPayload
	addPlaylistID               string
	addPlaylistItem             abs.PlaylistItemPayload
	deletePlaylistID            string
	removePlaylistID            string
	removePlaylistItem          abs.PlaylistItemPayload
	removeIssuesCalled          bool
	removeIssuesLibraryID       string
	err                         error
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
					Media: abs.Media{
						Metadata: abs.Metadata{Title: "Alice", AuthorName: "Lewis Carroll"},
						EbookFile: &abs.EbookFile{LibraryFile: abs.LibraryFile{
							FileType: "ebook",
							Metadata: abs.FileMetadata{
								Filename: "alice.epub",
								Path:     "/books/alice/alice.epub",
								RelPath:  "alice/alice.epub",
								Size:     456,
							},
						}},
					},
				},
				{
					ID:        "book-2",
					LibraryID: "lib-books",
					Path:      "/books/alice-two",
					MediaType: "book",
					Media: abs.Media{
						Metadata: abs.Metadata{Title: "Alice Sequel", AuthorName: "Lewis Carroll"},
						EbookFile: &abs.EbookFile{LibraryFile: abs.LibraryFile{
							FileType: "ebook",
							Metadata: abs.FileMetadata{
								Filename: "alice-two.epub",
								Path:     "/books/alice-two/alice-two.epub",
								RelPath:  "alice-two/alice-two.epub",
								Size:     789,
							},
						}},
					},
				},
				{
					ID:        "audio-only-book",
					LibraryID: "lib-books",
					Path:      "/books/audio-only",
					MediaType: "book",
					Media:     abs.Media{Metadata: abs.Metadata{Title: "Audio Only", AuthorName: "Narrator"}},
				},
			},
		},
	}
}

func (f *fakeABSClient) ListBackups(context.Context) ([]abs.Backup, error) {
	if f.err != nil {
		return nil, f.err
	}
	return []abs.Backup{
		{ID: "backup-1", Filename: "backup-1.audiobookshelf", CreatedAt: 123, Size: 456},
	}, nil
}

func (f *fakeABSClient) CreateBackup(context.Context) (*abs.Backup, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.createBackupCalled = true
	return &abs.Backup{ID: "backup-created", Filename: "backup-created.audiobookshelf"}, nil
}

func (f *fakeABSClient) SendEbookToDevice(_ context.Context, payload abs.EbookDevicePayload) (abs.JSONValue, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.sendEbookPayload = payload
	return map[string]any{"success": true}, nil
}

func (f *fakeABSClient) GetEmailSettings(context.Context) (*abs.EmailSettings, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &abs.EmailSettings{
		EReaderDevices: []abs.EReaderDevice{
			{Name: "Kindle", Email: "kindle@example.com", AvailabilityOption: "specificUsers", Users: []string{"user-1"}},
			{Name: "Kobo", Email: "kobo@example.com", AvailabilityOption: "allUsers"},
		},
	}, nil
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

func (f *fakeABSClient) ListLibraryAuthors(_ context.Context, libraryID string, options abs.CatalogListOptions) (abs.JSONValue, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.lastCatalogListOptions = options
	return map[string]any{
		"libraryId": libraryID,
		"results":   []any{map[string]any{"id": "author-1", "name": "Lewis Carroll"}},
		"total":     1,
	}, nil
}

func (f *fakeABSClient) GetAuthor(_ context.Context, authorID string, include []string) (abs.JSONValue, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.lastInclude = include
	return map[string]any{"id": authorID, "name": "Lewis Carroll"}, nil
}

func (f *fakeABSClient) ListLibrarySeries(_ context.Context, libraryID string, options abs.CatalogListOptions) (abs.JSONValue, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.lastCatalogListOptions = options
	return map[string]any{
		"libraryId": libraryID,
		"results":   []any{map[string]any{"id": "series-1", "name": "Alice Books"}},
		"total":     1,
	}, nil
}

func (f *fakeABSClient) GetSeries(_ context.Context, seriesID string, include []string) (abs.JSONValue, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.lastInclude = include
	return map[string]any{"id": seriesID, "name": "Alice Books"}, nil
}

func (f *fakeABSClient) ListCollections(context.Context) (abs.JSONValue, error) {
	if f.err != nil {
		return nil, f.err
	}
	return map[string]any{"collections": []any{map[string]any{"id": "col-1", "name": "Favorites"}}}, nil
}

func (f *fakeABSClient) GetCollection(_ context.Context, collectionID string, include []string) (abs.JSONValue, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.lastInclude = include
	return map[string]any{"id": collectionID, "name": "Favorites"}, nil
}

func (f *fakeABSClient) GetItemsInProgress(_ context.Context, limit int) (abs.JSONValue, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.itemsInProgressLimit = limit
	return map[string]any{
		"libraryItems": []any{
			map[string]any{"id": "item-1", "progressLastUpdate": 123},
		},
	}, nil
}

func (f *fakeABSClient) GetListeningStats(context.Context) (abs.JSONValue, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.getListeningStatsCalled = true
	return map[string]any{"totalTime": 123}, nil
}

func (f *fakeABSClient) ListListeningSessions(_ context.Context, limit int, page int) (abs.JSONValue, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.listListeningSessionsLimit = limit
	f.listListeningSessionsPage = page
	return map[string]any{
		"sessions": []any{
			map[string]any{"id": "session-1"},
		},
	}, nil
}

func (f *fakeABSClient) GetItemProgress(_ context.Context, itemID string, episodeID string) (*abs.MediaProgress, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &abs.MediaProgress{
		ID:            "progress-1",
		UserID:        "user-1",
		LibraryItemID: itemID,
		EpisodeID:     episodeID,
		CurrentTime:   42,
		Progress:      0.5,
	}, nil
}

func (f *fakeABSClient) ListBookmarks(context.Context) ([]abs.Bookmark, error) {
	if f.err != nil {
		return nil, f.err
	}
	return []abs.Bookmark{
		{LibraryItemID: "item-1", Time: 12.5, Title: "Start", CreatedAt: 123},
		{LibraryItemID: "item-2", Time: 24, Title: "Middle", CreatedAt: 456},
	}, nil
}

func (f *fakeABSClient) UpdateItemProgress(_ context.Context, itemID string, episodeID string, payload abs.ProgressUpdatePayload) error {
	if f.err != nil {
		return f.err
	}
	f.updateItemProgressID = itemID
	f.updateItemProgressEpisodeID = episodeID
	f.updateItemProgressPayload = payload
	return nil
}

func (f *fakeABSClient) CreateBookmark(_ context.Context, itemID string, payload abs.BookmarkPayload) (*abs.Bookmark, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.createBookmarkID = itemID
	f.createBookmarkPayload = payload
	return &abs.Bookmark{LibraryItemID: itemID, Time: payload.Time, Title: payload.Title}, nil
}

func (f *fakeABSClient) UpdateBookmark(_ context.Context, itemID string, payload abs.BookmarkPayload) (*abs.Bookmark, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.updateBookmarkID = itemID
	f.updateBookmarkPayload = payload
	return &abs.Bookmark{LibraryItemID: itemID, Time: payload.Time, Title: payload.Title}, nil
}

func (f *fakeABSClient) GetItemMetadataObject(_ context.Context, itemID string) (abs.JSONValue, error) {
	if f.err != nil {
		return nil, f.err
	}
	return map[string]any{"itemId": itemID, "title": "Alice"}, nil
}

func (f *fakeABSClient) UpdateItemMetadata(_ context.Context, itemID string, payload abs.ItemMetadataPayload) (abs.JSONValue, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.updateItemMetadataID = itemID
	f.updateItemMetadataPayload = payload
	return map[string]any{"updated": true, "libraryItem": map[string]any{"id": itemID}}, nil
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

func (f *fakeABSClient) UpdateItemCover(_ context.Context, itemID string, cover string) (abs.JSONValue, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.updateItemCoverID = itemID
	f.updateItemCoverPath = cover
	return map[string]any{"itemId": itemID, "cover": cover}, nil
}

func (f *fakeABSClient) RemoveItemCover(_ context.Context, itemID string) error {
	if f.err != nil {
		return f.err
	}
	f.removeItemCoverID = itemID
	return nil
}

func (f *fakeABSClient) UpdateItemChapters(_ context.Context, itemID string, chapters []abs.Chapter) (abs.JSONValue, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.updateItemChaptersID = itemID
	f.updateItemChapters = chapters
	return map[string]any{"itemId": itemID, "chapterCount": len(chapters)}, nil
}

func (f *fakeABSClient) CreateCollection(_ context.Context, payload abs.CollectionPayload) (abs.JSONValue, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.createCollectionPayload = payload
	return map[string]any{"id": "col-1", "name": payload.Name}, nil
}

func (f *fakeABSClient) UpdateCollection(_ context.Context, collectionID string, payload abs.CollectionPayload) (abs.JSONValue, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.updateCollectionID = collectionID
	f.updateCollectionPayload = payload
	return map[string]any{"id": collectionID, "name": payload.Name}, nil
}

func (f *fakeABSClient) AddCollectionItem(_ context.Context, collectionID string, itemID string) (abs.JSONValue, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.addCollectionID = collectionID
	f.addCollectionItemID = itemID
	return map[string]any{"id": collectionID, "addedItemId": itemID}, nil
}

func (f *fakeABSClient) DeleteCollection(_ context.Context, collectionID string) (abs.JSONValue, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.deleteCollectionID = collectionID
	return map[string]any{"id": collectionID, "deleted": true}, nil
}

func (f *fakeABSClient) RemoveCollectionItem(_ context.Context, collectionID string, itemID string) (abs.JSONValue, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.removeCollectionID = collectionID
	f.removeCollectionItemID = itemID
	return map[string]any{"id": collectionID, "removedItemId": itemID}, nil
}

func (f *fakeABSClient) CreatePlaylist(_ context.Context, payload abs.PlaylistPayload) (abs.JSONValue, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.createPlaylistPayload = payload
	return map[string]any{"id": "pl-1", "name": payload.Name}, nil
}

func (f *fakeABSClient) UpdatePlaylist(_ context.Context, playlistID string, payload abs.PlaylistPayload) (abs.JSONValue, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.updatePlaylistID = playlistID
	f.updatePlaylistPayload = payload
	return map[string]any{"id": playlistID, "name": payload.Name}, nil
}

func (f *fakeABSClient) AddPlaylistItem(_ context.Context, playlistID string, payload abs.PlaylistItemPayload) (abs.JSONValue, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.addPlaylistID = playlistID
	f.addPlaylistItem = payload
	return map[string]any{"id": playlistID, "addedItemId": payload.LibraryItemID}, nil
}

func (f *fakeABSClient) DeletePlaylist(_ context.Context, playlistID string) (abs.JSONValue, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.deletePlaylistID = playlistID
	return map[string]any{"id": playlistID, "deleted": true}, nil
}

func (f *fakeABSClient) RemovePlaylistItem(_ context.Context, playlistID string, payload abs.PlaylistItemPayload) (abs.JSONValue, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.removePlaylistID = playlistID
	f.removePlaylistItem = payload
	return map[string]any{"id": playlistID, "removedItemId": payload.LibraryItemID}, nil
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
