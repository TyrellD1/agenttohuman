import type { AppConfig } from "./config.js";
import { loadConfig, methodSettings } from "./config.js";
import { A2hError } from "./errors.js";
import { resolveMethod } from "./methods/registry.js";
import type { SendResult } from "./methods/types.js";

export type NotifyOptions = {
  method?: string;
  username?: string;
  config?: AppConfig;
};

export async function notify(text: string, options: NotifyOptions = {}): Promise<SendResult> {
  const config = options.config ?? (await loadConfig());
  const method = resolveMethod(config, options.method);
  const settings = methodSettings(config, method.id);
  if (!method.isConfigured(settings)) {
    throw new A2hError(`${method.name} is not configured. Run: a2h init ${method.id}`);
  }
  return method.send(settings, { text, username: options.username });
}
