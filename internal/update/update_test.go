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

func TestLatestFollowsReleaseRedirectWithoutAPI(t *testing.T) {
	binary := []byte("new-a2h-binary")
	sum := sha256.Sum256(binary)
	sums := hex.EncodeToString(sum[:]) + "  a2h-linux-amd64\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/TyrellD1/agenttohuman/releases/latest":
			http.Redirect(w, r, "/TyrellD1/agenttohuman/releases/tag/v0.3.1", http.StatusFound)
		case "/TyrellD1/agenttohuman/releases/tag/v0.3.1":
			t.Error("Latest should not follow the redirect")
		case "/TyrellD1/agenttohuman/releases/download/v0.3.1/a2h-linux-amd64":
			// Real GitHub bounces downloads to a CDN host; Download must follow.
			http.Redirect(w, r, "/cdn/a2h-linux-amd64", http.StatusFound)
		case "/cdn/a2h-linux-amd64":
			_, _ = w.Write(binary)
		case "/TyrellD1/agenttohuman/releases/download/v0.3.1/SHA256SUMS":
			_, _ = w.Write([]byte(sums))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	client := &Client{HTTP: server.Client(), WebBase: server.URL, Repo: Repo}
	rel, err := client.Latest()
	if err != nil {
		t.Fatal(err)
	}
	if rel.Tag != "v0.3.1" || rel.Version != "0.3.1" {
		t.Fatalf("release = %#v", rel)
	}
	for _, name := range []string{"a2h-darwin-arm64", "a2h-linux-amd64", "a2h-linux-arm64", ChecksumsName} {
		want := server.URL + "/TyrellD1/agenttohuman/releases/download/v0.3.1/" + name
		if rel.Assets[name] != want {
			t.Fatalf("asset %s = %q, want %q", name, rel.Assets[name], want)
		}
	}
	gotSums, err := client.Download(rel.Assets[ChecksumsName])
	if err != nil {
		t.Fatal(err)
	}
	want, err := ChecksumFor(gotSums, "a2h-linux-amd64")
	if err != nil {
		t.Fatal(err)
	}
	gotBin, err := client.Download(rel.Assets["a2h-linux-amd64"])
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifySHA256(gotBin, want); err != nil {
		t.Fatal(err)
	}
}

func TestLatestWithoutReleaseFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/TyrellD1/agenttohuman/releases/latest":
			// GitHub sends repos with no releases to the releases index.
			http.Redirect(w, r, "/TyrellD1/agenttohuman/releases", http.StatusFound)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	client := &Client{HTTP: server.Client(), WebBase: server.URL, Repo: Repo}
	if _, err := client.Latest(); err == nil {
		t.Fatal("expected an error when there is no release")
	}
	missing := &Client{HTTP: server.Client(), WebBase: server.URL, Repo: "someone/else"}
	if _, err := missing.Latest(); err == nil {
		t.Fatal("expected an error for a 404")
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
