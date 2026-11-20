# strata

Checks the structure of a Markdown note vault in CI.

## Description

strata is a Go command line tool that reads a vault of Markdown notes and reports structural problems as findings: note size outside the configured caps, too many tags, and changes to governance paths that lack a signed commit from an allowed signer. Size is measured in Unicode code points of the note body after the frontmatter.

Findings have a level of error or warning. The command exits 0 when there are no error findings, 1 when there is at least one, and 2 on a usage or internal error. Settings live in `strata.yml`; every key has a default.

## Installation

Not released yet. Build from source with Go 1.27 or later:

```
go install github.com/lucawalz/strata/cmd/strata@latest
```

## Usage

Check the notes tree of a repository (defaults to the current directory):

```
strata check tree --root DIR --format text
```

Check a pull request range for add-only changes under `raw/`, append-only edits, governance isolation and commit signatures:

```
strata check diff --base SHA --head SHA --root DIR
```

Audit a pushed range for governance changes without an owner signature:

```
strata audit --before SHA --after SHA --root DIR
```

Measure the characters-per-token ratio of sample notes and print proposed size caps; the key is read from `ANTHROPIC_API_KEY`:

```
strata calibrate --model MODEL DIR
```

Copy the embedded capture and curate skills into a skills directory; directories with other names are left alone:

```
strata skills sync --to .claude/skills
```

`--format github` prints GitHub workflow commands instead of text lines.

## Support

Report vulnerabilities privately as described in [SECURITY.md](SECURITY.md). For other questions, open an issue.

## Contributing

Contributions follow the shared guidelines in the [lucawalz/.github](https://github.com/lucawalz/.github) repository. Run the local checks before opening a pull request:

```
just check
```

## Authors and acknowledgment

Maintained by Luca Walz.

## License

[MIT](LICENSE)

## Project status

In development.
