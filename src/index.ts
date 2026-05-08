import * as core from "@actions/core";
import * as exec from "@actions/exec";
import * as github from "@actions/github";
import { gzipSync } from "node:zlib";
import { access, mkdir, readFile, writeFile } from "node:fs/promises";
import { join, resolve } from "node:path";

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
  const targetPath = resolve(
    process.env.GITHUB_WORKSPACE || process.cwd(),
    core.getInput("path") || ".",
  );
  const mode = normalizeMode(core.getInput("mode") || "warn");
  const writeSarif = getBooleanInput("sarif");
  const writeJson = getBooleanInput("json");
  const writeMarkdown = getBooleanInput("markdown");
  const uploadSarif = getBooleanInput("upload-sarif");
  const token = process.env.GITHUB_TOKEN || "";

  const resultDirectory = resolve(process.cwd(), outputDirectory);
  await mkdir(resultDirectory, { recursive: true });

  const jsonPath = join(resultDirectory, "cifence.json");
  const sarifPath = join(resultDirectory, "cifence.sarif");
  const markdownPath = join(resultDirectory, "cifence.md");

  const actionPath = process.env.GITHUB_ACTION_PATH || process.cwd();
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
  const bundledBinary = join(
    actionPath,
    "dist",
    "bin",
    `${process.platform}-${process.arch}`,
    binaryName,
  );
  if (await fileExists(bundledBinary)) {
    return { command: bundledBinary, args: [] };
  }

  const builtBinary = join(resultDirectory, binaryName);
  const build = await exec.getExecOutput(
    "go",
    ["build", "-trimpath", "-o", builtBinary, "./cmd/cifence"],
    {
      cwd: actionPath,
      ignoreReturnCode: true,
      silent: true,
    },
  );
  if (build.exitCode !== 0) {
    throw new Error("CIFence CLI build failed. Ensure Go is available on the runner.");
  }
  return { command: builtBinary, args: [] };
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
