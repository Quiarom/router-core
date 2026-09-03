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

Plus **four binaries** (commands you can actually run):

- `router-core` — the runtime CLI. Authenticates against the verified
  TP-Link adapter and exposes the read-only API on loopback.
- `router-core-learn` — the experimental probe. Authenticates against
  the physical router using five candidate recipes, captures
  sanitized evidence, and writes it to `fixtures/captured/`. This is
  what produced the Phase 2 evidence.
- `router-core-agent` — the reasoning layer. It consumes local
  observations and answers questions in deterministic or OpenRouter mode.
- `router-core-desktop` — the Fedora desktop shell. It starts the other
  services after the operator submits credentials and keeps the session
  in memory.

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
2026-08-30 and 2026-08-31 and produced sanitized evidence.

**Authentication** (verified 2026-08-30):

| Endpoint | Recipe verified | Fingerprint match |
|---|---|---|
| Login (root `/`) | `GET /` with `Authorization: Basic base64("admin:<plaintext>")` (plaintext password, NOT md5hex) | n/a |
| `GET /userRpm/StatusRpm.htm` (with session) | Same Basic Auth header | firmware `3.13.33 Build 130506 Rel.48660n` and hardware `WR841N v8 00000000` both matched exactly |

**Observation surface** (verified 2026-08-31 via `router-core-learn observe`):

| Capability | Path | State on physical lab unit | Result |
|---|---|---|---|
| `status` | `/userRpm/StatusRpm.htm` | 200 OK with dashboard | **verified** via session-token URL prefix |
| `wireless_security` | `/userRpm/WlanSecurityRpm.htm` | 200 OK with body | **verified** via session-token URL prefix |
| `clients` | `/userRpm/AssignedIpAddrListRpm.htm` | 200 OK with DHCPDynList | **verified** via Basic Auth header only |
| `wps` | `/userRpm/WpsRpm.htm` | HTTP 501 | **unsupported** (endpoint not present on this firmware build) |
| `dmz` | `/userRpm/DMZRpm.htm` | 200 OK with body | **verified** via session-token URL prefix |
| `upnp` | `/userRpm/UpnpRpm.htm` | HTTP 501 | **unsupported** (endpoint not present) |
| `remote_management` | `/userRpm/AccessCtrlRpm.htm` | HTTP 501 | **unsupported** (endpoint not present) |
| `forwarding` | `/userRpm/VirtualServerRpm.htm` | 200 OK with body | **verified** via session-token URL prefix |

**Key finding:** the WR841N v8.4 firmware has **two auth modes**:

- **Basic Auth header** for some endpoints (`/userRpm/AssignedIpAddrListRpm.htm`).
- **Session token URL prefix** (`/userRpm/<16-char-token>/<path>`) for the rest. The Basic Auth header is required in both cases — but the URL prefix is also required for protected endpoints.

The probe's `authedGetWithToken` implements the fallback strategy:
first try with header only, then with token-URL prefix on 68-byte "no
authority" response. This two-step approach is what makes 5 of 8
capabilities reachable.

The full per-capability matrix is persisted at
`fixtures/captured/tplink-wr841n-v8/capability-evidence.json` after
each `observe` run.

The runtime registry marks `status`, DHCP clients, and wireless security
as verified endpoints. The wireless parser still returns unavailable, and
DMZ, WPS, UPnP, Remote Management, and Forwarding remain blocked until their
recipes and parsers are verified. The desktop therefore reports unknown
families and unverified capabilities without substituting fixture data.

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
- **No universal adapter.** The desktop flow currently accepts the
  verified TP-Link WR841N v8.4 adapter. Other router families remain
  unsupported until their authentication and observation recipes are
  captured and tested.
- **No mutation surface.** The adapter cannot change settings, reboot,
  reset, or write anything to the router. This is enforced by the
  architecture test in `internal/architecture_test.go` (which would
  fail if any mutating endpoint path appeared in `internal/`).

## What you can do right now

If you have a WR841N v8.4 on your local network with the stock
factory credentials, capture a session with `router-core-learn`:

```bash
go build -o ./bin/router-core-learn ./cmd/router-core-learn
echo '<your-password>' | ./bin/router-core-learn learn \
    --host <your-router-ip> \
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

For Fedora, build the desktop RPM with:

```bash
make desktop-build
```

The package is written under
`frontend/src-tauri/target/release/bundle/rpm/`.

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
- **Phase 4 — Local Service:** done. `router-core serve` keeps the
  authenticated session in memory and exposes a typed HTTP API on
  `127.0.0.1`.
- **Phase 5 — M3 Agent Loop:** done. The agent calls Phase 4's API,
  records observation steps, and runs in deterministic or OpenRouter
  mode.
- **Phase 6 — One Safe Write:** not started. Any mutation would require
  a new safety decision and explicit human approval.
- **Phase 7 — UI / Demo:** in progress. The Fedora Tauri desktop app
  provides the connection flow and dashboard; broader adapter support
  remains pending.

The **biggest unverified surface** is WPS, DMZ, UPnP, Remote Management,
and Forwarding. Each one needs its own physical capture pass and parser
verification. Until then, the `Verified: false` flag in `endpoints.go`
blocks the request. This is intentional: the runtime adapter must never
guess a recipe; it must only use recipes that have been physically verified.

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