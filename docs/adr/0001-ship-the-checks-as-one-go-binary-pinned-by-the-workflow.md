# 1. Ship the checks as one Go binary pinned by the workflow

Date: 2026-11-20

## Status

Accepted

## Context

A notes repository needs schema, link, size, personal-data, hidden-content and governance checks, run the same way in CI and on a laptop. Shell scripts spread over a workflow are hard to test and drift between the two places. A caller pins strata to a commit, and the checks it runs must come from exactly that commit. `go run` flattens the exit code of the program it runs, so a warning and an error look alike to the workflow.

## Decision

All checks live in one Go command, `strata`, with subcommands for the tree, the diff, the push audit, calibration and skill sync. The reusable workflows fetch strata at the commit in `job.workflow_sha` into the runner's temporary directory, outside the checked-out repository, build it with `go install`, and run the installed binary; a pull request merge commit cannot be fetched as a Go module version. Schemas and skills are embedded in the binary.

## Consequences

- The version a notes repository pins is the version that checks it, schemas included.
- Every check has Go unit tests, and `just check` runs the same code as CI.
- Each CI run compiles strata once before checking.
