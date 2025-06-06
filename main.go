package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"portaptable/pkg/config"
)

const (
	defaultRepoPath = "./repository"
	defaultPort     = "8080"
)

func main() {
	var config config.Config
	var downloadMode, serveMode, helpMode bool

	// Define command line flags
	flag.BoolVar(&downloadMode, "download", false, "Download mode: fetch packages and dependencies")
	flag.BoolVar(&serveMode, "serve", false, "Serve mode: start local repository server")
	flag.BoolVar(&helpMode, "help", false, "Show help information")
	flag.StringVar(&config.RepoPath, "repo", defaultRepoPath, "Repository directory path")
	flag.StringVar(&config.Port, "port", defaultPort, "Port for serve mode")
	flag.StringVar(&config.ConfigFile, "config", "", "Configuration file path")
	flag.StringVar(&config.Architecture, "arch", "amd64", "Target architecture")
	flag.StringVar(&config.Distribution, "dist", "focal", "Target distribution (e.g., focal, jammy)")

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
		config.Packages = flag.Args()

		if len(config.Packages) == 0 {
			log.Fatal("Error: No packages specified for download mode")
		}
	}

	// Ensure repository path exists
	if err := ensureRepoPath(config.RepoPath); err != nil {
		log.Fatalf("Error creating repository path: %v", err)
	}

	// Execute the appropriate mode
	switch {
	case downloadMode:
		fmt.Printf("Starting download mode...\n")
		fmt.Printf("Repository: %s\n", config.RepoPath)
		fmt.Printf("Architecture: %s\n", config.Architecture)
		fmt.Printf("Distribution: %s\n", config.Distribution)
		fmt.Printf("Packages: %v\n", config.Packages)

		if err := runDownloadMode(&config); err != nil {
			log.Fatalf("Download mode failed: %v", err)
		}

		fmt.Println("Download completed successfully")

	case serveMode:
		fmt.Printf("Starting serve mode...\n")
		fmt.Printf("Repository: %s\n", config.RepoPath)
		fmt.Printf("Port: %s\n", config.Port)

		if err := runServeMode(&config); err != nil {
			log.Fatalf("Serve mode failed: %v", err)
		}
	}
}

func showHelp() {
	fmt.Printf(`portaptable - Offline APT Package Management Tool

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

func runDownloadMode(config *config.Config) error {
	// TODO: Implement download functionality
	// This will call functions from cmd/download.go
	fmt.Println("Download mode implementation pending...")

	return nil
}

func runServeMode(config *config.Config) error {
	// TODO: Implement serve functionality
	// This will call functions from cmd/install.go
	fmt.Println("Serve mode implementation pending...")

	return nil
}
