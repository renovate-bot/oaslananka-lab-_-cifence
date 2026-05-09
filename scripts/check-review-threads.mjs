#!/usr/bin/env node
import { readFile, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";

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
const threads = await listReviewThreads(owner, name, Number(pullNumber));
for (const thread of threads) {
  thread.comments.nodes = await listThreadComments(thread.id, thread.comments);
}
const botApprovals = await listLatestBotApprovals(owner, name, Number(pullNumber));

const blocking = threads
  .filter((thread) => !thread.isResolved && !thread.isOutdated)
  .map((thread) => blockingThread(thread, botApprovals))
  .filter(Boolean);

await writeSummary({ skipped: false, pull_number: pullNumber, blocking });

if (blocking.length > 0) {
  process.stderr.write(`review thread gate blocked ${blocking.length} unresolved thread(s)\n`);
  process.exit(1);
}

process.stdout.write("review thread gate clean\n");

async function listReviewThreads(owner, name, number) {
  const query = `
query($owner: String!, $name: String!, $number: Int!, $after: String) {
  repository(owner: $owner, name: $name) {
    pullRequest(number: $number) {
      reviewThreads(first: 100, after: $after) {
        pageInfo {
          hasNextPage
          endCursor
        }
        nodes {
          id
          isResolved
          isOutdated
          comments(first: 100) {
            pageInfo {
              hasNextPage
              endCursor
            }
            nodes {
              author {
                login
                __typename
              }
              body
              url
              path
              line
              createdAt
            }
          }
        }
      }
    }
  }
}`;
  const threads = [];
  let after = null;
  for (;;) {
    const payload = await graphql(query, { owner, name, number, after });
    const connection = payload.data.repository.pullRequest.reviewThreads;
    threads.push(...connection.nodes);
    if (!connection.pageInfo.hasNextPage) {
      return threads;
    }
    after = connection.pageInfo.endCursor;
  }
}

async function listThreadComments(threadID, initialConnection) {
  const comments = [...initialConnection.nodes];
  if (!initialConnection.pageInfo.hasNextPage) {
    return comments;
  }
  const query = `
query($id: ID!, $after: String) {
  node(id: $id) {
    ... on PullRequestReviewThread {
      comments(first: 100, after: $after) {
        pageInfo {
          hasNextPage
          endCursor
        }
        nodes {
          author {
            login
            __typename
          }
          body
          url
          path
          line
          createdAt
        }
      }
    }
  }
}`;
  let after = initialConnection.pageInfo.endCursor;
  for (;;) {
    const payload = await graphql(query, { id: threadID, after });
    const connection = payload.data.node.comments;
    comments.push(...connection.nodes);
    if (!connection.pageInfo.hasNextPage) {
      return comments;
    }
    after = connection.pageInfo.endCursor;
  }
}

async function graphql(query, variables) {
  const response = await fetch("https://api.github.com/graphql", {
    method: "POST",
    headers: {
      authorization: `Bearer ${token}`,
      "content-type": "application/json",
      "user-agent": "cifence-review-thread-gate",
    },
    body: JSON.stringify({ query, variables }),
  });

  if (!response.ok) {
    throw new Error(`GitHub GraphQL request failed with ${response.status}`);
  }

  const payload = await response.json();
  if (payload.errors?.length) {
    throw new Error(payload.errors.map((error) => error.message).join("; "));
  }
  return payload;
}

async function listLatestBotApprovals(owner, name, number) {
  const query = `
query($owner: String!, $name: String!, $number: Int!, $after: String) {
  repository(owner: $owner, name: $name) {
    pullRequest(number: $number) {
      reviews(first: 100, after: $after) {
        pageInfo {
          hasNextPage
          endCursor
        }
        nodes {
          state
          submittedAt
          author {
            login
            __typename
          }
        }
      }
    }
  }
}`;
  const approvals = new Map();
  let after = null;
  for (;;) {
    const payload = await graphql(query, { owner, name, number, after });
    const connection = payload.data.repository.pullRequest.reviews;
    for (const review of connection.nodes) {
      if (review.state !== "APPROVED" || review.author?.__typename !== "Bot") {
        continue;
      }
      const login = review.author.login;
      const submittedAt = Date.parse(review.submittedAt);
      if (!Number.isFinite(submittedAt)) {
        continue;
      }
      approvals.set(login, Math.max(approvals.get(login) ?? 0, submittedAt));
    }
    if (!connection.pageInfo.hasNextPage) {
      return approvals;
    }
    after = connection.pageInfo.endCursor;
  }
}

function blockingThread(thread, botApprovals) {
  const comments = thread.comments.nodes;
  if (comments.length === 0) {
    return {
      id: thread.id,
      author: "unknown",
      path: "",
      line: null,
      url: "",
      reason: "empty unresolved thread",
    };
  }
  const humanComment = comments.find((comment) => comment.author?.__typename !== "Bot");
  if (humanComment) {
    return summaryItem(thread.id, humanComment, "human unresolved comment");
  }
  const botComment = comments.find((comment) => actionableBotComment(comment.body ?? ""));
  if (botComment) {
    if (hasLaterBotApproval(botComment, botApprovals)) {
      return null;
    }
    return summaryItem(thread.id, botComment, "actionable bot comment");
  }
  return null;
}

function hasLaterBotApproval(comment, botApprovals) {
  const login = comment.author?.login;
  const approvalTime = botApprovals.get(login);
  if (!approvalTime) {
    return false;
  }
  const commentTime = Date.parse(comment.createdAt ?? "");
  return Number.isFinite(commentTime) && approvalTime >= commentTime;
}

function summaryItem(threadID, comment, reason) {
  return {
    id: threadID,
    author: comment.author?.login ?? "unknown",
    path: comment.path ?? "",
    line: comment.line ?? null,
    url: comment.url ?? "",
    reason,
  };
}

function actionableBotComment(body) {
  return /must|should|fix|change|bug|error|fail|required|security|vulnerab|broken|incorrect/i.test(
    body,
  );
}

async function writeSummary(summary) {
  const summaryPath =
    process.env.CIFENCE_REVIEW_THREAD_SUMMARY ||
    join(process.env.RUNNER_TEMP || tmpdir(), "review-thread-summary.json");
  await writeFile(summaryPath, `${JSON.stringify(summary, null, 2)}\n`, {
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
        lines.push(`- ${item.path}:${item.line ?? ""} ${item.url} (${item.reason})`);
      }
    }
    await writeFile(process.env.GITHUB_STEP_SUMMARY, `${lines.join("\n")}\n`, { flag: "a" });
  }
}
