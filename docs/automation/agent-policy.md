# Automation Policy

Repository automation must preserve the dual-repository model, avoid broad credentials, and keep generated operational text out of commits.

Repository roles:

- `oaslananka/cifence` is the personal source repository.
- `oaslananka-lab/cifence` carries the same git content and runs CI/CD workflows.
- Workflows are guarded so CI/CD authority executes only in the organization repository.
- Tags and releases should be kept aligned between both repositories.

Allowed automation behavior:

- inspect repository state
- run local validation
- create a draft pull request for review
- report CI and review-thread status
- verify personal and organization repository state

Forbidden automation behavior:

- auto-merge
- auto-approve
- force-push
- publish packages
- create production releases outside release-please
- leave the personal and organization repositories intentionally divergent
