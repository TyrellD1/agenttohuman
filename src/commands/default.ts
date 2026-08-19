import { loadConfig, saveConfig } from "../config.js";
import { A2hError } from "../errors.js";
import { emit, type CliContext } from "../output.js";
import { requireMethod } from "../methods/registry.js";

export async function handleDefault(ctx: CliContext): Promise<void> {
  const config = await loadConfig();
  const id = ctx.rest[0] ?? ctx.values.method;

  if (!id) {
    if (!config.defaultMethod) {
      throw new A2hError("No default method set. Run a2h init <method> first.");
    }
    emit(ctx, config.defaultMethod, { defaultMethod: config.defaultMethod });
    return;
  }

  requireMethod(id);
  const next = { ...config, defaultMethod: id };
  await saveConfig(next);
  emit(ctx, `default method is ${id}`, { defaultMethod: id });
}
