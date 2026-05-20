package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/jeeftor/abs-mcp/internal/abs"
	"github.com/jeeftor/abs-mcp/internal/config"
	"github.com/jeeftor/abs-mcp/internal/mcpserver"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "abs-mcp: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	command := newRootCommand(ctx, runServer)
	command.SetArgs(args)
	return command.ExecuteContext(ctx)
}

func newRootCommand(ctx context.Context, runner func(context.Context, config.Config) error) *cobra.Command {
	settings := config.NewViper()
	command := &cobra.Command{
		Use:           "abs-mcp",
		Short:         "Run the Audiobookshelf MCP server over stdio",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(command *cobra.Command, args []string) error {
			cfg, err := config.LoadFromViper(settings)
			if err != nil {
				return err
			}
			return runner(command.Context(), cfg)
		},
	}
	command.SetContext(ctx)
	bindFlags(command.Flags(), settings)
	return command
}

func bindFlags(flags *pflag.FlagSet, settings *viper.Viper) {
	flags.String(config.KeyBaseURL, "", "Audiobookshelf base URL (env ABS_BASE_URL)")
	flags.String(config.KeyAPIKey, "", "Audiobookshelf API key or bearer token (env ABS_API_KEY)")
	flags.String(config.KeyTimeout, "", "Audiobookshelf request timeout as a Go duration or seconds (env ABS_TIMEOUT)")
	flags.Bool(config.KeyReadOnly, true, "Block mutating MCP tools (env ABS_READ_ONLY)")
	flags.String(config.KeyFixtureDir, "", "ABS fixture directory used by fixture resources (env ABS_FIXTURE_DIR)")
	flags.String(config.KeyExtraHeadersFile, "", "JSON file of extra ABS request headers (env ABS_EXTRA_HEADERS_FILE)")
	flags.StringArray(config.KeyExtraHeader, nil, "Extra ABS request header as NAME=VALUE; repeatable and overrides duplicate file headers")
	flags.String(config.KeyTLSCACertFile, "", "PEM CA bundle for private or corporate ABS TLS certificates (env ABS_TLS_CA_CERT_FILE)")
	flags.Bool(config.KeyTLSSkipVerify, false, "Skip ABS TLS certificate verification; use only as a temporary fallback (env ABS_TLS_INSECURE_SKIP_VERIFY)")

	mustBindFlag(settings, config.KeyBaseURL, flags)
	mustBindFlag(settings, config.KeyAPIKey, flags)
	mustBindFlag(settings, config.KeyTimeout, flags)
	mustBindFlag(settings, config.KeyReadOnly, flags)
	mustBindFlag(settings, config.KeyFixtureDir, flags)
	mustBindFlag(settings, config.KeyExtraHeadersFile, flags)
	mustBindFlag(settings, config.KeyExtraHeader, flags)
	mustBindFlag(settings, config.KeyTLSCACertFile, flags)
	mustBindFlag(settings, config.KeyTLSSkipVerify, flags)
}

func mustBindFlag(settings *viper.Viper, key string, flags *pflag.FlagSet) {
	if err := settings.BindPFlag(key, flags.Lookup(key)); err != nil {
		panic(fmt.Sprintf("bind flag %s: %v", key, err))
	}
}

func runServer(ctx context.Context, cfg config.Config) error {
	client, err := abs.NewClient(cfg.ABSBaseURL, cfg.ABSAPIKey)
	if err != nil {
		return err
	}
	httpClient, err := newHTTPClient(cfg)
	if err != nil {
		return err
	}
	client.SetHTTPClient(httpClient)
	if err := client.SetExtraHeaders(cfg.ExtraHeaders); err != nil {
		return err
	}

	server := mcpserver.New(cfg, client).MCPServer()
	return server.Run(ctx, &mcp.StdioTransport{})
}

func newHTTPClient(cfg config.Config) (*http.Client, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if cfg.TLSCACertFile != "" || cfg.TLSSkipVerify {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: cfg.TLSSkipVerify,
		}
		if cfg.TLSCACertFile != "" {
			certPool, err := x509.SystemCertPool()
			if err != nil {
				certPool = x509.NewCertPool()
			}
			data, err := os.ReadFile(cfg.TLSCACertFile)
			if err != nil {
				return nil, fmt.Errorf("read ABS_TLS_CA_CERT_FILE: %w", err)
			}
			if !certPool.AppendCertsFromPEM(data) {
				return nil, fmt.Errorf("ABS_TLS_CA_CERT_FILE must contain at least one PEM certificate")
			}
			tlsConfig.RootCAs = certPool
		}
		transport.TLSClientConfig = tlsConfig
	}
	return &http.Client{
		Timeout:   cfg.Timeout,
		Transport: transport,
	}, nil
}
