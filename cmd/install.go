package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"portaptable/pkg/config"
	"portaptable/pkg/manifest"
)

type RepositoryServer struct {
	config   *config.Config
	manifest *manifest.Manifest
}

func runServeMode(config *config.Config) error {
	server := &RepositoryServer{config: config}

	// Load and validate repository
	if err := server.loadRepository(); err != nil {
		return fmt.Errorf("failed to load repository: %w", err)
	}

	// Setup HTTP handlers
	server.setupRoutes()

	fmt.Printf("Starting repository server on http://localhost:%s\n", config.Port)
	fmt.Printf("Repository path: %s\n", config.RepoPath)
	fmt.Printf("Serving %d packages\n", len(server.manifest.Packages))
	fmt.Println("\nTo use this repository on the target machine:")
	fmt.Printf("  echo 'deb [trusted=yes] http://localhost:%s/ %s main' | sudo tee /etc/apt/sources.list.d/portaptable.list\n",
		config.Port, config.Distribution)
	fmt.Println("  sudo apt update")
	fmt.Println("\nPress Ctrl+C to stop the server")

	return http.ListenAndServe(":"+config.Port, nil)
}

func (s *RepositoryServer) loadRepository() error {
	// Check if repository directory exists
	if _, err := os.Stat(s.config.RepoPath); os.IsNotExist(err) {
		return fmt.Errorf("repository directory does not exist: %s", s.config.RepoPath)
	}

	// Load manifest
	manifestPath := filepath.Join(s.config.RepoPath, "manifest.json")
	manifestData, err := os.ReadFile(manifestPath)

	if err != nil {
		return fmt.Errorf("failed to read manifest: %w", err)
	}

	s.manifest = &manifest.Manifest{}

	if err := json.Unmarshal(manifestData, s.manifest); err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	// Validate that packages exist
	poolPath := filepath.Join(s.config.RepoPath, "pool")
	missingCount := 0

	for _, pkg := range s.manifest.Packages {
		if pkg.Downloaded {
			pkgPath := filepath.Join(poolPath, pkg.Filename)

			if _, err := os.Stat(pkgPath); os.IsNotExist(err) {
				fmt.Printf("Warning: Package file missing: %s\n", pkg.Filename)
				missingCount++
			}
		}
	}

	if missingCount > 0 {
		fmt.Printf("Warning: %d package files are missing from the repository\n", missingCount)
	}

	return nil
}

func (s *RepositoryServer) setupRoutes() {
	// Serve the repository root
	http.HandleFunc("/", s.handleRepositoryRoot)

	// Serve distribution metadata
	http.HandleFunc("/dists/", s.handleDists)

	// Serve package pool
	http.HandleFunc("/pool/", s.handlePool)

	// Serve generated Packages file
	http.HandleFunc(fmt.Sprintf("/dists/%s/main/binary-%s/Packages",
		s.manifest.Distribution, s.manifest.Architecture), s.handlePackagesFile)

	// Health check endpoint
	http.HandleFunc("/health", s.handleHealth)

	// Repository info endpoint
	http.HandleFunc("/info", s.handleInfo)
}

func (s *RepositoryServer) handleRepositoryRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		// Serve a simple index page
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
    <title>Portaptable Repository</title>
</head>
<body>
    <h1>Portaptable - Portable APT Repository</h1>
    <p>This is a local APT repository serving %d packages.</p>
    <h2>Usage:</h2>
    <pre>echo 'deb [trusted=yes] http://localhost:%s/ %s main' | sudo tee /etc/apt/sources.list.d/portaptable.list
sudo apt update</pre>
    <h2>Available Endpoints:</h2>
    <ul>
        <li><a href="/info">/info</a> - Repository information</li>
        <li><a href="/health">/health</a> - Health check</li>
        <li><a href="/dists/">/dists/</a> - Distribution metadata</li>
        <li><a href="/pool/">/pool/</a> - Package files</li>
    </ul>
</body>
</html>`, len(s.manifest.Packages), s.config.Port, s.manifest.Distribution)
		return
	}

	http.NotFound(w, r)
}

func (s *RepositoryServer) handleDists(w http.ResponseWriter, r *http.Request) {
	// Remove /dists/ prefix
	path := strings.TrimPrefix(r.URL.Path, "/dists/")

	// Serve files from the dists directory
	filePath := filepath.Join(s.config.RepoPath, "dists", path)

	// Security check - ensure we're not serving files outside the repository
	absRepoPath, _ := filepath.Abs(s.config.RepoPath)
	absFilePath, _ := filepath.Abs(filePath)

	if !strings.HasPrefix(absFilePath, absRepoPath) {
		http.Error(w, "Access denied", http.StatusForbidden)

		return
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.NotFound(w, r)

		return
	}

	// Serve the file
	http.ServeFile(w, r, filePath)
}

func (s *RepositoryServer) handlePool(w http.ResponseWriter, r *http.Request) {
	// Remove /pool/ prefix
	filename := strings.TrimPrefix(r.URL.Path, "/pool/")

	// Serve files from the pool directory
	filePath := filepath.Join(s.config.RepoPath, "pool", filename)

	// Security check
	absRepoPath, _ := filepath.Abs(s.config.RepoPath)
	absFilePath, _ := filepath.Abs(filePath)

	if !strings.HasPrefix(absFilePath, absRepoPath) {
		http.Error(w, "Access denied", http.StatusForbidden)

		return
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.NotFound(w, r)

		return
	}

	// Set appropriate headers for .deb files
	if strings.HasSuffix(filename, ".deb") {
		w.Header().Set("Content-Type", "application/vnd.debian.binary-package")
	}

	// Serve the file
	http.ServeFile(w, r, filePath)

	return
}

func (s *RepositoryServer) handlePackagesFile(w http.ResponseWriter, r *http.Request) {
	// Generate Packages file content on-demand
	w.Header().Set("Content-Type", "text/plain")

	poolPath := filepath.Join(s.config.RepoPath, "pool")

	for _, pkg := range s.manifest.Packages {
		if !pkg.Downloaded {
			continue
		}

		pkgPath := filepath.Join(poolPath, pkg.Filename)

		if _, err := os.Stat(pkgPath); os.IsNotExist(err) {
			continue // Skip missing files
		}

		// Generate basic package entry
		fmt.Fprintf(w, "Package: %s\n", pkg.Name)
		fmt.Fprintf(w, "Version: %s\n", pkg.Version)
		fmt.Fprintf(w, "Architecture: %s\n", pkg.Architecture)
		fmt.Fprintf(w, "Filename: pool/%s\n", pkg.Filename)
		fmt.Fprintf(w, "Size: %d\n", pkg.Size)

		// TODO: Add MD5sum, SHA1, SHA256 checksums
		// For now, apt will work without them if we use [trusted=yes]
		fmt.Fprintf(w, "Description: Package downloaded by portaptable\n")
		fmt.Fprintf(w, "\n") // Empty line separates packages
	}

	return
}

func (s *RepositoryServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	health := map[string]interface{}{
		"status":         "ok",
		"packages_total": len(s.manifest.Packages),
		"packages_downloaded": func() int {
			count := 0

			for _, pkg := range s.manifest.Packages {
				if pkg.Downloaded {
					count++
				}
			}

			return count
		}(),
		"repository_path": s.config.RepoPath,
		"distribution":    s.manifest.Distribution,
		"architecture":    s.manifest.Architecture,
		"created_at":      s.manifest.CreatedAt,
	}

	json.NewEncoder(w).Encode(health)

	return
}

func (s *RepositoryServer) handleInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	info := map[string]interface{}{
		"repository": map[string]interface{}{
			"path":         s.config.RepoPath,
			"distribution": s.manifest.Distribution,
			"architecture": s.manifest.Architecture,
			"created_at":   s.manifest.CreatedAt,
		},
		"packages": s.manifest.Packages,
		"usage": map[string]string{
			"add_repo": fmt.Sprintf("echo 'deb [trusted=yes] http://localhost:%s/ %s main' | sudo tee /etc/apt/sources.list.d/portaptable.list",
				s.config.Port, s.manifest.Distribution),
			"update": "sudo apt update",
		},
	}

	json.NewEncoder(w).Encode(info)

	return
}
