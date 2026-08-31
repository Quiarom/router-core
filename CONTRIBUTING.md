# Contributing to router-core

Thanks for your interest in `router-core`. This project follows
a few simple rules to keep the safety invariants intact and the
review surface small.

## Before you open an issue or pull request

1. Read [`README.md`](README.md), [`NOTICE`](NOTICE),
   [`docs/STATUS.md`](docs/STATUS.md), and the relevant ADR in
   [`docs/adr/`](docs/adr/).
2. Read [`AGENTS.md`](AGENTS.md) if you are an AI coding agent
   (or working with one).
3. Search the existing issues and pull requests to avoid
   duplication.

## Development setup

```bash
git clone https://github.com/Quiarom/router-core.git
cd router-core
go version         # 1.25 or newer
go build ./...
go test ./... -race
```

The Makefile wraps these:

```bash
make build
make test
make vet
make fmt
```

## Commit messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <short summary>

<body explaining the why>
```

Types: `feat`, `fix`, `docs`, `test`, `refactor`, `chore`,
`revert`. Scope is the package or area (e.g. `tplink`,
`serve`, `transport`, `docs`).

## Pull requests

Before opening a pull request:

- [ ] `gofmt -l .` is empty.
- [ ] `go vet ./...` is silent.
- [ ] `go test ./... -race` is green on all eight packages.
- [ ] New code has tests.
- [ ] New behavior is reflected in `docs/STATUS.md` if a
  capability flipped Verified.
- [ ] The diff is reviewable in one pass. If the change is
  > 400 lines, split it.

Use the [pull request template](.github/PULL_REQUEST_TEMPLATE.md).

## Issues

- **Bugs:** use the bug report template
  ([.github/ISSUE_TEMPLATE/bug_report.md](.github/ISSUE_TEMPLATE/bug_report.md)).
- **Features:** use the feature request template
  ([.github/ISSUE_TEMPLATE/feature_request.md](.github/ISSUE_TEMPLATE/feature_request.md)).
- **Security vulnerabilities:** open a **private security
  advisory** on GitHub rather than a public issue.

## Architecture decisions

If your change is a non-trivial design decision (new HTTP
method, new outbound target, new write path, new vendor
adapter, removal of an existing safety check), open an ADR in
`docs/adr/` **before** opening the pull request. ADRs are
reviewed for safety implications and license compatibility.

## License

By contributing, you certify that you have the right to submit
the work under the project's MIT license (see [LICENSE](LICENSE)).
Do not import code from GPL, AGPL, or unlicensed sources.

## Code of conduct

All participants are bound by
[`CODE_OF_CONDUCT.md`](CODE_OF_CONDUCT.md).
