import assert from "node:assert/strict";
import { mkdtemp } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { test } from "node:test";
import { main } from "./cli.js";

async function withHome(run: () => Promise<void>): Promise<void> {
  const dir = await mkdtemp(join(tmpdir(), "a2h-cli-"));
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

async function capture(run: () => Promise<void>): Promise<{ stdout: string; stderr: string }> {
  const stdoutChunks: string[] = [];
  const stderrChunks: string[] = [];
  const stdoutWrite = process.stdout.write.bind(process.stdout);
  const stderrWrite = process.stderr.write.bind(process.stderr);
  process.stdout.write = ((chunk: string | Uint8Array) => {
    stdoutChunks.push(String(chunk));
    return true;
  }) as typeof process.stdout.write;
  process.stderr.write = ((chunk: string | Uint8Array) => {
    stderrChunks.push(String(chunk));
    return true;
  }) as typeof process.stderr.write;
  try {
    await run();
  } finally {
    process.stdout.write = stdoutWrite;
    process.stderr.write = stderrWrite;
  }
  return { stdout: stdoutChunks.join(""), stderr: stderrChunks.join("") };
}

test("methods --json lists discord-webhook", async () => {
  await withHome(async () => {
    const { stdout } = await capture(() => main(["methods", "--json"]));
    const body = JSON.parse(stdout) as {
      methods: Array<{ id: string; configured: boolean }>;
    };
    assert.equal(body.methods[0]?.id, "discord-webhook");
    assert.equal(body.methods[0]?.configured, false);
  });
});

test("init then default and info round-trip without sending", async () => {
  await withHome(async () => {
    const url = "https://discord.com/api/webhooks/42/token-value";
    const { stdout: initOut } = await capture(() =>
      main(["init", "discord-webhook", "--webhook-url", url, "--json"]),
    );
    const initBody = JSON.parse(initOut) as { defaultMethod: string };
    assert.equal(initBody.defaultMethod, "discord-webhook");

    const { stdout: defaultOut } = await capture(() => main(["default", "--json"]));
    assert.equal(JSON.parse(defaultOut).defaultMethod, "discord-webhook");

    const { stdout: infoOut } = await capture(() => main(["info", "--json"]));
    const info = JSON.parse(infoOut) as {
      configured: boolean;
      metadata: { contentCharacterLimit: number };
    };
    assert.equal(info.configured, true);
    assert.equal(info.metadata.contentCharacterLimit, 2000);
  });
});

test("send without init fails with a configuration error", async () => {
  await withHome(async () => {
    await assert.rejects(
      () => capture(() => main(["send", "hello"])),
      /not configured/,
    );
  });
});
