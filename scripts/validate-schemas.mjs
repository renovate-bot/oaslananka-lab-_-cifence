#!/usr/bin/env node
import { execFileSync } from "node:child_process";
import { mkdtempSync, readFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import Ajv2020 from "ajv/dist/2020.js";
import addFormats from "ajv-formats";

const ajv = new Ajv2020({ allErrors: true, strict: true });
addFormats(ajv);

const reportSchema = loadJSON("schemas/report.schema.json");
const configSchema = loadJSON("schemas/config.schema.json");
const baselineSchema = loadJSON("schemas/baseline.schema.json");

const validateReport = ajv.compile(reportSchema);
const validateConfig = ajv.compile(configSchema);
const validateBaseline = ajv.compile(baselineSchema);

assertValid(
  validateReport,
  JSON.parse(
    execFileSync(
      "go",
      [
        "run",
        "./cmd/cifence",
        "scan",
        "tests/fixtures/workflows/mutable-action-ref.yml",
        "--format",
        "json",
      ],
      { encoding: "utf8" },
    ),
  ),
  "report",
);

assertValid(
  validateConfig,
  {
    version: 1,
    severity: { fail_on: "high" },
    rules: {
      "CF-ACT-001": {
        enabled: true,
        severity: "medium",
        allow: ["actions/checkout@0123456789abcdef0123456789abcdef01234567"],
      },
    },
    paths: {
      include: [".github/workflows/*.yml", ".github/workflows/*.yaml"],
      exclude: [".github/workflows/generated-*.yml"],
    },
    suppressions: [
      {
        rule: "CF-ACT-001",
        path: ".github/workflows/legacy.yml",
        yaml_path: "jobs.scan.steps[0].uses",
        evidence: "vendor/action@v1",
        reason: "Vendor action has no immutable release yet",
        expires: "2026-07-01",
      },
    ],
  },
  "config",
);

const tempRoot = mkdtempSync(join(tmpdir(), "cifence-schema-"));
try {
  const baselinePath = join(tempRoot, "cifence.baseline.json");
  execFileSync(
    "go",
    [
      "run",
      "./cmd/cifence",
      "scan",
      "tests/fixtures/workflows/missing-permissions.yml",
      "--baseline",
      baselinePath,
      "--update-baseline",
      "--format",
      "json",
    ],
    { encoding: "utf8", stdio: ["ignore", "pipe", "pipe"] },
  );
  assertValid(validateBaseline, loadJSON(baselinePath), "baseline");
} finally {
  rmSync(tempRoot, { recursive: true, force: true });
}

process.stdout.write("schemas valid\n");

function loadJSON(path) {
  return JSON.parse(readFileSync(path, "utf8"));
}

function assertValid(validate, data, name) {
  if (!validate(data)) {
    const detail = ajv.errorsText(validate.errors, { separator: "\n" });
    throw new Error(`${name} schema validation failed:\n${detail}`);
  }
}
