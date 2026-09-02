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
- `cmd/router-core-agent --serve 127.0.0.1:8585`: HTTP server
  mode for the agent, exposing `POST /v0/chat` and
  `GET /healthz` with loopback-only CORS so the frontend can
  drive the agent over the local HTTP surface.
- `cmd/router-core/serve`: `/v0/clients` route wired (the
  handler existed on main but the route registration was
  missing — a real bug; the frontend dev caught and fixed it).
- React 19 + Vite 8 + Tailwind 4 dashboard under `frontend/`,
  contributed by the third-party frontend dev. Hits every
  documented endpoint of the contract and respects the four
  capability states.

### In progress on `develop`

The next items land on the `develop` branch and merge to
`main` as a single release once the full MiniMax integration
is verified. The CHANGELOG will be split at that point.

- **Session-token path: not needed for v8.4.** The v8.4 firmware
  at `192.168.1.1` (3.15.9 Build 140724) does not expose a login
  endpoint: `/userRpm/LoginRpm.htm?Save=Save` and the other
  PA-1/PA-2 paths return HTTP 501 "File not found". The actual
  recipe is HTTP Basic Auth with the plaintext password on
  every request, and the v8.4 firmware expects every caller
  to send it. The `auth-evidence.json` capture that suggested a
  session token was from an older firmware build and does not
  match the current state. No production path is needed.
- **Wireless-security parser.** The endpoint is reachable
  (verified 2026-08-31) but the runtime returns 503 with
  reason "parser is pending". Wire the parser against a
  sanitized capture from the live lab unit.
- **`inspect` end-to-end.** Make sure the per-capability
  independent dispatch surfaces in the inspect output too.
- **Agent tests.** `cmd/router-core-agent/` has 0 test files.
  Add tests for the tool loop (deterministic stub mode) and
  the new HTTP server (`/v0/chat`, `/healthz`, CORS).
- **Live OpenRouter run.** Requires the operator's
  `OPENROUTER_API_KEY`. Document the trace.
- **Frontend end-to-end test.** Prove the round trip:
  frontend → `/v0/chat` → agent → `/v0/security/*` → frontend.
  The frontend dev's work is on `main`; the test belongs on
  `develop` or as a separate CI job.
- **Frontend tests.** Vitest not configured. Add a minimal
  smoke test for the contract integration.

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
- `Security()` aggregates per-capability observations
  independently. A failure in one capability does not poison
  the others.
- The `default` branch on GitHub is `develop` during the
  MiniMax-Week window; `main` is the released state.

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
- `/v0/clients` route registration was missing on main (the
  handler existed but was never wired to the mux). Caught and
  fixed by the frontend dev.

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
