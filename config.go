package main

import (
	"fmt"
	"os"
	"path/filepath"
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
