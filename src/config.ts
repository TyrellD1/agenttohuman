import { homedir } from "node:os";
import { dirname, join } from "node:path";
import { mkdir, readFile, rename, writeFile } from "node:fs/promises";
import { A2hError } from "./errors.js";

export const CONFIG_VERSION = 1;

export type MethodConfig = {
  webhookUrl?: string;
  username?: string;
  [key: string]: unknown;
};

export type AppConfig = {
  version: number;
  defaultMethod: string | null;
  methods: Record<string, MethodConfig>;
};

export const emptyConfig = (): AppConfig => ({
  version: CONFIG_VERSION,
  defaultMethod: null,
  methods: {},
});

export function getConfigDir(): string {
  return process.env.AGENTTOHUMAN_HOME?.trim() || join(homedir(), ".agenttohuman");
}

export function getConfigPath(): string {
  return join(getConfigDir(), "config.json");
}

export async function loadConfig(): Promise<AppConfig> {
  const path = getConfigPath();
  let raw: string;
  try {
    raw = await readFile(path, "utf8");
  } catch (error) {
    if (isNotFound(error)) {
      return emptyConfig();
    }
    throw error;
  }

  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    throw new A2hError(`Config at ${path} is not valid JSON.`);
  }

  return normalizeConfig(parsed, path);
}

export async function saveConfig(config: AppConfig): Promise<void> {
  const path = getConfigPath();
  const dir = dirname(path);
  await mkdir(dir, { recursive: true, mode: 0o700 });
  const tmp = `${path}.tmp`;
  const body = `${JSON.stringify(config, null, 2)}\n`;
  await writeFile(tmp, body, { encoding: "utf8", mode: 0o600 });
  await rename(tmp, path);
}

export function methodSettings(config: AppConfig, methodId: string): MethodConfig {
  return config.methods[methodId] ?? {};
}

export function withMethodSettings(
  config: AppConfig,
  methodId: string,
  settings: MethodConfig,
): AppConfig {
  const next: AppConfig = {
    ...config,
    methods: {
      ...config.methods,
      [methodId]: { ...methodSettings(config, methodId), ...settings },
    },
  };
  if (!next.defaultMethod) {
    next.defaultMethod = methodId;
  }
  return next;
}

function normalizeConfig(parsed: unknown, path: string): AppConfig {
  if (!isRecord(parsed)) {
    throw new A2hError(`Config at ${path} must be a JSON object.`);
  }

  const methodsValue = parsed.methods;
  const methods: Record<string, MethodConfig> = {};
  if (methodsValue !== undefined) {
    if (!isRecord(methodsValue)) {
      throw new A2hError(`Config at ${path} has an invalid "methods" field.`);
    }
    for (const [id, value] of Object.entries(methodsValue)) {
      if (!isRecord(value)) {
        throw new A2hError(`Config at ${path} has an invalid methods.${id} field.`);
      }
      methods[id] = { ...value };
    }
  }

  const defaultMethod =
    parsed.defaultMethod === null || parsed.defaultMethod === undefined
      ? null
      : typeof parsed.defaultMethod === "string"
        ? parsed.defaultMethod
        : null;

  return {
    version: typeof parsed.version === "number" ? parsed.version : CONFIG_VERSION,
    defaultMethod,
    methods,
  };
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function isNotFound(error: unknown): boolean {
  return (
    typeof error === "object" &&
    error !== null &&
    "code" in error &&
    (error as { code: unknown }).code === "ENOENT"
  );
}
