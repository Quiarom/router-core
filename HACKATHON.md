# router-core — GMI Cloud × MiniMax Week submission

## Track

**Reasoning.** router-core is a local-first, read-only observation
layer for legacy consumer routers. The current target is the
TP-Link TL-WR841N/ND v8.4 stock dashboard, but the architecture is
general enough to grow to other vendor families. MiniMax models are
used to interpret the structured facts the adapter emits and to
answer questions like "is my network exposed?" or "who is connected?"
in plain language.

## MiniMax models used

- **MiniMax M3 (coding, 1M ctx)** — primary reasoning model for the
  conversational interface.
- **MiniMax M2.7 (high-speed)** — fallback / low-latency model.

Both are accessed through GMI Cloud's inference API via OpenRouter
as the routing gateway, per the MiniMax-Week rules (the project
remains original work created during the 14-day campaign window).

## What this project does

router-core talks to a real, ten-year-old consumer router that the
stock web UI has long since rendered unmaintainable. Instead of
asking the user to click through menus, decode legacy JavaScript
arrays, and remember whether WPS is on, the user asks in English or
Spanish. The system turns each question into a structured request
against the read-only HTTP surface, parses the firmware's response
deterministically, and presents the result with provenance.

Three layers, all read-only:

1. `internal/domain/` — vendor-neutral types with first-class
   `Unknown` (never silently `false`), `Untrusted` strings, and
   explicit provenance.
2. `internal/transport/` — guarded HTTP client: loopback/RFC1918
   only, GET only, 2 MiB body cap, no cross-host redirects, no
   method other than what the caller declared. This is the hard
   safety boundary.
3. `internal/adapters/tplinkwr841v8/` — vendor-specific code for
   the WR841N v8.4. Parsers for the `var name = new Array(...)`
   JavaScript blocks the firmware emits in its HTML.

Two binaries:

- `router-core` — runtime CLI with `probe` and `inspect` against
  synthetic or live fixtures, and a new `serve` subcommand that
  exposes a typed HTTP API on the loopback interface for the
  MiniMax-driven frontend.
- `router-core-learn` — experimental probe that authenticates
  against the physical lab unit using five candidate recipes,
  captures sanitized evidence, and writes it to
  `fixtures/captured/`. This is what produced the verified
  authentication recipe (ADR 0005) and the per-capability
  observation matrix (2026-08-31).

## How the MiniMax models fit

The runtime model receives:

- The set of verified read-only capabilities (`/v0/device`,
  `/v0/status`, `/v0/clients`, `/v0/security/*`) with their
  current evidence.
- The user's natural-language question.
- A short prompt describing the safety model
  ("GET only, loopback, no mutations, `Unknown` is first-class").

The model emits a structured request (which capability, which
field, what filter) or a refusal when the question is outside the
read-only surface. The Go side executes the request, attaches
provenance, and returns the typed result for the model to phrase
in natural language.

`Unknown` and `Untrusted` types are passed through to the model as
explicit values, not absorbed into booleans, so the model can reason
honestly about absence vs. false vs. unsupported.

## What's verified against the physical lab unit

| Capability | Endpoint | State | Evidence |
| --- | --- | --- | --- |
| Authentication | `GET /` with `Authorization: Basic <base64(admin:plaintext)>` | verified 2026-08-30 | `fixtures/captured/tplink-wr841n-v8/auth-evidence.json` |
| Status | `/userRpm/StatusRpm.htm` | verified 2026-08-31 | `fixtures/captured/tplink-wr841n-v8/status.html` |
| Wireless security | `/userRpm/WlanSecurityRpm.htm` | verified 2026-08-31 | `fixtures/captured/tplink-wr841n-v8/wireless_security.html` |
| Clients | `/userRpm/AssignedIpAddrListRpm.htm` | verified 2026-08-31 | `fixtures/captured/tplink-wr841n-v8/clients.html` |
| DMZ | `/userRpm/DMZRpm.htm` | verified 2026-08-31 | `fixtures/captured/tplink-wr841n-v8/dmz.html` |
| Forwarding | `/userRpm/VirtualServerRpm.htm` | verified 2026-08-31 | `fixtures/captured/tplink-wr841n-v8/forwarding.html` |
| WPS | `/userRpm/WpsRpm.htm` | unsupported (HTTP 501) | `fixtures/captured/tplink-wr841n-v8/wps.html` |
| UPnP | `/userRpm/UpnpRpm.htm` | unsupported (HTTP 501) | `fixtures/captured/tplink-wr841n-v8/upnp.html` |
| Remote management | `/userRpm/AccessCtrlRpm.htm` | unsupported (HTTP 501) | `fixtures/captured/tplink-wr841n-v8/remote_management.html` |

## How to run

```sh
go build -o router-core ./cmd/router-core
go build -o router-core-learn ./cmd/router-core-learn

# Standalone: probe a synthetic fixture
./router-core probe --fixtures fixtures/synthetic/tplink-wr841n-v8

# Standalone: probe the physical lab unit
./router-core probe --host 192.168.0.1

# Serve: typed HTTP API on loopback for the MiniMax-driven frontend
./router-core serve --host 192.168.0.1 --addr 127.0.0.1:8484
# (then prompts for the admin password)
curl http://127.0.0.1:8484/v0/status
```

## Safety invariants (enforced in code)

- GET only. The transport layer's `Client.Dispatch` rejects every
  other method.
- Loopback or RFC1918 only. Public IPs and hostnames are refused at
  every layer.
- 2 MiB response body cap.
- No cross-host redirects.
- The session lives in process memory. The password is read from
  `/dev/tty` with echo disabled, held only for the duration of the
  `serve` process, and zeroed on exit.
- Mutations are unrepresentable: there is no `CapMutate` in the
  type system. Any future mutation path requires a new ADR,
  capability constant, and explicit operator approval.

## Repo layout

- `cmd/router-core/` — runtime CLI (`probe`, `inspect`, `serve`).
- `cmd/router-core-learn/` — experimental probe and observation
  capture.
- `internal/domain/` — vendor-neutral types and contracts.
- `internal/transport/` — guarded HTTP client.
- `internal/adapters/tplinkwr841v8/` — vendor-specific code.
- `internal/adapters/fixture/` — fixture-backed adapter for
  testing without hardware.
- `fixtures/synthetic/` — synthetic dashboard pages for replay.
- `fixtures/captured/` — sanitized evidence from the physical
  lab unit.
- `docs/adr/` — architecture decision records (0001, 0002, 0003,
  0005).
- `docs/STATUS.md` — non-engineer-friendly project status.

## License

MIT. See `LICENSE`.

## Demo

A 3-minute demo video is published with the submission per the
MiniMax-Week rules.
