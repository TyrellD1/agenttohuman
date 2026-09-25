package methods

import (
	"io"
	"net/http"
	"os"
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
