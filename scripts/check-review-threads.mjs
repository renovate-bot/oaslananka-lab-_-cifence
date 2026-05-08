#!/usr/bin/env node
import { readFile, writeFile } from "node:fs/promises";

const token = process.env.GITHUB_TOKEN;
const repository = process.env.GITHUB_REPOSITORY;
const eventPath = process.env.GITHUB_EVENT_PATH;

if (!token || !repository || !eventPath) {
  await writeSummary({ skipped: true, reason: "review thread context unavailable", blocking: [] });
  process.stdout.write("review thread gate skipped: context unavailable\n");
  process.exit(0);
}

const event = JSON.parse(await readFile(eventPath, "utf8"));
const pullNumber = event.pull_request?.number ?? event.number;
if (!pullNumber) {
  await writeSummary({ skipped: true, reason: "not a pull request event", blocking: [] });
  process.stdout.write("review thread gate skipped: not a pull request event\n");
  process.exit(0);
}

const [owner, name] = repository.split("/");
const query = `
query($owner: String!, $name: String!, $number: Int!) {
  repository(owner: $owner, name: $name) {
    pullRequest(number: $number) {
      reviewThreads(first: 100) {
        nodes {
          id
          isResolved
          isOutdated
          comments(first: 20) {
            nodes {
              author {
                login
                __typename
              }
              body
              url
              path
              line
            }
          }
        }
      }
    }
  }
}`;

const response = await fetch("https://api.github.com/graphql", {
  method: "POST",
  headers: {
    authorization: `Bearer ${token}`,
    "content-type": "application/json",
    "user-agent": "cifence-review-thread-gate",
  },
  body: JSON.stringify({ query, variables: { owner, name, number: Number(pullNumber) } }),
});

if (!response.ok) {
  throw new Error(`GitHub GraphQL request failed with ${response.status}`);
}

const payload = await response.json();
if (payload.errors?.length) {
  throw new Error(payload.errors.map((error) => error.message).join("; "));
}

const threads = payload.data.repository.pullRequest.reviewThreads.nodes;
const blocking = threads
  .filter((thread) => !thread.isResolved && !thread.isOutdated)
  .filter((thread) => shouldBlockThread(thread))
  .map((thread) => {
    const first = thread.comments.nodes[0];
    return {
      id: thread.id,
      author: first?.author?.login ?? "unknown",
      path: first?.path ?? "",
      line: first?.line ?? null,
      url: first?.url ?? "",
    };
  });

await writeSummary({ skipped: false, pull_number: pullNumber, blocking });

if (blocking.length > 0) {
  process.stderr.write(`review thread gate blocked ${blocking.length} unresolved thread(s)\n`);
  process.exit(1);
}

process.stdout.write("review thread gate clean\n");

function shouldBlockThread(thread) {
  const first = thread.comments.nodes[0];
  if (!first) {
    return true;
  }
  if (first.author?.__typename !== "Bot") {
    return true;
  }
  return /must|should|fix|change|bug|error|fail|required|security|vulnerab|broken|incorrect/i.test(
    first.body ?? "",
  );
}

async function writeSummary(summary) {
  await writeFile("review-thread-summary.json", `${JSON.stringify(summary, null, 2)}\n`, {
    mode: 0o600,
  });
  if (process.env.GITHUB_STEP_SUMMARY) {
    const lines = ["# Review Thread Gate", ""];
    if (summary.skipped) {
      lines.push(`Skipped: ${summary.reason}`);
    } else if (summary.blocking.length === 0) {
      lines.push("No blocking unresolved review threads.");
    } else {
      lines.push(`Blocking unresolved review threads: ${summary.blocking.length}`);
      for (const item of summary.blocking) {
        lines.push(`- ${item.path}:${item.line ?? ""} ${item.url}`);
      }
    }
    await writeFile(process.env.GITHUB_STEP_SUMMARY, `${lines.join("\n")}\n`, { flag: "a" });
  }
}
