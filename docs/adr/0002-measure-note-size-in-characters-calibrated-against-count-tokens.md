# 2. Measure note size in characters calibrated against count_tokens

Date: 2026-11-20

## Status

Accepted

## Context

Notes are read by Claude, so their useful size is a token budget. There is no offline Claude tokenizer, token counts differ between models, and Anthropic's count_tokens endpoint needs an API key that a CI run on a pull request should not hold. Characters can be counted anywhere without a key.

## Decision

The size caps in `strata.yml` are character counts. `strata calibrate` sends sample notes to count_tokens and proposes caps for 300, 1500 and 3000 tokens from the measured characters-per-token ratio. The first run, on 2026-11-20 over 64 notes with `claude-opus-5-5`, measured 2.31 characters per token and set the defaults to 700, 3500 and 6900 characters.

## Consequences

- The checks run offline and need no secret.
- The caps are only as accurate as the sample they were measured on.
- A new model or a different mix of notes needs a new calibration run.
