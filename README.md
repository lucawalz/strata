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

Usage is documented once the checks are implemented.

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
