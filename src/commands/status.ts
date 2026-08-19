import { getConfigDir, getConfigPath, loadConfig } from "../config.js";
import { emit, type CliContext } from "../output.js";
import { listMethods } from "../methods/registry.js";
import { describeMethod } from "../methods/types.js";

export async function handleStatus(ctx: CliContext): Promise<void> {
  const config = await loadConfig();
  const methods = listMethods().map((method) => describeMethod(method, config));
  const human = [
    `config: ${getConfigPath()}`,
    `default: ${config.defaultMethod ?? "(none)"}`,
    ...methods.map((method) => {
      const state = method.configured ? "configured" : "not configured";
      const mark = method.isDefault ? "*" : " ";
      return `${mark} ${method.id}  ${state}`;
    }),
  ].join("\n");

  emit(ctx, human, {
    configDir: getConfigDir(),
    configPath: getConfigPath(),
    defaultMethod: config.defaultMethod,
    methods,
  });
}
