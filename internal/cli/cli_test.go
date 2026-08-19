package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestMethodsJSONListsDiscord(t *testing.T) {
	t.Setenv("AGENTTOHUMAN_HOME", t.TempDir())
	stdout := &bytes.Buffer{}
	if err := Run([]string{"methods", "--json"}, strings.NewReader(""), stdout, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	var body struct {
		Methods []struct {
			ID         string `json:"id"`
			Configured bool   `json:"configured"`
		} `json:"methods"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Methods) != 1 || body.Methods[0].ID != "discord-webhook" {
		t.Fatalf("methods = %#v", body.Methods)
	}
	if body.Methods[0].Configured {
		t.Fatal("expected unconfigured")
	}
}

func TestInitDefaultInfoRoundTrip(t *testing.T) {
	t.Setenv("AGENTTOHUMAN_HOME", t.TempDir())
	url := "https://discord.com/api/webhooks/42/token-value"
	stdout := &bytes.Buffer{}
	if err := Run([]string{"init", "discord-webhook", "--webhook-url", url, "--json"}, strings.NewReader(""), stdout, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	var initBody struct {
		DefaultMethod string `json:"defaultMethod"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &initBody); err != nil {
		t.Fatal(err)
	}
	if initBody.DefaultMethod != "discord-webhook" {
		t.Fatalf("default = %s", initBody.DefaultMethod)
	}

	stdout.Reset()
	if err := Run([]string{"default", "--json"}, strings.NewReader(""), stdout, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	var defBody struct {
		DefaultMethod string `json:"defaultMethod"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &defBody); err != nil {
		t.Fatal(err)
	}
	if defBody.DefaultMethod != "discord-webhook" {
		t.Fatalf("default = %s", defBody.DefaultMethod)
	}

	stdout.Reset()
	if err := Run([]string{"info", "--json"}, strings.NewReader(""), stdout, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	var info struct {
		Configured bool `json:"configured"`
		Metadata   struct {
			ContentCharacterLimit int `json:"contentCharacterLimit"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &info); err != nil {
		t.Fatal(err)
	}
	if !info.Configured {
		t.Fatal("expected configured")
	}
	if info.Metadata.ContentCharacterLimit != 2000 {
		t.Fatalf("limit = %d", info.Metadata.ContentCharacterLimit)
	}
}

func TestSendWithoutInitFails(t *testing.T) {
	t.Setenv("AGENTTOHUMAN_HOME", t.TempDir())
	err := Run([]string{"send", "hello"}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("err = %v", err)
	}
}

func TestUnknownCommand(t *testing.T) {
	err := Run([]string{"nope"}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("err = %v", err)
	}
}
