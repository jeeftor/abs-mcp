package config

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

const (
	defaultTimeout = 30 * time.Second

	// KeyBaseURL is the configuration key for the Audiobookshelf base URL.
	KeyBaseURL = "base-url"
	// KeyAPIKey is the configuration key for the Audiobookshelf API key.
	KeyAPIKey = "api-key"
	// KeyEnvFile is the configuration key for a Docker-style environment file.
	KeyEnvFile = "env-file"
	// KeyTimeout is the configuration key for Audiobookshelf request timeout.
	KeyTimeout = "timeout"
	// KeyReadOnly is the configuration key for blocking mutating tools.
	KeyReadOnly = "read-only"
	// KeyFixtureDir is the configuration key for the local ABS fixture path.
	KeyFixtureDir = "fixture-dir"
	// KeyExtraHeadersFile is the configuration key for extra request headers.
	KeyExtraHeadersFile = "extra-headers-file"
	// KeyExtraHeader is the configuration key for one inline extra header.
	KeyExtraHeader = "header"
	// KeyTLSCACertFile is the configuration key for a custom TLS CA bundle.
	KeyTLSCACertFile = "tls-ca-cert-file"
	// KeyTLSSkipVerify is the configuration key for temporary TLS verification bypass.
	KeyTLSSkipVerify = "tls-insecure-skip-verify"
)

var envFileKeys = map[string]string{
	"ABS_BASE_URL":                 KeyBaseURL,
	"ABS_API_KEY":                  KeyAPIKey,
	"ABS_TIMEOUT":                  KeyTimeout,
	"ABS_READ_ONLY":                KeyReadOnly,
	"ABS_FIXTURE_DIR":              KeyFixtureDir,
	"ABS_EXTRA_HEADERS_FILE":       KeyExtraHeadersFile,
	"ABS_TLS_CA_CERT_FILE":         KeyTLSCACertFile,
	"ABS_TLS_INSECURE_SKIP_VERIFY": KeyTLSSkipVerify,
}

// Config contains runtime settings for the MCP server.
type Config struct {
	ABSBaseURL       string
	ABSAPIKey        string
	Timeout          time.Duration
	ReadOnly         bool
	FixtureDir       string
	ExtraHeadersFile string
	ExtraHeaders     map[string]string
	TLSCACertFile    string
	TLSSkipVerify    bool
}

// Load reads configuration from process environment variables.
func Load() (Config, error) {
	return LoadFromEnv(os.LookupEnv)
}

// LoadFromEnv reads configuration through lookup, which is useful for tests.
func LoadFromEnv(lookup func(string) (string, bool)) (Config, error) {
	return load(sourceValues{
		baseURL:              lookupString(lookup, "ABS_BASE_URL"),
		apiKey:               lookupString(lookup, "ABS_API_KEY"),
		timeout:              lookupString(lookup, "ABS_TIMEOUT"),
		readOnly:             lookupString(lookup, "ABS_READ_ONLY"),
		fixtureDir:           lookupString(lookup, "ABS_FIXTURE_DIR"),
		extraHeadersFile:     lookupString(lookup, "ABS_EXTRA_HEADERS_FILE"),
		extraHeaderValues:    nil,
		tlsCACertFile:        lookupString(lookup, "ABS_TLS_CA_CERT_FILE"),
		tlsSkipVerify:        lookupString(lookup, "ABS_TLS_INSECURE_SKIP_VERIFY"),
		baseURLName:          "ABS_BASE_URL",
		apiKeyName:           "ABS_API_KEY",
		timeoutName:          "ABS_TIMEOUT",
		readOnlyName:         "ABS_READ_ONLY",
		extraHeadersFileName: "ABS_EXTRA_HEADERS_FILE",
		tlsSkipVerifyName:    "ABS_TLS_INSECURE_SKIP_VERIFY",
	})
}

// NewViper returns a Viper instance configured for ABS environment variables.
func NewViper() *viper.Viper {
	settings := viper.New()
	settings.SetEnvPrefix("ABS")
	settings.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	settings.AutomaticEnv()
	settings.SetDefault(KeyTimeout, defaultTimeout.String())
	settings.SetDefault(KeyReadOnly, "true")
	settings.SetDefault(KeyFixtureDir, "test/abs")
	return settings
}

// ApplyEnvFile loads Docker-style dotenv values as Viper defaults.
func ApplyEnvFile(settings *viper.Viper, path string) error {
	values, err := LoadEnvFile(path)
	if err != nil {
		return err
	}
	for envKey, value := range values {
		configKey, ok := envFileKeys[envKey]
		if !ok {
			continue
		}
		settings.SetDefault(configKey, value)
	}
	return nil
}

// LoadEnvFile reads simple Docker-style KEY=VALUE lines.
func LoadEnvFile(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read --env-file: %w", err)
	}
	values := map[string]string{}
	for index, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		name, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("parse --env-file line %d: expected KEY=VALUE", index+1)
		}
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, fmt.Errorf("parse --env-file line %d: key is required", index+1)
		}
		value = strings.TrimSpace(value)
		unquoted, err := unquoteEnvFileValue(value)
		if err != nil {
			return nil, fmt.Errorf("parse --env-file line %d: %w", index+1, err)
		}
		values[name] = unquoted
	}
	return values, nil
}

// LoadFromViper reads configuration from a Cobra/Viper-backed settings source.
func LoadFromViper(settings *viper.Viper) (Config, error) {
	return load(sourceValues{
		baseURL:              strings.TrimSpace(settings.GetString(KeyBaseURL)),
		apiKey:               strings.TrimSpace(settings.GetString(KeyAPIKey)),
		timeout:              strings.TrimSpace(settings.GetString(KeyTimeout)),
		readOnly:             strings.TrimSpace(settings.GetString(KeyReadOnly)),
		fixtureDir:           strings.TrimSpace(settings.GetString(KeyFixtureDir)),
		extraHeadersFile:     strings.TrimSpace(settings.GetString(KeyExtraHeadersFile)),
		extraHeaderValues:    settings.GetStringSlice(KeyExtraHeader),
		tlsCACertFile:        strings.TrimSpace(settings.GetString(KeyTLSCACertFile)),
		tlsSkipVerify:        strings.TrimSpace(settings.GetString(KeyTLSSkipVerify)),
		baseURLName:          "ABS_BASE_URL or --base-url",
		apiKeyName:           "ABS_API_KEY or --api-key",
		timeoutName:          "ABS_TIMEOUT or --timeout",
		readOnlyName:         "ABS_READ_ONLY or --read-only",
		extraHeadersFileName: "ABS_EXTRA_HEADERS_FILE or --extra-headers-file",
		tlsSkipVerifyName:    "ABS_TLS_INSECURE_SKIP_VERIFY or --tls-insecure-skip-verify",
	})
}

// LoadExtraHeadersFile reads a JSON object of additional request headers.
func LoadExtraHeadersFile(path string) (map[string]string, error) {
	return loadExtraHeadersFile(path, "ABS_EXTRA_HEADERS_FILE")
}

func loadExtraHeadersFile(path string, sourceName string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", sourceName, err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse %s as JSON object: %w", sourceName, err)
	}
	headers := make(map[string]string, len(raw))
	for name, value := range raw {
		text, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("%s header %q must have a string value", sourceName, name)
		}
		canonicalName, err := normalizeExtraHeaderName(name, sourceName)
		if err != nil {
			return nil, err
		}
		headers[canonicalName] = text
	}
	return headers, nil
}

func normalizeExtraHeaderName(name string, sourceName string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("%s header name is required", sourceName)
	}
	if strings.EqualFold(name, "Authorization") {
		return "", fmt.Errorf("%s must not contain Authorization; use ABS_API_KEY or --api-key", sourceName)
	}
	for _, r := range name {
		if r <= 32 || r >= 127 || strings.ContainsRune("()<>@,;:\\\"/[]?={}", r) {
			return "", fmt.Errorf("%s header name %q is invalid", sourceName, name)
		}
	}
	return http.CanonicalHeaderKey(name), nil
}

func lookupString(lookup func(string) (string, bool), key string) string {
	value, ok := lookup(key)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func unquoteEnvFileValue(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	if strings.HasPrefix(value, `"`) || strings.HasPrefix(value, `'`) {
		quote := value[:1]
		if !strings.HasSuffix(value, quote) || len(value) == 1 {
			return "", fmt.Errorf("quoted value is not closed")
		}
		value = value[1 : len(value)-1]
		if quote == `"` {
			value = strings.ReplaceAll(value, `\"`, `"`)
			value = strings.ReplaceAll(value, `\\`, `\`)
		}
	}
	return value, nil
}

type sourceValues struct {
	baseURL              string
	apiKey               string
	timeout              string
	readOnly             string
	fixtureDir           string
	extraHeadersFile     string
	extraHeaderValues    []string
	tlsCACertFile        string
	tlsSkipVerify        string
	baseURLName          string
	apiKeyName           string
	timeoutName          string
	readOnlyName         string
	extraHeadersFileName string
	tlsSkipVerifyName    string
}

func load(values sourceValues) (Config, error) {
	if values.baseURL == "" {
		return Config{}, fmt.Errorf("%s is required", values.baseURLName)
	}
	if values.apiKey == "" {
		return Config{}, fmt.Errorf("%s is required", values.apiKeyName)
	}

	timeout, err := parseDuration(values.timeout, values.timeoutName, defaultTimeout)
	if err != nil {
		return Config{}, err
	}
	readOnly, err := parseBool(values.readOnly, values.readOnlyName, true)
	if err != nil {
		return Config{}, err
	}
	tlsSkipVerify, err := parseBool(values.tlsSkipVerify, values.tlsSkipVerifyName, false)
	if err != nil {
		return Config{}, err
	}
	fixtureDir := values.fixtureDir
	if fixtureDir == "" {
		fixtureDir = "test/abs"
	}
	extraHeaders := map[string]string{}
	if values.extraHeadersFile != "" {
		extraHeaders, err = loadExtraHeadersFile(values.extraHeadersFile, values.extraHeadersFileName)
		if err != nil {
			return Config{}, err
		}
	}
	for _, header := range values.extraHeaderValues {
		name, value, err := parseExtraHeaderValue(header)
		if err != nil {
			return Config{}, err
		}
		extraHeaders[name] = value
	}

	return Config{
		ABSBaseURL:       values.baseURL,
		ABSAPIKey:        values.apiKey,
		Timeout:          timeout,
		ReadOnly:         readOnly,
		FixtureDir:       fixtureDir,
		ExtraHeadersFile: values.extraHeadersFile,
		ExtraHeaders:     extraHeaders,
		TLSCACertFile:    values.tlsCACertFile,
		TLSSkipVerify:    tlsSkipVerify,
	}, nil
}

func parseExtraHeaderValue(header string) (string, string, error) {
	name, value, ok := strings.Cut(header, "=")
	if !ok {
		return "", "", fmt.Errorf("--header must use NAME=VALUE")
	}
	canonicalName, err := normalizeExtraHeaderName(name, "--header")
	if err != nil {
		return "", "", err
	}
	return canonicalName, value, nil
}

func parseDuration(value string, key string, defaultValue time.Duration) (time.Duration, error) {
	if value == "" {
		return defaultValue, nil
	}
	duration, err := time.ParseDuration(value)
	if err == nil {
		return duration, nil
	}
	seconds, secondsErr := strconv.Atoi(value)
	if secondsErr != nil {
		return 0, fmt.Errorf("%s must be a Go duration or seconds: %w", key, err)
	}
	return time.Duration(seconds) * time.Second, nil
}

func parseBool(value string, key string, defaultValue bool) (bool, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return defaultValue, nil
	}
	switch value {
	case "1", "true", "yes", "y", "on":
		return true, nil
	case "0", "false", "no", "n", "off":
		return false, nil
	default:
		return false, fmt.Errorf("%s must be a boolean", key)
	}
}
