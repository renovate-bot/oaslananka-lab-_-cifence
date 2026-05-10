#!/usr/bin/env node
import { spawnSync } from "node:child_process";

run("pnpm", ["run", "build"]);
run("git", ["diff", "--exit-code", "--", "dist/index.js"]);

if (process.platform === "linux") {
  run("pnpm", ["run", "binaries"]);
  run("git", ["diff", "--exit-code", "--", "dist/bin"]);
} else {
  process.stdout.write("dist/bin freshness is checked on linux\n");
}

process.stdout.write("dist fresh\n");

function run(command, args) {
  const invocation = resolveInvocation(command, args);
  const result = spawnSync(invocation.command, invocation.args, {
    encoding: "utf8",
    stdio: "inherit",
  });
  if (result.error) {
    process.stderr.write(`${command} failed: ${result.error.message}\n`);
    process.exit(1);
  }
  if (result.status !== 0) {
    process.exit(result.status ?? 1);
  }
}

function resolveInvocation(command, args) {
  if (process.platform === "win32" && command === "pnpm") {
    return { command: "cmd.exe", args: ["/d", "/s", "/c", ["pnpm", ...args].join(" ")] };
  }
  return { command, args };
}
