package mcpserver

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jeeftor/abs-mcp/internal/config"
)

func TestReadServerInfoResource(t *testing.T) {
	t.Parallel()

	result, err := newTestServer().ReadServerInfoResource(context.Background(), readResourceRequest("abs://server/info"))
	if err != nil {
		t.Fatalf("ReadServerInfoResource failed: %v", err)
	}
	var output HealthOutput
	unmarshalResource(t, result, &output)
	if !output.OK || output.Username != "root" {
		t.Fatalf("unexpected health resource: %#v", output)
	}
}

func TestReadLibrariesResource(t *testing.T) {
	t.Parallel()

	result, err := newTestServer().ReadLibrariesResource(context.Background(), readResourceRequest("abs://libraries"))
	if err != nil {
		t.Fatalf("ReadLibrariesResource failed: %v", err)
	}
	var output LibrariesOutput
	unmarshalResource(t, result, &output)
	if output.Count != 2 {
		t.Fatalf("Count = %d, want 2", output.Count)
	}
}

func TestReadLibraryResource(t *testing.T) {
	t.Parallel()

	result, err := newTestServer().ReadLibraryResource(context.Background(), readResourceRequest("abs://libraries/lib-books"))
	if err != nil {
		t.Fatalf("ReadLibraryResource failed: %v", err)
	}
	var output LibraryOutput
	unmarshalResource(t, result, &output)
	if output.Library.Name != "Ebooks" {
		t.Fatalf("library name = %q, want Ebooks", output.Library.Name)
	}
}

func TestReadLibraryItemsResource(t *testing.T) {
	t.Parallel()

	result, err := newTestServer().ReadLibraryItemsResource(context.Background(), readResourceRequest("abs://libraries/lib-audio/items?limit=2&offset=2&sort=media.metadata.title&desc=1&filter=issues.true&include=rssfeed,progress&minified=1&collapseSeries=true"))
	if err != nil {
		t.Fatalf("ReadLibraryItemsResource failed: %v", err)
	}
	var output LibraryItemsOutput
	unmarshalResource(t, result, &output)
	if output.Total != 3 || output.Count != 1 {
		t.Fatalf("Total/Count = %d/%d, want 3/1", output.Total, output.Count)
	}
	if output.Page != 1 || output.Sort != "media.metadata.title" || !output.Desc || output.Filter != "issues.true" {
		t.Fatalf("unexpected library items resource output: %#v", output)
	}
}

func TestReadItemResource(t *testing.T) {
	t.Parallel()

	result, err := newTestServer().ReadItemResource(context.Background(), readResourceRequest("abs://items/item-1"))
	if err != nil {
		t.Fatalf("ReadItemResource failed: %v", err)
	}
	var output LibraryItemOutput
	unmarshalResource(t, result, &output)
	if output.Item.Title != "Alice" {
		t.Fatalf("title = %q, want Alice", output.Item.Title)
	}
}

func TestReadLibraryStatsResource(t *testing.T) {
	t.Parallel()

	result, err := newTestServer().ReadLibraryStatsResource(context.Background(), readResourceRequest("abs://libraries/lib-audio/stats"))
	if err != nil {
		t.Fatalf("ReadLibraryStatsResource failed: %v", err)
	}
	var output LibraryRawOutput
	unmarshalResource(t, result, &output)
	if output.Data == nil {
		t.Fatal("expected stats data")
	}
}

func TestReadLibraryFilterDataResource(t *testing.T) {
	t.Parallel()

	result, err := newTestServer().ReadLibraryFilterDataResource(context.Background(), readResourceRequest("abs://libraries/lib-audio/filterdata"))
	if err != nil {
		t.Fatalf("ReadLibraryFilterDataResource failed: %v", err)
	}
	var output LibraryRawOutput
	unmarshalResource(t, result, &output)
	if output.Data == nil {
		t.Fatal("expected filter data")
	}
}

func TestReadItemMetadataObjectResource(t *testing.T) {
	t.Parallel()

	result, err := newTestServer().ReadItemMetadataObjectResource(context.Background(), readResourceRequest("abs://items/item-1/metadata-object"))
	if err != nil {
		t.Fatalf("ReadItemMetadataObjectResource failed: %v", err)
	}
	var output MetadataObjectOutput
	unmarshalResource(t, result, &output)
	if output.Data == nil {
		t.Fatal("expected metadata object data")
	}
}

func TestReadAPIInventoryResource(t *testing.T) {
	t.Parallel()

	result, err := newTestServer().ReadAPIInventoryResource(context.Background(), readResourceRequest("abs://api-inventory/current"))
	if err != nil {
		t.Fatalf("ReadAPIInventoryResource failed: %v", err)
	}
	text := result.Contents[0].Text
	if !strings.Contains(text, "total_routes") && !strings.Contains(text, "available") {
		t.Fatalf("unexpected API inventory resource text: %s", text)
	}
}

func TestReadFixtureStatusResource(t *testing.T) {
	t.Parallel()

	fixtureDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(fixtureDir, "docker-compose.yml"), []byte("services: {}\n"), 0o600); err != nil {
		t.Fatalf("write docker-compose.yml: %v", err)
	}
	env := strings.Join([]string{
		"ABS_PLAIN_URL=http://localhost:13388",
		"ABS_METADATA_URL=http://localhost:13389",
		"ABS_TOKEN=secret-token",
		"ABS_PLAIN_SQLITE=test/abs/state/plain/config/absdatabase.sqlite",
		"ABS_METADATA_SQLITE=test/abs/state/metadata-enabled/config/absdatabase.sqlite",
		"ABS_EXPECT_AUDIOBOOKS=2",
		"ABS_EXPECT_BOOKS=3",
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(fixtureDir, ".env.testing"), []byte(env), 0o600); err != nil {
		t.Fatalf("write .env.testing: %v", err)
	}

	server := New(config.Config{
		ABSBaseURL: "http://abs",
		ReadOnly:   true,
		FixtureDir: fixtureDir,
	}, newFakeABSClient())
	result, err := server.ReadFixtureStatusResource(context.Background(), readResourceRequest("abs://fixture/status"))
	if err != nil {
		t.Fatalf("ReadFixtureStatusResource failed: %v", err)
	}
	var output FixtureStatusOutput
	unmarshalResource(t, result, &output)
	if !output.Configured || !output.Exists || !output.ComposeFilePresent || !output.EnvFilePresent {
		t.Fatalf("unexpected fixture file status: %#v", output)
	}
	if output.PlainURL != "http://localhost:13388" || output.MetadataURL != "http://localhost:13389" {
		t.Fatalf("unexpected fixture URLs: %#v", output)
	}
	if !output.TokenConfigured || output.TokenLength != len("secret-token") {
		t.Fatalf("unexpected token status: %#v", output)
	}
	if strings.Contains(result.Contents[0].Text, "secret-token") {
		t.Fatalf("fixture status leaked token: %s", result.Contents[0].Text)
	}
	if output.ExpectedAudiobooks != 2 || output.ExpectedBooks != 3 {
		t.Fatalf("unexpected expected counts: %#v", output)
	}
}

func TestReadFixtureStatusResourceWhenMissing(t *testing.T) {
	t.Parallel()

	server := New(config.Config{
		ABSBaseURL: "http://abs",
		ReadOnly:   true,
		FixtureDir: filepath.Join(t.TempDir(), "missing"),
	}, newFakeABSClient())
	result, err := server.ReadFixtureStatusResource(context.Background(), readResourceRequest("abs://fixture/status"))
	if err != nil {
		t.Fatalf("ReadFixtureStatusResource failed: %v", err)
	}
	var output FixtureStatusOutput
	unmarshalResource(t, result, &output)
	if !output.Configured || output.Exists || output.EnvFilePresent || output.ComposeFilePresent {
		t.Fatalf("unexpected missing fixture status: %#v", output)
	}
}

func TestReadResourceRejectsWrongURI(t *testing.T) {
	t.Parallel()

	if _, err := newTestServer().ReadLibrariesResource(context.Background(), readResourceRequest("abs://items/item-1")); err == nil {
		t.Fatal("expected resource not found error")
	}
	if _, err := newTestServer().ReadLibraryItemsResource(context.Background(), readResourceRequest("abs://libraries/lib-audio/items?limit=nope")); err == nil {
		t.Fatal("expected invalid query error")
	}
	if _, err := newTestServer().ReadFixtureStatusResource(context.Background(), readResourceRequest("abs://libraries")); err == nil {
		t.Fatal("expected fixture resource not found error")
	}
}

func readResourceRequest(uri string) *mcp.ReadResourceRequest {
	return &mcp.ReadResourceRequest{Params: &mcp.ReadResourceParams{URI: uri}}
}

func unmarshalResource(t *testing.T, result *mcp.ReadResourceResult, target any) {
	t.Helper()
	if len(result.Contents) != 1 {
		t.Fatalf("resource contents length = %d, want 1", len(result.Contents))
	}
	if result.Contents[0].MIMEType != resourceMIMETypeJSON {
		t.Fatalf("resource MIME type = %q", result.Contents[0].MIMEType)
	}
	if err := json.Unmarshal([]byte(result.Contents[0].Text), target); err != nil {
		t.Fatalf("unmarshal resource JSON: %v\n%s", err, result.Contents[0].Text)
	}
}
