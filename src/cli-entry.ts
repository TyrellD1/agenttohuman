#!/usr/bin/env node
import { main } from "./cli.js";
import { emitError } from "./output.js";

const json = process.argv.includes("--json");

main().catch((error: unknown) => {
  process.exitCode = emitError(json, error);
});
