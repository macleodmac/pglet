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
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/macleodmac/pglet/pkg/api"
	"github.com/macleodmac/pglet/pkg/client"
	"github.com/macleodmac/pglet/pkg/repository"
	"github.com/macleodmac/pglet/static"
	"github.com/lmittmann/tint"
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

type Config struct {
	Host        string
	Port        int
	User        string
	Pass        string
	DbName      string
	URL         string
	SSL         string
	Bind        string
	Listen      int
	Prefix      string
	OpenBrowser bool
	RepoDir     string
	Dev         bool
	Cors        bool
}

func parseConfig() Config {
	cfg := Config{
		SSL:    "disable",
		Bind:   "localhost",
		Listen: 8081,
		Prefix: "/",
	}

	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			printUsage()
			os.Exit(0)
		case "-v", "--version":
			fmt.Println(getVersion())
			os.Exit(0)
		case "--host":
			if i+1 < len(args) {
				cfg.Host = args[i+1]
				i++
			}
		case "--port":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &cfg.Port)
				i++
			}
		case "--user":
			if i+1 < len(args) {
				cfg.User = args[i+1]
				i++
			}
		case "--pass":
			if i+1 < len(args) {
				cfg.Pass = args[i+1]
				i++
			}
		case "--db":
			if i+1 < len(args) {
				cfg.DbName = args[i+1]
				i++
			}
		case "--url":
			if i+1 < len(args) {
				cfg.URL = args[i+1]
				i++
			}
		case "--ssl":
			if i+1 < len(args) {
				cfg.SSL = args[i+1]
				i++
			}
		case "--bind":
			if i+1 < len(args) {
				cfg.Bind = args[i+1]
				i++
			}
		case "--listen":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &cfg.Listen)
				i++
			}
		case "--prefix":
			if i+1 < len(args) {
				cfg.Prefix = args[i+1]
				i++
			}
		case "--open":
			cfg.OpenBrowser = true
		case "--store-dir", "--repo-dir":
			if i+1 < len(args) {
				cfg.RepoDir = args[i+1]
				i++
			}
		case "--dev":
			cfg.Dev = true
		case "--cors":
			cfg.Cors = true
		}
	}

	// Also check env vars
	if os.Getenv("PGLET_DEV") == "1" {
		cfg.Dev = true
	}

	return cfg
}

func resolveRepoDir(cfg Config) string {
	if cfg.RepoDir != "" {
		return cfg.RepoDir
	}

	// Walk from CWD upward looking for .pglet/
	cwd, err := os.Getwd()
	if err == nil {
		dir := cwd
		for {
			candidate := filepath.Join(dir, ".pglet")
			if info, err := os.Stat(candidate); err == nil && info.IsDir() {
				return candidate
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	// Default to ~/.pglet/
	home, err := os.UserHomeDir()
	if err != nil {
		return ".pglet"
	}
	return filepath.Join(home, ".pglet")
}

func buildConnectionURL(cfg Config) string {
	if cfg.URL != "" {
		return cfg.URL
	}
	if cfg.Host == "" {
		return ""
	}

	port := cfg.Port
	if port == 0 {
		port = 5432
	}

	url := fmt.Sprintf("postgres://%s", cfg.Host)
	if cfg.User != "" {
		userPart := cfg.User
		if cfg.Pass != "" {
			userPart += ":" + cfg.Pass
		}
		url = fmt.Sprintf("postgres://%s@%s", userPart, cfg.Host)
	}
	url += fmt.Sprintf(":%d", port)
	if cfg.DbName != "" {
		url += "/" + cfg.DbName
	}
	url += fmt.Sprintf("?sslmode=%s", cfg.SSL)
	return url
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
	if err := repo.ImportSharedQueries(context.Background(), repoDir); err != nil {
		slog.Warn("failed to import shared queries", "err", err)
	}

	// Setup server
	server := &api.Server{
		Repo: repo,
	}

	// Auto-connect if URL provided
	connURL := buildConnectionURL(cfg)
	if connURL != "" {
		cl, err := client.New(connURL)
		if err != nil {
			slog.Warn("failed to connect", "err", err)
		} else {
			server.SetClient(cl)
			slog.Debug("connected", "database", cl.Database(), "elapsed", time.Since(start))
		}
	}

	// Setup Gin
	if !cfg.Dev {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(api.LoggingMiddleware())

	if cfg.Cors || cfg.Dev {
		r.Use(api.CorsMiddleware())
	}

	api.RegisterHandlers(r, server)

	// Serve frontend
	if !cfg.Dev {
		frontendFS, err := fs.Sub(static.Frontend, "dist")
		if err != nil {
			slog.Error("failed to setup frontend", "err", err)
			os.Exit(1)
		}
		fileServer := http.FileServer(http.FS(frontendFS))

		r.NoRoute(func(c *gin.Context) {
			path := c.Request.URL.Path
			// Try to serve the file directly
			if path != "/" && !strings.HasPrefix(path, "/api") {
				c.FileFromFS(path, http.FS(frontendFS))
				return
			}
			// Fallback to index.html for SPA
			fileServer.ServeHTTP(c.Writer, c.Request)
		})
	}

	addr := fmt.Sprintf("%s:%d", cfg.Bind, cfg.Listen)
	slog.Info("listening", "version", getVersion(), "addr", fmt.Sprintf("http://%s", addr), "startup", time.Since(start))

	if !cfg.Dev {
		go openBrowser(fmt.Sprintf("http://%s", addr))
	}

	srv := &http.Server{Addr: addr, Handler: r}

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
