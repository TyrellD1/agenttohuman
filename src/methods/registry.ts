import { A2hError } from "../errors.js";
import type { AppConfig } from "../config.js";
import { discordWebhook } from "./discord-webhook.js";
import type { MethodDefinition } from "./types.js";

const methods: MethodDefinition[] = [discordWebhook];

const byId = new Map(methods.map((method) => [method.id, method]));

export function listMethods(): MethodDefinition[] {
  return [...methods];
}

export function getMethod(id: string): MethodDefinition | undefined {
  return byId.get(id);
}

export function requireMethod(id: string): MethodDefinition {
  const method = getMethod(id);
  if (!method) {
    const known = methods.map((item) => item.id).join(", ") || "(none)";
    throw new A2hError(`Unknown method "${id}". Known methods: ${known}`);
  }
  return method;
}

export function resolveMethod(config: AppConfig, methodId?: string): MethodDefinition {
  const id = methodId?.trim() || config.defaultMethod;
  if (id) {
    return requireMethod(id);
  }
  if (methods.length === 1) {
    return methods[0]!;
  }
  throw new A2hError(
    "No default method is set. Run a2h init <method>, or pass --method <id>.",
  );
}
