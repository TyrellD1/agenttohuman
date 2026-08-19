package methods

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/TyrellD1/agenttohuman/internal/config"
)

const DiscordWebhookID = "discord-webhook"

var webhookURLRe = regexp.MustCompile(`(?i)^https://(?:(?:ptb|canary)\.)?discord(?:app)?\.com/api(?:/v\d+)?/webhooks/\d+/[\w-]+$`)

var httpClient = &http.Client{Timeout: 30 * time.Second}

var discordDo = func(req *http.Request) (*http.Response, error) {
	return httpClient.Do(req)
}

type DiscordWebhook struct{}

func (DiscordWebhook) ID() string          { return DiscordWebhookID }
func (DiscordWebhook) Name() string        { return "Discord webhook" }
func (DiscordWebhook) Description() string { return "POST a message to a Discord channel webhook." }

func (DiscordWebhook) Metadata() Metadata {
	return Metadata{
		ContentCharacterLimit:  IntPtr(2000),
		UsernameCharacterLimit: IntPtr(80),
		RateLimit:              "5 requests per 2 seconds per webhook",
		DocsURL:                "https://discord.com/developers/docs/resources/webhook#execute-webhook",
		Notes: []string{
			"content is required for a2h send (plain text / Discord markdown).",
			"Discord may silently truncate content over 2000 characters; a2h rejects it first.",
			"username overrides the webhook's default display name for that message.",
		},
		PayloadShape: map[string]string{
			"content":    "string, required, max 2000 characters",
			"username":   "string, optional, max 80 characters",
			"avatar_url": "string URL, optional (not exposed by a2h send yet)",
			"embeds":     "array, optional, max 10 (not exposed by a2h send yet)",
		},
	}
}

func (DiscordWebhook) Configured(settings config.MethodConfig) bool {
	return settings.String("webhookUrl") != ""
}

func (d DiscordWebhook) Init(settings config.MethodConfig, req InitRequest) (config.MethodConfig, error) {
	webhookURL, err := resolveWebhookURL(req)
	if err != nil {
		return nil, err
	}
	if err := AssertWebhookURL(webhookURL); err != nil {
		return nil, err
	}
	username := strings.TrimSpace(req.Username)
	if username == "" {
		username = settings.String("username")
	}
	if username != "" {
		if err := AssertUsername(username); err != nil {
			return nil, err
		}
	}
	next := settings.SetString("webhookUrl", webhookURL)
	if username != "" {
		next = next.SetString("username", username)
	}
	if req.Test {
		if _, err := d.Send(next, SendRequest{Text: "a2h connected. Discord webhook is ready.", Username: username}); err != nil {
			return nil, err
		}
	}
	return next, nil
}

func (DiscordWebhook) Send(settings config.MethodConfig, req SendRequest) (SendResult, error) {
	webhookURL := settings.String("webhookUrl")
	if webhookURL == "" {
		return SendResult{}, fmt.Errorf("Discord webhook is not configured. Run: a2h init discord-webhook")
	}
	if err := AssertWebhookURL(webhookURL); err != nil {
		return SendResult{}, err
	}
	if err := AssertContent(req.Text); err != nil {
		return SendResult{}, err
	}
	username := strings.TrimSpace(req.Username)
	if username == "" {
		username = settings.String("username")
	}
	if username != "" {
		if err := AssertUsername(username); err != nil {
			return SendResult{}, err
		}
	}

	payload := map[string]string{"content": req.Text}
	if username != "" {
		payload["username"] = username
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return SendResult{}, err
	}

	parsed, err := url.Parse(webhookURL)
	if err != nil {
		return SendResult{}, err
	}
	query := parsed.Query()
	query.Set("wait", "true")
	parsed.RawQuery = query.Encode()

	httpReq, err := http.NewRequest(http.MethodPost, parsed.String(), bytes.NewReader(body))
	if err != nil {
		return SendResult{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	res, err := discordDo(httpReq)
	if err != nil {
		return SendResult{}, err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)

	if res.StatusCode == http.StatusTooManyRequests {
		retry := res.Header.Get("Retry-After")
		if retry == "" {
			retry = "unknown"
		}
		return SendResult{}, fmt.Errorf("Discord rate-limited this webhook. Retry after %ss", retry)
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		detail := discordError(raw)
		if detail != "" {
			return SendResult{}, fmt.Errorf("Discord webhook failed (%d): %s", res.StatusCode, detail)
		}
		return SendResult{}, fmt.Errorf("Discord webhook failed (%d).", res.StatusCode)
	}

	var msg struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(raw, &msg)
	return SendResult{Method: DiscordWebhookID, ID: msg.ID}, nil
}

func AssertWebhookURL(raw string) error {
	if !webhookURLRe.MatchString(raw) {
		return fmt.Errorf("that does not look like a Discord webhook URL. Expected https://discord.com/api/webhooks/<id>/<token>")
	}
	return nil
}

func AssertUsername(username string) error {
	if utf8.RuneCountInString(username) > 80 {
		return fmt.Errorf("Discord webhook username must be 80 characters or fewer")
	}
	banned := strings.ToLower(strings.TrimSpace(username))
	if banned == "clyde" || banned == "discord" {
		return fmt.Errorf("Discord rejects webhook usernames \"Clyde\" and \"Discord\"")
	}
	return nil
}

func AssertContent(text string) error {
	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("message is empty")
	}
	n := utf8.RuneCountInString(text)
	if n > 2000 {
		return fmt.Errorf("message is %d characters; Discord webhook content max is 2000. Split it and send again", n)
	}
	return nil
}

func resolveWebhookURL(req InitRequest) (string, error) {
	if url := strings.TrimSpace(req.WebhookURL); url != "" {
		return url, nil
	}
	if !req.Interactive {
		return "", fmt.Errorf("no webhook URL provided. Re-run with --webhook-url <url>, or run a2h init discord-webhook in a terminal")
	}
	fmt.Fprint(os.Stderr, "Discord webhook URL\n  Channel Settings → Integrations → Webhooks → Copy Webhook URL\n> ")
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	url := strings.TrimSpace(line)
	if url == "" {
		return "", fmt.Errorf("no webhook URL entered")
	}
	return url, nil
}

func discordError(raw []byte) string {
	var body struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return ""
	}
	parts := make([]string, 0, 2)
	if body.Message != "" {
		parts = append(parts, body.Message)
	}
	if body.Code != 0 {
		parts = append(parts, fmt.Sprintf("code %d", body.Code))
	}
	return strings.Join(parts, " ")
}
