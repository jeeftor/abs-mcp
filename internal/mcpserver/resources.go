package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const resourceMIMETypeJSON = "application/json"

// RegisterResources adds the server's resource surface.
func (s *Server) RegisterResources(server *mcp.Server) {
	server.AddResource(&mcp.Resource{
		URI:         "abs://server/info",
		Name:        "abs_server_info",
		Title:       "Audiobookshelf server info",
		Description: "Sanitized authenticated Audiobookshelf server summary.",
		MIMEType:    resourceMIMETypeJSON,
	}, s.ReadServerInfoResource)
	server.AddResource(&mcp.Resource{
		URI:         "abs://libraries",
		Name:        "abs_libraries",
		Title:       "Audiobookshelf libraries",
		Description: "Visible Audiobookshelf libraries.",
		MIMEType:    resourceMIMETypeJSON,
	}, s.ReadLibrariesResource)
	server.AddResource(&mcp.Resource{
		URI:         "abs://api-inventory/current",
		Name:        "abs_api_inventory_current",
		Title:       "Audiobookshelf API inventory",
		Description: "Generated source-backed Audiobookshelf API route inventory.",
		MIMEType:    resourceMIMETypeJSON,
	}, s.ReadAPIInventoryResource)
	server.AddResource(&mcp.Resource{
		URI:         "abs://fixture/status",
		Name:        "abs_fixture_status",
		Title:       "Audiobookshelf fixture status",
		Description: "Local Docker fixture configuration status without exposing secrets.",
		MIMEType:    resourceMIMETypeJSON,
	}, s.ReadFixtureStatusResource)
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "abs://libraries/{library_id}",
		Name:        "abs_library",
		Title:       "Audiobookshelf library",
		Description: "One Audiobookshelf library by ID.",
		MIMEType:    resourceMIMETypeJSON,
	}, s.ReadLibraryResource)
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "abs://libraries/{library_id}/items{?limit,offset}",
		Name:        "abs_library_items",
		Title:       "Audiobookshelf library items",
		Description: "A bounded page of Audiobookshelf library items.",
		MIMEType:    resourceMIMETypeJSON,
	}, s.ReadLibraryItemsResource)
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "abs://libraries/{library_id}/stats",
		Name:        "abs_library_stats",
		Title:       "Audiobookshelf library stats",
		Description: "Raw Audiobookshelf stats for one library.",
		MIMEType:    resourceMIMETypeJSON,
	}, s.ReadLibraryStatsResource)
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "abs://libraries/{library_id}/filterdata",
		Name:        "abs_library_filter_data",
		Title:       "Audiobookshelf library filter data",
		Description: "Raw Audiobookshelf filter data for one library.",
		MIMEType:    resourceMIMETypeJSON,
	}, s.ReadLibraryFilterDataResource)
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "abs://items/{item_id}",
		Name:        "abs_item",
		Title:       "Audiobookshelf item",
		Description: "One Audiobookshelf library item by ID.",
		MIMEType:    resourceMIMETypeJSON,
	}, s.ReadItemResource)
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "abs://items/{item_id}/metadata-object",
		Name:        "abs_item_metadata_object",
		Title:       "Audiobookshelf item metadata object",
		Description: "Raw Audiobookshelf metadata object for one item.",
		MIMEType:    resourceMIMETypeJSON,
	}, s.ReadItemMetadataObjectResource)
}

// ReadServerInfoResource reads abs://server/info.
func (s *Server) ReadServerInfoResource(ctx context.Context, request *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	if request.Params.URI != "abs://server/info" {
		return nil, mcp.ResourceNotFoundError(request.Params.URI)
	}
	_, output, err := s.HealthCheck(ctx, nil, EmptyInput{})
	if err != nil {
		return nil, err
	}
	return jsonResource(request.Params.URI, output)
}

// ReadLibrariesResource reads abs://libraries.
func (s *Server) ReadLibrariesResource(ctx context.Context, request *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	if request.Params.URI != "abs://libraries" {
		return nil, mcp.ResourceNotFoundError(request.Params.URI)
	}
	_, output, err := s.ListLibraries(ctx, nil, EmptyInput{})
	if err != nil {
		return nil, err
	}
	return jsonResource(request.Params.URI, output)
}

// ReadLibraryResource reads abs://libraries/{library_id}.
func (s *Server) ReadLibraryResource(ctx context.Context, request *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	parsed, err := parseABSURI(request.Params.URI)
	if err != nil {
		return nil, mcp.ResourceNotFoundError(request.Params.URI)
	}
	parts := splitPath(parsed.Path)
	if len(parts) != 1 || parsed.Host != "libraries" {
		return nil, mcp.ResourceNotFoundError(request.Params.URI)
	}
	_, output, err := s.GetLibrary(ctx, nil, LibraryInput{LibraryID: parts[0]})
	if err != nil {
		return nil, err
	}
	return jsonResource(request.Params.URI, output)
}

// ReadLibraryItemsResource reads abs://libraries/{library_id}/items.
func (s *Server) ReadLibraryItemsResource(ctx context.Context, request *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	parsed, err := parseABSURI(request.Params.URI)
	if err != nil {
		return nil, mcp.ResourceNotFoundError(request.Params.URI)
	}
	parts := splitPath(parsed.Path)
	if len(parts) != 2 || parsed.Host != "libraries" || parts[1] != "items" {
		return nil, mcp.ResourceNotFoundError(request.Params.URI)
	}
	limit, err := parseOptionalInt(parsed.Query().Get("limit"))
	if err != nil {
		return nil, err
	}
	offset, err := parseOptionalInt(parsed.Query().Get("offset"))
	if err != nil {
		return nil, err
	}
	_, output, err := s.ListLibraryItems(ctx, nil, LibraryItemsInput{
		LibraryID:      parts[0],
		Limit:          limit,
		Offset:         offset,
		Sort:           parsed.Query().Get("sort"),
		Desc:           parsed.Query().Get("desc") == "1" || strings.EqualFold(parsed.Query().Get("desc"), "true"),
		Filter:         parsed.Query().Get("filter"),
		Include:        splitComma(parsed.Query().Get("include")),
		Minified:       parsed.Query().Get("minified") == "1" || strings.EqualFold(parsed.Query().Get("minified"), "true"),
		CollapseSeries: parsed.Query().Get("collapseseries") == "1" || strings.EqualFold(parsed.Query().Get("collapseSeries"), "true"),
	})
	if err != nil {
		return nil, err
	}
	return jsonResource(request.Params.URI, output)
}

// ReadItemResource reads abs://items/{item_id}.
func (s *Server) ReadItemResource(ctx context.Context, request *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	parsed, err := parseABSURI(request.Params.URI)
	if err != nil {
		return nil, mcp.ResourceNotFoundError(request.Params.URI)
	}
	parts := splitPath(parsed.Path)
	if len(parts) != 1 || parsed.Host != "items" {
		return nil, mcp.ResourceNotFoundError(request.Params.URI)
	}
	_, output, err := s.GetLibraryItem(ctx, nil, LibraryItemInput{ItemID: parts[0]})
	if err != nil {
		return nil, err
	}
	return jsonResource(request.Params.URI, output)
}

// ReadLibraryStatsResource reads abs://libraries/{library_id}/stats.
func (s *Server) ReadLibraryStatsResource(ctx context.Context, request *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	parsed, err := parseABSURI(request.Params.URI)
	if err != nil {
		return nil, mcp.ResourceNotFoundError(request.Params.URI)
	}
	parts := splitPath(parsed.Path)
	if len(parts) != 2 || parsed.Host != "libraries" || parts[1] != "stats" {
		return nil, mcp.ResourceNotFoundError(request.Params.URI)
	}
	_, output, err := s.GetLibraryStats(ctx, nil, LibraryRawInput{LibraryID: parts[0]})
	if err != nil {
		return nil, err
	}
	return jsonResource(request.Params.URI, output)
}

// ReadLibraryFilterDataResource reads abs://libraries/{library_id}/filterdata.
func (s *Server) ReadLibraryFilterDataResource(ctx context.Context, request *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	parsed, err := parseABSURI(request.Params.URI)
	if err != nil {
		return nil, mcp.ResourceNotFoundError(request.Params.URI)
	}
	parts := splitPath(parsed.Path)
	if len(parts) != 2 || parsed.Host != "libraries" || parts[1] != "filterdata" {
		return nil, mcp.ResourceNotFoundError(request.Params.URI)
	}
	_, output, err := s.GetLibraryFilterData(ctx, nil, LibraryRawInput{LibraryID: parts[0]})
	if err != nil {
		return nil, err
	}
	return jsonResource(request.Params.URI, output)
}

// ReadItemMetadataObjectResource reads abs://items/{item_id}/metadata-object.
func (s *Server) ReadItemMetadataObjectResource(ctx context.Context, request *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	parsed, err := parseABSURI(request.Params.URI)
	if err != nil {
		return nil, mcp.ResourceNotFoundError(request.Params.URI)
	}
	parts := splitPath(parsed.Path)
	if len(parts) != 2 || parsed.Host != "items" || parts[1] != "metadata-object" {
		return nil, mcp.ResourceNotFoundError(request.Params.URI)
	}
	_, output, err := s.GetItemMetadataObject(ctx, nil, LibraryItemInput{ItemID: parts[0]})
	if err != nil {
		return nil, err
	}
	return jsonResource(request.Params.URI, output)
}

// ReadAPIInventoryResource reads abs://api-inventory/current.
func (s *Server) ReadAPIInventoryResource(_ context.Context, request *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	if request.Params.URI != "abs://api-inventory/current" {
		return nil, mcp.ResourceNotFoundError(request.Params.URI)
	}
	return jsonResource(request.Params.URI, s.apiInventory)
}

// FixtureStatusOutput describes the local Docker ABS fixture without secrets.
type FixtureStatusOutput struct {
	Configured         bool   `json:"configured"`
	FixtureDir         string `json:"fixtureDir,omitempty"`
	Exists             bool   `json:"exists"`
	ComposeFilePresent bool   `json:"composeFilePresent"`
	EnvFilePresent     bool   `json:"envFilePresent"`
	PlainURL           string `json:"plainUrl,omitempty"`
	MetadataURL        string `json:"metadataUrl,omitempty"`
	PlainSQLite        string `json:"plainSqlite,omitempty"`
	MetadataSQLite     string `json:"metadataSqlite,omitempty"`
	TokenConfigured    bool   `json:"tokenConfigured"`
	TokenLength        int    `json:"tokenLength,omitempty"`
	ExpectedAudiobooks int    `json:"expectedAudiobooks,omitempty"`
	ExpectedBooks      int    `json:"expectedBooks,omitempty"`
}

// ReadFixtureStatusResource reads abs://fixture/status.
func (s *Server) ReadFixtureStatusResource(_ context.Context, request *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	if request.Params.URI != "abs://fixture/status" {
		return nil, mcp.ResourceNotFoundError(request.Params.URI)
	}
	status := s.fixtureStatus()
	return jsonResource(request.Params.URI, status)
}

func (s *Server) fixtureStatus() FixtureStatusOutput {
	status := FixtureStatusOutput{
		Configured: s.cfg.FixtureDir != "",
		FixtureDir: s.cfg.FixtureDir,
	}
	if s.cfg.FixtureDir == "" {
		return status
	}

	if info, err := os.Stat(s.cfg.FixtureDir); err == nil && info.IsDir() {
		status.Exists = true
	}
	status.ComposeFilePresent = regularFileExists(filepath.Join(s.cfg.FixtureDir, "docker-compose.yml"))
	envPath := filepath.Join(s.cfg.FixtureDir, ".env.testing")
	status.EnvFilePresent = regularFileExists(envPath)
	if !status.EnvFilePresent {
		return status
	}

	values, err := readFixtureEnvFile(envPath)
	if err != nil {
		return status
	}
	status.PlainURL = values["ABS_PLAIN_URL"]
	status.MetadataURL = values["ABS_METADATA_URL"]
	status.PlainSQLite = values["ABS_PLAIN_SQLITE"]
	status.MetadataSQLite = values["ABS_METADATA_SQLITE"]
	token := values["ABS_TOKEN"]
	status.TokenConfigured = token != ""
	status.TokenLength = len(token)
	status.ExpectedAudiobooks = parseEnvInt(values["ABS_EXPECT_AUDIOBOOKS"])
	status.ExpectedBooks = parseEnvInt(values["ABS_EXPECT_BOOKS"])
	return status
}

func jsonResource(uri string, value any) (*mcp.ReadResourceResult, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal resource %s: %w", uri, err)
	}
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      uri,
				MIMEType: resourceMIMETypeJSON,
				Text:     string(data),
			},
		},
	}, nil
}

func parseABSURI(rawURI string) (*url.URL, error) {
	parsed, err := url.Parse(rawURI)
	if err != nil {
		return nil, err
	}
	if parsed.Scheme != "abs" {
		return nil, fmt.Errorf("unsupported resource scheme: %s", parsed.Scheme)
	}
	return parsed, nil
}

func splitPath(path string) []string {
	path = strings.Trim(path, "/")
	if path == "" {
		return nil
	}
	return strings.Split(path, "/")
}

func parseOptionalInt(value string) (int, error) {
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("resource query value %q must be an integer", value)
	}
	return parsed, nil
}

func splitComma(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			values = append(values, part)
		}
	}
	return values
}

func regularFileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func readFixtureEnvFile(path string) (map[string]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	values := make(map[string]string)
	for _, rawLine := range strings.Split(string(content), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		values[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"'`)
	}
	return values, nil
}

func parseEnvInt(value string) int {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return parsed
}
