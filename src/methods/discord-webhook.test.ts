import assert from "node:assert/strict";
import { test } from "node:test";
import { A2hError } from "../errors.js";
import {
  assertContent,
  assertUsername,
  assertWebhookUrl,
  discordWebhook,
} from "./discord-webhook.js";

test("accepts discord.com and discordapp.com webhook URLs", () => {
  assertWebhookUrl("https://discord.com/api/webhooks/123/abc-DEF_456");
  assertWebhookUrl("https://discordapp.com/api/webhooks/123/abc");
  assertWebhookUrl("https://ptb.discord.com/api/v10/webhooks/123/abc");
});

test("rejects non-webhook URLs", () => {
  assert.throws(() => assertWebhookUrl("https://example.com/hooks/1"), A2hError);
  assert.throws(() => assertWebhookUrl("https://discord.com/channels/1/2"), A2hError);
});

test("rejects empty or oversized content", () => {
  assert.throws(() => assertContent(""), A2hError);
  assert.throws(() => assertContent("a".repeat(2001)), A2hError);
  assertContent("a".repeat(2000));
});

test("rejects banned Discord webhook usernames", () => {
  assert.throws(() => assertUsername("Clyde"), A2hError);
  assert.throws(() => assertUsername("x".repeat(81)), A2hError);
  assertUsername("a2h");
});

test("discord-webhook is unconfigured without a URL", () => {
  assert.equal(discordWebhook.isConfigured({}), false);
  assert.equal(
    discordWebhook.isConfigured({
      webhookUrl: "https://discord.com/api/webhooks/1/token",
    }),
    true,
  );
});

test("send posts content and optional username", async () => {
  const original = globalThis.fetch;
  const calls: Array<{ url: string; body: unknown }> = [];
  globalThis.fetch = (async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = String(input);
    calls.push({ url, body: JSON.parse(String(init?.body)) });
    return new Response(JSON.stringify({ id: "msg_1" }), { status: 200 });
  }) as typeof fetch;

  try {
    const result = await discordWebhook.send(
      { webhookUrl: "https://discord.com/api/webhooks/1/token" },
      { text: "hello from a2h", username: "a2h" },
    );
    assert.equal(result.method, "discord-webhook");
    assert.equal(result.id, "msg_1");
    assert.equal(calls.length, 1);
    assert.match(calls[0]!.url, /wait=true/);
    assert.deepEqual(calls[0]!.body, { content: "hello from a2h", username: "a2h" });
  } finally {
    globalThis.fetch = original;
  }
});
