---
name: curate
description: Turns items in inbox/ into well-formed notes in a notes repository by searching for an existing note first, then merging, creating or superseding. Use when the owner asks to curate, process or file inbox items, or to resolve a contradiction between notes.
license: MIT
metadata:
  repo: strata
---

# Curate

Curation turns captured material into notes that stay short, current and findable.
The repository's AGENTS.md is authoritative; this skill summarises the workflow.

## Note format

- Path: `notes/<id>.md`, where `id` equals the file name without the extension and is unique across the repository.
- Frontmatter fields: `id`, `type`, `status`, `summary`, `tags`, `sources`, `refs`, `superseded_by`.
- Tags form a block-style YAML list without spaces inside a tag, at most the number set in `strata.yml`.
- The first heading after the frontmatter is the only H1. No line starting with "# " appears before it.
- Write one sentence per line (Semantic Line Breaks) and give each H2 a self-describing title, because later chunks carry only the document title.
- Link related notes with relative Markdown links in the body, where the relation is explained, never wikilinks. `refs` holds only references to records outside the repository, for example `paperless:1234` or `pr:42`.
- Stay within the size band from `strata.yml`; split a note that outgrows it.
- For German notes, put key terms in both German and English in the summary or tags.

## Steps

1. Search before creating: query the notes by the item's key terms (`qmd` when installed, otherwise a text search over `notes/`).
2. When a matching note exists, merge: integrate the new material into the body where it belongs, rewrite the `summary` so it describes the whole note, and add the item's `raw/` path to `sources`. Never append a dated "Update" paragraph.
3. When nothing matches, create a new note with a fresh `id`, status `active`, a one-sentence summary and relative Markdown links to related notes in the body.
4. When the item contradicts an existing note, decide which one holds. Set the older note to status `superseded`, set its `superseded_by` to the newer id, and move it to `archive/`. Fix links that pointed to it. When the answer is unclear, record the contradiction under an "Open questions" H2 instead.
5. Move every processed inbox item into `raw/` with `git mv`, in the same commit as the note, and list the new `raw/` path in the note's `sources`. Never delete an inbox item, and never modify, rename or delete anything already under `raw/`.
6. Archive instead of deleting notes; `archive/` keeps history.
7. One pull request per topic, labelled `kind/note`, from a branch named `feat/<topic>`. Web sessions push only to non-main branches.
8. Run `strata check tree` before opening the PR and fix every error finding.

## Rules

- Governance changes (`.github/`, `AGENTS.md` and `CLAUDE.md` at any depth, `CLAUDE.local.md`, `.claude/`, `.mcp.json`, `justfile`, `scripts/`, `strata.yml`) go in their own pull request, never mixed with notes.
- Commit with a one-line Conventional Commit of 7 to 12 words, for example `feat(notes): merge tcp congestion lecture into transport note`.
- Never put personal identifiers into a note; stop and ask the owner instead.

See scenarios.md for worked cases.
