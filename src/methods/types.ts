import type { AppConfig, MethodConfig } from "../config.js";

export type MethodMetadata = {
  contentCharacterLimit: number | null;
  usernameCharacterLimit?: number;
  rateLimit?: string;
  docsUrl: string;
  notes: string[];
  payloadShape: Record<string, string>;
};

export type SendRequest = {
  text: string;
  username?: string;
};

export type SendResult = {
  method: string;
  id?: string;
};

export type InitRequest = {
  flags: Record<string, string | boolean | undefined>;
  interactive: boolean;
};

export type MethodDefinition = {
  id: string;
  name: string;
  description: string;
  metadata: MethodMetadata;
  isConfigured: (settings: MethodConfig) => boolean;
  init: (settings: MethodConfig, request: InitRequest) => Promise<MethodConfig>;
  send: (settings: MethodConfig, request: SendRequest) => Promise<SendResult>;
};

export type MethodInfo = {
  id: string;
  name: string;
  description: string;
  configured: boolean;
  isDefault: boolean;
  metadata: MethodMetadata;
};

export function describeMethod(
  method: MethodDefinition,
  config: AppConfig,
): MethodInfo {
  return {
    id: method.id,
    name: method.name,
    description: method.description,
    configured: method.isConfigured(config.methods[method.id] ?? {}),
    isDefault: config.defaultMethod === method.id,
    metadata: method.metadata,
  };
}
