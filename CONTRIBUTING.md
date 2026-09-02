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

## Branching model

- `main` is the default branch. It always points to the latest
  released state (or the current tip of the alpha during the
  MiniMax-Week window).
- All work happens on short-lived branches off `main`. Use a
  Conventional Commits-style prefix in the branch name so the
  intent is obvious from `git branch -a`:
  - `feat/<short-name>` — new capability.
  - `fix/<short-name>` — bug fix.
  - `test/<short-name>` — test-only change.
  - `docs/<short-name>` — documentation only.
  - `refactor/<short-name>` — no behavior change.
  - `chore/<short-name>` — build, CI, or repo hygiene.
- Keep the branch focused. One concern per branch. If the change
  is > 400 lines, split it; the project follows conventional
  small-PR practice.
- Rebase onto `main` before opening the PR. Do not merge
  `main` into your feature branch.

## Commit messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <short summary>

<body explaining the why>
```

Types: `feat`, `fix`, `docs`, `test`, `refactor`, `chore`,
`revert`. Scope is the package or area (e.g. `tplink`,
`serve`, `transport`, `docs`).

Rules:

- Subject line: imperative mood, ≤ 72 chars, no trailing period.
- Body: explain the **why**, not the **what**. The diff already
  shows the what.
- Footer (optional): `Closes #<issue>`, `Refs ADR-XXXX`, or
  `BREAKING CHANGE: <note>`.
- One logical change per commit. Atomic commits make `git
  revert` and `git bisect` cheap.

## Pull requests

Before opening a pull request:

- [ ] `gofmt -l .` is empty.
- [ ] `go vet ./...` is silent.
- [ ] `go build ./...` succeeds.
- [ ] `go test ./... -race` is green on all eight packages.
- [ ] New code has tests. A bug fix without a regression test
      is not merged.
- [ ] New behavior is reflected in `docs/STATUS.md` if a
      capability flipped Verified.
- [ ] The diff is reviewable in one pass. If the change is
      > 400 lines, split it.
- [ ] The PR is opened against `main`, not against a personal
      branch.
- [ ] The PR description answers the three questions a reviewer
      asks first: "what does this change?", "why now?", and
      "what did I miss?"

Use the [pull request template](.github/PULL_REQUEST_TEMPLATE.md).
The CI workflow must be green before review starts.

## Issues

- **Bugs:** use the bug report template
  ([.github/ISSUE_TEMPLATE/bug_report.md](.github/ISSUE_TEMPLATE/bug_report.md)).
  Include the `router-core` version, the Go version, the OS,
  the router model and firmware, and the exact command that
  reproduces the issue.
- **Features:** use the feature request template
  ([.github/ISSUE_TEMPLATE/feature_request.md](.github/ISSUE_TEMPLATE/feature_request.md)).
  Explain the problem first, then the proposed solution.

## Architecture decisions

If your change is a non-trivial design decision (new HTTP
method, new outbound target, new write path, new vendor
adapter, removal of an existing safety check), open an ADR in
`docs/adr/` **before** opening the pull request. The ADR is
reviewed for safety implications and license compatibility.

The ADR file name is `NNNN-<short-kebab-title>.md` and the status
starts at `proposed` and moves to `accepted` or `rejected`.

## License

By contributing, you certify that you have the right to submit
the work under the project's MIT license (see [LICENSE](LICENSE)).
Do not import code from GPL, AGPL, or unlicensed sources.

## Code of conduct

All participants are bound by
[`CODE_OF_CONDUCT.md`](CODE_OF_CONDUCT.md).

## Security

If you find a security vulnerability, do not open a public
issue. See [`SECURITY.md`](SECURITY.md) for the private
disclosure workflow.
