#!/usr/bin/env node
import { spawnSync } from "node:child_process";
import { chmodSync, mkdirSync, readFileSync, statSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const packageJson = JSON.parse(readFileSync(join(root, "package.json"), "utf8"));

const targets = [
  { name: "linux-x64", goos: "linux", goarch: "amd64", binary: "cifence" },
  { name: "linux-arm64", goos: "linux", goarch: "arm64", binary: "cifence" },
  { name: "darwin-x64", goos: "darwin", goarch: "amd64", binary: "cifence" },
  { name: "darwin-arm64", goos: "darwin", goarch: "arm64", binary: "cifence" },
  { name: "win32-x64", goos: "windows", goarch: "amd64", binary: "cifence.exe" },
];

for (const target of targets) {
  const outputDir = join(root, "dist", "bin", target.name);
  const output = join(outputDir, target.binary);
  mkdirSync(outputDir, { recursive: true });

  const result = spawnSync(
    "go",
    [
      "build",
      "-trimpath",
      `-ldflags=-s -w -X github.com/oaslananka/cifence/internal/analyzer.Version=${packageJson.version}`,
      "-o",
      output,
      "./cmd/cifence",
    ],
    {
      cwd: root,
      env: {
        ...process.env,
        CGO_ENABLED: "0",
        GOOS: target.goos,
        GOARCH: target.goarch,
      },
      encoding: "utf8",
      stdio: ["ignore", "pipe", "pipe"],
    },
  );

  if (result.status !== 0) {
    process.stderr.write(`failed to build ${target.name}\n`);
    if (result.stderr) {
      process.stderr.write(result.stderr);
    }
    if (result.stdout) {
      process.stderr.write(result.stdout);
    }
    process.exit(result.status ?? 1);
  }

  if (target.goos !== "windows") {
    chmodSync(output, 0o755);
  }

  const size = statSync(output).size;
  if (size <= 0) {
    process.stderr.write(`${output} is empty\n`);
    process.exit(1);
  }

  process.stdout.write(`${target.name} ${size} bytes\n`);
}

const hostTarget = targets.find((target) => target.name === `${process.platform}-${process.arch}`);

if (hostTarget) {
  const hostBinary = join(root, "dist", "bin", hostTarget.name, hostTarget.binary);
  const version = runHostBinary(hostBinary, ["version"]).trim();
  if (version !== packageJson.version) {
    process.stderr.write(
      `${hostBinary} version mismatch: expected ${packageJson.version}, got ${version}\n`,
    );
    process.exit(1);
  }

  const rules = runHostBinary(hostBinary, ["rules"]);
  if (!rules.includes("CF-ACT-001") || !rules.includes("CF-TRG-001")) {
    process.stderr.write(`${hostBinary} rules output did not include expected rule IDs\n`);
    process.exit(1);
  }
}

function runHostBinary(command, args) {
  const result = spawnSync(command, args, {
    cwd: root,
    encoding: "utf8",
    stdio: ["ignore", "pipe", "pipe"],
  });
  if (result.status !== 0) {
    process.stderr.write(`${command} ${args.join(" ")} failed\n`);
    if (result.stderr) {
      process.stderr.write(result.stderr);
    }
    process.exit(result.status ?? 1);
  }
  return result.stdout;
}
