#!/usr/bin/env node
import { spawnSync } from "node:child_process";
import { mkdtempSync, readFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

const packageJson = JSON.parse(readFileSync("package.json", "utf8"));
const tempDir = mkdtempSync(join(tmpdir(), "cifence-security-"));
const binary = join(tempDir, process.platform === "win32" ? "cifence.exe" : "cifence");

try {
  run("pnpm", ["audit", "--audit-level", "moderate"]);
  run("go", ["run", "golang.org/x/vuln/cmd/govulncheck@v1.3.0", "./..."]);
  run("go", [
    "build",
    "-trimpath",
    `-ldflags=-s -w -X github.com/oaslananka/cifence/internal/analyzer.Version=${packageJson.version}`,
    "-o",
    binary,
    "./cmd/cifence",
  ]);
  run(binary, ["scan", ".", "--format", "json", "--mode", "enforce", "--fail-on", "high"]);
} finally {
  rmSync(tempDir, { recursive: true, force: true });
}

process.stdout.write("security checks valid\n");

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
