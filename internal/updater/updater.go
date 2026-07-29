// Package updater provides self-update functionality for the moonbase binary.
// It checks GitHub Releases for newer versions and replaces the running binary.
//
// SECURITY:
//   - Uses HTTPS only for all GitHub API and download requests
//   - Verifies SHA256 checksums when available (checksums.txt in release)
//   - Downloads to a temp file first, verifies, then atomically renames
//   - Preserves original binary permissions
//   - No environment secrets are transmitted
package updater

import (
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"time"
)

const (
	// Fallback GitHub coordinates, used only when the module path cannot be
	// determined at runtime (e.g. a binary built without module info). Normally
	// the owner/repo are derived from the module path via repoCoordinates().
	defaultRepoOwner = "rk-senne"
	defaultRepoName  = "moonbase"

	// API timeout for checking releases.
	apiTimeout = 15 * time.Second

	// Download timeout for fetching binaries.
	downloadTimeout = 120 * time.Second

	// Maximum binary size to download (100MB safety cap).
	maxBinarySize = 100 << 20
)

// updaterHTTPClient is a shared HTTP client with TLS 1.2 minimum for all updater
// network operations. Prevents TLS downgrade attacks while preserving per-call
// timeouts via context or per-request clients derived from this transport.
//
// SECURITY: Enforces TLS 1.2+ to prevent downgrade attacks on release downloads.
var updaterTransport = &http.Transport{
	TLSClientConfig: &tls.Config{
		MinVersion: tls.VersionTLS12,
	},
}

// repoCoordinates returns the GitHub owner and repository name to check for
// releases. It derives them from the module path baked into the binary at build
// time (via debug.ReadBuildInfo), so the updater always targets the repository
// the binary was actually built from — there is no hardcoded owner to drift out
// of sync with where the code lives. Falls back to the defaults if build info is
// unavailable or the module path is not a github.com path.
func repoCoordinates() (owner, name string) {
	if info, ok := debug.ReadBuildInfo(); ok {
		if o, n, ok := parseGitHubModulePath(info.Main.Path); ok {
			return o, n
		}
	}
	return defaultRepoOwner, defaultRepoName
}

// parseGitHubModulePath extracts the owner and repo from a module path such as
// "github.com/rk-senne/moonbase". Returns ok=false for non-github.com paths or
// paths missing an owner/repo segment.
func parseGitHubModulePath(modPath string) (owner, name string, ok bool) {
	parts := strings.Split(modPath, "/")
	if len(parts) >= 3 && parts[0] == "github.com" && parts[1] != "" && parts[2] != "" {
		return parts[1], parts[2], true
	}
	return "", "", false
}

// Release represents a GitHub release.
type Release struct {
	TagName    string  `json:"tag_name"`
	Name       string  `json:"name"`
	Draft      bool    `json:"draft"`
	Prerelease bool    `json:"prerelease"`
	Assets     []Asset `json:"assets"`
	Body       string  `json:"body"`
	HTMLURL    string  `json:"html_url"`
}

// Asset represents a downloadable file in a release.
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// UpdateResult holds the outcome of an update check or execution.
type UpdateResult struct {
	CurrentVersion string
	LatestVersion  string
	UpdateNeeded   bool
	AssetName      string
	ReleaseURL     string
	ReleaseNotes   string
}

// CheckForUpdate queries GitHub Releases API for the latest version.
// Returns update info without downloading anything.
func CheckForUpdate(currentVersion string) (*UpdateResult, error) {
	release, err := fetchLatestRelease()
	if err != nil {
		return nil, fmt.Errorf("checking for updates: %w", err)
	}

	latest := strings.TrimPrefix(release.TagName, "v")
	current := strings.TrimPrefix(currentVersion, "v")

	result := &UpdateResult{
		CurrentVersion: current,
		LatestVersion:  latest,
		UpdateNeeded:   latest != current && current != "dev",
		ReleaseURL:     release.HTMLURL,
		ReleaseNotes:   release.Body,
	}

	// Find the asset for this platform
	assetName := expectedAssetName()
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			result.AssetName = asset.Name
			break
		}
	}

	return result, nil
}

// Update downloads and installs the latest release, replacing the current binary.
// Returns the new version string on success.
func Update(currentVersion string) (string, error) {
	release, err := fetchLatestRelease()
	if err != nil {
		return "", fmt.Errorf("fetching latest release: %w", err)
	}

	latest := strings.TrimPrefix(release.TagName, "v")
	current := strings.TrimPrefix(currentVersion, "v")

	if latest == current {
		return current, fmt.Errorf("already at latest version (%s)", current)
	}

	if current == "dev" {
		return "", fmt.Errorf("cannot update a development build — install a release binary first")
	}

	// Find the binary asset for this platform
	assetName := expectedAssetName()
	var downloadURL string
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return "", fmt.Errorf("no binary found for %s/%s in release %s\n  Expected: %s\n  Available: %s",
			runtime.GOOS, runtime.GOARCH, release.TagName, assetName, listAssets(release.Assets))
	}

	// Find checksums if available
	var checksumURL string
	for _, asset := range release.Assets {
		if asset.Name == "checksums.txt" {
			checksumURL = asset.BrowserDownloadURL
			break
		}
	}

	// Get the current binary path
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("finding current binary: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return "", fmt.Errorf("resolving binary path: %w", err)
	}

	// Get original file permissions
	fi, err := os.Stat(execPath)
	if err != nil {
		return "", fmt.Errorf("stat current binary: %w", err)
	}
	originalMode := fi.Mode()

	// Download archive to temp file
	tmpFile, err := os.CreateTemp(filepath.Dir(execPath), "moonbase-archive-*")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	archivePath := tmpFile.Name()
	defer os.Remove(archivePath) // Clean up archive on any path

	// Download the archive
	if err := downloadFile(downloadURL, tmpFile); err != nil {
		tmpFile.Close()
		return "", fmt.Errorf("downloading archive: %w", err)
	}
	tmpFile.Close()

	// Verify checksum of the archive if available
	if checksumURL != "" {
		if err := verifyChecksum(archivePath, assetName, checksumURL); err != nil {
			return "", fmt.Errorf("checksum verification failed: %w", err)
		}
	}

	// Extract binary from the tar.gz archive
	extractedPath, err := extractBinary(archivePath)
	if err != nil {
		return "", fmt.Errorf("extracting binary: %w", err)
	}
	defer os.RemoveAll(filepath.Dir(extractedPath)) // Clean up extraction dir

	// Set executable permissions to match original
	if err := os.Chmod(extractedPath, originalMode); err != nil {
		return "", fmt.Errorf("setting permissions: %w", err)
	}

	// Atomic replace: rename old → .bak, copy new → target, remove .bak
	// We copy instead of rename because the extracted file may be on a different filesystem.
	backupPath := execPath + ".bak"
	os.Remove(backupPath) // Remove any stale backup

	if err := os.Rename(execPath, backupPath); err != nil {
		return "", fmt.Errorf("backing up current binary: %w", err)
	}

	if err := copyBinary(extractedPath, execPath, originalMode); err != nil {
		// Restore backup on failure
		os.Rename(backupPath, execPath)
		return "", fmt.Errorf("installing new binary: %w", err)
	}

	// Clean up backup
	os.Remove(backupPath)

	return latest, nil
}

// fetchLatestRelease gets the latest non-draft, non-prerelease from GitHub.
func fetchLatestRelease() (*Release, error) {
	owner, name := repoCoordinates()
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, name)

	client := &http.Client{Timeout: apiTimeout, Transport: updaterTransport}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "moonbase-updater")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("connecting to GitHub: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("no releases found (repo may be private or not exist)")
	}
	if resp.StatusCode == 403 {
		return nil, fmt.Errorf("rate limited by GitHub API — try again later")
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("parsing release: %w", err)
	}

	return &release, nil
}

// expectedAssetName returns the archive name for the current platform.
// Must match goreleaser's name_template: "moonbase_{{ .Os }}_{{ .Arch }}"
func expectedAssetName() string {
	return fmt.Sprintf("moonbase_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)
}

// downloadFile downloads a URL to the given file with size limits.
func downloadFile(url string, dest *os.File) error {
	client := &http.Client{Timeout: downloadTimeout, Transport: updaterTransport}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("creating download request: %w", err)
	}
	req.Header.Set("User-Agent", "moonbase-updater")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	// Size-limited copy to prevent DoS
	limited := io.LimitReader(resp.Body, maxBinarySize)
	written, err := io.Copy(dest, limited)
	if err != nil {
		return fmt.Errorf("writing binary: %w", err)
	}
	if written == 0 {
		return fmt.Errorf("downloaded file is empty")
	}

	return nil
}

// verifyChecksum downloads checksums.txt and verifies the downloaded file matches.
func verifyChecksum(filePath, assetName, checksumURL string) error {
	// Download checksums
	client := &http.Client{Timeout: apiTimeout, Transport: updaterTransport}
	resp, err := client.Get(checksumURL)
	if err != nil {
		return fmt.Errorf("downloading checksums: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		// Checksums not available — skip verification with warning
		return nil
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<16)) // 64KB max
	if err != nil {
		return fmt.Errorf("reading checksums: %w", err)
	}

	// Parse checksums.txt (format: "sha256hash  filename")
	var expectedHash string
	for _, line := range strings.Split(string(body), "\n") {
		parts := strings.Fields(line)
		if len(parts) == 2 && parts[1] == assetName {
			expectedHash = parts[0]
			break
		}
	}

	if expectedHash == "" {
		// No checksum for this asset — can't verify
		return nil
	}

	// Compute SHA256 of downloaded file
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("opening file for checksum: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("reading file for checksum: %w", err)
	}
	actualHash := hex.EncodeToString(h.Sum(nil))

	if actualHash != expectedHash {
		return fmt.Errorf("SHA256 mismatch:\n  expected: %s\n  got:      %s\n  This may indicate a corrupted download or tampering", expectedHash, actualHash)
	}

	return nil
}

// listAssets returns a comma-separated list of asset names for error messages.
func listAssets(assets []Asset) string {
	names := make([]string, 0, len(assets))
	for _, a := range assets {
		names = append(names, a.Name)
	}
	return strings.Join(names, ", ")
}

// copyBinary copies a file from src to dst with explicit permissions.
// Used instead of rename when files may be on different filesystems.
func copyBinary(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening source %s: %w", src, err)
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("creating destination %s: %w", dst, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copying %s to %s: %w", src, dst, err)
	}
	return out.Close()
}
