import { loadConfig } from "../config.js";
import { emit, type CliContext } from "../output.js";
import { requireMethod, resolveMethod } from "../methods/registry.js";
import { describeMethod } from "../methods/types.js";

export async function handleInfo(ctx: CliContext): Promise<void> {
  const config = await loadConfig();
  const id = ctx.rest[0] ?? ctx.values.method;
  const method = id ? requireMethod(id) : resolveMethod(config);
  const info = describeMethod(method, config);
  const human = [
    `${info.name} (${info.id})`,
    info.description,
    `configured: ${info.configured}`,
    `default: ${info.isDefault}`,
    `content limit: ${info.metadata.contentCharacterLimit ?? "n/a"}`,
    info.metadata.rateLimit ? `rate limit: ${info.metadata.rateLimit}` : null,
    `docs: ${info.metadata.docsUrl}`,
    "payload:",
    ...Object.entries(info.metadata.payloadShape).map(
      ([key, value]) => `  ${key}: ${value}`,
    ),
    ...info.metadata.notes.map((note) => `note: ${note}`),
  ]
    .filter((line) => line !== null)
    .join("\n");

  emit(ctx, human, info);
}
