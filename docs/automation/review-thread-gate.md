# Review Thread Gate

`scripts/check-review-threads.mjs` queries GitHub GraphQL `PullRequest.reviewThreads(first: 100)`.

The gate:

- ignores resolved threads
- ignores outdated threads
- blocks unresolved human review threads
- blocks actionable automated review threads
- does not resolve threads
- does not approve or merge
- writes `review-thread-summary.json`
- writes a Markdown job summary in GitHub Actions
