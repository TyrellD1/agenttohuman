import { parseArgs } from "node:util";
import { readFile } from "node:fs/promises";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";
import { A2hError } from "./errors.js";
import type { CliContext, CliValues } from "./output.js";
import { handleDefault } from "./commands/default.js";
import { handleInfo } from "./commands/info.js";
import { handleInit } from "./commands/init.js";
import { handleMethods } from "./commands/methods.js";
import { handleSend } from "./commands/send.js";
import { handleStatus } from "./commands/status.js";

const HELP = `a2h — agent-to-human notifications

Usage:
  a2h send <message>          Send via the default method
  a2h init [method]           Save credentials for a method
  a2h methods                 List available methods
  a2h info [method]           Method metadata and payload shape
  a2h default [method]        Get or set the default method
  a2h status                  Local config summary

Options:
  -m, --method <id>           Method for send / info / init
  --webhook-url <url>         Discord webhook URL (init)
  --username <name>           Webhook username override
  --stdin                     Read send message from stdin
  --test                      Send a test message during init
  --json                      Machine-readable output
  -h, --help
  -V, --version

Data is stored in ~/.agenttohuman (override with AGENTTOHUMAN_HOME).
`;

export async function main(argv = process.argv.slice(2)): Promise<void> {
  const { values, positionals } = parseArgs({
    args: argv,
    allowPositionals: true,
    options: {
      method: { type: "string", short: "m" },
      json: { type: "boolean", default: false },
      help: { type: "boolean", short: "h", default: false },
      version: { type: "boolean", short: "V", default: false },
      stdin: { type: "boolean", default: false },
      test: { type: "boolean", default: false },
      username: { type: "string" },
      "webhook-url": { type: "string" },
    },
  });

  const typed = values as CliValues;
  if (typed.help || argv.includes("--help")) {
    process.stdout.write(HELP);
    return;
  }
  if (typed.version) {
    process.stdout.write(`${await readVersion()}\n`);
    return;
  }

  const [command, ...rest] = positionals;
  const ctx: CliContext = {
    command,
    rest,
    values: typed,
    json: typed.json === true,
  };

  if (!command) {
    process.stdout.write(HELP);
    return;
  }

  switch (command) {
    case "send":
      await handleSend(ctx);
      return;
    case "init":
      await handleInit(ctx);
      return;
    case "methods":
      await handleMethods(ctx);
      return;
    case "info":
      await handleInfo(ctx);
      return;
    case "default":
      await handleDefault(ctx);
      return;
    case "status":
      await handleStatus(ctx);
      return;
    default:
      throw new A2hError(`Unknown command "${command}". Run a2h --help.`);
  }
}

async function readVersion(): Promise<string> {
  const here = dirname(fileURLToPath(import.meta.url));
  const pkgPath = join(here, "..", "package.json");
  const raw = await readFile(pkgPath, "utf8");
  const pkg = JSON.parse(raw) as { version?: string };
  return pkg.version ?? "0.0.0";
}
