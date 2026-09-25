package methods

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/TyrellD1/agenttohuman/internal/config"
)

const DiscordWebhookID = "discord-webhook"

const (
	DiscordMaxFiles     = 10
	DiscordMaxFileBytes = int64(20 << 20) // 20 MiB per file (Discord default)
)

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
		MaxFiles:               IntPtr(DiscordMaxFiles),
		MaxFileBytes:           Int64Ptr(DiscordMaxFileBytes),
		RateLimit:              "5 requests per 2 seconds per webhook",
		DocsURL:                "https://discord.com/developers/docs/resources/webhook#execute-webhook",
		Notes: []string{
			"content is required unless --file is given (plain text / Discord markdown).",
			"Discord may silently truncate content over 2000 characters; a2h rejects it first.",
			"username overrides the webhook's default display name for that message.",
			"files: any type uploads as an attachment (images render inline, .md/.html download). Max 10 files, 20 MiB each.",
		},
		PayloadShape: map[string]string{
			"content":    "string, required unless files given, max 2000 characters",
			"username":   "string, optional, max 80 characters",
			"files":      "repeatable --file <path>, max 10 files, 20 MiB each",
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
	if err := AssertFiles(req.Files); err != nil {
		return SendResult{}, err
	}
	text := strings.TrimSpace(req.Text)
	if len(req.Files) == 0 {
		if err := AssertContent(text); err != nil {
			return SendResult{}, err
		}
	} else if text != "" {
		if err := AssertContent(text); err != nil {
			return SendResult{}, err
		}
	} else {
		text = ""
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

	parsed, err := url.Parse(webhookURL)
	if err != nil {
		return SendResult{}, err
	}
	query := parsed.Query()
	query.Set("wait", "true")
	parsed.RawQuery = query.Encode()

	var body io.Reader
	var contentType string
	if len(req.Files) == 0 {
		payload := map[string]string{"content": text}
		if username != "" {
			payload["username"] = username
		}
		raw, err := json.Marshal(payload)
		if err != nil {
			return SendResult{}, err
		}
		body = bytes.NewReader(raw)
		contentType = "application/json"
	} else {
		buf, ctype, err := buildMultipartBody(text, username, req.Files)
		if err != nil {
			return SendResult{}, err
		}
		body = buf
		contentType = ctype
	}

	httpReq, err := http.NewRequest(http.MethodPost, parsed.String(), body)
	if err != nil {
		return SendResult{}, err
	}
	httpReq.Header.Set("Content-Type", contentType)

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

func AssertFiles(paths []string) error {
	if len(paths) > DiscordMaxFiles {
		return fmt.Errorf("got %d files; Discord webhooks accept max %d per message", len(paths), DiscordMaxFiles)
	}
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return fmt.Errorf("file %q: %v", p, err)
		}
		if info.IsDir() {
			return fmt.Errorf("file %q is a directory; pass a file path", p)
		}
		if info.Size() > DiscordMaxFileBytes {
			return fmt.Errorf("file %q is %.1f MiB; max is 20 MiB per file", p, float64(info.Size())/(1<<20))
		}
	}
	return nil
}

func buildMultipartBody(text, username string, paths []string) (*bytes.Buffer, string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	payload := map[string]any{}
	if strings.TrimSpace(text) != "" {
		payload["content"] = text
	}
	if username != "" {
		payload["username"] = username
	}
	attachments := make([]map[string]any, 0, len(paths))
	for i, p := range paths {
		attachments = append(attachments, map[string]any{"id": i, "filename": filepath.Base(p)})
	}
	if len(attachments) > 0 {
		payload["attachments"] = attachments
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, "", err
	}
	part, err := w.CreatePart(textproto.MIMEHeader{
		"Content-Disposition": []string{`form-data; name="payload_json"`},
		"Content-Type":        []string{"application/json"},
	})
	if err != nil {
		return nil, "", err
	}
	if _, err := part.Write(raw); err != nil {
		return nil, "", err
	}

	for i, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, "", fmt.Errorf("file %q: %v", p, err)
		}
		ctype := mime.TypeByExtension(strings.ToLower(filepath.Ext(p)))
		if ctype == "" {
			ctype = http.DetectContentType(data)
			if ctype == "" {
				ctype = "application/octet-stream"
			}
		}
		fw, err := w.CreatePart(textproto.MIMEHeader{
			"Content-Disposition": []string{fmt.Sprintf(`form-data; name="files[%d]"; filename="%s"`, i, filepath.Base(p))},
			"Content-Type":        []string{ctype},
		})
		if err != nil {
			return nil, "", err
		}
		if _, err := fw.Write(data); err != nil {
			return nil, "", err
		}
	}
	if err := w.Close(); err != nil {
		return nil, "", err
	}
	return &buf, w.FormDataContentType(), nil
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
