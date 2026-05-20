package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadFromEnv(t *testing.T) {
	t.Parallel()

	cfg, err := LoadFromEnv(mapLookup(map[string]string{
		"ABS_BASE_URL":    "http://localhost:13388",
		"ABS_API_KEY":     "test-token",
		"ABS_TIMEOUT":     "45s",
		"ABS_READ_ONLY":   "false",
		"ABS_FIXTURE_DIR": "/tmp/abs-fixture",
	}))
	if err != nil {
		t.Fatalf("LoadFromEnv failed: %v", err)
	}
	if cfg.ABSBaseURL != "http://localhost:13388" {
		t.Fatalf("ABSBaseURL = %q", cfg.ABSBaseURL)
	}
	if cfg.ABSAPIKey != "test-token" {
		t.Fatalf("ABSAPIKey = %q", cfg.ABSAPIKey)
	}
	if cfg.Timeout != 45*time.Second {
		t.Fatalf("Timeout = %s, want 45s", cfg.Timeout)
	}
	if cfg.ReadOnly {
		t.Fatal("ReadOnly = true, want false")
	}
	if cfg.FixtureDir != "/tmp/abs-fixture" {
		t.Fatalf("FixtureDir = %q", cfg.FixtureDir)
	}
}

func TestLoadFromEnvLoadsExtraHeadersFile(t *testing.T) {
	t.Parallel()

	headersPath := filepath.Join(t.TempDir(), "headers.json")
	if err := os.WriteFile(headersPath, []byte(`{"X-Corp-Trace":"trace-1","X-Test":"ok"}`), 0o600); err != nil {
		t.Fatalf("write headers file: %v", err)
	}

	cfg, err := LoadFromEnv(mapLookup(map[string]string{
		"ABS_BASE_URL":           "http://localhost:13388",
		"ABS_API_KEY":            "test-token",
		"ABS_EXTRA_HEADERS_FILE": headersPath,
	}))
	if err != nil {
		t.Fatalf("LoadFromEnv failed: %v", err)
	}
	if cfg.ExtraHeadersFile != headersPath {
		t.Fatalf("ExtraHeadersFile = %q", cfg.ExtraHeadersFile)
	}
	if cfg.ExtraHeaders["X-Corp-Trace"] != "trace-1" {
		t.Fatalf("X-Corp-Trace = %q", cfg.ExtraHeaders["X-Corp-Trace"])
	}
	if cfg.ExtraHeaders["X-Test"] != "ok" {
		t.Fatalf("X-Test = %q", cfg.ExtraHeaders["X-Test"])
	}
}

func TestLoadFromEnvDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := LoadFromEnv(mapLookup(map[string]string{
		"ABS_BASE_URL": "http://localhost:13388",
		"ABS_API_KEY":  "test-token",
	}))
	if err != nil {
		t.Fatalf("LoadFromEnv failed: %v", err)
	}
	if cfg.Timeout != defaultTimeout {
		t.Fatalf("Timeout = %s, want %s", cfg.Timeout, defaultTimeout)
	}
	if !cfg.ReadOnly {
		t.Fatal("ReadOnly = false, want true")
	}
	if cfg.FixtureDir != "test/abs" {
		t.Fatalf("FixtureDir = %q, want test/abs", cfg.FixtureDir)
	}
	if cfg.ExtraHeadersFile != "" {
		t.Fatalf("ExtraHeadersFile = %q, want empty", cfg.ExtraHeadersFile)
	}
	if len(cfg.ExtraHeaders) != 0 {
		t.Fatalf("ExtraHeaders = %#v, want empty", cfg.ExtraHeaders)
	}
}

func TestLoadFromEnvRequiresURLAndAPIKey(t *testing.T) {
	t.Parallel()

	if _, err := LoadFromEnv(mapLookup(map[string]string{"ABS_API_KEY": "token"})); err == nil {
		t.Fatal("expected missing ABS_BASE_URL error")
	}
	if _, err := LoadFromEnv(mapLookup(map[string]string{"ABS_BASE_URL": "http://localhost:13388"})); err == nil {
		t.Fatal("expected missing ABS_API_KEY error")
	}
}

func TestLoadFromEnvRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	if _, err := LoadFromEnv(mapLookup(map[string]string{
		"ABS_BASE_URL": "http://localhost:13388",
		"ABS_API_KEY":  "test-token",
		"ABS_TIMEOUT":  "not-a-duration",
	})); err == nil {
		t.Fatal("expected invalid timeout error")
	}

	if _, err := LoadFromEnv(mapLookup(map[string]string{
		"ABS_BASE_URL":  "http://localhost:13388",
		"ABS_API_KEY":   "test-token",
		"ABS_READ_ONLY": "sometimes",
	})); err == nil {
		t.Fatal("expected invalid read-only error")
	}
}

func TestLoadFromEnvRejectsInvalidExtraHeadersFile(t *testing.T) {
	t.Parallel()

	for name, content := range map[string]string{
		"invalid JSON":         `{`,
		"non-string value":     `{"X-Test": 123}`,
		"authorization header": `{"Authorization": "Bearer other-token"}`,
		"empty header name":    `{"": "value"}`,
		"control header name":  "{\"X-Test\nBad\":\"value\"}",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			headersPath := filepath.Join(t.TempDir(), "headers.json")
			if err := os.WriteFile(headersPath, []byte(content), 0o600); err != nil {
				t.Fatalf("write headers file: %v", err)
			}
			if _, err := LoadFromEnv(mapLookup(map[string]string{
				"ABS_BASE_URL":           "http://localhost:13388",
				"ABS_API_KEY":            "test-token",
				"ABS_EXTRA_HEADERS_FILE": headersPath,
			})); err == nil {
				t.Fatal("expected invalid extra headers error")
			}
		})
	}
}

func TestLoadFromViperReadsEnv(t *testing.T) {
	t.Setenv("ABS_BASE_URL", "http://localhost:13388")
	t.Setenv("ABS_API_KEY", "test-token")
	t.Setenv("ABS_TIMEOUT", "15s")
	t.Setenv("ABS_READ_ONLY", "false")
	t.Setenv("ABS_FIXTURE_DIR", "/tmp/abs-fixture")

	cfg, err := LoadFromViper(NewViper())
	if err != nil {
		t.Fatalf("LoadFromViper failed: %v", err)
	}
	if cfg.ABSBaseURL != "http://localhost:13388" {
		t.Fatalf("ABSBaseURL = %q", cfg.ABSBaseURL)
	}
	if cfg.ABSAPIKey != "test-token" {
		t.Fatalf("ABSAPIKey = %q", cfg.ABSAPIKey)
	}
	if cfg.Timeout != 15*time.Second {
		t.Fatalf("Timeout = %s, want 15s", cfg.Timeout)
	}
	if cfg.ReadOnly {
		t.Fatal("ReadOnly = true, want false")
	}
	if cfg.FixtureDir != "/tmp/abs-fixture" {
		t.Fatalf("FixtureDir = %q", cfg.FixtureDir)
	}
}

func TestLoadFromViperDefaults(t *testing.T) {
	settings := NewViper()
	settings.Set(KeyBaseURL, "http://localhost:13388")
	settings.Set(KeyAPIKey, "test-token")

	cfg, err := LoadFromViper(settings)
	if err != nil {
		t.Fatalf("LoadFromViper failed: %v", err)
	}
	if cfg.Timeout != defaultTimeout {
		t.Fatalf("Timeout = %s, want %s", cfg.Timeout, defaultTimeout)
	}
	if !cfg.ReadOnly {
		t.Fatal("ReadOnly = false, want true")
	}
	if cfg.FixtureDir != "test/abs" {
		t.Fatalf("FixtureDir = %q, want test/abs", cfg.FixtureDir)
	}
}

func mapLookup(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}
