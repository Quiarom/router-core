# Project Status — Hackathon Submission

## TL;DR for non-engineers

We're building a tool that lets you talk to your old WiFi router in plain
language. Instead of opening a web page, clicking through menus, and
trying to remember whether "WPS" is on or off, you ask the question in
English (or Spanish): *"Is my network exposed?" "Who is connected?"*
The system figures out what your router is doing and tells you — without
you ever needing to look at HTML pages, cookies, or login forms.

The router we're using is a TP-Link TL-WR841N v8.4 that was manufactured
in 2013. The web interface on it is ugly, hard to use, and missing most
of what a normal user needs. We're teaching our software to talk to it
the way the original firmware wants to be talked to, then letting an
AI model reason about what the router is doing.

## What's actually been built

This is a **read-only** tool. It cannot change the router's settings,
reboot it, or do anything destructive. It can only look.

The codebase is Go (the language), about 4,200 lines of Go code plus
2,300 lines of design documents and ADRs (architecture decision records).

There are **three layers**:

1. **Domain types** (`internal/domain/`) — the clean, vendor-neutral
   data model: `DeviceInfo`, `RouterStatus`, `Client`, `SecurityState`.
   "Unknown" is a first-class value (never silently turned into
   "false"). All human-readable strings are wrapped as `Untrusted`
   with a marker so the AI layer knows they came from the network.

2. **Transport** (`internal/transport/`) — a tightly-restricted HTTP
   client that only talks to loopback / RFC1918 addresses, only does
   GET, never lets the body exceed 2 MiB, and never follows cross-host
   redirects. This is the hard safety boundary.

3. **Adapter** (`internal/adapters/tplinkwr841v8/`) — the vendor-
   specific code that knows how to talk to a WR841N. Parsers for the
   weird `var name = new Array(...)` JavaScript blocks the firmware
   emits in its HTML.

Plus **two binaries** (commands you can actually run):

- `router-core` — the runtime CLI. Tells you what the router is
  doing. Currently returns `ErrCaptureMissing` because the adapter
  hasn't been wired to live authentication yet.
- `router-core-learn` — the experimental probe. Authenticates against
  the physical router using five candidate recipes, captures
  sanitized evidence, and writes it to `fixtures/captured/`. This is
  what produced the Phase 2 evidence.

## Hardcoded values (no secrets stored)

These are hardcoded in the source. They are **defaults**, not secrets:

- **`"admin"`** — the TP-Link default username, passed to the probe's
  five candidate recipes. The probe is allowed to try this username
  because it is the public TP-Link factory default. The actual
  password is **never hardcoded**: the operator types it interactively
  in the terminal (via `/dev/tty` with echo disabled) or pipes it via
  `--password-stdin`. The password never touches the source code, the
  build output, or the model's context.

- **`192.168.0.1`** — fallback default host in `cmd/router-core/main.go`
  when no `--host` flag is given. Always overridden by the operator
  via the flag (e.g. `--host 192.168.1.1`).

- **`192.168.1.1`** — only used in **docs and example commands** as
  the canonical example of the lab unit's IP. No code uses this
  literal at runtime; it is always read from `--host`.

- **`3.13.33 Build 130506 Rel.48660n`** — the firmware string we
  expect to see when parsing the Status page. Used in probe error
  messages and tests to verify the parser returned the right shape.

- **`WR841N v8 00000000`** — same purpose: the expected hardware
  string for the lab unit, used in tests and probe diagnostics.

- **TL-WR841N/ND** — the model name in `ModelName`. Hardcoded
  because the adapter is vendor-and-model-specific. Future routers
  would have their own adapter package.

- **`hunter2`** — appears ONLY in test files (`*_test.go`) as the
  example password for httptest-sidecar mock servers. Never in
  production code paths.

- **`admin/admin`** — appears in docs/PRIOR_ART_PROTOCOL.md as the
  TP-Link factory default documented by both prior-art
  implementations. Used in the probe's candidate recipes as the
  username input; the operator types the password at runtime.

## What is verified against the physical lab unit

The probe executed against the WR841N v8.4 at `192.168.1.1` on
2026-08-30 and produced sanitized evidence:

| Endpoint | Recipe verified | Fingerprint match |
|---|---|---|
| Login (root `/`) | `GET /` with `Authorization: Basic base64("admin:<plaintext>")` (plaintext password, NOT md5hex) | n/a |
| `GET /userRpm/StatusRpm.htm` (with session) | Same Basic Auth header | firmware `3.13.33 Build 130506 Rel.48660n` and hardware `WR841N v8 00000000` both matched exactly |

All other endpoints (DHCP, WPS, DMZ, UPnP, Remote Management,
Forwarding) remain **unverified** — `Verified: false` in
`endpoints.go`. They require their own physical capture pass before
their `Verified` flag can be flipped.

The verification is encoded in three layers:

1. `internal/adapters/tplinkwr841v8/endpoints.go` — `OpStatus.Verified = true`.
2. `internal/adapters/tplinkwr841v8/login_test.go` — five tests
   that exercise `Adapter.Login` and `Adapter.Status` against an
   httptest sidecar that emulates the WR841N behavior.
3. `docs/adr/0005-verified-wr841n-auth-recipe.md` — the human-
   readable evidence trail: what recipe was verified, when, why,
   and what remains unverified.

## What is NOT in this repo

- **No real router credentials.** None. The probe asks the operator
  for the password at runtime and never persists it.
- **No captured HTTP traces from a real session.** Sanitized evidence
  exists (status body, login page, headers) but no HAR, no cookies,
  no session tokens.
- **No live router integration tests** beyond an opt-in `TestLiveOptIn`
  that runs only when `ROUTER_LIVE_TESTS=1` is set. Default behavior
  is fixture-based or httptest-mock-based.
- **No AI model integration.** The reasoning layer that would consume
  the `domain` types and propose remediations is Phase 5 work.
- **No mutation surface.** The adapter cannot change settings, reboot,
  reset, or write anything to the router. This is enforced by the
  architecture test in `internal/architecture_test.go` (which would
  fail if any mutating endpoint path appeared in `internal/`).

## What you can do right now

If you have a WR841N v8.4 at `192.168.1.1` with `admin/admin` credentials:

```bash
cd /home/quiarom/Documents/Hackathon/router-core
go build -o /tmp/router-core-learn ./cmd/router-core-learn
echo '<your-password>' | /tmp/router-core-learn learn \
    --host 192.168.1.1 \
    --password-stdin \
    --timeout 5s
```

The probe tries five recipes in order. The fifth (Basic Auth with
plaintext password via the `Authorization` header) is the one that
works on the lab unit. On success, sanitized evidence is written to
`fixtures/captured/tplink-wr841n-v8/`.

If you don't have a WR841N at hand, the unit tests cover the same
scenarios with httptest mocks:

```bash
go test ./... -count=1 -race
```

Eight packages, all green.

## Where we're going

The hackathon scope is "Reasoning" — we want an AI agent to be able
to ask the router "what's your current state?" and get a typed,
sanitized answer back. Phase-by-phase:

- **Phase 0 — Bootstrap:** done (Devin's overnight pass).
- **Phase 1 — Physical Lab:** done (the lab router is online and
  observable).
- **Phase 2 — Real Capture:** **done** (Phase 2B, the candidate-probe
  path, succeeded against the physical unit; five candidates tested;
  one verified recipe).
- **Phase 3 — Verified Driver:** **done** (the minimal change: the
  adapter now uses the verified recipe; `OpStatus` is `Verified: true`;
  ADR 0005 documents the evidence).
- **Phase 4 — Local Service:** not started. Would add a
  `router-core serve` binary that keeps the authenticated session
  alive in memory and exposes a typed HTTP API on `127.0.0.1`. This
  is the seam where the AI agent plugs in.
- **Phase 5 — M3 Agent Loop:** not started. The AI agent (the
  reasoning layer that the hackathon judges on) calls Phase 4's
  HTTP API to ask "what's the router doing?" and reasons about the
  answer.
- **Phase 6 — One Safe Write:** not started. Would add the first
  mutation (e.g. disabling UPnP) gated by policy + human approval.
- **Phase 7 — UI / Demo:** not started. Final wrap-up.

The **biggest unverified surface** is the rest of the endpoints
(DHCP, WPS, DMZ, UPnP, Remote Management, Forwarding). Each one needs
its own physical capture pass. Until they are captured and verified,
the adapter cannot read them at runtime — the `Verified: false` flag
in `endpoints.go` blocks the request. This is intentional: the runtime
adapter must never guess a recipe; it must only ever use recipes that
have been physically verified.

## License and attribution

This codebase is MIT-licensed.

Two public prior-art implementations were studied as **research** but
no code was copied:

- `mkubicek/tpylink` — no declared license.
- `maesoser/tplink_exporter` — GPL-3.0.

See `docs/PRIOR_ART_PROTOCOL.md` for the full comparison and per-
observation attribution.

The verified recipe (Basic Auth with plaintext password, no md5hex)
**diverges** from both prior-art implementations, which both assumed
md5hex. This divergence was discovered by physical capture, not by
copying.