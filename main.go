package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"syscall"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
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

	storeDir := resolveStoreDir(cfg)
	repoPath := filepath.Join(storeDir, "pglet.db")
	repo, err := repository.Open(repoPath, resolveSharedQueriesDir(storeDir))
	if err != nil {
		slog.Error("failed to open repository", "err", err)
		os.Exit(1)
	}
	defer repo.Close()

	slog.Debug("repository opened", "path", repoPath, "elapsed", time.Since(start))

	// Setup service and server
	svc := service.New(repo, getVersion())
	server := api.NewServer(svc)

	// Auto-connect in background so the UI is available immediately
	connURL := buildConnectionURL(cfg)
	if connURL != "" {
		svc.SetConnecting(true)
		go func() {
			cl, err := client.New(connURL)
			if err != nil {
				slog.Warn("failed to connect", "err", err)
				svc.SetConnecting(false)
				return
			}
			if svc.SetInitialClient(cl) {
				slog.Info("connected", "database", cl.Database(), "elapsed", time.Since(start))
			}
		}()
	}

	// Load OpenAPI spec for request validation
	swagger, err := api.GetSwagger()
	if err != nil {
		slog.Error("failed to load OpenAPI spec", "err", err)
		os.Exit(1)
	}
	// A relative server URL lets the validator match paths under the prefix
	// without checking the host.
	swagger.Servers = openapi3.Servers{{URL: cfg.Prefix + "/"}}

	var handler http.Handler
	handler = api.HandlerWithOptions(server, api.StdHTTPServerOptions{
		BaseURL: cfg.Prefix,
		Middlewares: []api.MiddlewareFunc{nethttpmiddleware.OapiRequestValidatorWithOptions(swagger, &nethttpmiddleware.Options{
			SilenceServersWarning: true,
		})},
	})
	handler = api.LoggingMiddleware(handler)

	if !cfg.Dev {
		frontendFS, err := fs.Sub(static.Frontend, "dist")
		if err == nil {
			handler, err = spaHandler(handler, frontendFS, cfg.Prefix)
		}
		if err != nil {
			slog.Error("failed to setup frontend", "err", err)
			os.Exit(1)
		}
	}

	loopback := isLoopbackBind(cfg.Bind)
	guard := api.Guard{
		LoopbackOnly: loopback,
		Prefix:       cfg.Prefix,
		CookieName:   fmt.Sprintf("pglet_token_%d", cfg.Listen),
	}
	// The dev UI is served by Vite, which never sees the token link.
	if !cfg.Dev {
		guard.Token, err = repo.AuthToken()
		if err != nil {
			slog.Error("failed to load auth token", "err", err)
			os.Exit(1)
		}
	}
	handler = guard.Middleware(handler)
	if len(cfg.CorsOrigins) > 0 {
		handler = api.CorsMiddleware(cfg.CorsOrigins)(handler)
	}
	handler = api.RecoveryMiddleware(handler)

	addr := net.JoinHostPort(cfg.Bind, strconv.Itoa(cfg.Listen))
	appURL := fmt.Sprintf("http://%s%s/", addr, cfg.Prefix)
	if guard.Token != "" {
		appURL += "?token=" + guard.Token
	}
	slog.Info("listening", "version", getVersion(), "addr", addr, "startup", time.Since(start))
	fmt.Fprintf(os.Stderr, "\n  Open pglet: %s\n\n", appURL)
	if !loopback {
		slog.Warn("listening beyond this machine; anyone who has the link above can use your database", "bind", cfg.Bind)
	}

	if cfg.OpenBrowser && !cfg.Dev {
		go openBrowser(appURL)
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

Server:
  --bind <addr>     Bind address (default: localhost)
  --listen <port>   Listen port (default: 8081)
  --prefix <path>   URL prefix (default: /)
  --open            Open browser on start
  --store-dir <dir> Data directory (default: ~/.pglet/)
  --dev             Development mode (UI served by Vite, no sign-in token)
  --cors-origin <o> Allow API calls from this origin (repeatable)

The password can also come from the PGPASSWORD environment variable, which
keeps it out of the process list.

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
