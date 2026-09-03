// Release discovery and artifact download for the self-updater.
//
// This file owns everything that talks to GitHub — locating the latest release,
// downloading the platform archive, and verifying its checksum. updater.go owns
// the decision to update and the installation of the binary. The two change for
// different reasons (API/wire format vs. local filesystem behaviour), so they
// live apart.
package updater

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
)

// fetchLatestRelease gets the latest non-draft, non-prerelease from GitHub.
func fetchLatestRelease(baseURL string) (*Release, error) {
	owner, name := repoCoordinates()
	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", baseURL, owner, name)

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
