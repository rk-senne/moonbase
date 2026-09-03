package updater

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// releaseJSON builds a GitHub release payload with the given tag and assets.
func releaseJSON(t *testing.T, tag string, assets map[string]string) string {
	t.Helper()
	type asset struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	}
	payload := struct {
		TagName string  `json:"tag_name"`
		HTMLURL string  `json:"html_url"`
		Body    string  `json:"body"`
		Assets  []asset `json:"assets"`
	}{
		TagName: tag,
		HTMLURL: "https://example.test/releases/" + tag,
		Body:    "release notes for " + tag,
	}
	for name, url := range assets {
		payload.Assets = append(payload.Assets, asset{Name: name, BrowserDownloadURL: url})
	}
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshalling release: %v", err)
	}
	return string(b)
}

// releaseServer serves a fixed release payload at the latest-release endpoint.
func releaseServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/releases/latest") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(status)
		fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// === fetchLatestRelease ===

func TestFetchLatestRelease_Success(t *testing.T) {
	srv := releaseServer(t, 200, releaseJSON(t, "v9.9.9", nil))

	rel, err := fetchLatestRelease(srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rel.TagName != "v9.9.9" {
		t.Errorf("TagName = %q, want v9.9.9", rel.TagName)
	}
	if rel.Body != "release notes for v9.9.9" {
		t.Errorf("Body = %q", rel.Body)
	}
}

func TestFetchLatestRelease_StatusErrors(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		wantSubstr string
	}{
		{name: "not found", status: 404, wantSubstr: "no releases found"},
		{name: "rate limited", status: 403, wantSubstr: "rate limited"},
		{name: "server error", status: 500, wantSubstr: "status 500"},
		{name: "teapot", status: 418, wantSubstr: "status 418"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := releaseServer(t, tt.status, "{}")
			_, err := fetchLatestRelease(srv.URL)
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), tt.wantSubstr) {
				t.Errorf("error = %q, want substring %q", err, tt.wantSubstr)
			}
		})
	}
}

func TestFetchLatestRelease_MalformedJSON(t *testing.T) {
	srv := releaseServer(t, 200, "{not json")
	_, err := fetchLatestRelease(srv.URL)
	if err == nil || !strings.Contains(err.Error(), "parsing release") {
		t.Fatalf("expected parsing error, got %v", err)
	}
}

func TestFetchLatestRelease_UnreachableHost(t *testing.T) {
	// Closed server → connection refused.
	srv := releaseServer(t, 200, "{}")
	url := srv.URL
	srv.Close()

	_, err := fetchLatestRelease(url)
	if err == nil || !strings.Contains(err.Error(), "connecting to GitHub") {
		t.Fatalf("expected connection error, got %v", err)
	}
}

// === checkForUpdate ===

func TestCheckForUpdate_UpdateAvailable(t *testing.T) {
	asset := expectedAssetName()
	srv := releaseServer(t, 200, releaseJSON(t, "v2.0.0", map[string]string{
		asset: "https://example.test/" + asset,
	}))

	res, err := checkForUpdate("1.0.0", srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.UpdateNeeded {
		t.Error("expected UpdateNeeded = true")
	}
	if res.LatestVersion != "2.0.0" || res.CurrentVersion != "1.0.0" {
		t.Errorf("versions = %q/%q", res.CurrentVersion, res.LatestVersion)
	}
	if res.AssetName != asset {
		t.Errorf("AssetName = %q, want %q", res.AssetName, asset)
	}
}

func TestCheckForUpdate_AlreadyLatestNeedsNoUpdate(t *testing.T) {
	srv := releaseServer(t, 200, releaseJSON(t, "v1.2.3", nil))

	res, err := checkForUpdate("v1.2.3", srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.UpdateNeeded {
		t.Error("expected UpdateNeeded = false at the same version")
	}
}

// A dev build must never report that an update is needed — that would prompt a
// developer to overwrite a local build with a release binary.
func TestCheckForUpdate_DevBuildNeverNeedsUpdate(t *testing.T) {
	srv := releaseServer(t, 200, releaseJSON(t, "v5.0.0", nil))

	res, err := checkForUpdate("dev", srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.UpdateNeeded {
		t.Error("dev build must not report UpdateNeeded")
	}
}

func TestCheckForUpdate_NoAssetForPlatformLeavesNameEmpty(t *testing.T) {
	srv := releaseServer(t, 200, releaseJSON(t, "v2.0.0", map[string]string{
		"moonbase_plan9_sparc.tar.gz": "https://example.test/other",
	}))

	res, err := checkForUpdate("1.0.0", srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.AssetName != "" {
		t.Errorf("AssetName = %q, want empty when no platform asset exists", res.AssetName)
	}
}

func TestCheckForUpdate_PropagatesFetchError(t *testing.T) {
	srv := releaseServer(t, 500, "{}")
	if _, err := checkForUpdate("1.0.0", srv.URL); err == nil ||
		!strings.Contains(err.Error(), "checking for updates") {
		t.Fatalf("expected wrapped fetch error, got %v", err)
	}
}

// === update (paths that return before touching the binary) ===

func TestUpdate_AlreadyAtLatestReturnsCurrentAndError(t *testing.T) {
	srv := releaseServer(t, 200, releaseJSON(t, "v3.1.4", nil))

	got, err := update("3.1.4", srv.URL)
	if err == nil || !strings.Contains(err.Error(), "already at latest") {
		t.Fatalf("expected already-at-latest error, got %v", err)
	}
	if got != "3.1.4" {
		t.Errorf("expected current version returned alongside error, got %q", got)
	}
}

// Refusing to update a dev build protects an in-progress local build from being
// silently replaced by a release artifact.
func TestUpdate_RefusesDevBuild(t *testing.T) {
	srv := releaseServer(t, 200, releaseJSON(t, "v3.0.0", nil))

	if _, err := update("dev", srv.URL); err == nil ||
		!strings.Contains(err.Error(), "development build") {
		t.Fatalf("expected dev-build refusal, got %v", err)
	}
}

func TestUpdate_MissingPlatformAssetAborts(t *testing.T) {
	srv := releaseServer(t, 200, releaseJSON(t, "v4.0.0", map[string]string{
		"moonbase_plan9_sparc.tar.gz": "https://example.test/other",
	}))

	_, err := update("1.0.0", srv.URL)
	if err == nil || !strings.Contains(err.Error(), "no binary found") {
		t.Fatalf("expected missing-asset error, got %v", err)
	}
	// The error must name what was expected and what was available so a user can act.
	if !strings.Contains(err.Error(), expectedAssetName()) ||
		!strings.Contains(err.Error(), "plan9") {
		t.Errorf("error should list expected and available assets, got %q", err)
	}
}

func TestUpdate_PropagatesFetchError(t *testing.T) {
	srv := releaseServer(t, 404, "{}")
	if _, err := update("1.0.0", srv.URL); err == nil ||
		!strings.Contains(err.Error(), "fetching latest release") {
		t.Fatalf("expected wrapped fetch error, got %v", err)
	}
}

// === selectReleaseAssets (pure) ===

func TestSelectReleaseAssets_FindsBinaryAndChecksums(t *testing.T) {
	asset := expectedAssetName()
	rel := &Release{
		TagName: "v1.0.0",
		Assets: []Asset{
			{Name: "checksums.txt", BrowserDownloadURL: "https://example.test/checksums.txt"},
			{Name: asset, BrowserDownloadURL: "https://example.test/" + asset},
		},
	}

	name, dl, sum, err := selectReleaseAssets(rel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != asset {
		t.Errorf("assetName = %q, want %q", name, asset)
	}
	if dl != "https://example.test/"+asset {
		t.Errorf("downloadURL = %q", dl)
	}
	if sum != "https://example.test/checksums.txt" {
		t.Errorf("checksumURL = %q", sum)
	}
}

func TestSelectReleaseAssets_ChecksumsOptional(t *testing.T) {
	asset := expectedAssetName()
	rel := &Release{
		TagName: "v1.0.0",
		Assets:  []Asset{{Name: asset, BrowserDownloadURL: "https://example.test/" + asset}},
	}

	_, dl, sum, err := selectReleaseAssets(rel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dl == "" {
		t.Error("expected a download URL")
	}
	if sum != "" {
		t.Errorf("expected empty checksumURL, got %q", sum)
	}
}

func TestSelectReleaseAssets_NoMatchingBinary(t *testing.T) {
	rel := &Release{TagName: "v1.0.0", Assets: []Asset{{Name: "unrelated.zip"}}}

	if _, _, _, err := selectReleaseAssets(rel); err == nil ||
		!strings.Contains(err.Error(), "no binary found") {
		t.Fatalf("expected no-binary error, got %v", err)
	}
}

func TestSelectReleaseAssets_EmptyAssetList(t *testing.T) {
	rel := &Release{TagName: "v1.0.0"}
	if _, _, _, err := selectReleaseAssets(rel); err == nil {
		t.Fatal("expected an error for a release with no assets")
	}
}

// === downloadFile ===

func TestDownloadFile_WritesBody(t *testing.T) {
	const payload = "binary-contents"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, payload)
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "out.bin")
	f, err := os.Create(dest)
	if err != nil {
		t.Fatalf("creating dest: %v", err)
	}
	if err := downloadFile(srv.URL, f); err != nil {
		t.Fatalf("downloadFile: %v", err)
	}
	f.Close()

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("reading dest: %v", err)
	}
	if string(got) != payload {
		t.Errorf("downloaded %q, want %q", got, payload)
	}
}

func TestDownloadFile_NonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	f, _ := os.Create(filepath.Join(t.TempDir(), "out.bin"))
	defer f.Close()

	if err := downloadFile(srv.URL, f); err == nil ||
		!strings.Contains(err.Error(), "status 404") {
		t.Fatalf("expected status error, got %v", err)
	}
}

// An empty download must be rejected rather than installed as a zero-byte binary.
func TestDownloadFile_RejectsEmptyBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	f, _ := os.Create(filepath.Join(t.TempDir(), "out.bin"))
	defer f.Close()

	if err := downloadFile(srv.URL, f); err == nil ||
		!strings.Contains(err.Error(), "empty") {
		t.Fatalf("expected empty-file error, got %v", err)
	}
}

func TestDownloadFile_UnreachableHost(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	f, _ := os.Create(filepath.Join(t.TempDir(), "out.bin"))
	defer f.Close()

	if err := downloadFile(url, f); err == nil ||
		!strings.Contains(err.Error(), "download request") {
		t.Fatalf("expected request error, got %v", err)
	}
}

// === verifyChecksum (security critical) ===

// checksumServer serves a checksums.txt body.
func checksumServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func writeTempFile(t *testing.T, contents string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "artifact.tar.gz")
	if err := os.WriteFile(p, []byte(contents), 0o644); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}
	return p
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func TestVerifyChecksum_MatchingHashPasses(t *testing.T) {
	const contents = "artifact-bytes"
	path := writeTempFile(t, contents)
	asset := "moonbase_test.tar.gz"
	srv := checksumServer(t, 200, sha256Hex(contents)+"  "+asset+"\n")

	if err := verifyChecksum(path, asset, srv.URL); err != nil {
		t.Fatalf("expected checksum to verify, got %v", err)
	}
}

// The core security property: a tampered artifact must be rejected.
func TestVerifyChecksum_MismatchedHashIsRejected(t *testing.T) {
	path := writeTempFile(t, "tampered-bytes")
	asset := "moonbase_test.tar.gz"
	// Publish the hash of different content.
	srv := checksumServer(t, 200, sha256Hex("original-bytes")+"  "+asset+"\n")

	err := verifyChecksum(path, asset, srv.URL)
	if err == nil {
		t.Fatal("SECURITY: tampered artifact passed checksum verification")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "checksum") &&
		!strings.Contains(strings.ToLower(err.Error()), "mismatch") {
		t.Errorf("error should describe a checksum mismatch, got %q", err)
	}
}

func TestVerifyChecksum_PicksTheLineForThisAsset(t *testing.T) {
	const contents = "mine"
	path := writeTempFile(t, contents)
	asset := "moonbase_mine.tar.gz"
	body := strings.Join([]string{
		sha256Hex("other") + "  moonbase_other.tar.gz",
		sha256Hex(contents) + "  " + asset,
		sha256Hex("third") + "  moonbase_third.tar.gz",
	}, "\n")
	srv := checksumServer(t, 200, body)

	if err := verifyChecksum(path, asset, srv.URL); err != nil {
		t.Fatalf("expected the matching line to be used, got %v", err)
	}
}

// Unavailable checksums are treated as "skip verification", not as failure.
// This is a deliberate availability trade-off; the test pins it so the behaviour
// cannot change silently.
func TestVerifyChecksum_MissingChecksumsFileSkipsVerification(t *testing.T) {
	path := writeTempFile(t, "whatever")
	srv := checksumServer(t, 404, "not found")

	if err := verifyChecksum(path, "moonbase_test.tar.gz", srv.URL); err != nil {
		t.Fatalf("expected missing checksums to skip verification, got %v", err)
	}
}

func TestVerifyChecksum_UnreachableHost(t *testing.T) {
	path := writeTempFile(t, "whatever")
	srv := checksumServer(t, 200, "")
	url := srv.URL
	srv.Close()

	if err := verifyChecksum(path, "moonbase_test.tar.gz", url); err == nil ||
		!strings.Contains(err.Error(), "downloading checksums") {
		t.Fatalf("expected download error, got %v", err)
	}
}

func TestVerifyChecksum_MissingFileOnDisk(t *testing.T) {
	asset := "moonbase_test.tar.gz"
	srv := checksumServer(t, 200, sha256Hex("x")+"  "+asset+"\n")

	err := verifyChecksum(filepath.Join(t.TempDir(), "absent.tar.gz"), asset, srv.URL)
	if err == nil {
		t.Fatal("expected an error when the artifact is missing")
	}
}

// === expectedAssetName ===

func TestExpectedAssetName_MatchesGoreleaserTemplate(t *testing.T) {
	want := fmt.Sprintf("moonbase_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)
	if got := expectedAssetName(); got != want {
		t.Errorf("expectedAssetName() = %q, want %q", got, want)
	}
}
