# Changelog

All notable changes to `router-core` are documented here. The
format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
and the project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- `router-core-agent` binary: read-only network auditor over
  the `serve` HTTP surface. Gathers device identity + capability
  matrix, sends the user's question to MiniMax M3 via OpenRouter,
  and runs a tool-call loop against `/v0/security/*`. Includes a
  deterministic stub for offline demos. See `docs/PHASE5.md`.
- `/v0/capabilities` endpoint: live matrix of all capabilities
  with one of the four documented states (`verified`,
  `absent`, `unsupported_or_unverified`, `unavailable`).
- Per-capability security endpoints (`/v0/security/wps|dmz|
  upnp|remote-management|forwarding`): each capability is
  fetched and parsed independently; a failure in one does not
  poison the others.
- `/v0/security/wireless` endpoint: returns `503 unavailable`
  with a clear reason (the parser is pending).
- `docs/FRONTEND_CONTRACT.md` and `fixtures/frontend-mocks/`:
  stable JSON contract and mock fixtures for the third-party
  frontend.
- `HACKATHON_FAQ.md`: MiniMax-Week judging context.
- `NOTICE`: third-party attribution (Devin AI baseline, prior
  art).
- `AGENTS.md`: conventions and hard rules for AI coding agents.
- `CODE_OF_CONDUCT.md` (Contributor Covenant v2.1).
- `CONTRIBUTING.md`, `SECURITY.md`, `CHANGELOG.md`, `CODEOWNERS`,
  `.editorconfig`: OSS team-contribution files.
- `.github/ISSUE_TEMPLATE/` (bug report, feature request) and
  `.github/PULL_REQUEST_TEMPLATE.md`.
- `humanizer` skill bundled at `.claude/skills/humanizer/`.

### Changed

- `cmd/router-core/serve.go` subcommands wired: `probe`,
  `inspect`, `serve` (`serve` runs as a loopback-only HTTP
  service).
- `router-core probe` prompts for the admin password on
  `/dev/tty` and calls `Login` before `Identify` when running
  live. Fixture mode is unchanged.
- `internal/adapters/tplinkwr841v8` `Adapter.Login` uses the
  recipe verified live 2026-08-31 against the lab unit at
  `192.168.1.1`: `GET /` with `Authorization: Basic <base64(
  admin:plaintext)>` (NOT md5hex).
- `Adapter.authedFetchWithFallback` and `Adapter.Identify` use
  a `Referer: http://<host>/` header on `/userRpm/<path>`
  requests so the v8.4 firmware returns the real dashboard
  body instead of the 68-byte "no authority" rejection.
- `Adapter.Identify` reads the authenticated `Status` dashboard
  to extract firmware and hardware fingerprints via regex
  (works across firmware builds 3.13.33 and 3.15.9).
- `cmd/router-core/main.go` `probe` subcommand no longer
  reports "missing capture" against a live router; it
  authenticates first.
- `internal/architecture_test.go`: scope excludes
  `cmd/router-core-agent/` (the agent's POST is to OpenRouter,
  not to the router).

### Fixed

- `authedFetchWithFallback` was constructing a double-slash URL
  (`http://host//userRpm/<path>`) that the firmware returned
  HTTP 501 for. Trimmed the trailing slash before appending
  the path.
- Hardcoded `statusPara` indices in `ParseIdentity` were
  firmware-build-specific (3.13.33 vs 3.15.9). Replaced with
  regex patterns.
- `Security()` short-circuited on the first failing capability.
  Refactored to `SecurityCapability(ctx, name)` so each
  capability is independent.

### Known limitations

- `/v0/security/wireless` returns `503 unavailable` (parser
  pending). The endpoint is reachable on v8.4 (verified
  2026-08-31).
- The session-token URL-prefix path for /userRpm/<path>
  endpoints is not exercised in production. The plumbing
  exists in the adapter (`authedFetchWithFallback`,
  `SessionTokenForTest`) but the production path for
  fetching the token from the firmware is not written.

## [0.1.0] - 2026-08-31

### Added

- Initial public release.
- Three-layer architecture: `internal/domain/`,
  `internal/transport/`, `internal/adapters/tplinkwr841v8/`.
- Two binaries: `router-core` (probe, inspect, serve) and
  `router-core-learn` (5-recipe auth probe + observation
  capture).
- Fixture-backed adapter for testing without hardware.
- CI: `gofmt`, `vet`, `build`, `go test -race`.
- MIT license.
