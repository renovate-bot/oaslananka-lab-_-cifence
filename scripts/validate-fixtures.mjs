#!/usr/bin/env node
import { execFileSync } from "node:child_process";
import { readFileSync, readdirSync } from "node:fs";
import { basename, join } from "node:path";

const schemaOnly = process.argv.includes("--schema-only");
const workflowDir = join("tests", "fixtures", "workflows");
const expectedDir = join("tests", "fixtures", "expected");

if (schemaOnly) {
  process.stdout.write("fixture schema available\n");
  process.exit(0);
}

const workflowFiles = readdirSync(workflowDir)
  .filter((file) => file.endsWith(".yml") || file.endsWith(".yaml"))
  .sort();

const failures = [];
for (const file of workflowFiles) {
  const stem = basename(file).replace(/\.(ya?ml)$/, "");
  const actual = execFileSync(
    "go",
    ["run", "./cmd/cifence", "scan", join(workflowDir, file), "--format", "json"],
    { encoding: "utf8" },
  ).trim();
  const expected = readFileSync(join(expectedDir, `${stem}.json`), "utf8").trim();
  if (actual !== expected) {
    failures.push(`${file} did not match expected JSON.`);
  }
}

if (failures.length > 0) {
  process.stderr.write(`${failures.join("\n")}\n`);
  process.exit(1);
}

process.stdout.write("fixtures valid\n");
