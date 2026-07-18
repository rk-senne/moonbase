package updater

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// extractBinary extracts the "moonbase" binary from a .tar.gz archive.
// Returns the path to the extracted binary (in a temp dir).
//
// SECURITY:
//   - Only extracts files named "moonbase" (the binary we want)
//   - Validates path components to prevent zip-slip (CWE-22)
//   - Size-limited extraction (maxBinarySize)
func extractBinary(archivePath string) (string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return "", fmt.Errorf("opening archive: %w", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", fmt.Errorf("not a valid gzip archive: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)

	// Create temp dir for extraction
	tmpDir, err := os.MkdirTemp("", "moonbase-extract-*")
	if err != nil {
		return "", fmt.Errorf("creating temp directory: %w", err)
	}

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			os.RemoveAll(tmpDir)
			return "", fmt.Errorf("reading archive: %w", err)
		}

		// Only extract the moonbase binary
		name := filepath.Base(header.Name)
		if name != "moonbase" {
			continue
		}

		// Security: validate no path traversal
		if filepath.IsAbs(header.Name) || containsDotDot(header.Name) {
			os.RemoveAll(tmpDir)
			return "", fmt.Errorf("suspicious path in archive: %s", header.Name)
		}

		// Only regular files
		if header.Typeflag != tar.TypeReg {
			continue
		}

		// Size check
		if header.Size > maxBinarySize {
			os.RemoveAll(tmpDir)
			return "", fmt.Errorf("binary too large: %d bytes (max %d)", header.Size, maxBinarySize)
		}

		destPath := filepath.Join(tmpDir, "moonbase")
		dest, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
		if err != nil {
			os.RemoveAll(tmpDir)
			return "", fmt.Errorf("creating extraction destination: %w", err)
		}

		written, err := io.Copy(dest, io.LimitReader(tr, maxBinarySize))
		dest.Close()
		if err != nil {
			os.RemoveAll(tmpDir)
			return "", fmt.Errorf("extracting binary: %w", err)
		}
		if written == 0 {
			os.RemoveAll(tmpDir)
			return "", fmt.Errorf("extracted binary is empty")
		}

		return destPath, nil
	}

	os.RemoveAll(tmpDir)
	return "", fmt.Errorf("moonbase binary not found in archive")
}

// containsDotDot checks for path traversal components.
func containsDotDot(path string) bool {
	for _, part := range filepath.SplitList(path) {
		if part == ".." {
			return true
		}
	}
	// Also check individual path components
	for path != "" {
		var component string
		component, path = splitFirst(path)
		if component == ".." {
			return true
		}
	}
	return false
}

// splitFirst splits a path into its first component and the rest.
func splitFirst(path string) (string, string) {
	i := 0
	for i < len(path) && path[i] != '/' {
		i++
	}
	if i >= len(path) {
		return path, ""
	}
	return path[:i], path[i+1:]
}
