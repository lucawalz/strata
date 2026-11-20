# 6. Run gitleaks as the MIT binary

Date: 2026-11-20

## Status

Accepted

## Context

Secret scanning in the notes repository needs gitleaks, because a private repository on GitHub Pro gets no secret scanning or push protection. The gitleaks GitHub Action is no longer under the MIT licence, while the gitleaks binary is.

## Decision

The check workflow downloads the gitleaks release named by its `gitleaks-version` input, verifies it against the release checksums, and runs it with `--redact` on the pull request range, or on the full history on a push to main.

## Consequences

- No licence key and no third-party action in the check path.
- The gitleaks version is bumped by hand in the workflow input.
