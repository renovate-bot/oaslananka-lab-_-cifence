#!/usr/bin/env node
import { execFileSync } from "node:child_process";
import { existsSync, readFileSync } from "node:fs";

const packageJson = JSON.parse(readFileSync("package.json", "utf8"));
const manifest = JSON.parse(readFileSync(".release-please-manifest.json", "utf8"));
const sourceRepo = process.env.CIFENCE_SOURCE_REPO || "oaslananka/cifence";
const ciRepo = process.env.CIFENCE_CI_REPO || "oaslananka-lab/cifence";
const tags = run("git", ["tag", "--list", "v*"]).split("\n").filter(Boolean).sort();
const sourceReleases = listReleases(sourceRepo);
const ciReleases = listReleases(ciRepo);
const sourceReleasePrs = listReleasePrs(sourceRepo);
const ciReleasePrs = listReleasePrs(ciRepo);

const blockers = [];
if (packageJson.version !== manifest["."]) {
  blockers.push("package.json version does not match release-please manifest.");
}
if (!existsSync("CHANGELOG.md")) {
  blockers.push("CHANGELOG.md is missing.");
}

const state = {
  package_version: packageJson.version,
  release_please_manifest_version: manifest["."],
  source_repository: sourceRepo,
  ci_repository: ciRepo,
  changelog_present: existsSync("CHANGELOG.md"),
  tags,
  source_releases_accessible: sourceReleases.accessible,
  ci_releases_accessible: ciReleases.accessible,
  source_releases: sourceReleases.items,
  ci_releases: ciReleases.items,
  source_open_release_prs: sourceReleasePrs,
  ci_open_release_prs: ciReleasePrs,
  blockers,
  next_safe_command:
    blockers.length === 0
      ? "Run local validation and let release-please manage the next release PR from main."
      : "Fix blockers before any release automation is trusted.",
  safe_to_publish: false,
};

process.stdout.write(`${JSON.stringify(state, null, 2)}\n`);

function run(command, args, allowFailure = false) {
  try {
    return execFileSync(command, args, {
      encoding: "utf8",
      stdio: ["ignore", "pipe", "ignore"],
    }).trim();
  } catch (error) {
    if (allowFailure) {
      return "";
    }
    throw error;
  }
}

function listReleases(repo) {
  const result = runStatus("gh", ["release", "list", "--repo", repo, "--limit", "20"]);
  const output = result.output;
  return {
    accessible: result.ok,
    items: output.split("\n").filter(Boolean),
  };
}

function listReleasePrs(repo) {
  const output = run(
    "gh",
    [
      "pr",
      "list",
      "--repo",
      repo,
      "--search",
      "release-please in:title state:open",
      "--json",
      "number,title,url",
    ],
    true,
  );
  return output ? JSON.parse(output) : [];
}

function runStatus(command, args) {
  try {
    return {
      ok: true,
      output: execFileSync(command, args, {
        encoding: "utf8",
        stdio: ["ignore", "pipe", "ignore"],
      }).trim(),
    };
  } catch {
    return { ok: false, output: "" };
  }
}
