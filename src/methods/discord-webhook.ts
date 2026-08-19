import { createInterface } from "node:readline/promises";
import { stdin as input, stdout as output } from "node:process";
import { A2hError } from "../errors.js";
import type { MethodConfig } from "../config.js";
import type { InitRequest, MethodDefinition, SendRequest, SendResult } from "./types.js";

export const DISCORD_WEBHOOK_ID = "discord-webhook";

const WEBHOOK_URL_RE =
  /^https:\/\/(?:(?:ptb|canary)\.)?discord(?:app)?\.com\/api(?:\/v\d+)?\/webhooks\/\d+\/[\w-]+$/i;

export const discordWebhook: MethodDefinition = {
  id: DISCORD_WEBHOOK_ID,
  name: "Discord webhook",
  description: "POST a message to a Discord channel webhook.",
  metadata: {
    contentCharacterLimit: 2000,
    usernameCharacterLimit: 80,
    rateLimit: "5 requests per 2 seconds per webhook",
    docsUrl: "https://discord.com/developers/docs/resources/webhook#execute-webhook",
    notes: [
      "content is required for a2h send (plain text / Discord markdown).",
      "Discord may silently truncate content over 2000 characters; a2h rejects it first.",
      "username overrides the webhook's default display name for that message.",
    ],
    payloadShape: {
      content: "string, required, max 2000 characters",
      username: "string, optional, max 80 characters",
      avatar_url: "string URL, optional (not exposed by a2h send yet)",
      embeds: "array, optional, max 10 (not exposed by a2h send yet)",
    },
  },
  isConfigured(settings) {
    return typeof settings.webhookUrl === "string" && settings.webhookUrl.length > 0;
  },
  async init(settings, request) {
    const webhookUrl = await resolveWebhookUrl(request);
    assertWebhookUrl(webhookUrl);

    const usernameFlag = request.flags.username;
    const username =
      typeof usernameFlag === "string" && usernameFlag.trim()
        ? usernameFlag.trim()
        : typeof settings.username === "string"
          ? settings.username
          : undefined;

    if (username) {
      assertUsername(username);
    }

    const next: MethodConfig = { ...settings, webhookUrl };
    if (username) {
      next.username = username;
    }

    if (request.flags.test === true) {
      await sendWithSettings(next, {
        text: "a2h connected. Discord webhook is ready.",
        username,
      });
    }

    return next;
  },
  async send(settings, request) {
    return sendWithSettings(settings, request);
  },
};

export function assertWebhookUrl(url: string): void {
  if (!WEBHOOK_URL_RE.test(url)) {
    throw new A2hError(
      "That does not look like a Discord webhook URL. Expected https://discord.com/api/webhooks/<id>/<token>",
    );
  }
}

export function assertUsername(username: string): void {
  if (username.length > 80) {
    throw new A2hError("Discord webhook username must be 80 characters or fewer.");
  }
  const banned = username.trim().toLowerCase();
  if (banned === "clyde" || banned === "discord") {
    throw new A2hError('Discord rejects webhook usernames "Clyde" and "Discord".');
  }
}

export function assertContent(text: string): void {
  if (!text.trim()) {
    throw new A2hError("Message is empty.");
  }
  if (text.length > 2000) {
    throw new A2hError(
      `Message is ${text.length} characters; Discord webhook content max is 2000. Split it and send again.`,
    );
  }
}

async function resolveWebhookUrl(request: InitRequest): Promise<string> {
  const fromFlag = request.flags["webhook-url"];
  if (typeof fromFlag === "string" && fromFlag.trim()) {
    return fromFlag.trim();
  }
  if (!request.interactive) {
    throw new A2hError(
      "No webhook URL provided. Re-run with --webhook-url <url>, or run a2h init discord-webhook in a terminal.",
    );
  }
  return promptWebhookUrl();
}

async function promptWebhookUrl(): Promise<string> {
  const rl = createInterface({ input, output });
  try {
    const answer = await rl.question(
      "Discord webhook URL\n  Channel Settings → Integrations → Webhooks → Copy Webhook URL\n> ",
    );
    const url = answer.trim();
    if (!url) {
      throw new A2hError("No webhook URL entered.");
    }
    return url;
  } finally {
    rl.close();
  }
}

async function sendWithSettings(
  settings: MethodConfig,
  request: SendRequest,
): Promise<SendResult> {
  const webhookUrl = settings.webhookUrl;
  if (typeof webhookUrl !== "string" || !webhookUrl) {
    throw new A2hError(
      "Discord webhook is not configured. Run: a2h init discord-webhook",
    );
  }
  assertWebhookUrl(webhookUrl);
  assertContent(request.text);

  const username =
    request.username?.trim() ||
    (typeof settings.username === "string" ? settings.username : undefined);
  if (username) {
    assertUsername(username);
  }

  const payload: Record<string, string> = { content: request.text };
  if (username) {
    payload.username = username;
  }

  const url = new URL(webhookUrl);
  url.searchParams.set("wait", "true");

  const response = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });

  if (response.status === 429) {
    const retryAfter = response.headers.get("retry-after") ?? "unknown";
    throw new A2hError(`Discord rate-limited this webhook. Retry after ${retryAfter}s.`);
  }

  if (!response.ok) {
    const detail = await readErrorBody(response);
    throw new A2hError(
      `Discord webhook failed (${response.status})${detail ? `: ${detail}` : "."}`,
    );
  }

  let id: string | undefined;
  try {
    const body = (await response.json()) as { id?: unknown };
    if (typeof body.id === "string") {
      id = body.id;
    }
  } catch {
    // wait=true should return JSON; ignore if Discord sent an empty body.
  }

  return { method: DISCORD_WEBHOOK_ID, id };
}

async function readErrorBody(response: Response): Promise<string | undefined> {
  try {
    const body = (await response.json()) as { message?: unknown; code?: unknown };
    const parts: string[] = [];
    if (typeof body.message === "string") {
      parts.push(body.message);
    }
    if (typeof body.code === "number") {
      parts.push(`code ${body.code}`);
    }
    return parts.join(" ");
  } catch {
    return undefined;
  }
}
