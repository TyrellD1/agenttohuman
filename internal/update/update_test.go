package update

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestNeedsUpdate(t *testing.T) {
	if !NeedsUpdate("dev", "0.2.0") {
		t.Fatal("dev should update")
	}
	if NeedsUpdate("0.2.0", "0.2.0") {
		t.Fatal("same version should not update")
	}
	if !NeedsUpdate("0.1.0", "0.2.0") {
		t.Fatal("older should update")
	}
}

func TestChecksumForAndVerify(t *testing.T) {
	data := []byte("hello")
	sum := sha256.Sum256(data)
	hexSum := hex.EncodeToString(sum[:])
	sums := hexSum + "  a2h-darwin-arm64\n"
	got, err := ChecksumFor([]byte(sums), "a2h-darwin-arm64")
	if err != nil {
		t.Fatal(err)
	}
	if got != hexSum {
		t.Fatalf("got %s", got)
	}
	if err := VerifySHA256(data, hexSum); err != nil {
		t.Fatal(err)
	}
	if err := VerifySHA256(data, "deadbeef"); err == nil {
		t.Fatal("expected mismatch")
	}
}

func TestLatestParsesGitHubJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/TyrellD1/agenttohuman/releases/latest" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"tag_name": "v0.2.0",
			"assets": [
				{"name": "a2h-darwin-arm64", "browser_download_url": "https://example.com/a2h"},
				{"name": "SHA256SUMS", "browser_download_url": "https://example.com/sums"}
			]
		}`))
	}))
	t.Cleanup(server.Close)

	client := &Client{HTTP: server.Client(), APIBase: server.URL, Repo: Repo}
	rel, err := client.Latest()
	if err != nil {
		t.Fatal(err)
	}
	if rel.Version != "0.2.0" {
		t.Fatalf("version = %s", rel.Version)
	}
	if rel.Assets["a2h-darwin-arm64"] != "https://example.com/a2h" {
		t.Fatalf("assets = %#v", rel.Assets)
	}
}

func TestReplaceExecutable(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "a2h")
	if err := os.WriteFile(dest, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := ReplaceExecutable(dest, []byte("new-binary")); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new-binary" {
		t.Fatalf("got %q", got)
	}
}

func TestReplaceExecutableRejectsOtherNames(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "not-a2h")
	if err := os.WriteFile(dest, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := ReplaceExecutable(dest, []byte("new")); err == nil {
		t.Fatal("expected refusal")
	}
}
