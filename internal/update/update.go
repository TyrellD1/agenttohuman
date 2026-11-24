package update

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/TyrellD1/agenttohuman/internal/version"
)

const (
	Repo           = "TyrellD1/agenttohuman"
	DefaultWebBase = "https://github.com"
	ChecksumsName  = "SHA256SUMS"
)

// Targets are the prebuilt GOOS/GOARCH pairs each release publishes.
var Targets = [][2]string{{"darwin", "arm64"}, {"linux", "amd64"}, {"linux", "arm64"}}

var httpClient = &http.Client{Timeout: 60 * time.Second}

type Release struct {
	Tag     string
	Version string
	Assets  map[string]string
}

type Client struct {
	HTTP    *http.Client
	WebBase string
	Repo    string
}

func NewClient() *Client {
	return &Client{
		HTTP:    httpClient,
		WebBase: DefaultWebBase,
		Repo:    Repo,
	}
}

func AssetName(goos, goarch string) string {
	return "a2h-" + goos + "-" + goarch
}

func CurrentAsset() string {
	return AssetName(runtime.GOOS, runtime.GOARCH)
}

func Supported() bool {
	for _, target := range Targets {
		if target[0] == runtime.GOOS && target[1] == runtime.GOARCH {
			return true
		}
	}
	return false
}

// Latest resolves the newest release from the github.com redirect
// (/releases/latest -> /releases/tag/<tag>) instead of the REST API, so it is
// not subject to the API's 60 requests/hour limit for anonymous callers.
// install.sh downloads the same way.
func (c *Client) Latest() (Release, error) {
	base := strings.TrimRight(c.webBase(), "/") + "/" + c.repo()
	req, err := http.NewRequest(http.MethodGet, base+"/releases/latest", nil)
	if err != nil {
		return Release{}, err
	}
	req.Header.Set("User-Agent", "a2h/"+version.String())
	noFollow := *c.http()
	noFollow.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	res, err := noFollow.Do(req)
	if err != nil {
		return Release{}, err
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, res.Body)
	if res.StatusCode < 300 || res.StatusCode >= 400 {
		return Release{}, fmt.Errorf("GitHub latest release lookup failed (%d)", res.StatusCode)
	}
	loc, err := res.Location()
	if err != nil {
		return Release{}, fmt.Errorf("GitHub latest release lookup: %v", err)
	}
	_, tag, found := strings.Cut(loc.Path, "/releases/tag/")
	tag = strings.Trim(tag, "/")
	if !found || tag == "" || strings.Contains(tag, "/") {
		return Release{}, fmt.Errorf("no GitHub release found for %s", c.repo())
	}
	download := base + "/releases/download/" + url.PathEscape(tag) + "/"
	assets := map[string]string{ChecksumsName: download + ChecksumsName}
	for _, target := range Targets {
		name := AssetName(target[0], target[1])
		assets[name] = download + name
	}
	return Release{
		Tag:     tag,
		Version: strings.TrimPrefix(tag, "v"),
		Assets:  assets,
	}, nil
}

func (c *Client) Download(assetURL string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, assetURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "a2h/"+version.String())
	res, err := c.http().Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed (%d)", res.StatusCode)
	}
	return body, nil
}

func ChecksumFor(sums []byte, asset string) (string, error) {
	for _, line := range strings.Split(string(sums), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := strings.TrimPrefix(fields[1], "*")
		if filepath.Base(name) == asset {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("no checksum for %s", asset)
}

func VerifySHA256(data []byte, expected string) error {
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	if !strings.EqualFold(got, expected) {
		return fmt.Errorf("checksum mismatch for download")
	}
	return nil
}

func NeedsUpdate(current, latest string) bool {
	if current == "" || current == "dev" {
		return true
	}
	return current != latest
}

func ReplaceExecutable(dest string, data []byte) error {
	resolved, err := filepath.EvalSymlinks(dest)
	if err != nil {
		resolved = dest
	}
	if filepath.Base(resolved) != "a2h" {
		return fmt.Errorf("refusing to replace %s (expected an a2h binary)", resolved)
	}
	dir := filepath.Dir(resolved)
	tmp, err := os.CreateTemp(dir, "a2h-update-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	ok := false
	defer func() {
		if !ok {
			_ = os.Remove(tmpName)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o755); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, resolved); err != nil {
		return err
	}
	ok = true
	return nil
}

func (c *Client) http() *http.Client {
	if c != nil && c.HTTP != nil {
		return c.HTTP
	}
	return httpClient
}

func (c *Client) webBase() string {
	if c != nil && c.WebBase != "" {
		return c.WebBase
	}
	return DefaultWebBase
}

func (c *Client) repo() string {
	if c != nil && c.Repo != "" {
		return c.Repo
	}
	return Repo
}
