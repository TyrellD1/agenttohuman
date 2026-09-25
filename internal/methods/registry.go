package methods

import (
	"fmt"
	"strings"

	"github.com/TyrellD1/agenttohuman/internal/config"
)

type Metadata struct {
	ContentCharacterLimit  *int              `json:"contentCharacterLimit"`
	UsernameCharacterLimit *int              `json:"usernameCharacterLimit,omitempty"`
	MaxFiles               *int              `json:"maxFiles,omitempty"`
	MaxFileBytes           *int64            `json:"maxFileBytes,omitempty"`
	RateLimit              string            `json:"rateLimit,omitempty"`
	DocsURL                string            `json:"docsUrl"`
	Notes                  []string          `json:"notes"`
	PayloadShape           map[string]string `json:"payloadShape"`
}

type SendRequest struct {
	Text     string
	Username string
	Files    []string
}

type SendResult struct {
	Method string `json:"method"`
	ID     string `json:"id,omitempty"`
}

type InitRequest struct {
	WebhookURL  string
	Username    string
	Test        bool
	Interactive bool
}

type Method interface {
	ID() string
	Name() string
	Description() string
	Metadata() Metadata
	Configured(settings config.MethodConfig) bool
	Init(settings config.MethodConfig, req InitRequest) (config.MethodConfig, error)
	Send(settings config.MethodConfig, req SendRequest) (SendResult, error)
}

type Info struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Configured  bool     `json:"configured"`
	IsDefault   bool     `json:"isDefault"`
	Metadata    Metadata `json:"metadata"`
}

var registry = []Method{DiscordWebhook{}}

func List() []Method {
	out := make([]Method, len(registry))
	copy(out, registry)
	return out
}

func Get(id string) (Method, bool) {
	for _, method := range registry {
		if method.ID() == id {
			return method, true
		}
	}
	return nil, false
}

func Require(id string) (Method, error) {
	method, ok := Get(id)
	if !ok {
		ids := make([]string, 0, len(registry))
		for _, item := range registry {
			ids = append(ids, item.ID())
		}
		known := strings.Join(ids, ", ")
		if known == "" {
			known = "(none)"
		}
		return nil, fmt.Errorf("unknown method %q. Known methods: %s", id, known)
	}
	return method, nil
}

func Resolve(cfg config.AppConfig, methodID string) (Method, error) {
	id := strings.TrimSpace(methodID)
	if id == "" {
		id = config.DefaultID(cfg)
	}
	if id != "" {
		return Require(id)
	}
	if len(registry) == 1 {
		return registry[0], nil
	}
	return nil, fmt.Errorf("no default method is set. Run a2h init <method>, or pass --method <id>")
}

func Describe(method Method, cfg config.AppConfig) Info {
	return Info{
		ID:          method.ID(),
		Name:        method.Name(),
		Description: method.Description(),
		Configured:  method.Configured(config.Settings(cfg, method.ID())),
		IsDefault:   config.DefaultID(cfg) == method.ID(),
		Metadata:    method.Metadata(),
	}
}

func IntPtr(n int) *int {
	return &n
}

func Int64Ptr(n int64) *int64 {
	return &n
}
