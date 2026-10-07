package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/lmittmann/tint"
	"github.com/macleodmac/pglet/internal/api"
	"github.com/macleodmac/pglet/internal/client"
	"github.com/macleodmac/pglet/internal/repository"
	"github.com/macleodmac/pglet/internal/service"
	"github.com/macleodmac/pglet/static"
	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"
)

var version = "dev"

func getVersion() string {
	if version != "dev" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return version
}

func main() {
	start := time.Now()

	slog.SetDefault(slog.New(tint.NewHandler(os.Stderr, &tint.Options{
		TimeFormat: "15:04:05",
	})))

	cfg := parseConfig()

	// Setup repository
	repoDir := resolveRepoDir(cfg)
	repoPath := filepath.Join(repoDir, "pglet.db")
	repo, err := repository.Open(repoPath)
	if err != nil {
		slog.Error("failed to open repository", "err", err)
		os.Exit(1)
	}
	defer repo.Close()

	slog.Debug("repository opened", "path", repoPath, "elapsed", time.Since(start))

	// Import shared queries from .pglet/queries/
	if err := repo.ImportSharedQueries(repoDir); err != nil {
		slog.Warn("failed to import shared queries", "err", err)
	}

	// Setup service and server
	svc := service.New(repo, getVersion())
	server := api.NewServer(svc)

	// Auto-connect in background so the UI is available immediately
	connURL := buildConnectionURL(cfg)
	if connURL != "" {
		svc.SetConnecting(true)
		go func() {
			cl, err := client.New(connURL, client.Options{ReadOnly: cfg.ReadOnly, StatementTimeoutMs: cfg.StatementTimeoutSec * 1000})
			if err != nil {
				slog.Warn("failed to connect", "err", err)
				svc.SetConnecting(false)
				return
			}
			svc.SetClient(cl) // also clears connecting
			slog.Info("connected", "database", cl.Database(), "elapsed", time.Since(start))
		}()
	}

	// Load OpenAPI spec for request validation
	swagger, err := api.GetSwagger()
	if err != nil {
		slog.Error("failed to load OpenAPI spec", "err", err)
		os.Exit(1)
	}
	swagger.Servers = nil // clear servers so validator doesn't check host

	// Build handler with validation middleware
	baseURL := cfg.Prefix
	if baseURL == "/" {
		baseURL = ""
	}
	var handler http.Handler
	handler = api.HandlerWithOptions(server, api.StdHTTPServerOptions{
		BaseURL:     baseURL,
		Middlewares: []api.MiddlewareFunc{nethttpmiddleware.OapiRequestValidator(swagger)},
	})

	// Apply middleware
	handler = api.LoggingMiddleware(handler)
	handler = api.RecoveryMiddleware(handler)
	if cfg.Cors || cfg.Dev {
		handler = api.CorsMiddleware(handler)
	}

	// Serve frontend (SPA fallback)
	if !cfg.Dev {
		frontendFS, err := fs.Sub(static.Frontend, "dist")
		if err != nil {
			slog.Error("failed to setup frontend", "err", err)
			os.Exit(1)
		}
		handler = spaHandler(handler, http.FS(frontendFS))
	}

	addr := fmt.Sprintf("%s:%d", cfg.Bind, cfg.Listen)
	slog.Info("listening", "version", getVersion(), "addr", fmt.Sprintf("http://%s", addr), "startup", time.Since(start))

	if !cfg.Dev {
		go openBrowser(fmt.Sprintf("http://%s", addr))
	}

	srv := &http.Server{Addr: addr, Handler: handler}

	// Graceful shutdown on SIGINT/SIGTERM
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		sig := <-quit
		slog.Info("shutting down", "signal", sig.String())

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			slog.Error("shutdown error", "err", err)
		}
	}()

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
	slog.Info("server stopped")
}

func printUsage() {
	fmt.Printf(`pglet %s — A PostgreSQL browser

Usage:
  pglet [flags]

Examples:
  pglet                                          # start and connect via UI
  pglet --url postgres://localhost:5432/mydb      # connect on startup
  pglet --host localhost --port 5432 --db mydb    # connect with flags

Connection:
  --url <url>       PostgreSQL connection URL
  --host <host>     Database host
  --port <port>     Database port (default: 5432)
  --user <user>     Database user
  --pass <pass>     Database password
  --db <name>       Database name
  --ssl <mode>      SSL mode (default: disable)
  --read-only       Start sessions read-only (default_transaction_read_only)
  --statement-timeout <sec>
                    Cancel statements running longer than this (default: none)

Server:
  --bind <addr>     Bind address (default: localhost)
  --listen <port>   Listen port (default: 8081)
  --prefix <path>   URL prefix (default: /)
  --open            Open browser on start
  --store-dir <dir> Data directory (default: ~/.pglet/)
  --dev             Development mode (CORS, verbose logging)
  --cors            Enable CORS

Other:
  -h, --help        Show this help
  -v, --version     Show version
`, getVersion())
}

func openBrowser(url string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "linux":
		cmd = "xdg-open"
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	}
	args = append(args, url)
	exec.Command(cmd, args...).Start()
}
