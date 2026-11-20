# 4. Report the gate through a dedicated app from pull_request_target

Date: 2026-11-20

## Status

Accepted

## Context

On a `pull_request` trigger the workflow definition comes from the pull request branch. An agent could rewrite the checks, or add a job named `ci-gate` that always passes, and the required check would be satisfied, because any workflow can report a job of that name as `github-actions`.

## Decision

The notes repository runs its checks on `pull_request_target`, so the workflow definition always comes from main; the pull request head is checked out only as data, with a read-only token, and nothing from it is built or run. The result is posted as the check run `strata/gate` by a dedicated GitHub App, gatebot, with Checks read and write and nothing else. Its key lives in an environment named `gate` that only main may deploy to. The ruleset requires `strata/gate` from gatebot's integration id.

## Consequences

- A pull request cannot change the checks that judge it or forge the required check.
- One more GitHub App to register and one more key to keep.
- The design relies on private repositories being exempt from GitHub's 2026-12-02 restriction on `pull_request_target`.
