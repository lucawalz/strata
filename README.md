# strata

[![ci](https://github.com/lucawalz/strata/actions/workflows/ci.yml/badge.svg)](https://github.com/lucawalz/strata/actions/workflows/ci.yml)
[![release](https://img.shields.io/github/v/release/lucawalz/strata)](https://github.com/lucawalz/strata/releases)
[![license: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
![Go](https://img.shields.io/badge/Go-1.27-00ADD8?logo=go&logoColor=white)

A self-hosted knowledge base on git that AI agents maintain through pull requests.

```mermaid
flowchart LR
  claude[Claude Code] -->|pull request| notes[notes repository]
  notes -->|calls| check[strata check workflow]
  check -->|findings| notes
  owner[owner] -->|signed governance commits, merge| notes
```

## Description

strata turns a private git repository of Markdown notes into a knowledge base that agents keep organised and every Claude surface can search. Every change arrives as a pull request that has to pass checks, and governance files change only through commits signed by the owner.

This repository holds the note format, the `strata` command that checks a notes repository in CI, and the reusable workflows a notes repository calls.

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

## Roadmap

1. Foundation: note format, checks, governance, local search (in progress)
2. First agent identity and `@strata` commands
3. Connector for every Claude surface
4. Earned autonomy with canary-tested reviews
5. Paperless ingest
6. Hardened self-hosted runners and vault import
7. First public release with self-hosting documentation

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
