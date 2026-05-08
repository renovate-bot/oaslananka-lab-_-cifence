#!/usr/bin/env node
import { execFileSync, spawnSync } from "node:child_process";
import { writeFileSync } from "node:fs";

const sourceRepo = process.env.CIFENCE_SOURCE_REPO || "oaslananka/cifence";
const targetRepo = process.env.CIFENCE_CI_REPO || "oaslananka-lab/cifence";
const sourceUrl = `https://github.com/${sourceRepo}.git`;
const targetUrl = `https://github.com/${targetRepo}.git`;
const args = new Set(process.argv.slice(2));
const checkOnly = args.has("--check") || !args.has("--apply");
const apply = args.has("--apply");
const force = args.has("--force");

const sourceRefs = listRemoteRefs(sourceUrl);
const targetRefs = listRemoteRefs(targetUrl);
const gitPlan = buildGitPlan(sourceRefs, targetRefs);
const metadata = buildMetadataDiff(sourceRepo, targetRepo);

const plan = {
  source_repository: sourceRepo,
  target_repository: targetRepo,
  mode: apply ? "apply" : "check",
  force_enabled: force,
  git: gitPlan,
  metadata,
  safe_to_publish: false,
  creates_release: false,
  creates_pull_request: false,
  creates_issue: false,
  next_safe_command: apply
    ? "Review this plan, then rerun with --apply only from the CI mirror with appropriate repository credentials."
    : "Use --apply to fast-forward missing or outdated Git refs after reviewing the plan.",
};

process.stdout.write(`${JSON.stringify(plan, null, 2)}\n`);
writeFileSync("repository-sync-plan.json", `${JSON.stringify(plan, null, 2)}\n`, { mode: 0o600 });

if (checkOnly) {
  process.exit(0);
}

if (gitPlan.divergent_refs.length > 0 && !force) {
  process.stderr.write(
    "Ref divergence requires explicit --force. No refs were updated in this run.\n",
  );
  process.exit(1);
}

applyGitSync(gitPlan, force);

function listRemoteRefs(url) {
  const output = run("git", ["ls-remote", "--heads", "--tags", url]);
  const refs = new Map();
  for (const line of output.split("\n").filter(Boolean)) {
    const [sha, ref] = line.trim().split(/\s+/);
    if (!ref || ref.endsWith("^{}")) {
      continue;
    }
    if (ref.startsWith("refs/heads/") || ref.startsWith("refs/tags/")) {
      refs.set(ref, sha);
    }
  }
  return refs;
}

function buildGitPlan(source, target) {
  const missingRefs = [];
  const divergentRefs = [];
  const matchingRefs = [];
  const extraTargetRefs = [];

  for (const [ref, sourceSha] of source.entries()) {
    const targetSha = target.get(ref);
    if (!targetSha) {
      missingRefs.push({ ref, source_sha: sourceSha });
    } else if (targetSha !== sourceSha) {
      divergentRefs.push({ ref, source_sha: sourceSha, target_sha: targetSha });
    } else {
      matchingRefs.push({ ref, sha: sourceSha });
    }
  }

  for (const [ref, targetSha] of target.entries()) {
    if (!source.has(ref)) {
      extraTargetRefs.push({ ref, target_sha: targetSha });
    }
  }

  return {
    source_ref_count: source.size,
    target_ref_count: target.size,
    synced: missingRefs.length === 0 && divergentRefs.length === 0 && extraTargetRefs.length === 0,
    missing_refs: missingRefs.sort(byRef),
    divergent_refs: divergentRefs.sort(byRef),
    extra_target_refs: extraTargetRefs.sort(byRef),
    matching_refs: matchingRefs.sort(byRef),
    deletes_unknown_refs: false,
  };
}

function applyGitSync(gitPlan, forceSync) {
  ensureLocalRemote("cifence-source", sourceUrl);
  runInherited("git", [
    "fetch",
    "--prune",
    "cifence-source",
    "+refs/heads/*:refs/remotes/cifence-source/*",
  ]);
  runInherited("git", ["fetch", "--tags", "cifence-source"]);

  for (const item of gitPlan.missing_refs.concat(gitPlan.divergent_refs)) {
    if (item.ref.startsWith("refs/heads/")) {
      const branch = item.ref.slice("refs/heads/".length);
      const sourceRef = `refs/remotes/cifence-source/${branch}`;
      const targetRef = `refs/heads/${branch}`;
      const refspec = `${forceSync ? "+" : ""}${sourceRef}:${targetRef}`;
      runInherited("git", ["push", "origin", refspec]);
    } else if (item.ref.startsWith("refs/tags/")) {
      const refspec = `${forceSync ? "+" : ""}${item.ref}:${item.ref}`;
      runInherited("git", ["push", "origin", refspec]);
    }
  }
}

function ensureLocalRemote(name, url) {
  const remotes = run("git", ["remote"], true).split("\n").filter(Boolean);
  if (!remotes.includes(name)) {
    runInherited("git", ["remote", "add", name, url]);
    return;
  }
  runInherited("git", ["remote", "set-url", name, url]);
}

function buildMetadataDiff(source, target) {
  if (!commandAvailable("gh")) {
    return {
      accessible: false,
      mode: "read-only diff",
      blocker: "GitHub CLI is unavailable. Metadata sync requires GitHub API access.",
      required_secret: "PERSONAL_REPO_SYNC_TOKEN",
    };
  }

  const sections = {
    releases: compareLists(listGh("release", source), listGh("release", target), "tagName"),
    issues: compareLists(listGh("issue", source), listGh("issue", target), "number"),
    pull_requests: compareLists(
      listGh("pull-request", source),
      listGh("pull-request", target),
      "number",
    ),
    labels: compareLists(listGh("label", source), listGh("label", target), "name"),
    milestones: compareLists(listGh("milestone", source), listGh("milestone", target), "title"),
  };

  return {
    accessible: true,
    mode: "read-only diff",
    sections,
    write_sync_enabled: false,
    blocker:
      "Conservative metadata write sync requires PERSONAL_REPO_SYNC_TOKEN and explicit operator approval to avoid duplicate issues, PRs, comments, releases, labels, or milestones.",
    required_secret: "PERSONAL_REPO_SYNC_TOKEN",
  };
}

function listGh(kind, repo) {
  if (kind === "release") {
    return parseJson(
      run(
        "gh",
        [
          "release",
          "list",
          "--repo",
          repo,
          "--limit",
          "100",
          "--json",
          "tagName,name,isDraft,isPrerelease",
        ],
        true,
      ),
    );
  }
  if (kind === "issue") {
    return parseJson(
      run(
        "gh",
        [
          "issue",
          "list",
          "--repo",
          repo,
          "--state",
          "all",
          "--limit",
          "100",
          "--json",
          "number,title,state",
        ],
        true,
      ),
    );
  }
  if (kind === "pull-request") {
    return parseJson(
      run(
        "gh",
        [
          "pr",
          "list",
          "--repo",
          repo,
          "--state",
          "all",
          "--limit",
          "100",
          "--json",
          "number,title,state,headRefName,baseRefName",
        ],
        true,
      ),
    );
  }
  if (kind === "label") {
    return parseJson(
      run(
        "gh",
        ["label", "list", "--repo", repo, "--limit", "100", "--json", "name,color,description"],
        true,
      ),
    );
  }
  if (kind === "milestone") {
    return parseJson(
      run(
        "gh",
        ["api", "--method", "GET", `repos/${repo}/milestones`, "--paginate", "-f", "state=all"],
        true,
      ),
    );
  }
  return [];
}

function compareLists(source, target, key) {
  const sourceKeys = new Set(source.map((item) => String(item[key] ?? "")));
  const targetKeys = new Set(target.map((item) => String(item[key] ?? "")));
  return {
    source_count: source.length,
    target_count: target.length,
    missing_in_target: [...sourceKeys].filter((value) => value && !targetKeys.has(value)).sort(),
    extra_in_target: [...targetKeys].filter((value) => value && !sourceKeys.has(value)).sort(),
  };
}

function parseJson(value) {
  if (!value.trim()) {
    return [];
  }
  try {
    return JSON.parse(value);
  } catch {
    return [];
  }
}

function commandAvailable(command) {
  const result = spawnSync(command, ["--version"], { stdio: "ignore" });
  return !result.error && result.status === 0;
}

function run(command, args, allowFailure = false) {
  try {
    return execFileSync(command, args, {
      encoding: "utf8",
      stdio: ["ignore", "pipe", "pipe"],
    }).trim();
  } catch (error) {
    if (allowFailure) {
      return "";
    }
    throw error;
  }
}

function runInherited(command, args) {
  execFileSync(command, args, { stdio: "inherit" });
}

function byRef(left, right) {
  return left.ref.localeCompare(right.ref);
}
