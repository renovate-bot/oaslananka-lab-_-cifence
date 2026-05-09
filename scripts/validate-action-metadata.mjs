#!/usr/bin/env node
import { readFile } from "node:fs/promises";
import YAML from "yaml";

const metadata = YAML.parse(await readFile("action.yml", "utf8"));

const requiredInputs = [
  "path",
  "mode",
  "fail-on",
  "allow-outside-workspace",
  "sarif",
  "json",
  "markdown",
  "upload-sarif",
];
const requiredOutputs = [
  "findings",
  "critical",
  "high",
  "medium",
  "low",
  "sarif-path",
  "json-path",
  "markdown-path",
];

const failures = [];
if (metadata?.runs?.using !== "node24") {
  failures.push("action.yml must use node24.");
}
if (metadata?.runs?.main !== "dist/index.js") {
  failures.push("action.yml must point runs.main at dist/index.js.");
}
for (const input of requiredInputs) {
  if (!metadata.inputs?.[input]) {
    failures.push(`missing input: ${input}`);
  }
}
for (const output of requiredOutputs) {
  if (!metadata.outputs?.[output]) {
    failures.push(`missing output: ${output}`);
  }
}
if (!metadata.name || !metadata.description || !metadata.author || !metadata.branding) {
  failures.push("metadata must include name, description, author, and branding.");
}

if (failures.length > 0) {
  process.stderr.write(`${failures.join("\n")}\n`);
  process.exit(1);
}

process.stdout.write("action metadata valid\n");
