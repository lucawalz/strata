# 3. Govern agent instructions with owner-signed merge commits

Date: 2026-11-20

## Status

Accepted

## Context

Agents write to the notes repository through pull requests. Some paths steer the agents themselves or run as code: `AGENTS.md`, `CLAUDE.md`, `.claude/`, `.github/`, `justfile`, `scripts/`, `strata.yml`. An agent that changes those paths can change its own rules. A squash merge replaces the author's signature with GitHub's, so the signature on main says nothing about who wrote the change.

## Decision

Every commit that touches a governance path must verify against `.github/allowed_signers`, read from the base branch, never from the pull request. A pull request that touches governance paths touches nothing else and is merged with a merge commit, which keeps the signed commits on main. On every push to main the audit checks the pushed range against `strata.yml` and `allowed_signers` from the last commit already on main, so a push cannot vouch for itself, and opens an issue when a governance change lacks a valid signature.

## Consequences

- Governance changes need the owner's signing key and a separate pull request.
- A governance pull request squashed by mistake is caught after the merge, by the audit, and reverted through a signed pull request.
- A rotated key is honoured by the audit one push later.
- Governance pull requests are updated by a local rebase, never by the GitHub "Update branch" button, which creates an unsigned merge.
