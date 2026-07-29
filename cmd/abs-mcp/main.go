package main

import (
	"context"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/modelcontextprotocol/go-sdk/auth"
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
		Short:         "Run the Audiobookshelf MCP server",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(command *cobra.Command, args []string) error {
			envFile, err := command.Flags().GetString(config.KeyEnvFile)
			if err != nil {
				return err
			}
			if envFile != "" {
				if err := config.ApplyEnvFile(settings, envFile); err != nil {
					return err
				}
			}
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
	flags.String(config.KeyEnvFile, "", "Docker-style env file with ABS_* settings")
	flags.String(config.KeyTimeout, "", "Audiobookshelf request timeout as a Go duration or seconds (env ABS_TIMEOUT)")
	flags.Bool(config.KeyReadOnly, true, "Block mutating MCP tools (env ABS_READ_ONLY)")
	flags.String(config.KeyFixtureDir, "", "ABS fixture directory used by fixture resources (env ABS_FIXTURE_DIR)")
	flags.String(config.KeyExtraHeadersFile, "", "JSON file of extra ABS request headers (env ABS_EXTRA_HEADERS_FILE)")
	flags.StringArray(config.KeyExtraHeader, nil, "Extra ABS request header as NAME=VALUE; repeatable and overrides duplicate file headers")
	flags.String(config.KeyTLSCACertFile, "", "PEM CA bundle for private or corporate ABS TLS certificates (env ABS_TLS_CA_CERT_FILE)")
	flags.Bool(config.KeyTLSSkipVerify, false, "Skip ABS TLS certificate verification; use only as a temporary fallback (env ABS_TLS_INSECURE_SKIP_VERIFY)")
	flags.String(config.KeyTransport, config.TransportStdio, "MCP transport: stdio or http (env ABS_TRANSPORT)")
	flags.String(config.KeyHTTPAddr, "127.0.0.1:3333", "Streamable HTTP bind address when --transport=http (env ABS_HTTP_ADDR)")
	flags.String(config.KeyHTTPPath, "/mcp", "Streamable HTTP endpoint path when --transport=http (env ABS_HTTP_PATH)")
	flags.String(config.KeyHTTPBearerToken, "", "Bearer token required by Streamable HTTP clients when set (env ABS_HTTP_BEARER_TOKEN)")

	mustBindFlag(settings, config.KeyBaseURL, flags)
	mustBindFlag(settings, config.KeyAPIKey, flags)
	mustBindFlag(settings, config.KeyEnvFile, flags)
	mustBindFlag(settings, config.KeyTimeout, flags)
	mustBindFlag(settings, config.KeyReadOnly, flags)
	mustBindFlag(settings, config.KeyFixtureDir, flags)
	mustBindFlag(settings, config.KeyExtraHeadersFile, flags)
	mustBindFlag(settings, config.KeyExtraHeader, flags)
	mustBindFlag(settings, config.KeyTLSCACertFile, flags)
	mustBindFlag(settings, config.KeyTLSSkipVerify, flags)
	mustBindFlag(settings, config.KeyTransport, flags)
	mustBindFlag(settings, config.KeyHTTPAddr, flags)
	mustBindFlag(settings, config.KeyHTTPPath, flags)
	mustBindFlag(settings, config.KeyHTTPBearerToken, flags)
}

func mustBindFlag(settings *viper.Viper, key string, flags *pflag.FlagSet) {
	if err := settings.BindPFlag(key, flags.Lookup(key)); err != nil {
		panic(fmt.Sprintf("bind flag %s: %v", key, err))
	}
}

func runServer(ctx context.Context, cfg config.Config) error {
	if cfg.Transport == config.TransportHTTP {
		return runStreamableHTTPServer(ctx, cfg)
	}

	server, err := newMCPServer(cfg)
	if err != nil {
		return err
	}
	return server.Run(ctx, &mcp.StdioTransport{})
}

func newMCPServer(cfg config.Config) (*mcp.Server, error) {
	client, err := abs.NewClient(cfg.ABSBaseURL, cfg.ABSAPIKey)
	if err != nil {
		return nil, err
	}
	httpClient, err := newHTTPClient(cfg)
	if err != nil {
		return nil, err
	}
	client.SetHTTPClient(httpClient)
	if err := client.SetExtraHeaders(cfg.ExtraHeaders); err != nil {
		return nil, err
	}

	return mcpserver.New(cfg, client).MCPServer(), nil
}

func newStreamableHTTPHandler(cfg config.Config) (http.Handler, error) {
	server, err := newMCPServer(cfg)
	if err != nil {
		return nil, err
	}
	handler := mcp.NewStreamableHTTPHandler(func(_ *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{
		Stateless:                    true,
		PropagateRequestCancellation: true,
	})
	mux := http.NewServeMux()
	mux.Handle(cfg.HTTPPath, streamableHTTPAuthMiddleware(cfg)(handler))
	return mux, nil
}

func streamableHTTPAuthMiddleware(cfg config.Config) func(http.Handler) http.Handler {
	if cfg.HTTPBearerToken == "" {
		return func(handler http.Handler) http.Handler {
			return handler
		}
	}
	return auth.RequireBearerToken(func(_ context.Context, token string, _ *http.Request) (*auth.TokenInfo, error) {
		if subtle.ConstantTimeCompare([]byte(token), []byte(cfg.HTTPBearerToken)) != 1 {
			return nil, auth.ErrInvalidToken
		}
		return &auth.TokenInfo{
			Expiration: time.Now().Add(time.Hour),
			UserID:     "abs-mcp-http",
		}, nil
	}, nil)
}

func runStreamableHTTPServer(ctx context.Context, cfg config.Config) error {
	handler, err := newStreamableHTTPHandler(cfg)
	if err != nil {
		return err
	}
	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}
	errc := make(chan error, 1)
	go func() {
		errc <- server.ListenAndServe()
	}()

	select {
	case err := <-errc:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return err
		}
		err := <-errc
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
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
