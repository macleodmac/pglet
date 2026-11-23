package main

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

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
	CorsOrigins []string
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
		case "--cors-origin":
			if i+1 < len(args) {
				cfg.CorsOrigins = append(cfg.CorsOrigins, strings.TrimSuffix(args[i+1], "/"))
				i++
			}
		default:
			fmt.Fprintf(os.Stderr, "unknown flag %s (see --help)\n", args[i])
			os.Exit(2)
		}
	}

	if os.Getenv("PGLET_DEV") == "1" {
		cfg.Dev = true
	}
	cfg.Prefix = normalizePrefix(cfg.Prefix)

	return cfg
}

// normalizePrefix turns "/", "pg", "/pg/" into "", "/pg", "/pg".
func normalizePrefix(p string) string {
	p = strings.Trim(p, "/")
	if p == "" {
		return ""
	}
	return "/" + p
}

// isLoopbackBind reports whether addr only accepts connections from this
// machine.
func isLoopbackBind(addr string) bool {
	if strings.EqualFold(addr, "localhost") {
		return true
	}
	ip := net.ParseIP(strings.Trim(addr, "[]"))
	return ip != nil && ip.IsLoopback()
}

// resolveStoreDir is where pglet keeps its private data (history, tab state,
// saved connections). It is never inside a project, so nothing private can end
// up committed to one.
func resolveStoreDir(cfg Config) string {
	if cfg.RepoDir != "" {
		return cfg.RepoDir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".pglet"
	}
	return filepath.Join(home, ".pglet")
}

// resolveSharedQueriesDir finds the nearest .pglet/queries directory at or
// above the working directory, falling back to the store's own queries/.
func resolveSharedQueriesDir(storeDir string) string {
	if cwd, err := os.Getwd(); err == nil {
		for dir := cwd; ; {
			candidate := filepath.Join(dir, ".pglet", "queries")
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
	return filepath.Join(storeDir, "queries")
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
	u := url.URL{
		Scheme:   "postgres",
		Host:     net.JoinHostPort(cfg.Host, strconv.Itoa(port)),
		Path:     "/" + cfg.DbName,
		RawQuery: url.Values{"sslmode": {cfg.SSL}}.Encode(),
	}
	switch {
	case cfg.User != "" && cfg.Pass != "":
		u.User = url.UserPassword(cfg.User, cfg.Pass)
	case cfg.User != "":
		u.User = url.User(cfg.User)
	}
	return u.String()
}
