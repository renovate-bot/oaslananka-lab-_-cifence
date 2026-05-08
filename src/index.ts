import * as core from "@actions/core";
import * as exec from "@actions/exec";
import * as github from "@actions/github";
import { gzipSync } from "node:zlib";
import { access, chmod, mkdir, readFile, writeFile } from "node:fs/promises";
import * as path from "node:path";

type CIFenceReport = {
  summary: {
    findings: number;
    critical: number;
    high: number;
    medium: number;
    low: number;
  };
};

const outputDirectory = "cifence-results";

async function main(): Promise<void> {
  const targetPath = path.resolve(
    process.env.GITHUB_WORKSPACE || process.cwd(),
    core.getInput("path") || ".",
  );
  const mode = normalizeMode(core.getInput("mode") || "warn");
  const writeSarif = getBooleanInput("sarif");
  const writeJson = getBooleanInput("json");
  const writeMarkdown = getBooleanInput("markdown");
  const uploadSarif = getBooleanInput("upload-sarif");
  const token = process.env.GITHUB_TOKEN || "";

  const resultDirectory = path.resolve(process.cwd(), outputDirectory);
  await mkdir(resultDirectory, { recursive: true });

  const jsonPath = path.join(resultDirectory, "cifence.json");
  const sarifPath = path.join(resultDirectory, "cifence.sarif");
  const markdownPath = path.join(resultDirectory, "cifence.md");

  const actionPath = resolveActionRoot();
  const cli = await resolveCIFenceCLI(actionPath, resultDirectory);
  const args = ["scan", "--path", targetPath, "--mode", mode];
  if (writeJson) {
    args.push("--json", jsonPath);
  }
  if (writeSarif) {
    args.push("--sarif", sarifPath);
  }
  if (writeMarkdown) {
    args.push("--markdown", markdownPath);
  }
  args.push("--format", "json");

  const result = await exec.getExecOutput(cli.command, [...cli.args, ...args], {
    cwd: actionPath,
    ignoreReturnCode: true,
    silent: true,
  });

  let report: CIFenceReport;
  if (writeJson) {
    report = JSON.parse(await readRequiredReport(jsonPath, result)) as CIFenceReport;
  } else {
    if (result.exitCode !== 0 && !result.stdout.trim()) {
      throw new Error(
        result.stderr.trim() || `CIFence completed with exit code ${result.exitCode}.`,
      );
    }
    report = JSON.parse(result.stdout) as CIFenceReport;
    await writeFile(jsonPath, `${JSON.stringify(report, null, 2)}\n`, { mode: 0o600 });
  }

  if (writeMarkdown && process.env.GITHUB_STEP_SUMMARY) {
    await core.summary.addRaw(await readFile(markdownPath, "utf8")).write();
  }

  if (uploadSarif) {
    if (!writeSarif) {
      throw new Error("upload-sarif=true requires sarif=true.");
    }
    if (!token) {
      throw new Error("upload-sarif=true requires GITHUB_TOKEN in the environment.");
    }
    await uploadSarifReport(token, sarifPath);
  }

  core.setOutput("findings", String(report.summary.findings));
  core.setOutput("critical", String(report.summary.critical));
  core.setOutput("high", String(report.summary.high));
  core.setOutput("medium", String(report.summary.medium));
  core.setOutput("low", String(report.summary.low));
  core.setOutput("sarif-path", writeSarif ? sarifPath : "");
  core.setOutput("json-path", jsonPath);
  core.setOutput("markdown-path", writeMarkdown ? markdownPath : "");

  if (result.exitCode !== 0) {
    core.setFailed(`CIFence completed with exit code ${result.exitCode}.`);
  }
}

async function readRequiredReport(
  path: string,
  result: { exitCode: number; stderr: string },
): Promise<string> {
  try {
    return await readFile(path, "utf8");
  } catch (error) {
    if (result.exitCode !== 0) {
      throw new Error(
        result.stderr.trim() || `CIFence completed with exit code ${result.exitCode}.`,
      );
    }
    throw error;
  }
}

async function resolveCIFenceCLI(
  actionPath: string,
  resultDirectory: string,
): Promise<{ command: string; args: string[] }> {
  const binaryName = process.platform === "win32" ? "cifence.exe" : "cifence";
  const bundledBinary = path.join(
    actionPath,
    "dist",
    "bin",
    `${process.platform}-${process.arch}`,
    binaryName,
  );
  if (await fileExists(bundledBinary)) {
    if (process.platform !== "win32") {
      await chmod(bundledBinary, 0o755);
    }
    return { command: bundledBinary, args: [] };
  }

  const builtBinary = path.join(resultDirectory, binaryName);
  const buildArgs = ["build", "-trimpath", "-o", builtBinary, "./cmd/cifence"];
  const build = await runBuildFallback("go", buildArgs, {
    cwd: actionPath,
    ignoreReturnCode: true,
    silent: true,
  });
  if (build.exitCode !== 0) {
    throw new Error(buildFailureDetails(actionPath, bundledBinary, builtBinary, build));
  }
  return { command: builtBinary, args: [] };
}

function resolveActionRoot(): string {
  const envPath = process.env.GITHUB_ACTION_PATH;
  if (envPath && envPath.trim() !== "") {
    return envPath;
  }

  if (__dirname.endsWith(`${path.sep}dist`)) {
    return path.dirname(__dirname);
  }

  return __dirname;
}

async function runBuildFallback(
  command: string,
  args: string[],
  options: exec.ExecOptions,
): Promise<{ exitCode: number; stdout: string; stderr: string }> {
  try {
    return await exec.getExecOutput(command, args, options);
  } catch (error) {
    return {
      exitCode: 127,
      stdout: "",
      stderr: error instanceof Error ? error.message : String(error),
    };
  }
}

function buildFailureDetails(
  actionPath: string,
  bundledBinary: string,
  builtBinary: string,
  build: { exitCode: number; stdout: string; stderr: string },
): string {
  const details = [
    "CIFence CLI build failed.",
    `actionPath: ${actionPath}`,
    `bundledBinary: ${bundledBinary}`,
    `builtBinary: ${builtBinary}`,
    "command: go build -trimpath -o <builtBinary> ./cmd/cifence",
    `buildExitCode: ${build.exitCode}`,
    process.env.PATH ? `PATH: ${process.env.PATH}` : "PATH: <empty>",
    build.stderr.trim() ? `stderr:\n${build.stderr.trim()}` : "",
    build.stdout.trim() ? `stdout:\n${build.stdout.trim()}` : "",
  ]
    .filter(Boolean)
    .join("\n");
  return redactSecrets(details);
}

function redactSecrets(value: string): string {
  return value
    .replace(/\bgh[pousr]_[A-Za-z0-9_]{20,}\b/g, "[REDACTED]")
    .replace(/\bgithub_pat_[A-Za-z0-9_]+\b/g, "[REDACTED]")
    .replace(/\b(Bearer\s+)[A-Za-z0-9._~+/=-]+\b/gi, "$1[REDACTED]")
    .replace(/\b(token|authorization|password|secret)=\S+/gi, "$1=[REDACTED]");
}

async function fileExists(path: string): Promise<boolean> {
  try {
    await access(path);
    return true;
  } catch {
    return false;
  }
}

function normalizeMode(value: string): "warn" | "enforce" {
  if (value === "warn" || value === "enforce") {
    return value;
  }
  throw new Error(`Invalid mode: ${value}`);
}

function getBooleanInput(name: string): boolean {
  const value = (core.getInput(name) || "false").toLowerCase();
  if (value === "true") {
    return true;
  }
  if (value === "false") {
    return false;
  }
  throw new Error(`Invalid boolean input ${name}: ${value}`);
}

async function uploadSarifReport(token: string, sarifPath: string): Promise<void> {
  const octokit = github.getOctokit(token);
  const sarif = await readFile(sarifPath);
  const compressed = gzipSync(sarif).toString("base64");
  const context = github.context;
  await octokit.rest.codeScanning.uploadSarif({
    owner: context.repo.owner,
    repo: context.repo.repo,
    commit_sha: context.sha,
    ref: context.ref,
    sarif: compressed,
    checkout_uri: `git+https://github.com/${context.repo.owner}/${context.repo.repo}`,
    tool_name: "CIFence",
  });
}

main().catch((error: unknown) => {
  core.setFailed(error instanceof Error ? error.message : String(error));
});
