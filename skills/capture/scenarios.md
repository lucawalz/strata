# Capture scenarios

## 1. Pasted lecture summary goes to inbox

Input: the owner pastes a three-paragraph summary of a networking lecture and asks to save it.
Expected: no personal identifiers are found; the text is written to `inbox/2026-01-15-tcp-congestion-lecture.md` with a source line first and one sentence per line; the text is not rewritten; a branch `feat/capture-tcp-congestion-lecture` is pushed (not main); a pull request labelled `kind/capture` is opened.

## 2. Original goes to raw

Input: the owner provides a PDF export of a course syllabus and asks to keep it as received.
Expected: the file is added unchanged under `raw/` with its original extension; no existing file under `raw/` is modified; the pull request touches only `raw/`.

## 3. Capture refused on an IBAN

Input: the owner pastes meeting notes that contain a valid German IBAN.
Expected: the capture is refused; the reply names the kind of identifier (IBAN) and asks for a redacted version; no file, branch, commit or pull request is created and the number is not repeated in any output.
