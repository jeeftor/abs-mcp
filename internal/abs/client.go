package abs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultTimeout = 30 * time.Second

const maxJSONPayloadBytes = 64 * 1024

// Client is an Audiobookshelf REST API client.
type Client struct {
	baseURL      *url.URL
	token        string
	httpClient   *http.Client
	extraHeaders map[string]string
}

// NewClient creates an authenticated Audiobookshelf API client.
func NewClient(baseURL string, token string) (*Client, error) {
	parsed, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil {
		return nil, fmt.Errorf("parse ABS base URL: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("parse ABS base URL: missing scheme or host")
	}

	return &Client{
		baseURL:      parsed,
		token:        token,
		httpClient:   &http.Client{Timeout: defaultTimeout},
		extraHeaders: map[string]string{},
	}, nil
}

// SetHTTPClient replaces the underlying HTTP client.
func (c *Client) SetHTTPClient(client *http.Client) {
	c.httpClient = client
}

// SetExtraHeaders replaces additional headers sent with every ABS request.
func (c *Client) SetExtraHeaders(headers map[string]string) error {
	extraHeaders := make(map[string]string, len(headers))
	for name, value := range headers {
		canonicalName, err := normalizeExtraHeaderName(name)
		if err != nil {
			return err
		}
		extraHeaders[canonicalName] = value
	}
	c.extraHeaders = extraHeaders
	return nil
}

// GetCurrentUser returns the authenticated ABS user.
func (c *Client) GetCurrentUser(ctx context.Context) (*User, error) {
	var user User
	if err := c.getJSON(ctx, "/api/me", nil, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// GetLibraries returns all ABS libraries visible to the token.
func (c *Client) GetLibraries(ctx context.Context) ([]Library, error) {
	var response struct {
		Libraries []Library `json:"libraries"`
	}
	if err := c.getJSON(ctx, "/api/libraries", nil, &response); err != nil {
		return nil, err
	}
	return response.Libraries, nil
}

// GetLibrary returns one ABS library by ID.
func (c *Client) GetLibrary(ctx context.Context, libraryID string) (*Library, error) {
	var library Library
	if err := c.getJSON(ctx, fmt.Sprintf("/api/libraries/%s", url.PathEscape(libraryID)), nil, &library); err != nil {
		return nil, err
	}
	return &library, nil
}

// GetLibraryItems returns one page of items from a library.
func (c *Client) GetLibraryItems(
	ctx context.Context,
	libraryID string,
	limit int,
	offset int,
) (*LibraryItemsResponse, error) {
	page := 0
	if limit > 0 && offset > 0 {
		page = offset / limit
	}
	return c.GetLibraryItemsWithOptions(ctx, libraryID, LibraryItemsOptions{Limit: limit, Page: page})
}

// GetLibraryItemsWithOptions returns one filtered page of items from a library.
func (c *Client) GetLibraryItemsWithOptions(
	ctx context.Context,
	libraryID string,
	options LibraryItemsOptions,
) (*LibraryItemsResponse, error) {
	query := url.Values{}
	if options.Limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", options.Limit))
	}
	if options.Page > 0 {
		query.Set("page", fmt.Sprintf("%d", options.Page))
	}
	if options.Sort != "" {
		query.Set("sort", options.Sort)
	}
	if options.Desc {
		query.Set("desc", "1")
	}
	if options.Filter != "" {
		query.Set("filter", options.Filter)
	}
	if len(options.Include) > 0 {
		query.Set("include", strings.Join(options.Include, ","))
	}
	if options.Minified {
		query.Set("minified", "1")
	}
	if options.CollapseSeries {
		query.Set("collapseseries", "1")
	}

	var response LibraryItemsResponse
	path := fmt.Sprintf("/api/libraries/%s/items", url.PathEscape(libraryID))
	if err := c.getJSON(ctx, path, query, &response); err != nil {
		return nil, err
	}
	if response.Offset == 0 && response.Page > 0 && response.Limit > 0 {
		response.Offset = response.Page * response.Limit
	}
	return &response, nil
}

// GetAllLibraryItems returns every item from a library using pagination.
func (c *Client) GetAllLibraryItems(ctx context.Context, libraryID string) ([]LibraryItem, error) {
	const limit = 100

	var items []LibraryItem
	offset := 0
	page := 0
	for {
		response, err := c.GetLibraryItemsWithOptions(ctx, libraryID, LibraryItemsOptions{Limit: limit, Page: page})
		if err != nil {
			return nil, err
		}
		items = append(items, response.Results...)
		if len(response.Results) == 0 || offset+len(response.Results) >= response.Total {
			return items, nil
		}
		offset += len(response.Results)
		page++
	}
}

// GetLibraryItem returns one ABS library item by ID.
func (c *Client) GetLibraryItem(ctx context.Context, itemID string) (*LibraryItem, error) {
	var item LibraryItem
	if err := c.getJSON(ctx, fmt.Sprintf("/api/items/%s", url.PathEscape(itemID)), nil, &item); err != nil {
		return nil, err
	}
	return &item, nil
}

// SearchLibrary searches items in one ABS library.
func (c *Client) SearchLibrary(
	ctx context.Context,
	libraryID string,
	query string,
	limit int,
) (JSONValue, error) {
	values := url.Values{}
	values.Set("q", query)
	if limit > 0 {
		values.Set("limit", fmt.Sprintf("%d", limit))
	}
	var response any
	path := fmt.Sprintf("/api/libraries/%s/search", url.PathEscape(libraryID))
	if err := c.getJSON(ctx, path, values, &response); err != nil {
		return nil, err
	}
	return response, nil
}

// GetLibraryStats returns raw ABS stats for one library.
func (c *Client) GetLibraryStats(ctx context.Context, libraryID string) (JSONValue, error) {
	var response any
	path := fmt.Sprintf("/api/libraries/%s/stats", url.PathEscape(libraryID))
	if err := c.getJSON(ctx, path, nil, &response); err != nil {
		return nil, err
	}
	return response, nil
}

// GetLibraryFilterData returns raw ABS filter data for one library.
func (c *Client) GetLibraryFilterData(ctx context.Context, libraryID string) (JSONValue, error) {
	var response any
	path := fmt.Sprintf("/api/libraries/%s/filterdata", url.PathEscape(libraryID))
	if err := c.getJSON(ctx, path, nil, &response); err != nil {
		return nil, err
	}
	return response, nil
}

// ListLibraryAuthors returns one source-backed page of authors from a library.
func (c *Client) ListLibraryAuthors(ctx context.Context, libraryID string, options CatalogListOptions) (JSONValue, error) {
	var response any
	path := fmt.Sprintf("/api/libraries/%s/authors", url.PathEscape(libraryID))
	if err := c.getJSON(ctx, path, catalogListQuery(options), &response); err != nil {
		return nil, err
	}
	return response, nil
}

// GetAuthor returns one source-backed author object.
func (c *Client) GetAuthor(ctx context.Context, authorID string, include []string) (JSONValue, error) {
	var response any
	path := fmt.Sprintf("/api/authors/%s", url.PathEscape(authorID))
	if err := c.getJSON(ctx, path, includeQuery(include), &response); err != nil {
		return nil, err
	}
	return response, nil
}

// ListLibrarySeries returns one source-backed page of series from a library.
func (c *Client) ListLibrarySeries(ctx context.Context, libraryID string, options CatalogListOptions) (JSONValue, error) {
	var response any
	path := fmt.Sprintf("/api/libraries/%s/series", url.PathEscape(libraryID))
	if err := c.getJSON(ctx, path, catalogListQuery(options), &response); err != nil {
		return nil, err
	}
	return response, nil
}

// GetSeries returns one source-backed series object.
func (c *Client) GetSeries(ctx context.Context, seriesID string, include []string) (JSONValue, error) {
	var response any
	path := fmt.Sprintf("/api/series/%s", url.PathEscape(seriesID))
	if err := c.getJSON(ctx, path, includeQuery(include), &response); err != nil {
		return nil, err
	}
	return response, nil
}

// ListCollections returns collections visible to the authenticated ABS user.
func (c *Client) ListCollections(ctx context.Context) (JSONValue, error) {
	var response any
	if err := c.getJSON(ctx, "/api/collections", nil, &response); err != nil {
		return nil, err
	}
	return response, nil
}

// GetCollection returns one source-backed collection object.
func (c *Client) GetCollection(ctx context.Context, collectionID string, include []string) (JSONValue, error) {
	var response any
	path := fmt.Sprintf("/api/collections/%s", url.PathEscape(collectionID))
	if err := c.getJSON(ctx, path, includeQuery(include), &response); err != nil {
		return nil, err
	}
	return response, nil
}

// GetItemsInProgress returns source-backed current-user items in progress.
func (c *Client) GetItemsInProgress(ctx context.Context, limit int) (JSONValue, error) {
	query := url.Values{}
	if limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", limit))
	}
	var response any
	if err := c.getJSON(ctx, "/api/me/items-in-progress", query, &response); err != nil {
		return nil, err
	}
	return response, nil
}

// GetItemProgress returns current-user progress for one library item or episode.
func (c *Client) GetItemProgress(ctx context.Context, itemID string, episodeID string) (*MediaProgress, error) {
	path := fmt.Sprintf("/api/me/progress/%s", url.PathEscape(itemID))
	if episodeID != "" {
		path += "/" + url.PathEscape(episodeID)
	}
	var progress MediaProgress
	if err := c.getJSON(ctx, path, nil, &progress); err != nil {
		return nil, err
	}
	return &progress, nil
}

// ListBookmarks returns current-user bookmarks without exposing the full user payload.
func (c *Client) ListBookmarks(ctx context.Context) ([]Bookmark, error) {
	var response struct {
		Bookmarks []Bookmark `json:"bookmarks"`
	}
	if err := c.getJSON(ctx, "/api/me", nil, &response); err != nil {
		return nil, err
	}
	return response.Bookmarks, nil
}

// UpdateItemProgress creates or updates current-user progress for one item.
func (c *Client) UpdateItemProgress(ctx context.Context, itemID string, episodeID string, payload ProgressUpdatePayload) error {
	path := fmt.Sprintf("/api/me/progress/%s", url.PathEscape(itemID))
	if episodeID != "" {
		path += "/" + url.PathEscape(episodeID)
	}
	return c.doJSON(ctx, http.MethodPatch, path, nil, payload, nil)
}

// CreateBookmark creates one current-user bookmark for a library item.
func (c *Client) CreateBookmark(ctx context.Context, itemID string, payload BookmarkPayload) (*Bookmark, error) {
	var response Bookmark
	path := fmt.Sprintf("/api/me/item/%s/bookmark", url.PathEscape(itemID))
	if err := c.doJSON(ctx, http.MethodPost, path, nil, payload, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// UpdateBookmark updates one current-user bookmark for a library item and time.
func (c *Client) UpdateBookmark(ctx context.Context, itemID string, payload BookmarkPayload) (*Bookmark, error) {
	var response Bookmark
	path := fmt.Sprintf("/api/me/item/%s/bookmark", url.PathEscape(itemID))
	if err := c.doJSON(ctx, http.MethodPatch, path, nil, payload, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// ListBackups returns backup records visible to the authenticated ABS user.
func (c *Client) ListBackups(ctx context.Context) ([]Backup, error) {
	var response backupListResponse
	if err := c.getJSON(ctx, "/api/backups", nil, &response); err != nil {
		return nil, err
	}
	return response.Backups, nil
}

// CreateBackup asks ABS to create a server backup.
func (c *Client) CreateBackup(ctx context.Context) (*Backup, error) {
	var response backupMutationResponse
	if err := c.doJSON(ctx, http.MethodPost, "/api/backups", nil, nil, &response); err != nil {
		return nil, err
	}
	return response.Backup(), nil
}

// SendEbookToDevice asks ABS to email one library item's ebook file to a saved ereader device.
func (c *Client) SendEbookToDevice(ctx context.Context, payload EbookDevicePayload) (JSONValue, error) {
	var response any
	if err := c.doJSON(ctx, http.MethodPost, "/api/emails/send-ebook-to-device", nil, payload, &response); err != nil {
		return nil, err
	}
	return response, nil
}

// GetEmailSettings returns ABS email settings for admin-capable tokens.
func (c *Client) GetEmailSettings(ctx context.Context) (*EmailSettings, error) {
	var response struct {
		Settings EmailSettings `json:"settings"`
	}
	if err := c.getJSON(ctx, "/api/emails/settings", nil, &response); err != nil {
		return nil, err
	}
	return &response.Settings, nil
}

// CreateCollection creates one ABS collection with an initial book list.
func (c *Client) CreateCollection(ctx context.Context, payload CollectionPayload) (JSONValue, error) {
	var response any
	if err := c.doJSON(ctx, http.MethodPost, "/api/collections", nil, payload, &response); err != nil {
		return nil, err
	}
	return response, nil
}

// UpdateCollection updates one ABS collection's non-membership fields.
func (c *Client) UpdateCollection(ctx context.Context, collectionID string, payload CollectionPayload) (JSONValue, error) {
	var response any
	path := fmt.Sprintf("/api/collections/%s", url.PathEscape(collectionID))
	if err := c.doJSON(ctx, http.MethodPatch, path, nil, payload, &response); err != nil {
		return nil, err
	}
	return response, nil
}

// AddCollectionItem adds one library item to an ABS collection.
func (c *Client) AddCollectionItem(ctx context.Context, collectionID string, itemID string) (JSONValue, error) {
	var response any
	path := fmt.Sprintf("/api/collections/%s/book", url.PathEscape(collectionID))
	payload := map[string]string{"id": itemID}
	if err := c.doJSON(ctx, http.MethodPost, path, nil, payload, &response); err != nil {
		return nil, err
	}
	return response, nil
}

// CreatePlaylist creates one ABS playlist.
func (c *Client) CreatePlaylist(ctx context.Context, payload PlaylistPayload) (JSONValue, error) {
	var response any
	if err := c.doJSON(ctx, http.MethodPost, "/api/playlists", nil, payload, &response); err != nil {
		return nil, err
	}
	return response, nil
}

// UpdatePlaylist updates one ABS playlist's non-membership fields.
func (c *Client) UpdatePlaylist(ctx context.Context, playlistID string, payload PlaylistPayload) (JSONValue, error) {
	var response any
	path := fmt.Sprintf("/api/playlists/%s", url.PathEscape(playlistID))
	if err := c.doJSON(ctx, http.MethodPatch, path, nil, payload, &response); err != nil {
		return nil, err
	}
	return response, nil
}

// AddPlaylistItem adds one item or podcast episode to an ABS playlist.
func (c *Client) AddPlaylistItem(ctx context.Context, playlistID string, payload PlaylistItemPayload) (JSONValue, error) {
	var response any
	path := fmt.Sprintf("/api/playlists/%s/item", url.PathEscape(playlistID))
	if err := c.doJSON(ctx, http.MethodPost, path, nil, payload, &response); err != nil {
		return nil, err
	}
	return response, nil
}

// GetItemMetadataObject returns the raw ABS metadata object for one item.
func (c *Client) GetItemMetadataObject(ctx context.Context, itemID string) (JSONValue, error) {
	var response any
	path := fmt.Sprintf("/api/items/%s/metadata-object", url.PathEscape(itemID))
	if err := c.getJSON(ctx, path, nil, &response); err != nil {
		return nil, err
	}
	return response, nil
}

// UpdateItemMetadata updates source-verified item metadata fields.
func (c *Client) UpdateItemMetadata(ctx context.Context, itemID string, payload ItemMetadataPayload) (JSONValue, error) {
	var response any
	path := fmt.Sprintf("/api/items/%s/media", url.PathEscape(itemID))
	if err := c.doJSON(ctx, http.MethodPatch, path, nil, payload, &response); err != nil {
		return nil, err
	}
	return response, nil
}

// ScanLibrary triggers an ABS library scan.
func (c *Client) ScanLibrary(ctx context.Context, libraryID string, force bool) error {
	query := url.Values{}
	if force {
		query.Set("force", "1")
	}
	return c.do(ctx, http.MethodPost, fmt.Sprintf("/api/libraries/%s/scan", url.PathEscape(libraryID)), query, nil)
}

// RemoveLibraryItemsWithIssues removes missing or invalid items from one ABS library.
func (c *Client) RemoveLibraryItemsWithIssues(ctx context.Context, libraryID string) error {
	path := fmt.Sprintf("/api/libraries/%s/issues", url.PathEscape(libraryID))
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

// ScanItem rescans one directory-backed ABS library item.
func (c *Client) ScanItem(ctx context.Context, itemID string) (*ScanItemResponse, error) {
	var response ScanItemResponse
	path := fmt.Sprintf("/api/items/%s/scan", url.PathEscape(itemID))
	if err := c.do(ctx, http.MethodPost, path, nil, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// Chapter is one audiobook chapter payload accepted by ABS.
type Chapter struct {
	Title string  `json:"title"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

// UpdateItemCover updates one ABS library item's cover from an ABS-visible path.
func (c *Client) UpdateItemCover(ctx context.Context, itemID string, cover string) (JSONValue, error) {
	var response any
	path := fmt.Sprintf("/api/items/%s/cover", url.PathEscape(itemID))
	payload := map[string]string{"cover": cover}
	if err := c.doJSON(ctx, http.MethodPatch, path, nil, payload, &response); err != nil {
		return nil, err
	}
	return response, nil
}

// RemoveItemCover removes one ABS library item's cover.
func (c *Client) RemoveItemCover(ctx context.Context, itemID string) error {
	path := fmt.Sprintf("/api/items/%s/cover", url.PathEscape(itemID))
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

// UpdateItemChapters replaces one ABS library item's chapters.
func (c *Client) UpdateItemChapters(ctx context.Context, itemID string, chapters []Chapter) (JSONValue, error) {
	var response any
	path := fmt.Sprintf("/api/items/%s/chapters", url.PathEscape(itemID))
	payload := map[string][]Chapter{"chapters": chapters}
	if err := c.doJSON(ctx, http.MethodPost, path, nil, payload, &response); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *Client) getJSON(
	ctx context.Context,
	path string,
	query url.Values,
	output any,
) error {
	requestURL := c.baseURL.ResolveReference(&url.URL{Path: path})
	if len(query) > 0 {
		requestURL.RawQuery = query.Encode()
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return fmt.Errorf("create ABS request: %w", err)
	}
	c.applyHeaders(request)

	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("call ABS %s: %w", request.URL.Path, err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return fmt.Errorf("ABS %s returned HTTP %d: %s", request.URL.Path, response.StatusCode, c.redact(strings.TrimSpace(string(body))))
	}

	if err := json.NewDecoder(response.Body).Decode(output); err != nil {
		return fmt.Errorf("decode ABS %s response: %w", request.URL.Path, err)
	}
	return nil
}

func (c *Client) doJSON(ctx context.Context, method string, path string, query url.Values, payload JSONValue, output any) error {
	requestURL := c.baseURL.ResolveReference(&url.URL{Path: path})
	if len(query) > 0 {
		requestURL.RawQuery = query.Encode()
	}

	if payload == nil {
		payload = map[string]any{}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode ABS %s request: %w", path, err)
	}
	if len(body) > maxJSONPayloadBytes {
		return fmt.Errorf("ABS %s request payload is %d bytes; maximum is %d bytes", path, len(body), maxJSONPayloadBytes)
	}

	request, err := http.NewRequestWithContext(ctx, method, requestURL.String(), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create ABS request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	c.applyHeaders(request)

	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("call ABS %s: %w", request.URL.Path, err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return fmt.Errorf("ABS %s returned HTTP %d: %s", request.URL.Path, response.StatusCode, c.redact(strings.TrimSpace(string(body))))
	}

	if output == nil {
		io.Copy(io.Discard, response.Body)
		return nil
	}
	if err := json.NewDecoder(response.Body).Decode(output); err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("decode ABS %s response: %w", request.URL.Path, err)
	}
	return nil
}

func (c *Client) do(ctx context.Context, method string, path string, query url.Values, output any) error {
	requestURL := c.baseURL.ResolveReference(&url.URL{Path: path})
	if len(query) > 0 {
		requestURL.RawQuery = query.Encode()
	}

	request, err := http.NewRequestWithContext(ctx, method, requestURL.String(), nil)
	if err != nil {
		return fmt.Errorf("create ABS request: %w", err)
	}
	c.applyHeaders(request)

	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("call ABS %s: %w", request.URL.Path, err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return fmt.Errorf("ABS %s returned HTTP %d: %s", request.URL.Path, response.StatusCode, c.redact(strings.TrimSpace(string(body))))
	}

	if output == nil {
		io.Copy(io.Discard, response.Body)
		return nil
	}
	if err := json.NewDecoder(response.Body).Decode(output); err != nil {
		return fmt.Errorf("decode ABS %s response: %w", request.URL.Path, err)
	}
	return nil
}

func (c *Client) applyHeaders(request *http.Request) {
	for name, value := range c.extraHeaders {
		request.Header.Set(name, value)
	}
	if c.token != "" {
		request.Header.Set("Authorization", "Bearer "+c.token)
	}
}

func (c *Client) redact(value string) string {
	if c.token == "" {
		return value
	}
	value = strings.ReplaceAll(value, "Bearer "+c.token, "Bearer [REDACTED]")
	return strings.ReplaceAll(value, c.token, "[REDACTED]")
}

func normalizeExtraHeaderName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("extra header name is required")
	}
	if strings.EqualFold(name, "Authorization") {
		return "", fmt.Errorf("extra headers must not contain Authorization")
	}
	for _, r := range name {
		if r <= 32 || r >= 127 || strings.ContainsRune("()<>@,;:\\\"/[]?={}", r) {
			return "", fmt.Errorf("extra header name %q is invalid", name)
		}
	}
	return http.CanonicalHeaderKey(name), nil
}

func catalogListQuery(options CatalogListOptions) url.Values {
	query := url.Values{}
	if options.Limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", options.Limit))
		query.Set("page", fmt.Sprintf("%d", options.Page))
	}
	if options.Sort != "" {
		query.Set("sort", options.Sort)
	}
	if options.Desc {
		query.Set("desc", "1")
	}
	if options.Filter != "" {
		query.Set("filter", options.Filter)
	}
	if len(options.Include) > 0 {
		query.Set("include", strings.Join(options.Include, ","))
	}
	if options.Minified {
		query.Set("minified", "1")
	}
	return query
}

func includeQuery(include []string) url.Values {
	if len(include) == 0 {
		return nil
	}
	query := url.Values{}
	query.Set("include", strings.Join(include, ","))
	return query
}
