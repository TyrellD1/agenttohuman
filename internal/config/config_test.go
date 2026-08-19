package config

import (
	"os"
	"strings"
	"testing"
)

func TestLoadMissingReturnsEmpty(t *testing.T) {
	t.Setenv("AGENTTOHUMAN_HOME", t.TempDir())
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DefaultMethod != nil {
		t.Fatalf("expected no default, got %v", *cfg.DefaultMethod)
	}
	if len(cfg.Methods) != 0 {
		t.Fatalf("expected empty methods, got %#v", cfg.Methods)
	}
}

func TestSaveRoundTripSetsDefault(t *testing.T) {
	t.Setenv("AGENTTOHUMAN_HOME", t.TempDir())
	saved := WithSettings(Empty(), "discord-webhook", MethodConfig{
		"webhookUrl": "https://discord.com/api/webhooks/1/token",
	})
	if err := Save(saved); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if DefaultID(loaded) != "discord-webhook" {
		t.Fatalf("default = %q", DefaultID(loaded))
	}
	got := Settings(loaded, "discord-webhook").String("webhookUrl")
	if got != "https://discord.com/api/webhooks/1/token" {
		t.Fatalf("webhookUrl = %q", got)
	}
	raw, err := os.ReadFile(Path())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "discord-webhook") {
		t.Fatalf("config file missing method: %s", raw)
	}
	info, err := os.Stat(Path())
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("config file perms: %s", info.Mode())
	}
}
