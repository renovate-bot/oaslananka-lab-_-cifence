# Failure Classifier

`scripts/classify-gh-failure.mjs` classifies failing logs into operational buckets:

- workflow syntax/actionlint
- zizmor issue
- secret scan finding
- CodeQL finding
- dependency audit finding
- test failure
- typecheck failure
- lint failure
- package build failure
- action metadata invalid
- SARIF invalid
- release tag/version mismatch
- release-please config error
- flaky/infra failure
- unknown

The script writes `failure-classification.json` and prints the same JSON to stdout.
