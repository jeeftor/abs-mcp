package main

import (
	"context"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jeeftor/abs-mcp/internal/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestCommandServesMCPOverStdio(t *testing.T) {
	t.Parallel()

	absServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer test-token" {
			http.Error(writer, "unauthorized", http.StatusUnauthorized)
			return
		}
		if request.Header.Get("X-Corp-Trace") != "trace-stdio" {
			http.Error(writer, "missing extra header", http.StatusForbidden)
			return
		}
		switch request.URL.Path {
		case "/api/me":
			writeJSON(t, writer, map[string]any{
				"id":       "user-1",
				"username": "root",
				"type":     "root",
				"isActive": true,
			})
		case "/api/libraries":
			writeJSON(t, writer, map[string]any{
				"libraries": []map[string]any{
					{
						"id":        "lib-audio",
						"name":      "Audiobooks",
						"mediaType": "book",
						"folders":   []map[string]any{{"id": "folder-audio", "fullPath": "/audiobooks"}},
					},
				},
			})
		default:
			http.NotFound(writer, request)
		}
	}))
	defer absServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	headersPath := filepath.Join(t.TempDir(), "headers.json")
	if err := os.WriteFile(headersPath, []byte(`{"X-Corp-Trace":"trace-stdio"}`), 0o600); err != nil {
		t.Fatalf("write headers file: %v", err)
	}

	command := exec.CommandContext(ctx, "go", "run", ".")
	command.Env = append(os.Environ(),
		"ABS_BASE_URL="+absServer.URL,
		"ABS_API_KEY=test-token",
		"ABS_READ_ONLY=true",
		"ABS_EXTRA_HEADERS_FILE="+headersPath,
	)

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "abs-mcp-command-test",
		Version: "0.1.0",
	}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: command}, nil)
	if err != nil {
		t.Fatalf("connect to command transport: %v", err)
	}
	defer session.Close()

	tools, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	if len(tools.Tools) < 6 {
		t.Fatalf("tool count = %d, want at least 6", len(tools.Tools))
	}

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "abs_health_check",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("call abs_health_check: %v", err)
	}
	if result.IsError {
		t.Fatalf("abs_health_check returned tool error: %#v", result.Content)
	}

	var output struct {
		OK           bool   `json:"ok"`
		Username     string `json:"username"`
		LibraryCount int    `json:"libraryCount"`
	}
	mustUnmarshalStructured(t, result.StructuredContent, &output)
	if !output.OK || output.Username != "root" || output.LibraryCount != 1 {
		t.Fatalf("unexpected health output: %#v", output)
	}
}

func TestCommandAuthFailureDoesNotLeakToken(t *testing.T) {
	t.Parallel()

	const badToken = "bad-token-secret"
	absServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.Error(writer, "unauthorized "+request.Header.Get("Authorization"), http.StatusUnauthorized)
	}))
	defer absServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	command := exec.CommandContext(ctx, "go", "run", ".")
	command.Env = append(os.Environ(),
		"ABS_BASE_URL="+absServer.URL,
		"ABS_API_KEY="+badToken,
		"ABS_READ_ONLY=true",
	)

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "abs-mcp-command-auth-test",
		Version: "0.1.0",
	}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: command}, nil)
	if err != nil {
		t.Fatalf("connect to command transport: %v", err)
	}
	defer session.Close()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "abs_health_check",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("call abs_health_check: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected abs_health_check to fail")
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal tool error: %v", err)
	}
	if strings.Contains(string(resultJSON), badToken) {
		t.Fatalf("tool error leaked token: %s", resultJSON)
	}
}

func TestRootCommandUsesEnvConfiguration(t *testing.T) {
	t.Setenv("ABS_BASE_URL", "http://localhost:13388")
	t.Setenv("ABS_API_KEY", "env-token")
	t.Setenv("ABS_TIMEOUT", "20s")
	t.Setenv("ABS_READ_ONLY", "false")
	t.Setenv("ABS_FIXTURE_DIR", "/tmp/env-fixture")

	var got config.Config
	command := newRootCommand(context.Background(), func(ctx context.Context, cfg config.Config) error {
		got = cfg
		return nil
	})

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext failed: %v", err)
	}
	if got.ABSBaseURL != "http://localhost:13388" {
		t.Fatalf("ABSBaseURL = %q", got.ABSBaseURL)
	}
	if got.ABSAPIKey != "env-token" {
		t.Fatalf("ABSAPIKey = %q", got.ABSAPIKey)
	}
	if got.Timeout != 20*time.Second {
		t.Fatalf("Timeout = %s, want 20s", got.Timeout)
	}
	if got.ReadOnly {
		t.Fatal("ReadOnly = true, want false")
	}
	if got.FixtureDir != "/tmp/env-fixture" {
		t.Fatalf("FixtureDir = %q", got.FixtureDir)
	}
}

func TestRootCommandFlagsOverrideEnvConfiguration(t *testing.T) {
	t.Setenv("ABS_BASE_URL", "http://env.example")
	t.Setenv("ABS_API_KEY", "env-token")
	t.Setenv("ABS_TIMEOUT", "10s")
	t.Setenv("ABS_READ_ONLY", "true")
	t.Setenv("ABS_FIXTURE_DIR", "/tmp/env-fixture")

	headersPath := filepath.Join(t.TempDir(), "headers.json")
	if err := os.WriteFile(headersPath, []byte(`{"X-Corp-Trace":"trace-flags"}`), 0o600); err != nil {
		t.Fatalf("write headers file: %v", err)
	}

	var got config.Config
	command := newRootCommand(context.Background(), func(ctx context.Context, cfg config.Config) error {
		got = cfg
		return nil
	})
	command.SetArgs([]string{
		"--base-url", "http://flag.example",
		"--api-key", "flag-token",
		"--timeout", "45s",
		"--read-only=false",
		"--fixture-dir", "/tmp/flag-fixture",
		"--extra-headers-file", headersPath,
		"--header", "X-Corp-Trace=trace-header",
		"--header", "X-Other=ok",
		"--tls-ca-cert-file", "/tmp/abs-ca.pem",
		"--tls-insecure-skip-verify",
	})

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext failed: %v", err)
	}
	if got.ABSBaseURL != "http://flag.example" {
		t.Fatalf("ABSBaseURL = %q", got.ABSBaseURL)
	}
	if got.ABSAPIKey != "flag-token" {
		t.Fatalf("ABSAPIKey = %q", got.ABSAPIKey)
	}
	if got.Timeout != 45*time.Second {
		t.Fatalf("Timeout = %s, want 45s", got.Timeout)
	}
	if got.ReadOnly {
		t.Fatal("ReadOnly = true, want false")
	}
	if got.FixtureDir != "/tmp/flag-fixture" {
		t.Fatalf("FixtureDir = %q", got.FixtureDir)
	}
	if got.ExtraHeadersFile != headersPath {
		t.Fatalf("ExtraHeadersFile = %q", got.ExtraHeadersFile)
	}
	if got.ExtraHeaders["X-Corp-Trace"] != "trace-header" {
		t.Fatalf("X-Corp-Trace = %q", got.ExtraHeaders["X-Corp-Trace"])
	}
	if got.ExtraHeaders["X-Other"] != "ok" {
		t.Fatalf("X-Other = %q", got.ExtraHeaders["X-Other"])
	}
	if got.TLSCACertFile != "/tmp/abs-ca.pem" {
		t.Fatalf("TLSCACertFile = %q", got.TLSCACertFile)
	}
	if !got.TLSSkipVerify {
		t.Fatal("TLSSkipVerify = false, want true")
	}
}

func TestRootCommandUsesEnvFileConfiguration(t *testing.T) {
	envPath := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(envPath, []byte(strings.Join([]string{
		"ABS_BASE_URL=http://env-file.example",
		"ABS_API_KEY=env-file-token",
		"ABS_TIMEOUT=35s",
		"ABS_READ_ONLY=false",
		"ABS_FIXTURE_DIR=/tmp/env-file-fixture",
		"ABS_TLS_INSECURE_SKIP_VERIFY=true",
	}, "\n")), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	var got config.Config
	command := newRootCommand(context.Background(), func(ctx context.Context, cfg config.Config) error {
		got = cfg
		return nil
	})
	command.SetArgs([]string{"--env-file", envPath})

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext failed: %v", err)
	}
	if got.ABSBaseURL != "http://env-file.example" {
		t.Fatalf("ABSBaseURL = %q", got.ABSBaseURL)
	}
	if got.ABSAPIKey != "env-file-token" {
		t.Fatalf("ABSAPIKey = %q", got.ABSAPIKey)
	}
	if got.Timeout != 35*time.Second {
		t.Fatalf("Timeout = %s, want 35s", got.Timeout)
	}
	if got.ReadOnly {
		t.Fatal("ReadOnly = true, want false")
	}
	if got.FixtureDir != "/tmp/env-file-fixture" {
		t.Fatalf("FixtureDir = %q", got.FixtureDir)
	}
	if !got.TLSSkipVerify {
		t.Fatal("TLSSkipVerify = false, want true")
	}
}

func TestRootCommandPrecedenceFlagsEnvThenEnvFile(t *testing.T) {
	envPath := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(envPath, []byte(strings.Join([]string{
		"ABS_BASE_URL=http://env-file.example",
		"ABS_API_KEY=env-file-token",
		"ABS_READ_ONLY=true",
	}, "\n")), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}
	t.Setenv("ABS_BASE_URL", "http://process-env.example")

	var got config.Config
	command := newRootCommand(context.Background(), func(ctx context.Context, cfg config.Config) error {
		got = cfg
		return nil
	})
	command.SetArgs([]string{
		"--env-file", envPath,
		"--base-url", "http://flag.example",
		"--read-only=false",
	})

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext failed: %v", err)
	}
	if got.ABSBaseURL != "http://flag.example" {
		t.Fatalf("ABSBaseURL = %q", got.ABSBaseURL)
	}
	if got.ABSAPIKey != "env-file-token" {
		t.Fatalf("ABSAPIKey = %q", got.ABSAPIKey)
	}
	if got.ReadOnly {
		t.Fatal("ReadOnly = true, want false")
	}
}

func TestHTTPClientTrustsConfiguredCACertFile(t *testing.T) {
	t.Parallel()

	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	certPath := filepath.Join(t.TempDir(), "ca.pem")
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: server.Certificate().Raw,
	})
	if err := os.WriteFile(certPath, certPEM, 0o600); err != nil {
		t.Fatalf("write CA file: %v", err)
	}

	client, err := newHTTPClient(config.Config{
		Timeout:       time.Second,
		TLSCACertFile: certPath,
	})
	if err != nil {
		t.Fatalf("newHTTPClient failed: %v", err)
	}
	response, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("GET with configured CA failed: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusNoContent)
	}
}

func TestHTTPClientCanSkipTLSVerification(t *testing.T) {
	t.Parallel()

	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := newHTTPClient(config.Config{
		Timeout:       time.Second,
		TLSSkipVerify: true,
	})
	if err != nil {
		t.Fatalf("newHTTPClient failed: %v", err)
	}
	response, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("GET with skip verify failed: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusNoContent)
	}
}

func TestHTTPClientRejectsInvalidCACertFile(t *testing.T) {
	t.Parallel()

	certPath := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(certPath, []byte("not a certificate"), 0o600); err != nil {
		t.Fatalf("write CA file: %v", err)
	}

	_, err := newHTTPClient(config.Config{
		Timeout:       time.Second,
		TLSCACertFile: certPath,
	})
	if err == nil {
		t.Fatal("expected invalid CA file error")
	}
}

func writeJSON(t *testing.T, writer http.ResponseWriter, value any) {
	t.Helper()
	writer.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(writer).Encode(value); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}

func mustUnmarshalStructured(t *testing.T, value any, target any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal structured output: %v", err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatalf("unmarshal structured output: %v", err)
	}
}
