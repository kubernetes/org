# Alias Syncer

This tool syncs `OWNERS_ALIASES` from kubernetes/org repository.

Benefits:
- Automated syncing of `OWNERS_ALIASES`
- Group changes propagates within a day and doesn't require humans to update `OWNERS_ALIASES` or OWNERS file for group changes.
- Easier onboarding for new members of Kubernetes Project.

Prerequisites:
- OWNERS file must be using GitHub teams slug name instead of individual GitHub users.
- Current `OWNERS_ALIASES` must be based on GitHub teams instead of manually created aliases.
- Every user currently defined in an `OWNERS_ALIASES` must be a member of the GitHub organization that the repository belongs to.
- Each repo has to be onboarded manually as this tool changes existing behaviour. This tool will reduce the toil around GitHub Operations.


Components:
- ProwJobs that call alias-syncer/alias-syncer.sh for each active GitHub Organization.
- `cmd/aliases` binary that generates `OWNERS_ALIASES`.
