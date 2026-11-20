# Curate scenarios

## 1. Inbox item merged into an existing note

Input: `inbox/2026-01-15-tcp-congestion-lecture.md` covers slow start; a search for "tcp congestion" finds `notes/tcp-congestion-control.md`.
Expected: the new material is merged into the matching section; the `summary` is rewritten to describe the whole note; the inbox file is moved with `git mv` to `raw/2026-01-15-tcp-congestion-lecture.md` and that path is added to `sources`; no dated update paragraph is appended; `strata check tree` passes before the pull request.

## 2. New note created when search finds nothing

Input: an inbox item about a topic with no hit in `notes/`.
Expected: a new `notes/<id>.md` with complete frontmatter, one H1 first, one sentence per line, self-describing H2s and relative Markdown links in `refs`; the inbox item is moved into `raw/` with `git mv` and its `raw/` path is listed in `sources`; the pull request is labelled `kind/note`.

## 3. Contradiction resolved by superseding

Input: an inbox item shows that a stored figure in `notes/old-protocol-limits.md` is wrong and a newer note should replace it.
Expected: the old note gets status `superseded` and `superseded_by` pointing to the new id, and is moved to `archive/`; links to it are updated; nothing is deleted; `strata check tree` passes.
