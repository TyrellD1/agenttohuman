package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/TyrellD1/agenttohuman/internal/config"
	"github.com/TyrellD1/agenttohuman/internal/methods"
	"github.com/TyrellD1/agenttohuman/internal/update"
	"github.com/TyrellD1/agenttohuman/internal/version"
)

const help = `a2h — agent-to-human notifications

Usage:
  a2h send <message>          Send via the default method
  a2h init [method]           Save credentials for a method
  a2h methods                 List available methods
  a2h info [method]           Method metadata and payload shape
  a2h default [method]        Get or set the default method
  a2h status                  Local config summary
  a2h update                  Replace this binary with the latest release
  a2h version                 Print the version

Options:
  -m, --method <id>           Method for send / info / init
  --webhook-url <url>         Discord webhook URL (init)
  --username <name>           Webhook username override
  --stdin                     Read send message from stdin
  --test                      Send a test message during init
  --check                     Report whether an update is available
  --json                      Machine-readable output
  -h, --help
  -V, --version

Data is stored in ~/.agenttohuman (override with AGENTTOHUMAN_HOME).
`

type flags struct {
	Method     string
	JSON       bool
	Help       bool
	Version    bool
	Stdin      bool
	Test       bool
	Check      bool
	Username   string
	WebhookURL string
}

type context struct {
	command string
	rest    []string
	flags   flags
	stdin   io.Reader
	stdout  io.Writer
	stderr  io.Writer
}

func HasJSON(args []string) bool {
	for _, arg := range args {
		if arg == "--json" {
			return true
		}
	}
	return false
}

func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	f, rest, err := parseArgs(args)
	if err != nil {
		return err
	}
	if f.Help {
		fmt.Fprint(stdout, help)
		return nil
	}
	if f.Version {
		fmt.Fprintln(stdout, version.String())
		return nil
	}
	if len(rest) == 0 {
		fmt.Fprint(stdout, help)
		return nil
	}
	ctx := context{
		command: rest[0],
		rest:    rest[1:],
		flags:   f,
		stdin:   stdin,
		stdout:  stdout,
		stderr:  stderr,
	}
	switch ctx.command {
	case "send":
		return handleSend(ctx)
	case "init":
		return handleInit(ctx)
	case "methods":
		return handleMethods(ctx)
	case "info":
		return handleInfo(ctx)
	case "default":
		return handleDefault(ctx)
	case "status":
		return handleStatus(ctx)
	case "update":
		return handleUpdate(ctx)
	case "version":
		fmt.Fprintln(stdout, version.String())
		return nil
	default:
		return fmt.Errorf("unknown command %q. Run a2h --help", ctx.command)
	}
}

func WriteError(stderr io.Writer, jsonOut bool, err error) {
	if jsonOut {
		enc, _ := json.Marshal(map[string]string{"error": err.Error()})
		fmt.Fprintf(stderr, "%s\n", enc)
		return
	}
	fmt.Fprintln(stderr, err.Error())
}

func parseArgs(args []string) (flags, []string, error) {
	var f flags
	var rest []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help":
			f.Help = true
		case arg == "-V" || arg == "--version":
			f.Version = true
		case arg == "--json":
			f.JSON = true
		case arg == "--stdin":
			f.Stdin = true
		case arg == "--test":
			f.Test = true
		case arg == "--check":
			f.Check = true
		case arg == "-m" || arg == "--method":
			value, next, err := needValue(args, i, arg)
			if err != nil {
				return f, nil, err
			}
			f.Method = value
			i = next
		case strings.HasPrefix(arg, "--method="):
			f.Method = strings.TrimPrefix(arg, "--method=")
		case arg == "--username":
			value, next, err := needValue(args, i, arg)
			if err != nil {
				return f, nil, err
			}
			f.Username = value
			i = next
		case strings.HasPrefix(arg, "--username="):
			f.Username = strings.TrimPrefix(arg, "--username=")
		case arg == "--webhook-url":
			value, next, err := needValue(args, i, arg)
			if err != nil {
				return f, nil, err
			}
			f.WebhookURL = value
			i = next
		case strings.HasPrefix(arg, "--webhook-url="):
			f.WebhookURL = strings.TrimPrefix(arg, "--webhook-url=")
		case strings.HasPrefix(arg, "-"):
			return f, nil, fmt.Errorf("unknown flag %s", arg)
		default:
			rest = append(rest, arg)
		}
	}
	return f, rest, nil
}

func needValue(args []string, i int, flag string) (string, int, error) {
	if i+1 >= len(args) {
		return "", i, fmt.Errorf("%s requires a value", flag)
	}
	return args[i+1], i + 1, nil
}

func emit(ctx context, human string, data any) error {
	if ctx.flags.JSON {
		enc, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintf(ctx.stdout, "%s\n", enc)
		return nil
	}
	fmt.Fprintln(ctx.stdout, human)
	return nil
}

func handleSend(ctx context) error {
	var text string
	if ctx.flags.Stdin {
		raw, err := io.ReadAll(ctx.stdin)
		if err != nil {
			return err
		}
		text = strings.TrimSpace(string(raw))
	} else {
		text = strings.TrimSpace(strings.Join(ctx.rest, " "))
	}
	if text == "" {
		return fmt.Errorf("no message given. Usage: a2h send <message>  (or --stdin)")
	}
	result, err := notify(text, ctx.flags.Method, ctx.flags.Username)
	if err != nil {
		return err
	}
	human := fmt.Sprintf("sent (%s)", result.Method)
	if result.ID != "" {
		human = fmt.Sprintf("sent (%s %s)", result.Method, result.ID)
	}
	return emit(ctx, human, map[string]any{
		"ok":     true,
		"method": result.Method,
		"id":     result.ID,
	})
}

func handleInit(ctx context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	id := ""
	if len(ctx.rest) > 0 {
		id = ctx.rest[0]
	}
	if id == "" {
		id = ctx.flags.Method
	}
	method, err := methods.Resolve(cfg, id)
	if err != nil {
		return err
	}
	interactive := isTTY(ctx.stdin) && ctx.flags.WebhookURL == ""
	settings, err := method.Init(config.Settings(cfg, method.ID()), methods.InitRequest{
		WebhookURL:  ctx.flags.WebhookURL,
		Username:    ctx.flags.Username,
		Test:        ctx.flags.Test,
		Interactive: interactive,
	})
	if err != nil {
		return err
	}
	next := config.WithSettings(cfg, method.ID(), settings)
	if err := config.Save(next); err != nil {
		return err
	}
	return emit(ctx, fmt.Sprintf("saved %s (default: %s)", method.ID(), config.DefaultID(next)), map[string]any{
		"ok":            true,
		"method":        method.ID(),
		"defaultMethod": config.DefaultID(next),
	})
}

func handleMethods(ctx context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	infos := make([]methods.Info, 0, len(methods.List()))
	lines := make([]string, 0, len(methods.List()))
	for _, method := range methods.List() {
		info := methods.Describe(method, cfg)
		infos = append(infos, info)
		flags := "not configured"
		if info.Configured {
			flags = "configured"
		}
		if info.IsDefault {
			flags = "default, " + flags
		}
		lines = append(lines, fmt.Sprintf("%s  %s  (%s)", info.ID, info.Name, flags))
	}
	human := strings.Join(lines, "\n")
	if human == "" {
		human = "No methods registered."
	}
	return emit(ctx, human, map[string]any{
		"defaultMethod": cfg.DefaultMethod,
		"methods":       infos,
	})
}

func handleInfo(ctx context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	id := ""
	if len(ctx.rest) > 0 {
		id = ctx.rest[0]
	}
	if id == "" {
		id = ctx.flags.Method
	}
	var method methods.Method
	if id != "" {
		method, err = methods.Require(id)
	} else {
		method, err = methods.Resolve(cfg, "")
	}
	if err != nil {
		return err
	}
	info := methods.Describe(method, cfg)
	limit := "n/a"
	if info.Metadata.ContentCharacterLimit != nil {
		limit = fmt.Sprintf("%d", *info.Metadata.ContentCharacterLimit)
	}
	lines := []string{
		fmt.Sprintf("%s (%s)", info.Name, info.ID),
		info.Description,
		fmt.Sprintf("configured: %v", info.Configured),
		fmt.Sprintf("default: %v", info.IsDefault),
		fmt.Sprintf("content limit: %s", limit),
	}
	if info.Metadata.RateLimit != "" {
		lines = append(lines, "rate limit: "+info.Metadata.RateLimit)
	}
	lines = append(lines, "docs: "+info.Metadata.DocsURL, "payload:")
	for _, key := range []string{"content", "username", "avatar_url", "embeds"} {
		if value, ok := info.Metadata.PayloadShape[key]; ok {
			lines = append(lines, fmt.Sprintf("  %s: %s", key, value))
		}
	}
	for k, v := range info.Metadata.PayloadShape {
		if k == "content" || k == "username" || k == "avatar_url" || k == "embeds" {
			continue
		}
		lines = append(lines, fmt.Sprintf("  %s: %s", k, v))
	}
	for _, note := range info.Metadata.Notes {
		lines = append(lines, "note: "+note)
	}
	return emit(ctx, strings.Join(lines, "\n"), info)
}

func handleDefault(ctx context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	id := ""
	if len(ctx.rest) > 0 {
		id = ctx.rest[0]
	}
	if id == "" {
		id = ctx.flags.Method
	}
	if id == "" {
		current := config.DefaultID(cfg)
		if current == "" {
			return fmt.Errorf("no default method set. Run a2h init <method> first")
		}
		return emit(ctx, current, map[string]any{"defaultMethod": current})
	}
	if _, err := methods.Require(id); err != nil {
		return err
	}
	cfg.DefaultMethod = &id
	if err := config.Save(cfg); err != nil {
		return err
	}
	return emit(ctx, "default method is "+id, map[string]any{"defaultMethod": id})
}

func handleStatus(ctx context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	infos := make([]methods.Info, 0, len(methods.List()))
	def := config.DefaultID(cfg)
	if def == "" {
		def = "(none)"
	}
	lines := []string{
		"config: " + config.Path(),
		"default: " + def,
	}
	for _, method := range methods.List() {
		info := methods.Describe(method, cfg)
		infos = append(infos, info)
		state := "not configured"
		if info.Configured {
			state = "configured"
		}
		mark := " "
		if info.IsDefault {
			mark = "*"
		}
		lines = append(lines, fmt.Sprintf("%s %s  %s", mark, info.ID, state))
	}
	return emit(ctx, strings.Join(lines, "\n"), map[string]any{
		"configDir":     config.Dir(),
		"configPath":    config.Path(),
		"defaultMethod": cfg.DefaultMethod,
		"methods":       infos,
	})
}

func handleUpdate(ctx context) error {
	client := update.NewClient()
	latest, err := client.Latest()
	if err != nil {
		return err
	}
	current := version.String()
	needed := update.NeedsUpdate(current, latest.Version)
	payload := map[string]any{
		"current":  current,
		"latest":   latest.Version,
		"upToDate": !needed,
	}
	if ctx.flags.Check {
		human := fmt.Sprintf("up to date (%s)", current)
		if needed {
			human = fmt.Sprintf("%s -> %s available", current, latest.Version)
		}
		return emit(ctx, human, payload)
	}
	if !needed {
		return emit(ctx, fmt.Sprintf("already up to date (%s)", current), payload)
	}
	if !update.Supported() {
		return fmt.Errorf("no prebuilt binary for this platform; supported: darwin/arm64, linux/amd64, linux/arm64")
	}
	asset := update.CurrentAsset()
	assetURL, ok := latest.Assets[asset]
	if !ok {
		return fmt.Errorf("release %s has no asset %s", latest.Tag, asset)
	}
	sumsURL, ok := latest.Assets[update.ChecksumsName]
	if !ok {
		return fmt.Errorf("release %s is missing %s", latest.Tag, update.ChecksumsName)
	}
	sums, err := client.Download(sumsURL)
	if err != nil {
		return err
	}
	want, err := update.ChecksumFor(sums, asset)
	if err != nil {
		return err
	}
	bin, err := client.Download(assetURL)
	if err != nil {
		return err
	}
	if err := update.VerifySHA256(bin, want); err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if err := update.ReplaceExecutable(exe, bin); err != nil {
		return err
	}
	payload["ok"] = true
	payload["asset"] = asset
	return emit(ctx, fmt.Sprintf("updated %s -> %s", current, latest.Version), payload)
}

func notify(text, methodID, username string) (methods.SendResult, error) {
	cfg, err := config.Load()
	if err != nil {
		return methods.SendResult{}, err
	}
	method, err := methods.Resolve(cfg, methodID)
	if err != nil {
		return methods.SendResult{}, err
	}
	settings := config.Settings(cfg, method.ID())
	if !method.Configured(settings) {
		return methods.SendResult{}, fmt.Errorf("%s is not configured. Run: a2h init %s", method.Name(), method.ID())
	}
	return method.Send(settings, methods.SendRequest{Text: text, Username: username})
}

func isTTY(r io.Reader) bool {
	file, ok := r.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
