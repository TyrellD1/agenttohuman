import { stdin } from "node:process";
import { loadConfig, saveConfig, withMethodSettings } from "../config.js";
import { emit, type CliContext } from "../output.js";
import { resolveMethod } from "../methods/registry.js";

export async function handleInit(ctx: CliContext): Promise<void> {
  const config = await loadConfig();
  const id = ctx.rest[0] ?? ctx.values.method;
  const method = resolveMethod(config, id);

  const interactive = Boolean(stdin.isTTY) && !ctx.values["webhook-url"];
  const settings = await method.init(config.methods[method.id] ?? {}, {
    flags: {
      "webhook-url": ctx.values["webhook-url"],
      username: ctx.values.username,
      test: ctx.values.test,
    },
    interactive,
  });

  const next = withMethodSettings(config, method.id, settings);
  await saveConfig(next);

  emit(ctx, `saved ${method.id} (default: ${next.defaultMethod})`, {
    ok: true,
    method: method.id,
    defaultMethod: next.defaultMethod,
  });
}
