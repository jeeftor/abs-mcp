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
