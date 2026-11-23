---
name: capture
description: Puts raw material such as a pasted lecture summary, article excerpt or original document into a notes repository as a pull request. Use when the owner asks to save, file or capture text, before any curation into notes.
license: MIT
metadata:
  repo: strata
---

# Capture

Capture stores material in the notes repository without interpreting it.
Turning material into notes is the curate skill.

## Layout

- `inbox/` holds material that still needs curation: summaries, pasted text, excerpts, links with context.
- `raw/` holds originals that must stay unchanged: source documents and verbatim exports.
- `raw/` is add-only; an existing file there is never modified, renamed or deleted.

## Steps

1. Scan the text for personal identifiers before writing anything. Refuse the capture when the text contains an IBAN, a payment card number, a tax ID, a social insurance number or an ID card or passport number. Tell the owner which kind was found and ask for a redacted version. Never write the text to disk, a branch or a PR description.
2. Pick the target. A summary, paste or excerpt goes to `inbox/`. An original that must be preserved as received goes to `raw/`.
3. Name the file `YYYY-MM-DD-short-slug.md` in lowercase ASCII with hyphens. A `raw/` original keeps its extension.
4. For `inbox/`, write one source line at the top (where the text came from and when), then the text. Use Semantic Line Breaks: one sentence per line. Do not summarise, merge or correct the text.
5. Create a branch named `feat/capture-<slug>`. Web sessions push only to non-main branches; never push to main.
6. Commit with a one-line Conventional Commit such as `feat(inbox): capture the lecture summary on tcp congestion control`.
7. Open a pull request with the label `kind/capture`. State in at most five lines what was captured and where.

## Rules

- One capture per pull request unless the items belong together.
- Never touch `notes/`, `archive/` or governance paths (`.github/`, `AGENTS.md` and `CLAUDE.md` at any depth, `CLAUDE.local.md`, `.claude/`, `.mcp.json`, `justfile`, `scripts/`, `strata.yml`) in a capture.
- Run `strata check tree` when the binary is available and fix any finding before opening the pull request.

See scenarios.md for worked cases.
