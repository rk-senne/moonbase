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
	"crypto/tls"
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

	// githubAPIBase is the GitHub REST API root. It is passed explicitly down to
	// fetchLatestRelease rather than hardcoded there, so tests can point the
	// updater at an httptest server without a mutable package-level override.
	githubAPIBase = "https://api.github.com"
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
	return checkForUpdate(currentVersion, githubAPIBase)
}

// checkForUpdate is CheckForUpdate with the API root injected so tests can
// exercise every branch against an httptest server.
func checkForUpdate(currentVersion, baseURL string) (*UpdateResult, error) {
	release, err := fetchLatestRelease(baseURL)
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

// selectReleaseAssets picks the platform binary archive and the optional
// checksums file out of a release's asset list.
//
// This is deliberately a pure function: it is the part of the update decision
// that can be exercised exhaustively in tests, separated from the filesystem and
// network work in update() which cannot be (that path replaces the running
// executable). Errors here describe exactly what was expected versus available.
func selectReleaseAssets(release *Release) (assetName, downloadURL, checksumURL string, err error) {
	assetName = expectedAssetName()
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}
	if downloadURL == "" {
		return "", "", "", fmt.Errorf("no binary found for %s/%s in release %s\n  Expected: %s\n  Available: %s",
			runtime.GOOS, runtime.GOARCH, release.TagName, assetName, listAssets(release.Assets))
	}

	for _, asset := range release.Assets {
		if asset.Name == "checksums.txt" {
			checksumURL = asset.BrowserDownloadURL
			break
		}
	}
	return assetName, downloadURL, checksumURL, nil
}

// Update downloads and installs the latest release, replacing the current binary.
// Returns the new version string on success.
func Update(currentVersion string) (string, error) {
	return update(currentVersion, githubAPIBase)
}

// update is Update with the API root injected for testing. Note that the tail of
// this function replaces the running executable, so tests only exercise the
// paths that return before that point.
func update(currentVersion, baseURL string) (string, error) {
	release, err := fetchLatestRelease(baseURL)
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

	assetName, downloadURL, checksumURL, err := selectReleaseAssets(release)
	if err != nil {
		return "", err
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

	// Atomic-ish replace with rollback. Extracted so the backup/restore logic can
	// be tested against temp files rather than the running executable.
	if err := replaceBinary(execPath, extractedPath, originalMode); err != nil {
		return "", err
	}

	return latest, nil
}

// replaceBinary swaps newBinaryPath into execPath, keeping a backup so a failed
// install can be rolled back.
//
// We copy rather than rename the new binary because the extracted file may live
// on a different filesystem. If the copy fails, the backup is restored so the
// caller is left with a working binary rather than a missing one — that rollback
// is the reason this is a separate, tested function.
func replaceBinary(execPath, newBinaryPath string, mode os.FileMode) error {
	backupPath := execPath + ".bak"
	os.Remove(backupPath) // Remove any stale backup

	if err := os.Rename(execPath, backupPath); err != nil {
		return fmt.Errorf("backing up current binary: %w", err)
	}

	if err := copyBinary(newBinaryPath, execPath, mode); err != nil {
		// Restore backup on failure so the install is not left half-applied.
		if restoreErr := os.Rename(backupPath, execPath); restoreErr != nil {
			return fmt.Errorf("installing new binary: %w (and restoring backup failed: %v)", err, restoreErr)
		}
		return fmt.Errorf("installing new binary: %w", err)
	}

	// Clean up backup
	os.Remove(backupPath)
	return nil
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
