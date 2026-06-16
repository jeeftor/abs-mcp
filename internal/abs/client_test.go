package abs

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewClientRejectsInvalidBaseURL(t *testing.T) {
	t.Parallel()

	if _, err := NewClient("localhost:13388", "token"); err == nil {
		t.Fatal("expected invalid base URL error")
	}
}

func TestClientGetCurrentUserSendsBearerToken(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/me" {
			t.Fatalf("path = %s, want /api/me", request.URL.Path)
		}
		if got := request.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("Authorization = %q, want Bearer test-token", got)
		}
		writeJSON(t, writer, User{ID: "user-1", Username: "root", Type: "root", IsActive: true})
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	user, err := client.GetCurrentUser(context.Background())
	if err != nil {
		t.Fatalf("GetCurrentUser failed: %v", err)
	}
	if user.Username != "root" {
		t.Fatalf("username = %q, want root", user.Username)
	}
}

func TestClientSendsExtraHeaders(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if got := request.Header.Get("X-Corp-Trace"); got != "trace-1" {
			t.Fatalf("X-Corp-Trace = %q, want trace-1", got)
		}
		if got := request.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("Authorization = %q, want Bearer test-token", got)
		}
		writeJSON(t, writer, User{ID: "user-1", Username: "root", Type: "root", IsActive: true})
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	if err := client.SetExtraHeaders(map[string]string{"X-Corp-Trace": "trace-1"}); err != nil {
		t.Fatalf("SetExtraHeaders failed: %v", err)
	}

	if _, err := client.GetCurrentUser(context.Background()); err != nil {
		t.Fatalf("GetCurrentUser failed: %v", err)
	}
}

func TestClientRejectsAuthorizationExtraHeader(t *testing.T) {
	t.Parallel()

	client, err := NewClient("http://localhost:13388", "test-token")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	if err := client.SetExtraHeaders(map[string]string{"Authorization": "Bearer other-token"}); err == nil {
		t.Fatal("expected authorization header rejection")
	}
}

func TestClientGetLibraries(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/libraries" {
			t.Fatalf("path = %s, want /api/libraries", request.URL.Path)
		}
		writeJSON(t, writer, map[string][]Library{
			"libraries": {
				{ID: "lib-audio", Name: "Audiobooks", Folders: []Folder{{FullPath: "/audiobooks"}}},
				{ID: "lib-books", Name: "Ebooks", Folders: []Folder{{FullPath: "/books"}}},
			},
		})
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	libraries, err := client.GetLibraries(context.Background())
	if err != nil {
		t.Fatalf("GetLibraries failed: %v", err)
	}
	if len(libraries) != 2 {
		t.Fatalf("len(libraries) = %d, want 2", len(libraries))
	}
	if libraries[0].Name != "Audiobooks" || libraries[1].Name != "Ebooks" {
		t.Fatalf("libraries = %#v", libraries)
	}
}

func TestClientGetLibraryItemsAddsPaginationQuery(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/libraries/lib-main/items" {
			t.Fatalf("path = %s, want /api/libraries/lib-main/items", request.URL.Path)
		}
		if got := request.URL.Query().Get("limit"); got != "50" {
			t.Fatalf("limit = %q, want 50", got)
		}
		if got := request.URL.Query().Get("page"); got != "2" {
			t.Fatalf("page = %q, want 2", got)
		}
		writeJSON(t, writer, LibraryItemsResponse{
			Results: []LibraryItem{{ID: "item-1", Path: "/audiobooks/book"}},
			Total:   101,
			Limit:   50,
			Page:    2,
		})
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	response, err := client.GetLibraryItems(context.Background(), "lib-main", 50, 100)
	if err != nil {
		t.Fatalf("GetLibraryItems failed: %v", err)
	}
	if response.Results[0].ID != "item-1" {
		t.Fatalf("item id = %q, want item-1", response.Results[0].ID)
	}
	if response.Offset != 100 {
		t.Fatalf("offset = %d, want 100", response.Offset)
	}
}

func TestClientGetLibraryItemsWithOptionsAddsFilterQuery(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		query := request.URL.Query()
		expected := map[string]string{
			"limit":          "25",
			"page":           "3",
			"sort":           "media.metadata.title",
			"desc":           "1",
			"filter":         "issues.true",
			"include":        "rssfeed,progress",
			"minified":       "1",
			"collapseseries": "1",
		}
		for key, want := range expected {
			if got := query.Get(key); got != want {
				t.Fatalf("%s = %q, want %q", key, got, want)
			}
		}
		writeJSON(t, writer, LibraryItemsResponse{
			Results: []LibraryItem{{ID: "item-1"}},
			Total:   1,
			Limit:   25,
			Page:    3,
		})
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	_, err = client.GetLibraryItemsWithOptions(context.Background(), "lib-main", LibraryItemsOptions{
		Limit:          25,
		Page:           3,
		Sort:           "media.metadata.title",
		Desc:           true,
		Filter:         "issues.true",
		Include:        []string{"rssfeed", "progress"},
		Minified:       true,
		CollapseSeries: true,
	})
	if err != nil {
		t.Fatalf("GetLibraryItemsWithOptions failed: %v", err)
	}
}

func TestClientListBackups(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/backups" {
			t.Fatalf("path = %s, want /api/backups", request.URL.Path)
		}
		writeJSON(t, writer, []Backup{
			{ID: "backup-1", Filename: "backup-1.audiobookshelf", CreatedAt: 123, Size: 456},
		})
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	backups, err := client.ListBackups(context.Background())
	if err != nil {
		t.Fatalf("ListBackups failed: %v", err)
	}
	if len(backups) != 1 || backups[0].ID != "backup-1" {
		t.Fatalf("backups = %#v", backups)
	}
}

func TestClientCreateBackup(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/backups" {
			t.Fatalf("path = %s, want /api/backups", request.URL.Path)
		}
		if request.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", request.Method)
		}
		writeJSON(t, writer, Backup{ID: "backup-2", Filename: "backup-2.audiobookshelf"})
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	backup, err := client.CreateBackup(context.Background())
	if err != nil {
		t.Fatalf("CreateBackup failed: %v", err)
	}
	if backup.ID != "backup-2" {
		t.Fatalf("backup = %#v", backup)
	}
}

func TestClientGetAllLibraryItemsPaginates(t *testing.T) {
	t.Parallel()

	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		switch request.URL.Query().Get("page") {
		case "":
			writeJSON(t, writer, LibraryItemsResponse{
				Results: []LibraryItem{{ID: "item-1"}},
				Total:   2,
				Limit:   100,
				Page:    0,
			})
		case "1":
			writeJSON(t, writer, LibraryItemsResponse{
				Results: []LibraryItem{{ID: "item-2"}},
				Total:   2,
				Limit:   100,
				Page:    1,
			})
		default:
			t.Fatalf("unexpected page %q", request.URL.Query().Get("page"))
		}
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	items, err := client.GetAllLibraryItems(context.Background(), "lib-main")
	if err != nil {
		t.Fatalf("GetAllLibraryItems failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
}

func TestClientGetLibraryItem(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/items/item-1" {
			t.Fatalf("path = %s, want /api/items/item-1", request.URL.Path)
		}
		writeJSON(t, writer, LibraryItem{ID: "item-1", Media: Media{Metadata: Metadata{Title: "Test Book"}}})
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	item, err := client.GetLibraryItem(context.Background(), "item-1")
	if err != nil {
		t.Fatalf("GetLibraryItem failed: %v", err)
	}
	if item.Media.Metadata.Title != "Test Book" {
		t.Fatalf("title = %q, want Test Book", item.Media.Metadata.Title)
	}
}

func TestClientSearchLibrary(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/libraries/lib-main/search" {
			t.Fatalf("path = %s, want /api/libraries/lib-main/search", request.URL.Path)
		}
		if got := request.URL.Query().Get("q"); got != "alice" {
			t.Fatalf("q = %q, want alice", got)
		}
		if got := request.URL.Query().Get("limit"); got != "7" {
			t.Fatalf("limit = %q, want 7", got)
		}
		writeJSON(t, writer, map[string]any{"book": []map[string]any{{"id": "item-1"}}})
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	response, err := client.SearchLibrary(context.Background(), "lib-main", "alice", 7)
	if err != nil {
		t.Fatalf("SearchLibrary failed: %v", err)
	}
	if response == nil {
		t.Fatal("expected search response")
	}
}

func TestClientGetLibraryStats(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/libraries/lib-main/stats" {
			t.Fatalf("path = %s, want /api/libraries/lib-main/stats", request.URL.Path)
		}
		writeJSON(t, writer, map[string]any{"totalItems": 3})
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	response, err := client.GetLibraryStats(context.Background(), "lib-main")
	if err != nil {
		t.Fatalf("GetLibraryStats failed: %v", err)
	}
	if response == nil {
		t.Fatal("expected stats response")
	}
}

func TestClientGetLibraryFilterData(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/libraries/lib-main/filterdata" {
			t.Fatalf("path = %s, want /api/libraries/lib-main/filterdata", request.URL.Path)
		}
		writeJSON(t, writer, map[string]any{"genres": []string{"fiction"}})
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	response, err := client.GetLibraryFilterData(context.Background(), "lib-main")
	if err != nil {
		t.Fatalf("GetLibraryFilterData failed: %v", err)
	}
	if response == nil {
		t.Fatal("expected filter data response")
	}
}

func TestClientListLibraryAuthors(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/libraries/lib-main/authors" {
			t.Fatalf("path = %s, want /api/libraries/lib-main/authors", request.URL.Path)
		}
		query := request.URL.Query()
		expected := map[string]string{
			"limit":    "25",
			"page":     "2",
			"sort":     "name",
			"desc":     "1",
			"filter":   "authors.name.Lewis",
			"include":  "items,series",
			"minified": "1",
		}
		for key, want := range expected {
			if got := query.Get(key); got != want {
				t.Fatalf("%s = %q, want %q", key, got, want)
			}
		}
		writeJSON(t, writer, map[string]any{"results": []map[string]any{{"id": "author-1"}}, "total": 1})
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	response, err := client.ListLibraryAuthors(context.Background(), "lib-main", CatalogListOptions{
		Limit:    25,
		Page:     2,
		Sort:     "name",
		Desc:     true,
		Filter:   "authors.name.Lewis",
		Include:  []string{"items", "series"},
		Minified: true,
	})
	if err != nil {
		t.Fatalf("ListLibraryAuthors failed: %v", err)
	}
	if response == nil {
		t.Fatal("expected authors response")
	}
}

func TestClientListLibraryAuthorsSendsFirstPage(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/libraries/lib-main/authors" {
			t.Fatalf("path = %s, want /api/libraries/lib-main/authors", request.URL.Path)
		}
		query := request.URL.Query()
		if got := query.Get("limit"); got != "25" {
			t.Fatalf("limit = %q, want 25", got)
		}
		if got := query.Get("page"); got != "0" {
			t.Fatalf("page = %q, want 0", got)
		}
		writeJSON(t, writer, map[string]any{"results": []map[string]any{{"id": "author-1"}}, "total": 1})
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	response, err := client.ListLibraryAuthors(context.Background(), "lib-main", CatalogListOptions{
		Limit: 25,
		Page:  0,
	})
	if err != nil {
		t.Fatalf("ListLibraryAuthors failed: %v", err)
	}
	if response == nil {
		t.Fatal("expected authors response")
	}
}

func TestClientGetAuthor(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/authors/author-1" {
			t.Fatalf("path = %s, want /api/authors/author-1", request.URL.Path)
		}
		if got := request.URL.Query().Get("include"); got != "items,series" {
			t.Fatalf("include = %q, want items,series", got)
		}
		writeJSON(t, writer, map[string]any{"id": "author-1", "name": "Lewis Carroll"})
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	response, err := client.GetAuthor(context.Background(), "author-1", []string{"items", "series"})
	if err != nil {
		t.Fatalf("GetAuthor failed: %v", err)
	}
	if response == nil {
		t.Fatal("expected author response")
	}
}

func TestClientListLibrarySeries(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/libraries/lib-main/series" {
			t.Fatalf("path = %s, want /api/libraries/lib-main/series", request.URL.Path)
		}
		query := request.URL.Query()
		expected := map[string]string{
			"limit":    "10",
			"page":     "1",
			"sort":     "name",
			"desc":     "1",
			"filter":   "series.name.Alice",
			"include":  "progress,rssfeed",
			"minified": "1",
		}
		for key, want := range expected {
			if got := query.Get(key); got != want {
				t.Fatalf("%s = %q, want %q", key, got, want)
			}
		}
		writeJSON(t, writer, map[string]any{"results": []map[string]any{{"id": "series-1"}}, "total": 1})
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	response, err := client.ListLibrarySeries(context.Background(), "lib-main", CatalogListOptions{
		Limit:    10,
		Page:     1,
		Sort:     "name",
		Desc:     true,
		Filter:   "series.name.Alice",
		Include:  []string{"progress", "rssfeed"},
		Minified: true,
	})
	if err != nil {
		t.Fatalf("ListLibrarySeries failed: %v", err)
	}
	if response == nil {
		t.Fatal("expected series response")
	}
}

func TestClientGetSeries(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/series/series-1" {
			t.Fatalf("path = %s, want /api/series/series-1", request.URL.Path)
		}
		if got := request.URL.Query().Get("include"); got != "progress,rssfeed" {
			t.Fatalf("include = %q, want progress,rssfeed", got)
		}
		writeJSON(t, writer, map[string]any{"id": "series-1", "name": "Alice Books"})
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	response, err := client.GetSeries(context.Background(), "series-1", []string{"progress", "rssfeed"})
	if err != nil {
		t.Fatalf("GetSeries failed: %v", err)
	}
	if response == nil {
		t.Fatal("expected series response")
	}
}

func TestClientListCollections(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/collections" {
			t.Fatalf("path = %s, want /api/collections", request.URL.Path)
		}
		writeJSON(t, writer, map[string]any{"collections": []map[string]any{{"id": "col-1"}}})
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	response, err := client.ListCollections(context.Background())
	if err != nil {
		t.Fatalf("ListCollections failed: %v", err)
	}
	if response == nil {
		t.Fatal("expected collections response")
	}
}

func TestClientGetCollection(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/collections/col-1" {
			t.Fatalf("path = %s, want /api/collections/col-1", request.URL.Path)
		}
		if got := request.URL.Query().Get("include"); got != "items" {
			t.Fatalf("include = %q, want items", got)
		}
		writeJSON(t, writer, map[string]any{"id": "col-1", "name": "Favorites"})
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	response, err := client.GetCollection(context.Background(), "col-1", []string{"items"})
	if err != nil {
		t.Fatalf("GetCollection failed: %v", err)
	}
	if response == nil {
		t.Fatal("expected collection response")
	}
}

func TestClientGetItemsInProgress(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/me/items-in-progress" {
			t.Fatalf("path = %s, want /api/me/items-in-progress", request.URL.Path)
		}
		if got := request.URL.Query().Get("limit"); got != "7" {
			t.Fatalf("limit = %q, want 7", got)
		}
		writeJSON(t, writer, map[string]any{
			"libraryItems": []map[string]any{{"id": "item-1", "progressLastUpdate": 123}},
		})
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	response, err := client.GetItemsInProgress(context.Background(), 7)
	if err != nil {
		t.Fatalf("GetItemsInProgress failed: %v", err)
	}
	if response == nil {
		t.Fatal("expected items-in-progress response")
	}
}

func TestClientGetItemProgress(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/me/progress/item-1/episode-1" {
			t.Fatalf("path = %s, want /api/me/progress/item-1/episode-1", request.URL.Path)
		}
		writeJSON(t, writer, MediaProgress{
			ID:            "progress-1",
			LibraryItemID: "item-1",
			EpisodeID:     "episode-1",
			CurrentTime:   42,
			Progress:      0.5,
		})
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	progress, err := client.GetItemProgress(context.Background(), "item-1", "episode-1")
	if err != nil {
		t.Fatalf("GetItemProgress failed: %v", err)
	}
	if progress.ID != "progress-1" || progress.LibraryItemID != "item-1" || progress.EpisodeID != "episode-1" {
		t.Fatalf("unexpected progress: %#v", progress)
	}
}

func TestClientListBookmarksExtractsBookmarksOnly(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/me" {
			t.Fatalf("path = %s, want /api/me", request.URL.Path)
		}
		writeJSON(t, writer, map[string]any{
			"id":    "user-1",
			"token": "should-not-be-returned",
			"bookmarks": []Bookmark{
				{LibraryItemID: "item-1", Time: 12.5, Title: "Start", CreatedAt: 123},
				{LibraryItemID: "item-2", Time: 24, Title: "Middle", CreatedAt: 456},
			},
		})
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	bookmarks, err := client.ListBookmarks(context.Background())
	if err != nil {
		t.Fatalf("ListBookmarks failed: %v", err)
	}
	if len(bookmarks) != 2 {
		t.Fatalf("bookmark count = %d, want 2", len(bookmarks))
	}
	if bookmarks[0].LibraryItemID != "item-1" || bookmarks[0].Title != "Start" {
		t.Fatalf("unexpected first bookmark: %#v", bookmarks[0])
	}
}

func TestClientUpdateItemProgress(t *testing.T) {
	t.Parallel()

	currentTime := 42.5
	progress := 0.5
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPatch {
			t.Fatalf("method = %s, want PATCH", request.Method)
		}
		if request.URL.Path != "/api/me/progress/item-1/episode-1" {
			t.Fatalf("path = %s, want /api/me/progress/item-1/episode-1", request.URL.Path)
		}
		var payload ProgressUpdatePayload
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if payload.CurrentTime == nil || *payload.CurrentTime != currentTime {
			t.Fatalf("unexpected current time payload: %#v", payload)
		}
		if payload.Progress == nil || *payload.Progress != progress {
			t.Fatalf("unexpected progress payload: %#v", payload)
		}
		writer.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	if err := client.UpdateItemProgress(context.Background(), "item-1", "episode-1", ProgressUpdatePayload{
		CurrentTime: &currentTime,
		Progress:    &progress,
	}); err != nil {
		t.Fatalf("UpdateItemProgress failed: %v", err)
	}
}

func TestClientCreateAndUpdateBookmark(t *testing.T) {
	t.Parallel()

	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		if request.URL.Path != "/api/me/item/item-1/bookmark" {
			t.Fatalf("path = %s, want /api/me/item/item-1/bookmark", request.URL.Path)
		}
		var payload BookmarkPayload
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if payload.Time != 12.5 || payload.Title != "Start" {
			t.Fatalf("unexpected bookmark payload: %#v", payload)
		}
		switch requests {
		case 1:
			if request.Method != http.MethodPost {
				t.Fatalf("method = %s, want POST", request.Method)
			}
		case 2:
			if request.Method != http.MethodPatch {
				t.Fatalf("method = %s, want PATCH", request.Method)
			}
		default:
			t.Fatalf("unexpected request count %d", requests)
		}
		writeJSON(t, writer, Bookmark{LibraryItemID: "item-1", Time: payload.Time, Title: payload.Title})
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	created, err := client.CreateBookmark(context.Background(), "item-1", BookmarkPayload{Time: 12.5, Title: "Start"})
	if err != nil {
		t.Fatalf("CreateBookmark failed: %v", err)
	}
	if created.Title != "Start" {
		t.Fatalf("unexpected created bookmark: %#v", created)
	}
	updated, err := client.UpdateBookmark(context.Background(), "item-1", BookmarkPayload{Time: 12.5, Title: "Start"})
	if err != nil {
		t.Fatalf("UpdateBookmark failed: %v", err)
	}
	if updated.LibraryItemID != "item-1" {
		t.Fatalf("unexpected updated bookmark: %#v", updated)
	}
}

func TestClientGetItemMetadataObject(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/items/item-1/metadata-object" {
			t.Fatalf("path = %s, want /api/items/item-1/metadata-object", request.URL.Path)
		}
		writeJSON(t, writer, map[string]any{"title": "Alice"})
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	response, err := client.GetItemMetadataObject(context.Background(), "item-1")
	if err != nil {
		t.Fatalf("GetItemMetadataObject failed: %v", err)
	}
	if response == nil {
		t.Fatal("expected metadata object response")
	}
}

func TestClientUpdateItemMetadata(t *testing.T) {
	t.Parallel()

	title := "Alice Updated"
	description := "Clean description"
	explicit := true
	genres := []string{"fiction", "classic"}
	authors := []ItemMetadataAuthorPayload{{Name: "Lewis Carroll"}}
	series := []ItemMetadataSeriesPayload{{Name: "Alice Books", Sequence: "1"}}
	tags := []string{"favorite"}

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPatch {
			t.Fatalf("method = %s, want PATCH", request.Method)
		}
		if request.URL.Path != "/api/items/item-1/media" {
			t.Fatalf("path = %s, want /api/items/item-1/media", request.URL.Path)
		}
		var payload ItemMetadataPayload
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if payload.Metadata == nil || payload.Metadata.Title == nil || *payload.Metadata.Title != title {
			t.Fatalf("unexpected title payload: %#v", payload)
		}
		if payload.Metadata.Description == nil || *payload.Metadata.Description != description {
			t.Fatalf("unexpected description payload: %#v", payload)
		}
		if payload.Metadata.Explicit == nil || !*payload.Metadata.Explicit {
			t.Fatalf("unexpected explicit payload: %#v", payload)
		}
		if payload.Metadata.Genres == nil || len(*payload.Metadata.Genres) != 2 || (*payload.Metadata.Genres)[0] != "fiction" {
			t.Fatalf("unexpected genres payload: %#v", payload)
		}
		if payload.Metadata.Authors == nil || len(*payload.Metadata.Authors) != 1 || (*payload.Metadata.Authors)[0].Name != "Lewis Carroll" {
			t.Fatalf("unexpected authors payload: %#v", payload)
		}
		if payload.Metadata.Series == nil || len(*payload.Metadata.Series) != 1 || (*payload.Metadata.Series)[0].Sequence != "1" {
			t.Fatalf("unexpected series payload: %#v", payload)
		}
		if payload.Tags == nil || len(*payload.Tags) != 1 || (*payload.Tags)[0] != "favorite" {
			t.Fatalf("unexpected tags payload: %#v", payload)
		}
		writeJSON(t, writer, map[string]any{
			"updated":     true,
			"libraryItem": map[string]any{"id": "item-1"},
		})
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	response, err := client.UpdateItemMetadata(context.Background(), "item-1", ItemMetadataPayload{
		Metadata: &ItemMetadataFields{
			Title:       &title,
			Description: &description,
			Explicit:    &explicit,
			Genres:      &genres,
			Authors:     &authors,
			Series:      &series,
		},
		Tags: &tags,
	})
	if err != nil {
		t.Fatalf("UpdateItemMetadata failed: %v", err)
	}
	if response == nil {
		t.Fatal("expected metadata update response")
	}
}

func TestClientScanLibrary(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", request.Method)
		}
		if request.URL.Path != "/api/libraries/lib-main/scan" {
			t.Fatalf("path = %s, want /api/libraries/lib-main/scan", request.URL.Path)
		}
		if got := request.URL.Query().Get("force"); got != "1" {
			t.Fatalf("force = %q, want 1", got)
		}
		writer.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	if err := client.ScanLibrary(context.Background(), "lib-main", true); err != nil {
		t.Fatalf("ScanLibrary failed: %v", err)
	}
}

func TestClientScanLibraryWithoutForce(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if got := request.URL.RawQuery; got != "" {
			t.Fatalf("query = %q, want empty", got)
		}
		writer.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	if err := client.ScanLibrary(context.Background(), "lib-main", false); err != nil {
		t.Fatalf("ScanLibrary failed: %v", err)
	}
}

func TestClientRemoveLibraryItemsWithIssues(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodDelete {
			t.Fatalf("method = %s, want DELETE", request.Method)
		}
		if request.URL.Path != "/api/libraries/lib-main/issues" {
			t.Fatalf("path = %s, want /api/libraries/lib-main/issues", request.URL.Path)
		}
		writer.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	if err := client.RemoveLibraryItemsWithIssues(context.Background(), "lib-main"); err != nil {
		t.Fatalf("RemoveLibraryItemsWithIssues failed: %v", err)
	}
}

func TestClientScanItem(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", request.Method)
		}
		if request.URL.Path != "/api/items/item-1/scan" {
			t.Fatalf("path = %s, want /api/items/item-1/scan", request.URL.Path)
		}
		if got := request.URL.RawQuery; got != "" {
			t.Fatalf("query = %q, want empty", got)
		}
		writeJSON(t, writer, map[string]any{"result": "SUCCESS"})
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	response, err := client.ScanItem(context.Background(), "item-1")
	if err != nil {
		t.Fatalf("ScanItem failed: %v", err)
	}
	if response.Result != "SUCCESS" {
		t.Fatalf("Result = %q, want SUCCESS", response.Result)
	}
}

func TestClientUpdateItemCover(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPatch {
			t.Fatalf("method = %s, want PATCH", request.Method)
		}
		if request.URL.Path != "/api/items/item-1/cover" {
			t.Fatalf("path = %s, want /api/items/item-1/cover", request.URL.Path)
		}
		if got := request.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("Content-Type = %q, want application/json", got)
		}
		var payload map[string]string
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if payload["cover"] != "/covers/alice.jpg" {
			t.Fatalf("cover = %q, want /covers/alice.jpg", payload["cover"])
		}
		writeJSON(t, writer, map[string]any{"id": "item-1", "coverPath": "/covers/alice.jpg"})
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	response, err := client.UpdateItemCover(context.Background(), "item-1", "/covers/alice.jpg")
	if err != nil {
		t.Fatalf("UpdateItemCover failed: %v", err)
	}
	if response == nil {
		t.Fatal("expected cover update response")
	}
}

func TestClientRemoveItemCover(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodDelete {
			t.Fatalf("method = %s, want DELETE", request.Method)
		}
		if request.URL.Path != "/api/items/item-1/cover" {
			t.Fatalf("path = %s, want /api/items/item-1/cover", request.URL.Path)
		}
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	if err := client.RemoveItemCover(context.Background(), "item-1"); err != nil {
		t.Fatalf("RemoveItemCover failed: %v", err)
	}
}

func TestClientUpdateItemChapters(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", request.Method)
		}
		if request.URL.Path != "/api/items/item-1/chapters" {
			t.Fatalf("path = %s, want /api/items/item-1/chapters", request.URL.Path)
		}
		var payload struct {
			Chapters []Chapter `json:"chapters"`
		}
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if len(payload.Chapters) != 1 {
			t.Fatalf("chapter count = %d, want 1", len(payload.Chapters))
		}
		if payload.Chapters[0].Title != "Intro" || payload.Chapters[0].Start != 0 || payload.Chapters[0].End != 12.5 {
			t.Fatalf("unexpected chapter: %#v", payload.Chapters[0])
		}
		writeJSON(t, writer, map[string]any{"updated": true})
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	response, err := client.UpdateItemChapters(context.Background(), "item-1", []Chapter{
		{Title: "Intro", Start: 0, End: 12.5},
	})
	if err != nil {
		t.Fatalf("UpdateItemChapters failed: %v", err)
	}
	if response == nil {
		t.Fatal("expected chapter update response")
	}
}

func TestClientCreateCollection(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", request.Method)
		}
		if request.URL.Path != "/api/collections" {
			t.Fatalf("path = %s, want /api/collections", request.URL.Path)
		}
		var payload CollectionPayload
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if payload.LibraryID != "lib-audio" || payload.Name != "Favorites" || payload.Description != "Good books" {
			t.Fatalf("unexpected payload: %#v", payload)
		}
		if len(payload.Books) != 1 || payload.Books[0] != "item-1" {
			t.Fatalf("books = %#v, want item-1", payload.Books)
		}
		writeJSON(t, writer, map[string]any{"id": "col-1", "name": "Favorites"})
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	response, err := client.CreateCollection(context.Background(), CollectionPayload{
		LibraryID:   "lib-audio",
		Name:        "Favorites",
		Description: "Good books",
		Books:       []string{"item-1"},
	})
	if err != nil {
		t.Fatalf("CreateCollection failed: %v", err)
	}
	if response == nil {
		t.Fatal("expected collection create response")
	}
}

func TestClientUpdateCollection(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPatch {
			t.Fatalf("method = %s, want PATCH", request.Method)
		}
		if request.URL.Path != "/api/collections/col-1" {
			t.Fatalf("path = %s, want /api/collections/col-1", request.URL.Path)
		}
		var payload CollectionPayload
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if payload.Name != "Updated" || payload.Description != "New description" || payload.LibraryID != "" || len(payload.Books) != 0 {
			t.Fatalf("unexpected payload: %#v", payload)
		}
		writeJSON(t, writer, map[string]any{"id": "col-1", "name": "Updated"})
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	response, err := client.UpdateCollection(context.Background(), "col-1", CollectionPayload{Name: "Updated", Description: "New description"})
	if err != nil {
		t.Fatalf("UpdateCollection failed: %v", err)
	}
	if response == nil {
		t.Fatal("expected collection update response")
	}
}

func TestClientAddCollectionItem(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", request.Method)
		}
		if request.URL.Path != "/api/collections/col-1/book" {
			t.Fatalf("path = %s, want /api/collections/col-1/book", request.URL.Path)
		}
		var payload map[string]string
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if payload["id"] != "item-1" {
			t.Fatalf("id = %q, want item-1", payload["id"])
		}
		writeJSON(t, writer, map[string]any{"id": "col-1"})
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	response, err := client.AddCollectionItem(context.Background(), "col-1", "item-1")
	if err != nil {
		t.Fatalf("AddCollectionItem failed: %v", err)
	}
	if response == nil {
		t.Fatal("expected collection add response")
	}
}

func TestClientCreatePlaylist(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", request.Method)
		}
		if request.URL.Path != "/api/playlists" {
			t.Fatalf("path = %s, want /api/playlists", request.URL.Path)
		}
		var payload PlaylistPayload
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if payload.LibraryID != "lib-audio" || payload.Name != "Queue" || len(payload.Items) != 1 {
			t.Fatalf("unexpected payload: %#v", payload)
		}
		if payload.Items[0].LibraryItemID != "item-1" || payload.Items[0].EpisodeID != "episode-1" {
			t.Fatalf("unexpected playlist item: %#v", payload.Items[0])
		}
		writeJSON(t, writer, map[string]any{"id": "pl-1", "name": "Queue"})
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	response, err := client.CreatePlaylist(context.Background(), PlaylistPayload{
		LibraryID: "lib-audio",
		Name:      "Queue",
		Items:     []PlaylistItemPayload{{LibraryItemID: "item-1", EpisodeID: "episode-1"}},
	})
	if err != nil {
		t.Fatalf("CreatePlaylist failed: %v", err)
	}
	if response == nil {
		t.Fatal("expected playlist create response")
	}
}

func TestClientUpdatePlaylist(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPatch {
			t.Fatalf("method = %s, want PATCH", request.Method)
		}
		if request.URL.Path != "/api/playlists/pl-1" {
			t.Fatalf("path = %s, want /api/playlists/pl-1", request.URL.Path)
		}
		var payload PlaylistPayload
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if payload.Name != "Updated" || payload.Description != "New description" || payload.LibraryID != "" || len(payload.Items) != 0 {
			t.Fatalf("unexpected payload: %#v", payload)
		}
		writeJSON(t, writer, map[string]any{"id": "pl-1", "name": "Updated"})
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	response, err := client.UpdatePlaylist(context.Background(), "pl-1", PlaylistPayload{Name: "Updated", Description: "New description"})
	if err != nil {
		t.Fatalf("UpdatePlaylist failed: %v", err)
	}
	if response == nil {
		t.Fatal("expected playlist update response")
	}
}

func TestClientAddPlaylistItem(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", request.Method)
		}
		if request.URL.Path != "/api/playlists/pl-1/item" {
			t.Fatalf("path = %s, want /api/playlists/pl-1/item", request.URL.Path)
		}
		var payload PlaylistItemPayload
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if payload.LibraryItemID != "item-1" || payload.EpisodeID != "episode-1" {
			t.Fatalf("unexpected payload: %#v", payload)
		}
		writeJSON(t, writer, map[string]any{"id": "pl-1"})
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	response, err := client.AddPlaylistItem(context.Background(), "pl-1", PlaylistItemPayload{LibraryItemID: "item-1", EpisodeID: "episode-1"})
	if err != nil {
		t.Fatalf("AddPlaylistItem failed: %v", err)
	}
	if response == nil {
		t.Fatal("expected playlist add response")
	}
}

func TestClientReturnsHTTPError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		http.Error(writer, "nope", http.StatusUnauthorized)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "bad-token")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	if _, err := client.GetLibraries(context.Background()); err == nil {
		t.Fatal("expected HTTP error")
	}
}

func TestClientRedactsTokenFromHTTPError(t *testing.T) {
	t.Parallel()

	const token = "bad-token-secret"
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		http.Error(writer, "nope "+token+" Bearer "+token, http.StatusUnauthorized)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, token)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	_, err = client.GetLibraries(context.Background())
	if err == nil {
		t.Fatal("expected HTTP error")
	}
	if strings.Contains(err.Error(), token) {
		t.Fatalf("error leaked token: %v", err)
	}
	if !strings.Contains(err.Error(), "[REDACTED]") {
		t.Fatalf("error did not include redaction marker: %v", err)
	}
}

func writeJSON(t *testing.T, writer http.ResponseWriter, value any) {
	t.Helper()
	writer.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(writer).Encode(value); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}
