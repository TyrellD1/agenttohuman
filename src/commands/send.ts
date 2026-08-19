import { stdin } from "node:process";
import { A2hError } from "../errors.js";
import { notify } from "../notify.js";
import { emit, type CliContext } from "../output.js";

export async function handleSend(ctx: CliContext): Promise<void> {
  const text = ctx.values.stdin ? await readStdin() : ctx.rest.join(" ").trim();
  if (!text) {
    throw new A2hError("No message given. Usage: a2h send <message>  (or --stdin)");
  }

  const result = await notify(text, {
    method: ctx.values.method,
    username: ctx.values.username,
  });

  emit(ctx, result.id ? `sent (${result.method} ${result.id})` : `sent (${result.method})`, {
    ok: true,
    ...result,
  });
}

async function readStdin(): Promise<string> {
  const chunks: Buffer[] = [];
  for await (const chunk of stdin) {
    chunks.push(typeof chunk === "string" ? Buffer.from(chunk) : chunk);
  }
  return Buffer.concat(chunks).toString("utf8").trim();
}
