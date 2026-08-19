import { A2hError } from "./errors.js";

export type CliValues = {
  method?: string;
  json?: boolean;
  help?: boolean;
  version?: boolean;
  stdin?: boolean;
  test?: boolean;
  username?: string;
  "webhook-url"?: string;
};

export type CliContext = {
  command: string | undefined;
  rest: string[];
  values: CliValues;
  json: boolean;
};

export function emit(ctx: CliContext, human: string, data: unknown): void {
  if (ctx.json) {
    process.stdout.write(`${JSON.stringify(data, null, 2)}\n`);
    return;
  }
  process.stdout.write(`${human}\n`);
}

export function emitError(json: boolean, error: unknown): number {
  const message = error instanceof Error ? error.message : String(error);
  if (json) {
    process.stderr.write(`${JSON.stringify({ error: message })}\n`);
  } else {
    process.stderr.write(`${message}\n`);
  }
  return error instanceof A2hError ? error.exitCode : 1;
}
