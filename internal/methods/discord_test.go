package methods

import (
	"io"
	"net/http"
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
