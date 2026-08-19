package update

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/TyrellD1/agenttohuman/internal/version"
)

const (
	Repo           = "TyrellD1/agenttohuman"
	DefaultAPIBase = "https://api.github.com"
	ChecksumsName  = "SHA256SUMS"
)

var httpClient = &http.Client{Timeout: 60 * time.Second}

type Release struct {
	Tag     string
	Version string
	Assets  map[string]string
}

type Client struct {
	HTTP    *http.Client
	APIBase string
	Repo    string
}

func NewClient() *Client {
	return &Client{
		HTTP:    httpClient,
		APIBase: DefaultAPIBase,
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
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "darwin/arm64", "linux/amd64", "linux/arm64":
		return true
	default:
		return false
	}
}

func (c *Client) Latest() (Release, error) {
	httpClient := c.http()
	url := strings.TrimRight(c.apiBase(), "/") + "/repos/" + c.repo() + "/releases/latest"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return Release{}, err
	}
	req.Header.Set("User-Agent", "a2h/"+version.String())
	req.Header.Set("Accept", "application/vnd.github+json")
	res, err := httpClient.Do(req)
	if err != nil {
		return Release{}, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return Release{}, err
	}
	if res.StatusCode != http.StatusOK {
		return Release{}, fmt.Errorf("GitHub releases failed (%d)", res.StatusCode)
	}
	var parsed struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return Release{}, err
	}
	if parsed.TagName == "" {
		return Release{}, fmt.Errorf("latest GitHub release has no tag")
	}
	assets := map[string]string{}
	for _, asset := range parsed.Assets {
		assets[asset.Name] = asset.URL
	}
	return Release{
		Tag:     parsed.TagName,
		Version: strings.TrimPrefix(parsed.TagName, "v"),
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

func (c *Client) apiBase() string {
	if c != nil && c.APIBase != "" {
		return c.APIBase
	}
	return DefaultAPIBase
}

func (c *Client) repo() string {
	if c != nil && c.Repo != "" {
		return c.Repo
	}
	return Repo
}
