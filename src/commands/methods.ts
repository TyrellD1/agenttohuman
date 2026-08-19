import { loadConfig } from "../config.js";
import { emit, type CliContext } from "../output.js";
import { listMethods } from "../methods/registry.js";
import { describeMethod } from "../methods/types.js";

export async function handleMethods(ctx: CliContext): Promise<void> {
  const config = await loadConfig();
  const methods = listMethods().map((method) => describeMethod(method, config));
  const lines = methods.map((method) => {
    const flags = [
      method.isDefault ? "default" : null,
      method.configured ? "configured" : "not configured",
    ]
      .filter(Boolean)
      .join(", ");
    return `${method.id}  ${method.name}  (${flags})`;
  });

  emit(ctx, lines.join("\n") || "No methods registered.", {
    defaultMethod: config.defaultMethod,
    methods,
  });
}
