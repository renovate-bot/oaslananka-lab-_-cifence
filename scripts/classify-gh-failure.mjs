#!/usr/bin/env node
import { readFile, writeFile } from "node:fs/promises";

const inputPath = process.argv[2];
const input = inputPath ? await readFile(inputPath, "utf8") : await readStdin();

const classifiers = [
  ["workflow syntax/actionlint", /actionlint|workflow syntax|invalid workflow/i],
  ["zizmor issue", /zizmor|github actions security/i],
  ["secret scan finding", /gitleaks|secret|private key/i],
  ["CodeQL finding", /codeql|code scanning/i],
  ["dependency audit finding", /osv|vulnerabilit|pnpm audit|dependency review/i],
  ["test failure", /go test|FAIL:|--- FAIL|test failed/i],
  ["typecheck failure", /tsc|typecheck|typescript/i],
  ["lint failure", /lint|gofmt|go vet|prettier/i],
  ["package build failure", /ncc|pnpm pack|package build/i],
  ["action metadata invalid", /action metadata|action\.yml|metadata/i],
  ["SARIF invalid", /sarif/i],
  ["release tag/version mismatch", /tag.*version|version.*tag|manifest.*version/i],
  ["release-please config error", /release-please|release please/i],
  [
    "flaky/infra failure",
    /timed out|rate limit|connection reset|runner.*lost|service unavailable/i,
  ],
];

const match = classifiers.find(([, pattern]) => pattern.test(input));
const result = {
  classification: match?.[0] ?? "unknown",
  confidence: match ? "medium" : "low",
  next_step:
    match?.[0] === "unknown"
      ? "Inspect the failing job log and reproduce the failing command locally."
      : "Inspect the matching log section, fix the exact failing gate, and rerun validation.",
};

await writeFile("failure-classification.json", `${JSON.stringify(result, null, 2)}\n`, {
  mode: 0o600,
});
process.stdout.write(`${JSON.stringify(result, null, 2)}\n`);

async function readStdin() {
  const chunks = [];
  for await (const chunk of process.stdin) {
    chunks.push(chunk);
  }
  return Buffer.concat(chunks).toString("utf8");
}
