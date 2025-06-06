package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"portaptable/pkg/config"
	"portaptable/pkg/manifest"
	"portaptable/pkg/packageinfo"
	"regexp"
	"strings"
	"time"
)

func runDownloadMode(config *config.Config) error {
	fmt.Println("Resolving package dependencies...")

	// Get all dependencies for the requested packages
	allPackages, err := resolveAllDependencies(config.Packages, config.Architecture)

	if err != nil {
		return fmt.Errorf("failed to resolve dependencies: %w", err)
	}

	fmt.Printf("Found %d packages to download (including dependencies)\n", len(allPackages))

	// Create manifest
	mfest := manifest.Manifest{
		CreatedAt:    time.Now(),
		Architecture: config.Architecture,
		Distribution: config.Distribution,
		Packages:     make([]packageinfo.PackageInfo, 0, len(allPackages)),
	}

	// Download each package
	poolPath := filepath.Join(config.RepoPath, "pool")

	for i, pkg := range allPackages {
		fmt.Printf("[%d/%d] Processing %s...\n", i+1, len(allPackages), pkg)

		packageInfo, err := downloadPackage(pkg, poolPath, config.Architecture)

		if err != nil {
			fmt.Printf("Warning: Failed to download %s: %v\n", pkg, err)
			packageInfo = packageinfo.PackageInfo{
				Name:         pkg,
				Architecture: config.Architecture,
				Downloaded:   false,
			}
		} else {
			fmt.Printf("Downloaded %s (%d bytes)\n", packageInfo.Filename, packageInfo.Size)
		}

		mfest.Packages = append(mfest.Packages, packageInfo)
	}

	// Save manifest
	if err := saveManifest(config.RepoPath, mfest); err != nil {
		return fmt.Errorf("failed to save manifest: %w", err)
	}

	// Generate repository metadata
	if err := generateRepositoryMetadata(config); err != nil {
		return fmt.Errorf("failed to generate repository metadata: %w", err)
	}

	fmt.Printf("Successfully processed %d packages\n", len(mfest.Packages))

	return nil
}

func resolveAllDependencies(packages []string, architecture string) ([]string, error) {
	allPackages := make(map[string]bool)

	for _, pkg := range packages {
		deps, err := getDependencies(pkg, architecture)

		if err != nil {
			return nil, fmt.Errorf("failed to get dependencies for %s: %w", pkg, err)
		}

		// Add the package itself and all its dependencies
		allPackages[pkg] = true

		for _, dep := range deps {
			allPackages[dep] = true
		}
	}

	// Convert map to slice
	result := make([]string, 0, len(allPackages))

	for pkg := range allPackages {
		result = append(result, pkg)
	}

	return result, nil
}

func getDependencies(packageName, architecture string) ([]string, error) {
	// Use apt-cache to get recursive dependencies
	cmd := exec.Command("apt-cache", "depends", "--recurse", "--no-recommends",
		"--no-suggests", "--no-conflicts", "--no-breaks", "--no-replaces",
		"--no-enhances", packageName)

	output, err := cmd.Output()

	if err != nil {
		return nil, fmt.Errorf("apt-cache command failed: %w", err)
	}

	return parseDependencyOutput(string(output)), nil
}

func parseDependencyOutput(output string) []string {
	var packages []string
	seen := make(map[string]bool)

	// Regular expression to match package names from apt-cache depends output
	// Looks for lines like "  Depends: package-name" or "package-name"
	packageRegex := regexp.MustCompile(`^\s*(?:Depends:\s+)?([a-zA-Z0-9][a-zA-Z0-9\-\+\.]+)`)

	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and certain dependency types we don't want
		if line == "" || strings.Contains(line, "|") ||
			strings.Contains(line, "Recommends:") ||
			strings.Contains(line, "Suggests:") {

			continue
		}

		matches := packageRegex.FindStringSubmatch(line)

		if len(matches) > 1 {
			pkg := matches[1]

			// Filter out virtual packages and duplicates
			if !seen[pkg] && !strings.HasPrefix(pkg, "<") {
				packages = append(packages, pkg)
				seen[pkg] = true
			}
		}
	}

	return packages
}

func downloadPackage(packageName, poolPath, architecture string) (packageinfo.PackageInfo, error) {
	// Use apt-get download to get the package
	cmd := exec.Command("apt-get", "download", packageName)
	cmd.Dir = poolPath

	output, err := cmd.CombinedOutput()

	if err != nil {
		return packageinfo.PackageInfo{}, fmt.Errorf("apt-get download failed: %w, output: %s", err, string(output))
	}

	// Find the downloaded file
	files, err := filepath.Glob(filepath.Join(poolPath, fmt.Sprintf("%s_*.deb", packageName)))

	if err != nil {
		return packageinfo.PackageInfo{}, fmt.Errorf("failed to find downloaded file: %w", err)
	}

	if len(files) == 0 {
		return packageinfo.PackageInfo{}, fmt.Errorf("no .deb file found after download")
	}

	// Get the most recent file (in case there are multiple versions)
	filename := filepath.Base(files[len(files)-1])

	// Get file info
	stat, err := os.Stat(files[len(files)-1])

	if err != nil {
		return packageinfo.PackageInfo{}, fmt.Errorf("failed to stat downloaded file: %w", err)
	}

	// Parse version from filename (format: package_version_architecture.deb)
	version := "unknown"
	parts := strings.Split(filename, "_")

	if len(parts) >= 2 {
		version = parts[1]
	}

	return packageinfo.PackageInfo{
		Name:         packageName,
		Version:      version,
		Architecture: architecture,
		Filename:     filename,
		Size:         stat.Size(),
		Downloaded:   true,
	}, nil
}

func saveManifest(repoPath string, mfest manifest.Manifest) error {
	manifestPath := filepath.Join(repoPath, "manifest.json")
	data, err := json.MarshalIndent(mfest, "", "  ")

	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	return os.WriteFile(manifestPath, data, 0644)
}

func generateRepositoryMetadata(config *config.Config) error {
	// TODO: Generate proper Debian repository metadata (Packages, Release files)
	// This is complex and involves creating proper apt repository structure
	fmt.Println("Repository metadata generation - placeholder")

	// Create basic directory structure for now
	distPath := filepath.Join(config.RepoPath, "dists", config.Distribution)
	mainPath := filepath.Join(distPath, "main", "binary-"+config.Architecture)

	if err := os.MkdirAll(mainPath, 0755); err != nil {
		return fmt.Errorf("failed to create dist directories: %w", err)
	}

	// Create a basic Release file
	releasePath := filepath.Join(distPath, "Release")
	releaseContent := fmt.Sprintf(`Suite: %s
Components: main
Architectures: %s
Date: %s
`, config.Distribution, config.Architecture, time.Now().Format(time.RFC1123Z))

	return os.WriteFile(releasePath, []byte(releaseContent), 0644)
}
