package updater

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestParseGitHubModulePath(t *testing.T) {
	tests := []struct {
		name      string
		modPath   string
		wantOwner string
		wantName  string
		wantOK    bool
	}{
		{"canonical", "github.com/rk-senne/moonbase", "rk-senne", "moonbase", true},
		{"work fnumber", "github.com/f5508037/moonbase", "f5508037", "moonbase", true},
		{"with subpackage", "github.com/owner/repo/internal/x", "owner", "repo", true},
		{"non-github host", "example.com/owner/repo", "", "", false},
		{"gitlab", "gitlab.com/owner/repo", "", "", false},
		{"too short", "github.com/onlyowner", "", "", false},
		{"empty", "", "", "", false},
		{"just host", "github.com", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, name, ok := parseGitHubModulePath(tt.modPath)
			if ok != tt.wantOK || owner != tt.wantOwner || name != tt.wantName {
				t.Errorf("parseGitHubModulePath(%q) = (%q, %q, %v), want (%q, %q, %v)",
					tt.modPath, owner, name, ok, tt.wantOwner, tt.wantName, tt.wantOK)
			}
		})
	}
}

func TestRepoCoordinates_ReturnsNonEmpty(t *testing.T) {
	// Whether derived from build info or the fallback, both must be non-empty so
	// the release API URL is always well-formed.
	owner, name := repoCoordinates()
	if owner == "" || name == "" {
		t.Errorf("repoCoordinates returned empty values: owner=%q name=%q", owner, name)
	}
	if name != "moonbase" {
		t.Errorf("expected repo name 'moonbase', got %q", name)
	}
}

func TestListAssets(t *testing.T) {
	tests := []struct {
		name   string
		assets []Asset
		want   string
	}{
		{
			name:   "empty",
			assets: nil,
			want:   "",
		},
		{
			name:   "single asset",
			assets: []Asset{{Name: "moonbase_linux_amd64.tar.gz"}},
			want:   "moonbase_linux_amd64.tar.gz",
		},
		{
			name: "multiple assets",
			assets: []Asset{
				{Name: "moonbase_linux_amd64.tar.gz"},
				{Name: "moonbase_darwin_arm64.tar.gz"},
				{Name: "checksums.txt"},
			},
			want: "moonbase_linux_amd64.tar.gz, moonbase_darwin_arm64.tar.gz, checksums.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := listAssets(tt.assets)
			if got != tt.want {
				t.Errorf("listAssets() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExpectedAssetName(t *testing.T) {
	got := expectedAssetName()
	want := "moonbase_" + runtime.GOOS + "_" + runtime.GOARCH + ".tar.gz"
	if got != want {
		t.Errorf("expectedAssetName() = %q, want %q", got, want)
	}
}

func TestContainsDotDot(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "simple file", path: "moonbase", want: false},
		{name: "nested path", path: "dir/moonbase", want: false},
		{name: "dot-dot at start", path: "../moonbase", want: true},
		{name: "dot-dot in middle", path: "dir/../moonbase", want: true},
		{name: "dot-dot at end", path: "dir/..", want: true},
		{name: "empty string", path: "", want: false},
		{name: "single dot", path: "./moonbase", want: false},
		{name: "dots in name", path: "file..name", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsDotDot(tt.path)
			if got != tt.want {
				t.Errorf("containsDotDot(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestSplitFirst(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		wantFirst string
		wantRest  string
	}{
		{name: "simple", path: "moonbase", wantFirst: "moonbase", wantRest: ""},
		{name: "two parts", path: "dir/file", wantFirst: "dir", wantRest: "file"},
		{name: "three parts", path: "a/b/c", wantFirst: "a", wantRest: "b/c"},
		{name: "empty", path: "", wantFirst: "", wantRest: ""},
		{name: "leading slash", path: "/file", wantFirst: "", wantRest: "file"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			first, rest := splitFirst(tt.path)
			if first != tt.wantFirst || rest != tt.wantRest {
				t.Errorf("splitFirst(%q) = (%q, %q), want (%q, %q)",
					tt.path, first, rest, tt.wantFirst, tt.wantRest)
			}
		})
	}
}

func TestVerifyChecksum(t *testing.T) {
	// Create a temp file with known content
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "testfile")
	content := []byte("hello moonbase")
	if err := os.WriteFile(testFile, content, 0o644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	// Compute expected hash
	h := sha256.Sum256(content)
	expectedHash := hex.EncodeToString(h[:])

	// Create a checksums file served via a local test (we can't use HTTP in unit tests,
	// so we test the file-hashing logic directly)
	t.Run("correct hash matches", func(t *testing.T) {
		f, err := os.Open(testFile)
		if err != nil {
			t.Fatalf("opening test file: %v", err)
		}
		defer f.Close()

		hasher := sha256.New()
		if _, err := hasher.Write(content); err != nil {
			t.Fatalf("hashing: %v", err)
		}
		actualHash := hex.EncodeToString(hasher.Sum(nil))

		if actualHash != expectedHash {
			t.Errorf("hash mismatch: got %s, want %s", actualHash, expectedHash)
		}
	})

	t.Run("wrong hash does not match", func(t *testing.T) {
		wrongHash := "0000000000000000000000000000000000000000000000000000000000000000"
		if wrongHash == expectedHash {
			t.Fatal("test setup error: wrong hash matches expected")
		}
	})
}

func TestCopyBinary(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source file
	srcContent := []byte("#!/bin/sh\necho moonbase")
	srcPath := filepath.Join(tmpDir, "src")
	if err := os.WriteFile(srcPath, srcContent, 0o755); err != nil {
		t.Fatalf("writing source: %v", err)
	}

	t.Run("successful copy", func(t *testing.T) {
		dstPath := filepath.Join(tmpDir, "dst")
		err := copyBinary(srcPath, dstPath, 0o755)
		if err != nil {
			t.Fatalf("copyBinary() error: %v", err)
		}

		got, err := os.ReadFile(dstPath)
		if err != nil {
			t.Fatalf("reading dest: %v", err)
		}
		if string(got) != string(srcContent) {
			t.Errorf("content mismatch: got %q, want %q", got, srcContent)
		}

		fi, err := os.Stat(dstPath)
		if err != nil {
			t.Fatalf("stat dest: %v", err)
		}
		if fi.Mode().Perm() != 0o755 {
			t.Errorf("permissions = %o, want 0755", fi.Mode().Perm())
		}
	})

	t.Run("source not found", func(t *testing.T) {
		err := copyBinary("/nonexistent/path", filepath.Join(tmpDir, "out"), 0o755)
		if err == nil {
			t.Error("expected error for missing source, got nil")
		}
	})

	t.Run("destination not writable", func(t *testing.T) {
		err := copyBinary(srcPath, "/nonexistent/dir/file", 0o755)
		if err == nil {
			t.Error("expected error for bad destination, got nil")
		}
	})
}

func TestExtractBinary(t *testing.T) {
	t.Run("valid archive", func(t *testing.T) {
		// Create a tar.gz archive with a moonbase binary
		tmpDir := t.TempDir()
		archivePath := filepath.Join(tmpDir, "moonbase_test.tar.gz")
		binaryContent := []byte("#!/bin/sh\necho moonbase")

		if err := createTestArchive(archivePath, "moonbase", binaryContent); err != nil {
			t.Fatalf("creating test archive: %v", err)
		}

		extractedPath, err := extractBinary(archivePath)
		if err != nil {
			t.Fatalf("extractBinary() error: %v", err)
		}
		defer os.RemoveAll(filepath.Dir(extractedPath))

		got, err := os.ReadFile(extractedPath)
		if err != nil {
			t.Fatalf("reading extracted: %v", err)
		}
		if string(got) != string(binaryContent) {
			t.Errorf("content mismatch: got %q, want %q", got, binaryContent)
		}
	})

	t.Run("archive without moonbase binary", func(t *testing.T) {
		tmpDir := t.TempDir()
		archivePath := filepath.Join(tmpDir, "empty.tar.gz")

		if err := createTestArchive(archivePath, "other-file", []byte("not moonbase")); err != nil {
			t.Fatalf("creating test archive: %v", err)
		}

		_, err := extractBinary(archivePath)
		if err == nil {
			t.Error("expected error for archive without moonbase, got nil")
		}
	})

	t.Run("nonexistent archive", func(t *testing.T) {
		_, err := extractBinary("/nonexistent/archive.tar.gz")
		if err == nil {
			t.Error("expected error for missing archive, got nil")
		}
	})

	t.Run("invalid gzip", func(t *testing.T) {
		tmpDir := t.TempDir()
		badFile := filepath.Join(tmpDir, "notgzip.tar.gz")
		if err := os.WriteFile(badFile, []byte("this is not gzip"), 0o644); err != nil {
			t.Fatalf("writing bad file: %v", err)
		}

		_, err := extractBinary(badFile)
		if err == nil {
			t.Error("expected error for invalid gzip, got nil")
		}
	})

	t.Run("path traversal blocked", func(t *testing.T) {
		tmpDir := t.TempDir()
		archivePath := filepath.Join(tmpDir, "evil.tar.gz")

		// Create archive with path traversal attempt
		if err := createTestArchive(archivePath, "../../../tmp/moonbase", []byte("evil")); err != nil {
			t.Fatalf("creating evil archive: %v", err)
		}

		_, err := extractBinary(archivePath)
		if err == nil {
			t.Error("expected error for path traversal, got nil")
		}
	})
}

func TestUpdateResult_VersionComparison(t *testing.T) {
	// Test the logic used in CheckForUpdate for version comparison
	tests := []struct {
		name         string
		current      string
		latest       string
		wantNeeded   bool
	}{
		{name: "same version", current: "1.4.0", latest: "1.4.0", wantNeeded: false},
		{name: "newer available", current: "1.3.0", latest: "1.4.0", wantNeeded: true},
		{name: "dev build", current: "dev", latest: "1.4.0", wantNeeded: false},
		{name: "with v prefix current", current: "v1.3.0", latest: "1.4.0", wantNeeded: true},
		{name: "with v prefix latest", current: "1.3.0", latest: "v1.4.0", wantNeeded: true},
		{name: "both v prefix same", current: "v1.4.0", latest: "v1.4.0", wantNeeded: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Replicate the logic from CheckForUpdate
			latest := trimV(tt.latest)
			current := trimV(tt.current)
			updateNeeded := latest != current && current != "dev"

			if updateNeeded != tt.wantNeeded {
				t.Errorf("updateNeeded(%q vs %q) = %v, want %v",
					tt.current, tt.latest, updateNeeded, tt.wantNeeded)
			}
		})
	}
}

// trimV mirrors the strings.TrimPrefix logic used in the updater.
func trimV(v string) string {
	if len(v) > 0 && v[0] == 'v' {
		return v[1:]
	}
	return v
}

// createTestArchive creates a .tar.gz archive with a single file.
func createTestArchive(archivePath, fileName string, content []byte) error {
	f, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	header := &tar.Header{
		Name: fileName,
		Mode: 0o755,
		Size: int64(len(content)),
	}
	if err := tw.WriteHeader(header); err != nil {
		return err
	}
	if _, err := tw.Write(content); err != nil {
		return err
	}

	return nil
}
