package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"portaptable/cmd"
	"portaptable/pkg/config"
)

const (
	defaultRepoPath = "./repository"
	defaultPort     = "8080"
)

func main() {
	var cfg config.Config
	var downloadMode, serveMode, helpMode bool

	// Define command line flags
	flag.BoolVar(&downloadMode, "download", false, "Download mode: fetch packages and dependencies")
	flag.BoolVar(&serveMode, "serve", false, "Serve mode: start local repository server")
	flag.BoolVar(&helpMode, "help", false, "Show help information")
	flag.StringVar(&cfg.RepoPath, "repo", defaultRepoPath, "Repository directory path")
	flag.StringVar(&cfg.Port, "port", defaultPort, "Port for serve mode")
	flag.StringVar(&cfg.ConfigFile, "config", "", "Configuration file path")
	flag.StringVar(&cfg.Architecture, "arch", "amd64", "Target architecture")
	flag.StringVar(&cfg.Distribution, "dist", "focal", "Target distribution (e.g., focal, jammy)")

	flag.Parse()

	// Show help if requested or no mode specified
	if helpMode || (!downloadMode && !serveMode) {
		showHelp()
		return
	}

	// Validate that only one mode is specified
	if downloadMode && serveMode {
		log.Fatal("Error: Cannot specify both --download and --serve modes")
	}

	// Get remaining arguments as package names for download mode
	if downloadMode {
		cfg.Packages = flag.Args()

		if len(cfg.Packages) == 0 {
			log.Fatal("Error: No packages specified for download mode")
		}
	}

	// Ensure repository path exists
	if err := ensureRepoPath(cfg.RepoPath); err != nil {
		log.Fatalf("Error creating repository path: %v", err)
	}

	// Execute the appropriate mode
	switch {
	case downloadMode:
		fmt.Printf("Starting download mode...\n")
		fmt.Printf("Repository: %s\n", cfg.RepoPath)
		fmt.Printf("Architecture: %s\n", cfg.Architecture)
		fmt.Printf("Distribution: %s\n", cfg.Distribution)
		fmt.Printf("Packages: %v\n", cfg.Packages)

		if err := cmd.RunDownloadMode(&cfg); err != nil {
			log.Fatalf("Download mode failed: %v", err)
		}
		fmt.Println("Download completed successfully")

	case serveMode:
		fmt.Printf("Starting serve mode...\n")
		fmt.Printf("Repository: %s\n", cfg.RepoPath)
		fmt.Printf("Port: %s\n", cfg.Port)

		if err := cmd.RunServeMode(&cfg); err != nil {
			log.Fatalf("Serve mode failed: %v", err)
		}
	}

	return
}

func showHelp() {
	fmt.Printf(`apt-offline - Offline APT Package Management Tool

Usage:
  %s [OPTIONS] --download package1 [package2 ...]
  %s [OPTIONS] --serve

Modes:
  --download    Download packages and dependencies for offline installation
  --serve       Start local repository server for air-gapped installation

Options:
  --repo PATH   Repository directory (default: %s)
  --port PORT   Server port for serve mode (default: %s)
  --arch ARCH   Target architecture (default: amd64)
  --dist DIST   Target distribution (default: focal)
  --config FILE Configuration file path
  --help        Show this help message

Examples:
  # Download nginx and all dependencies
  %s --download nginx

  # Download multiple packages for specific architecture
  %s --arch arm64 --dist jammy --download curl vim git

  # Serve local repository on port 9000
  %s --serve --port 9000

  # Use custom repository location
  %s --repo /opt/offline-repo --serve

`, os.Args[0], os.Args[0], defaultRepoPath, defaultPort, os.Args[0], os.Args[0], os.Args[0], os.Args[0])

	return
}

func ensureRepoPath(repoPath string) error {
	// Create main repository directory
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		return fmt.Errorf("failed to create repository directory: %w", err)
	}

	// Create subdirectories
	subdirs := []string{"pool", "dists"}

	for _, subdir := range subdirs {
		path := filepath.Join(repoPath, subdir)

		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("failed to create %s directory: %w", subdir, err)
		}
	}

	return nil
}
