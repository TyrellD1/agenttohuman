package methods

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAssertWebhookURL(t *testing.T) {
	ok := []string{
		"https://discord.com/api/webhooks/123/abc-DEF_456",
		"https://discordapp.com/api/webhooks/123/abc",
		"https://ptb.discord.com/api/v10/webhooks/123/abc",
	}
	for _, url := range ok {
		if err := AssertWebhookURL(url); err != nil {
			t.Fatalf("%s: %v", url, err)
		}
	}
	bad := []string{
		"https://example.com/hooks/1",
		"https://discord.com/channels/1/2",
	}
	for _, url := range bad {
		if err := AssertWebhookURL(url); err == nil {
			t.Fatalf("expected error for %s", url)
		}
	}
}

func TestAssertContent(t *testing.T) {
	if err := AssertContent(""); err == nil {
		t.Fatal("expected empty error")
	}
	if err := AssertContent(strings.Repeat("a", 2001)); err == nil {
		t.Fatal("expected oversize error")
	}
	if err := AssertContent(strings.Repeat("a", 2000)); err != nil {
		t.Fatal(err)
	}
}

func TestAssertUsername(t *testing.T) {
	if err := AssertUsername("Clyde"); err == nil {
		t.Fatal("expected banned username error")
	}
	if err := AssertUsername(strings.Repeat("x", 81)); err == nil {
		t.Fatal("expected length error")
	}
	if err := AssertUsername("a2h"); err != nil {
		t.Fatal(err)
	}
}

func TestConfigured(t *testing.T) {
	var method DiscordWebhook
	if method.Configured(nil) {
		t.Fatal("empty should be unconfigured")
	}
	if !method.Configured(map[string]any{
		"webhookUrl": "https://discord.com/api/webhooks/1/token",
	}) {
		t.Fatal("expected configured")
	}
}

func TestSendPostsContent(t *testing.T) {
	var gotURL, gotBody string
	original := discordDo
	t.Cleanup(func() { discordDo = original })
	discordDo = func(req *http.Request) (*http.Response, error) {
		gotURL = req.URL.String()
		raw, _ := io.ReadAll(req.Body)
		gotBody = string(raw)
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"id":"msg_1"}`)),
			Header:     make(http.Header),
		}, nil
	}

	result, err := DiscordWebhook{}.Send(
		map[string]any{"webhookUrl": "https://discord.com/api/webhooks/1/token"},
		SendRequest{Text: "hello from a2h", Username: "a2h"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.Method != DiscordWebhookID || result.ID != "msg_1" {
		t.Fatalf("result = %#v", result)
	}
	if !strings.Contains(gotURL, "wait=true") {
		t.Fatalf("url missing wait: %s", gotURL)
	}
	if !strings.Contains(gotBody, `"content":"hello from a2h"`) || !strings.Contains(gotBody, `"username":"a2h"`) {
		t.Fatalf("body = %s", gotBody)
	}
}

func TestAssertFiles(t *testing.T) {
	if err := AssertFiles(nil); err != nil {
		t.Fatal(err)
	}
	if err := AssertFiles([]string{"nope-missing-file.txt"}); err == nil {
		t.Fatal("expected missing file error")
	}
	many := make([]string, DiscordMaxFiles+1)
	for i := range many {
		many[i] = "x"
	}
	if err := AssertFiles(many); err == nil {
		t.Fatal("expected too-many error")
	}
	dir := t.TempDir()
	if err := AssertFiles([]string{dir}); err == nil {
		t.Fatal("expected directory error")
	}
}

func TestSendMultipartWithFiles(t *testing.T) {
	dir := t.TempDir()
	f1 := dir + "/notes.md"
	f2 := dir + "/report.html"
	if err := os.WriteFile(f1, []byte("# hi"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f2, []byte("<h1>hi</h1>"), 0o600); err != nil {
		t.Fatal(err)
	}
	var gotType string
	var gotRaw []byte
	original := discordDo
	t.Cleanup(func() { discordDo = original })
	discordDo = func(req *http.Request) (*http.Response, error) {
		gotType = req.Header.Get("Content-Type")
		raw, _ := io.ReadAll(req.Body)
		gotRaw = raw
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"id":"msg_files"}`)),
			Header:     make(http.Header),
		}, nil
	}
	result, err := DiscordWebhook{}.Send(
		map[string]any{"webhookUrl": "https://discord.com/api/webhooks/1/token"},
		SendRequest{Text: "build files", Files: []string{f1, f2}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.ID != "msg_files" {
		t.Fatalf("result = %#v", result)
	}
	if !strings.HasPrefix(gotType, "multipart/form-data;") {
		t.Fatalf("content-type = %s", gotType)
	}
	body := string(gotRaw)
	for _, want := range []string{`name="payload_json"`, `"content":"build files"`, `name="files[0]"`, `filename="notes.md"`, `name="files[1]"`, `filename="report.html"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("multipart body missing %s\n%.500s", want, body)
		}
	}
}

func TestSendFileOnlyAllowsEmptyText(t *testing.T) {
	dir := t.TempDir()
	f := dir + "/a.txt"
	if err := os.WriteFile(f, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	original := discordDo
	t.Cleanup(func() { discordDo = original })
	discordDo = func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"id":"m"}`)), Header: make(http.Header)}, nil
	}
	if _, err := (DiscordWebhook{}).Send(
		map[string]any{"webhookUrl": "https://discord.com/api/webhooks/1/token"},
		SendRequest{Files: []string{f}},
	); err != nil {
		t.Fatal(err)
	}
	if _, err := (DiscordWebhook{}).Send(
		map[string]any{"webhookUrl": "https://discord.com/api/webhooks/1/token"},
		SendRequest{},
	); err == nil {
		t.Fatal("expected empty message+files error")
	}
}

func writePNG(t *testing.T, path string) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 4, 3))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeJPEG(t *testing.T, path string) {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 4, 3)), nil); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
}

// parts reads a multipart request body into name -> (filename, content type).
func parts(t *testing.T, req *http.Request) (map[string][2]string, string) {
	t.Helper()
	_, params, err := mime.ParseMediaType(req.Header.Get("Content-Type"))
	if err != nil {
		t.Fatal(err)
	}
	r := multipart.NewReader(req.Body, params["boundary"])
	out := map[string][2]string{}
	var payload string
	for {
		p, err := r.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		raw, _ := io.ReadAll(p)
		if p.FormName() == "payload_json" {
			payload = string(raw)
			continue
		}
		out[p.FormName()] = [2]string{p.FileName(), p.Header.Get("Content-Type")}
	}
	return out, payload
}

func TestSendImagesInlineBeforeFiles(t *testing.T) {
	dir := t.TempDir()
	shot := filepath.Join(dir, "shot") // no extension: a2h adds .png
	photo := filepath.Join(dir, "photo.jpeg")
	notes := filepath.Join(dir, "notes.md")
	writePNG(t, shot)
	writeJPEG(t, photo)
	if err := os.WriteFile(notes, []byte("# hi"), 0o600); err != nil {
		t.Fatal(err)
	}
	var got map[string][2]string
	var payload string
	original := discordDo
	t.Cleanup(func() { discordDo = original })
	discordDo = func(req *http.Request) (*http.Response, error) {
		got, payload = parts(t, req)
		return &http.Response{
			StatusCode: 200,
			Body: io.NopCloser(strings.NewReader(`{"id":"msg_img","attachments":[
				{"filename":"shot.png","content_type":"image/png","size":70,"width":4,"height":3},
				{"filename":"photo.jpeg","content_type":"image/jpeg","size":600,"width":4,"height":3},
				{"filename":"notes.md","content_type":"text/markdown","size":4}]}`)),
			Header: make(http.Header),
		}, nil
	}
	result, err := DiscordWebhook{}.Send(
		map[string]any{"webhookUrl": "https://discord.com/api/webhooks/1/token"},
		SendRequest{Text: "screens", Files: []string{notes}, Images: []string{shot, photo}},
	)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][2]string{
		"files[0]": {"shot.png", "image/png"},
		"files[1]": {"photo.jpeg", "image/jpeg"},
	}
	// .md's type comes from the OS mime table (text/markdown or text/plain).
	if f := got["files[2]"]; f[0] != "notes.md" || !strings.HasPrefix(f[1], "text/") {
		t.Fatalf("files[2] = %v", f)
	}
	for name, w := range want {
		if got[name] != w {
			t.Fatalf("%s = %v, want %v (all %v)", name, got[name], w, got)
		}
	}
	if !strings.Contains(payload, `"filename":"shot.png"`) || !strings.Contains(payload, `"content":"screens"`) {
		t.Fatalf("payload_json = %s", payload)
	}
	if len(result.Attachments) != 3 || result.Attachments[0].Width != 4 || result.Attachments[2].Width != 0 {
		t.Fatalf("attachments = %#v", result.Attachments)
	}
}

func TestImageOnlyNeedsNoText(t *testing.T) {
	dir := t.TempDir()
	shot := filepath.Join(dir, "a.png")
	writePNG(t, shot)
	original := discordDo
	t.Cleanup(func() { discordDo = original })
	discordDo = func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"id":"m"}`)), Header: make(http.Header)}, nil
	}
	if _, err := (DiscordWebhook{}).Send(
		map[string]any{"webhookUrl": "https://discord.com/api/webhooks/1/token"},
		SendRequest{Images: []string{shot}},
	); err != nil {
		t.Fatal(err)
	}
}

func TestAssertImagesRejectsNonInlineFormats(t *testing.T) {
	dir := t.TempDir()
	cases := map[string][]byte{
		"pic.bmp":  append([]byte("BM"), make([]byte, 64)...),
		"pic.heic": append([]byte("\x00\x00\x00\x1cftypheic"), make([]byte, 32)...),
		"pic.png":  []byte("not really a png"),
	}
	for name, data := range cases {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, data, 0o600); err != nil {
			t.Fatal(err)
		}
		err := AssertImages([]string{p})
		if err == nil || !strings.Contains(err.Error(), "not a format Discord shows inline") {
			t.Fatalf("%s: err = %v", name, err)
		}
		// The same file is fine as a plain attachment.
		if err := AssertFiles([]string{p}); err != nil {
			t.Fatalf("%s as file: %v", name, err)
		}
	}
}

func TestSniffImageTypeAVIF(t *testing.T) {
	head := append([]byte("\x00\x00\x00\x1cftypavif"), make([]byte, 32)...)
	if got := sniffImageType(head); got != "image/avif" {
		t.Fatalf("got %s", got)
	}
}

func TestImageFilename(t *testing.T) {
	cases := []struct {
		path string
		exts []string
		want string
	}{
		{"/x/shot.png", []string{".png"}, "shot.png"},
		{"/x/SHOT.PNG", []string{".png"}, "SHOT.PNG"},
		{"/x/shot", []string{".png"}, "shot.png"},
		{"/x/photo.jpg", []string{".jpg", ".jpeg"}, "photo.jpg"},
		{"/x/photo.png", []string{".jpg", ".jpeg"}, "photo.png.jpg"},
	}
	for _, c := range cases {
		if got := imageFilename(c.path, c.exts); got != c.want {
			t.Fatalf("%s: got %s, want %s", c.path, got, c.want)
		}
	}
}

func TestImagesAndFilesShareLimit(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.png")
	writePNG(t, p)
	images := make([]string, 6)
	files := make([]string, 5)
	for i := range images {
		images[i] = p
	}
	for i := range files {
		files[i] = p
	}
	if _, err := prepareUploads(images, files); err == nil || !strings.Contains(err.Error(), "max 10") {
		t.Fatalf("err = %v", err)
	}
	if _, err := prepareUploads(images, files[:4]); err != nil {
		t.Fatal(err)
	}
}

func TestMultipartEscapesFilename(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, `odd "name".png`)
	writePNG(t, p)
	uploads, err := prepareUploads([]string{p}, nil)
	if err != nil {
		t.Fatal(err)
	}
	buf, ctype, err := buildMultipartBody("", "", uploads)
	if err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest(http.MethodPost, "http://x", buf)
	req.Header.Set("Content-Type", ctype)
	got, _ := parts(t, req)
	if got["files[0]"][0] != `odd "name".png` {
		t.Fatalf("filename = %q", got["files[0]"][0])
	}
}
