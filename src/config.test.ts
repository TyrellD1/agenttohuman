import assert from "node:assert/strict";
import { mkdtemp, readFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { test } from "node:test";
import {
  emptyConfig,
  getConfigPath,
  loadConfig,
  saveConfig,
  withMethodSettings,
} from "./config.js";

async function withHome(run: () => Promise<void>): Promise<void> {
  const dir = await mkdtemp(join(tmpdir(), "a2h-"));
  const previous = process.env.AGENTTOHUMAN_HOME;
  process.env.AGENTTOHUMAN_HOME = dir;
  try {
    await run();
  } finally {
    if (previous === undefined) {
      delete process.env.AGENTTOHUMAN_HOME;
    } else {
      process.env.AGENTTOHUMAN_HOME = previous;
    }
  }
}

test("loadConfig returns empty config when the file is missing", async () => {
  await withHome(async () => {
    const config = await loadConfig();
    assert.deepEqual(config, emptyConfig());
  });
});

test("saveConfig round-trips and withMethodSettings sets the first default", async () => {
  await withHome(async () => {
    const saved = withMethodSettings(emptyConfig(), "discord-webhook", {
      webhookUrl: "https://discord.com/api/webhooks/1/token",
    });
    await saveConfig(saved);
    const loaded = await loadConfig();
    assert.equal(loaded.defaultMethod, "discord-webhook");
    assert.equal(
      loaded.methods["discord-webhook"]?.webhookUrl,
      "https://discord.com/api/webhooks/1/token",
    );
    const raw = await readFile(getConfigPath(), "utf8");
    assert.match(raw, /discord-webhook/);
  });
});
