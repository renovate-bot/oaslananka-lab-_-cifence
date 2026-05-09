#!/usr/bin/env node
import { spawnSync } from "node:child_process";
import {
  existsSync,
  mkdirSync,
  mkdtempSync,
  readFileSync,
  rmSync,
  statSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const requiredFiles = [
  "action.yml",
  "dist/index.js",
  "dist/bin/linux-x64/cifence",
  "dist/bin/linux-arm64/cifence",
  "dist/bin/darwin-x64/cifence",
  "dist/bin/darwin-arm64/cifence",
  "dist/bin/win32-x64/cifence.exe",
];

run("node", ["scripts/validate-action-metadata.mjs"], { cwd: root });

for (const file of requiredFiles) {
  const absolute = join(root, file);
  if (!existsSync(absolute)) {
    fail(`${file} is missing`);
  }
  if (statSync(absolute).size <= 0) {
    fail(`${file} is empty`);
  }
}

const tempRoot = mkdtempSync(join(tmpdir(), "cifence-action-smoke-"));
try {
  const noGoPath = join(tempRoot, "no-go-bin");
  mkdirSync(noGoPath, { recursive: true });

  runActionSmoke({
    name: "with GITHUB_ACTION_PATH",
    workspace: join(tempRoot, "with-action-path"),
    actionPath: root,
    noGoPath,
  });

  runActionSmoke({
    name: "without GITHUB_ACTION_PATH",
    workspace: join(tempRoot, "without-action-path"),
    actionPath: "",
    noGoPath,
  });
} finally {
  rmSync(tempRoot, { recursive: true, force: true });
}

process.stdout.write("action smoke valid\n");

function runActionSmoke({ name, workspace, actionPath, noGoPath }) {
  const workflowDir = join(workspace, ".github", "workflows");
  mkdirSync(workflowDir, { recursive: true });
  writeFileSync(
    join(workflowDir, "ci.yml"),
    [
      "name: Consumer",
      "on: push",
      "jobs:",
      "  scan:",
      "    runs-on: ubuntu-24.04",
      "    steps:",
      "      - uses: actions/checkout@v6",
      "      - uses: ./.github/actions/foo",
      "      - uses: ./actions/foo",
      "",
    ].join("\n"),
  );

  const outputFile = join(workspace, "github-output.txt");
  const summaryFile = join(workspace, "github-summary.md");
  writeFileSync(outputFile, "");
  writeFileSync(summaryFile, "");
  const env = {
    ...process.env,
    GITHUB_WORKSPACE: workspace,
    GITHUB_OUTPUT: outputFile,
    GITHUB_STEP_SUMMARY: summaryFile,
    GITHUB_REPOSITORY: "oaslananka-lab/test",
    GITHUB_SHA: "0000000000000000000000000000000000000000",
    GITHUB_REF: "refs/heads/main",
    INPUT_PATH: ".",
    INPUT_MODE: "warn",
    "INPUT_FAIL-ON": "high",
    "INPUT_ALLOW-OUTSIDE-WORKSPACE": "false",
    INPUT_SARIF: "true",
    INPUT_JSON: "true",
    INPUT_MARKDOWN: "true",
    "INPUT_UPLOAD-SARIF": "false",
    PATH: noGoPath,
    Path: noGoPath,
  };
  if (actionPath) {
    env.GITHUB_ACTION_PATH = actionPath;
  } else {
    delete env.GITHUB_ACTION_PATH;
  }

  const result = spawnSync(process.execPath, [join(root, "dist", "index.js")], {
    cwd: workspace,
    env,
    encoding: "utf8",
    stdio: ["ignore", "pipe", "pipe"],
  });

  if (result.status !== 0) {
    fail(
      [`${name} failed with exit ${result.status}`, result.stderr.trim(), result.stdout.trim()]
        .filter(Boolean)
        .join("\n"),
    );
  }

  const resultDir = join(workspace, "cifence-results");
  for (const report of ["cifence.json", "cifence.sarif", "cifence.md"]) {
    const reportPath = join(resultDir, report);
    if (!existsSync(reportPath) || statSync(reportPath).size <= 0) {
      fail(`${name} did not write ${report}`);
    }
  }

  const output = readFileSync(outputFile, "utf8");
  for (const outputName of [
    "findings",
    "critical",
    "high",
    "medium",
    "low",
    "sarif-path",
    "json-path",
  ]) {
    if (!output.includes(outputName)) {
      fail(`${name} did not set ${outputName}`);
    }
  }
}

function run(command, args, options) {
  const result = spawnSync(command, args, {
    ...options,
    encoding: "utf8",
    stdio: ["ignore", "pipe", "pipe"],
  });
  if (result.status !== 0) {
    fail([`${command} ${args.join(" ")} failed`, result.stderr, result.stdout].join("\n"));
  }
}

function fail(message) {
  process.stderr.write(`${message}\n`);
  process.exit(1);
}
